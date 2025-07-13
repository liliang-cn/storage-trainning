# Day 2: iSCSI 协议与块存储实践

## 🎯 学习目标
- **技能目标**: 深入理解 iSCSI 协议的架构（Target, Initiator, LUN），掌握其作为网络块存储的核心思想。
- **实践目标**: 能够独立配置 iSCSI Target (服务端) 和 Initiator (客户端)，并成功将网络块设备挂载到本地文件系统。
- **安全目标**: 学会配置 iSCSI 的 CHAP 认证，保障存储网络的基本安全。
- **成果产出**: 一个正常工作的 iSCSI 存储环境，一份详细的 iSCSI vs FC (Fibre Channel) 对比分析笔记，一个能自动发现并登录 Target 的 Shell 脚本。

## 📚 理论基础 (40%)

### 1. iSCSI 架构解析
iSCSI (Internet Small Computer System Interface) 是一种在 TCP/IP 网络上传输 SCSI 命令的协议。它允许我们将一个远程的物理磁盘或逻辑卷，当作一个本地连接的磁盘来使用。这与 NFS（提供文件级共享）有本质区别，iSCSI 提供的是**块级访问**。

- **核心组件**:
  - **iSCSI Target (目标器)**: 存储服务器。它管理着后备存储（可以是物理磁盘、LVM 卷、或镜像文件），并将它们作为 LUN 暴露给网络。
  - **iSCSI Initiator (启动器)**: 存储客户端。它通过网络向 Target 发送 SCSI 命令，并将 Target 提供的 LUN 识别为一个本地的块设备（如 `/dev/sdb`）。
  - **LUN (Logical Unit Number)**: 逻辑单元号。一个 Target 可以管理多个存储单元，每个单元用一个 LUN ID 来标识。一个 LUN 对 Initiator 来说就是一块“硬盘”。
  - **IQN (iSCSI Qualified Name)**: iSCSI 节点的全球唯一名称，格式通常为 `iqn.yyyy-mm.com.example:unique-id`。每个 Target 和 Initiator 都有一个 IQN，用于身份识别和访问控制。

### 2. iSCSI 认证与安全: CHAP
由于 iSCSI 在标准 IP 网络上传输数据，认证至关重要，以防止未经授权的客户端访问存储。CHAP 是 iSCSI 中最常用的认证机制。

- **CHAP (Challenge-Handshake Authentication Protocol)**: 质询握手认证协议。
  - **工作流程**: 
    1. Initiator 尝试连接 Target。
    2. Target 发送一个随机的“质询”(Challenge) 给 Initiator。
    3. Initiator 使用与 Target 预共享的密钥 (密码) 对质询进行哈希计算，并将结果发回给 Target。
    4. Target 使用相同的密钥和原始质询进行同样的哈希计算，如果结果匹配，则认证通过。
  - **优点**: 密码本身不会在网络上传输，只传输哈希值，因此相对安全。

## 🛠️ 实践操作 (40%)

我们将使用 LIO (Linux-IO Target) 作为 Target 实现，它是当前 Linux 内核的标准块存储目标框架。`targetcli` 是其用户态配置工具。

### 环境准备
- **Target 服务器**: IP `192.168.1.100`
- **Initiator 客户端**: IP `192.168.1.101`

### 1. iSCSI Target (服务器) 配置 (在 `192.168.1.100` 上)

```bash
# 1. 安装 targetcli 工具
sudo apt update
sudo apt install -y targetcli-fb

# 2. 准备一个后备存储 (Backstore)。这里我们用一个文件来模拟一个磁盘。
sudo fallocate -l 10G /var/lib/iscsi_storage.img

# 3. 启动 targetcli 配置界面
sudo targetcli

# --- 进入 targetcli shell 后执行以下命令 ---

# a. 创建一个 fileio 类型的后备存储
# 语法: /backstores/fileio create <name> <file_path>
/> /backstores/fileio create file_backend /var/lib/iscsi_storage.img

# b. 创建一个 iSCSI Target，系统会自动生成一个 IQN
/> /iscsi create
# 这会创建一个类似 iqn.2003-01.org.linux-iscsi.target.x8664:sn.somerandomstring 的 Target
# 为方便记，我们用一个自定义的 IQN
/> /iscsi create iqn.2025-07.com.example:storage.disk1

# c. 将后备存储关联到 Target，即创建一个 LUN
# 语法: /iscsi/<target_iqn>/tpg1/luns create /backstores/<type>/<name>
/> /iscsi/iqn.2025-07.com.example:storage.disk1/tpg1/luns create /backstores/fileio/file_backend

# d. 获取客户端的 IQN (需要在客户端上执行 `cat /etc/iscsi/initiatorname.iscsi`)
# 假设客户端 IQN 是 iqn.1993-08.org.debian:01:someid

# e. 创建 ACL (Access Control List)，只允许指定的客户端访问
# 语法: /iscsi/<target_iqn>/tpg1/acls create <client_iqn>
/> /iscsi/iqn.2025-07.com.example:storage.disk1/tpg1/acls create iqn.1993-08.org.debian:01:someid

# f. 保存配置并退出
/> saveconfig
/> exit
# --- 退出 targetcli shell ---
```

### 2. iSCSI Initiator (客户端) 配置 (在 `192.168.1.101` 上)

```bash
# 1. 安装 open-iscsi 工具
sudo apt update
sudo apt install -y open-iscsi

# 2. 获取本机的 IQN (用于在 Target 上配置 ACL)
sudo cat /etc/iscsi/initiatorname.iscsi
# 输出类似: InitiatorName=iqn.1993-08.org.debian:01:someid

# 3. 发现 Target 服务器上的可用目标
# -m discovery: 模式为发现
# -t sendtargets: 使用 SendTargets 发现类型
# -p <ip>: 指定 Target 的 IP 地址
sudo iscsiadm -m discovery -t sendtargets -p 192.168.1.100

# 4. 登录到 Target
# --login 会登录到所有已发现但未登录的 Target
sudo iscsiadm -m node --login

# 5. 验证块设备是否已连接
lsblk
# 你应该能看到一个新的磁盘，如 /dev/sdb，大小为 10G

# 6. 格式化、挂载并使用该设备
sudo mkfs.ext4 /dev/sdb
sudo mkdir /data
sudo mount /dev/sdb /data
df -hT /data
```

## 💻 Go 编程实现 (20%)

**目标**: 编写一个 Go 程序，调用 `iscsiadm` 命令，检查当前是否存在活动的 iSCSI 会话。

**`iscsi_checker.go`**
```go
package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// CheckActiveSessions 检查是否存在活动的 iSCSI 会话
func CheckActiveSessions() (bool, error) {
	// -m session: 查询会话信息
	// -P 1: 打印级别1，提供足够的信息
	cmd := exec.Command("iscsiadm", "-m", "session", "-P", "1")

	out, err := cmd.Output()
	if err != nil {
		// 如果命令执行失败，通常意味着 iscsid 服务未运行或没有会话
		if exitErr, ok := err.(*exec.ExitError); ok {
			// iscsiadm 在没有会话时会返回非零退出码
			if strings.Contains(string(exitErr.Stderr), "No active sessions") {
				return false, nil
			}
		}
		return false, fmt.Errorf("command failed: %w - %s", err, string(out))
	}

	// 如果命令成功并且有输出，说明存在会话
	return len(out) > 0, nil
}

func main() {
	log.Println("Checking for active iSCSI sessions...")
	active, err := CheckActiveSessions()
	if err != nil {
		log.Fatalf("Error checking iSCSI sessions: %v", err)
	}

	if active {
		fmt.Println("Result: Active iSCSI sessions found.")
	} else {
		fmt.Println("Result: No active iSCSI sessions.")
	}
}
```

## 🔍 故障排查与优化
- **发现失败 (Discovery Failed)**: 检查客户端和 Target 之间的网络连通性（`ping`）和防火墙（确保 TCP 端口 3260 开放）。
- **登录失败 (Login Failed)**: 
  - **认证失败**: 如果配置了 CHAP，请检查密码是否匹配。
  - **ACL 拒绝**: 检查 Target 上的 ACL 配置，确保客户端的 IQN 被正确添加。
  - **Target 端错误**: 查看 Target 服务器的系统日志（如 `journalctl -u iscsid`）获取详细错误信息。
- **优化**: 在生产环境中，iSCSI 流量应运行在专用的网络（独立的 VLAN 或物理网络）上，以减少与普通业务流量的干扰，保证性能和稳定性。

## 📝 实战项目
- **配置 CHAP 认证**: 
  1. 在 Target 的 `targetcli` 中，为 ACL 设置认证信息：`/> /iscsi/.../tpg1/acls/<client_iqn> set auth userid=user password=password123`
  2. 在 Initiator 的 `/etc/iscsi/iscsid.conf` 中，找到并修改 `node.session.auth.authmethod = CHAP`，并设置 `node.session.auth.username` 和 `node.session.auth.password`。
  3. 重启 `open-iscsi` 服务并重新登录。

## 🏠 课后作业
- **深入研究**: 详细对比 iSCSI 和 Fibre Channel (FC) 在性能、成本、部署复杂度、应用场景等方面的异同，并写出总结。
- **脚本编写**: 编写一个 Shell 脚本，该脚本接受一个 Target IP 作为参数，自动执行发现、登录，并在成功后打印出新出现的块设备名称（如 `sdb`）。
