# Day 1: NFS 协议与服务器配置

## 🎯 学习目标
- **技能目标**: 深入理解 NFS 协议 (v3, v4) 的工作原理、核心组件和版本差异。
- **实践目标**: 能够独立完成 NFS 服务器的安装、配置 (`/etc/exports`) 和管理，并在客户端上成功挂载和使用共享目录。
- **Go编程启蒙**: 编写一个简单的 Go 程序来解析 `/etc/exports` 文件，为后续开发监控工具打下基础。
- **成果产出**: 一个正常工作的 NFS 服务端和客户端环境，一份详细的 NFS v3 vs v4 对比笔记，一个 Go 语言的 exports 文件解析器。

## 📚 理论基础 (40%)

### 1. NFS 架构深度解析
NFS (Network File System) 是一种分布式文件系统协议，允许网络中的计算机之间通过 TCP/IP 网络共享文件和目录。其核心是客户端-服务器模型。

- **核心组件**:
  - **NFS Server**: 托管物理文件系统并将其共享给网络的机器。
  - **NFS Client**: 通过网络访问服务器上共享文件系统的机器。
  - **RPC (Remote Procedure Call)**: NFS 的基石。客户端通过 RPC 调用服务器上的程序（如 `mountd`, `nfsd`）来请求文件操作，就像调用本地函数一样。`rpcbind` 服务负责将 RPC 程序号映射到具体的端口号。

### 2. NFS v3 vs v4: 关键区别
| 特性 | NFSv3 | NFSv4 |
| :--- | :--- | :--- |
| **状态** | 无状态 (Stateless) | 有状态 (Stateful) |
| **端口** | 使用多个端口 (rpcbind, mountd, nfsd, etc.) | **仅使用 TCP 端口 2049**，易于防火墙管理 |
| **文件锁定** | 锁管理是独立的网络锁管理器 (NLM) | 锁管理集成在协议内部，更可靠 |
| **安全性** | 基础的 `AUTH_SYS` (基于 UID/GID) | **集成 Kerberos (krb5, krb5i, krb5p)**，支持强认证和加密 |
| **性能** | 简单高效 | 引入复合过程 (Compound Procedures)，可将多个操作捆绑在一次请求中，减少网络往返 |
| **文件系统模型** | 服务器导出多个路径 | 服务器导出一个统一的伪文件系统 (`/`)，客户端在此根下挂载 |

**企业级选择**: 除非有特殊的老旧设备兼容需求，**NFSv4 是现代环境的首选**，因为它更安全、防火墙友好且功能更强大。

## 🛠️ 实践操作 (40%)

### 环境准备
- **服务器**: IP `192.168.1.100`
- **客户端**: IP `192.168.1.101`

### 1. NFS 服务器端配置 (在 `192.168.1.100` 上操作)

```bash
# 1. 安装 NFS 服务 (以 Ubuntu/Debian 为例)
# nfs-kernel-server 包含 nfsd 和 mountd 等核心服务
sudo apt update
sudo apt install -y nfs-kernel-server

# 2. 创建共享目录
# 使用 /srv 目录是存放服务数据的好习惯
sudo mkdir -p /srv/share/public
# 将目录所有者设置为 nobody:nogroup，这是一个安全的默认设置
sudo chown nobody:nogroup /srv/share/public
# 允许任何人读写执行
sudo chmod 777 /srv/share/public

# 3. 配置 exports 文件 (`/etc/exports`)
# 这是 NFS 的核心配置文件，定义了哪个目录共享给哪个客户端，以及用什么权限
# 语法: /path/to/share client(options)
# 示例：将 /srv/share/public 共享给 192.168.1.101，并赋予读写权限
sudo echo "/srv/share/public 192.168.1.101(rw,sync,no_subtree_check)" | sudo tee /etc/exports

# 选项解释:
# rw: 允许读写
# sync: (安全) 要求服务器在响应前将更改写入稳定存储。性能稍低但数据安全。
# async: (不安全) 服务器可先响应再写入。性能高但断电可能丢数据。
# no_subtree_check: 禁用子树检查，可以提高可靠性，但有轻微安全风险。

# 4. 使配置生效并重启服务
sudo exportfs -arv
# -a: 全部导出
# -r: 重新导出
# -v: 显示详细信息
sudo systemctl restart nfs-kernel-server
```

### 2. NFS 客户端配置 (在 `192.168.1.101` 上操作)

```bash
# 1. 安装客户端工具
# nfs-common 提供了挂载 NFS 所需的库和工具
sudo apt update
sudo apt install -y nfs-common

# 2. 创建挂载点
sudo mkdir -p /mnt/nfs/public

# 3. 手动挂载
# -t nfs: 指定文件系统类型为 nfs
sudo mount -t nfs 192.168.1.100:/srv/share/public /mnt/nfs/public

# 4. 验证挂载和读写
# 检查挂载情况，应该能看到 nfs4 类型
df -hT /mnt/nfs/public

# 在挂载点创建文件，测试写入权限
touch /mnt/nfs/public/test_from_client.txt

# 在服务器上验证文件是否已同步
# (在 192.168.1.100 上执行)
ls -l /srv/share/public
```

## 💻 Go 编程实现 (20%)

**目标**: 编写一个简单的 Go 程序，用于解析 `/etc/exports` 文件，并以结构化的形式打印出来。

**`nfs_parser.go`**
```go
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// ExportRule 定义了一个 NFS 导出规则
type ExportRule struct {
	Path    string
	Client  string
	Options []string
}

// parseExportsFile 解析 /etc/exports 文件
func parseExportsFile(filePath string) ([]ExportRule, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var rules []ExportRule
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳过注释和空行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue // 无效行
		}

		path := parts[0]
		clientAndOptions := parts[1]

		// 解析客户端和选项，例如 192.168.1.101(rw,sync)
		client := clientAndOptions
		var options []string
		openParen := strings.Index(clientAndOptions, "(")
		if openParen != -1 {
			client = clientAndOptions[:openParen]
			closeParen := strings.Index(clientAndOptions, ")")
			if closeParen > openParen {
				optionsStr := clientAndOptions[openParen+1 : closeParen]
				options = strings.Split(optionsStr, ",")
			}
		}

		rules = append(rules, ExportRule{
			Path:    path,
			Client:  client,
			Options: options,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return rules, nil
}

func main() {
	log.Println("Parsing NFS exports file...")
	rules, err := parseExportsFile("/etc/exports")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println("--- NFS Export Rules ---")
	for _, rule := range rules {
		fmt.Printf("Path: %s\n", rule.Path)
		fmt.Printf("  Client: %s\n", rule.Client)
		fmt.Printf("  Options: %s\n", strings.Join(rule.Options, ", "))
		fmt.Println("------------------------")
	}
}
```

## 🔍 故障排查与优化
- **连接超时**: 检查客户端与服务器之间的网络连通性（`ping`）和防火墙规则（确保服务器的 TCP 2049 端口对客户端开放）。
- **权限被拒绝 (Permission Denied)**: 检查 `/etc/exports` 中的客户端 IP 或网段是否正确，以及共享目录在服务器上的文件系统权限。
- **优化**: 对于大量小文件的读写，可以调整挂载选项中的 `rsize` 和 `wsize` (如 `rsize=32768,wsize=32768`) 来增大单次读写的数据块大小，以提升性能。

## 📝 实战项目
- **多客户端配置**: 修改 `/etc/exports`，允许一个新的客户端（如 `192.168.1.102`）以只读（`ro`）方式挂载同一个共享目录。
- **fstab 自动挂载**: 将 NFS 挂载条目添加到客户端的 `/etc/fstab` 文件中，实现开机自动挂载。
  ```
  # 语法: <server>:<remote_path> <local_path> <type> <options> 0 0
  192.168.1.100:/srv/share/public /mnt/nfs/public nfs defaults 0 0
  ```
  然后使用 `sudo mount -a` 测试配置是否正确。

## 🏠 课后作业
- **深入研究**: 详细研究 NFS 的 `root_squash` 和 `no_root_squash` 选项的含义、安全影响以及默认行为。
- **脚本编写**: 编写一个 Shell 脚本，该脚本接受一个服务器 IP 作为参数，并使用 `showmount -e <server_ip>` 命令来检查该服务器上有哪些可用的 NFS 共享目录，并格式化输出。
