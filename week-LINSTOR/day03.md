# Day 3: 使用 LINSTOR 管理 DRBD 资源

## 🎯 学习目标
- **技能目标**: 熟练使用 `linstor` 命令行工具完成资源的完整生命周期管理（创建、查询、删除）。
- **实践目标**: 成功部署一个双副本的 DRBD 卷，并在节点上挂载使用。
- **核心概念**: 掌握 LINSTOR 如何通过“资源定义”和“卷定义”的抽象，来自动化和简化 DRBD 资源的放置与配置。
- **高级功能**: 学会使用 LINSTOR 对 DRBD 卷进行快照，为备份和测试提供支持。
- **成果产出**: 一个由 LINSTOR 管理的、可用的高可用 DRBD 卷，一个可以连接 LINSTOR API 并获取基本信息的 Go 程序。

## 📚 理论基础 (20%)

### LINSTOR 资源管理工作流
LINSTOR 将复杂的 DRBD 配置流程抽象为几个简单的、声明式的步骤。当你执行 `linstor resource create` 时，背后发生了以下一系列自动化操作：

1.  **用户请求**: `linstor` CLI 将用户的指令（例如：“在 node1 上为资源 `my-data` 创建一个副本”）发送到 Controller 的 REST API。
2.  **Controller 决策**: Controller 接收到请求，查询其内部数据库，获取 `my-data` 的卷定义（如大小）和 `node1` 的存储池信息。
3.  **下发指令给 Satellite**: Controller 向 `node1` 的 Satellite 发送指令：“请为资源 `my-data` 在存储池 `sp1` 中分配一个 10GB 的卷”。
4.  **Satellite 执行 (LVM)**: `node1` 的 Satellite 接收到指令，执行 `lvcreate -V 10G --name my-data_00000 -T lvm_vg/thin_pool` 来创建一个 LVM Thin Volume。
5.  **Satellite 执行 (DRBD)**: 当多个节点上都创建了资源副本后，Controller 会协调这些节点上的 Satellite，让它们自动生成临时的 `.res` 配置文件，并执行 `drbdadm up` 和 `drbdadm primary/secondary` 等命令来建立 DRBD 连接。
6.  **状态上报**: Satellite 将操作结果（如新创建的设备路径 `/dev/drbd1000`）上报给 Controller。
7.  **完成**: Controller 更新数据库状态，并向用户返回成功信息。

整个过程对用户透明，用户无需关心底层的 LVM 和 DRBD 命令细节。

## 🛠️ 实践操作 (50%)

我们将基于 Day 2 建立的三节点集群，创建一个双副本的卷。

### 1. 创建资源定义 (Resource Definition)
这是对一类存储的命名，比如 `web-server-logs` 或 `mysql-data`。

```bash
# 在任何一个节点上执行

# 语法: linstor resource-definition create <resource_name>
linstor resource-definition create web-data

# 查看已创建的资源定义
linstor resource-definition list
```

### 2. 创建卷定义 (Volume Definition)
这定义了该类存储的具体属性，最重要的是大小。

```bash
# 语法: linstor volume-definition create <resource_name> <size>
linstor volume-definition create web-data 10G

# 查看卷定义
linstor volume-definition list
```

### 3. 部署资源 (Deploy Resource)
这是最关键的一步，它会真正在节点上创建设备副本。

```bash
# 语法: linstor resource create <node_name> <resource_name> --storage-pool <pool_name>

# 在 node1 上创建第一个副本
linstor resource create node1 web-data --storage-pool sp1

# 在 node2 上创建第二个副本
linstor resource create node2 web-data --storage-pool sp1

# LINSTOR 会自动处理 DRBD 的配置和同步
```

### 4. 查看和使用资源

```bash
# 1. 查看资源列表和状态
# 你应该能看到 web-data 资源，以及它在 node1 和 node2 上的两个副本
# 初始状态可能是 Syncing，等待它变为 UpToDate
linstor resource list

# 2. 获取 DRBD 设备路径
# linstor resource list 命令的输出会显示设备路径，例如 /dev/drbd1000

# 3. 在其中一个节点上 (例如 node1) 挂载使用
# 注意：LINSTOR 创建的 DRBD 资源默认都是 Secondary 角色，你需要手动提升为主节点

# 将 node1 上的 web-data 资源提升为主节点
sudo drbdadm primary web-data

# 格式化并挂载
sudo mkfs.ext4 /dev/drbd/by-res/web-data/0 # 使用 by-res 路径更稳定
sudo mount /dev/drbd/by-res/web-data/0 /mnt

# 验证使用
df -h /mnt
touch /mnt/hello-linstor.txt
```

### 5. 创建快照
LINSTOR 的快照功能依赖于底层存储池的能力（如 LVM Thin Pool 或 ZFS）。

```bash
# 语法: linstor snapshot create <node_name> <resource_name> <snapshot_name>

# 在 node1 上为 web-data 资源创建一个快照
linstor snapshot create node1 web-data snap1

# 查看快照列表
linstor snapshot list

# 快照实际上是一个只读的、冻结的 LVM Thin 快照卷。
# 你可以基于这个快照创建一个新的可写卷，用于测试或恢复。
linstor snapshot restore node1 web-data snap1 web-data-restored
```

## 💻 Go 编程实现 (30%)

**目标**: 编写一个 Go 程序，连接到 LINSTOR Controller 的 REST API，并以结构化的形式打印出集群中的节点列表。

**`linstor_checker.go`**
```go
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// NodeInfo 结构体用于解析 LINSTOR API 返回的节点信息
// 我们只定义我们关心的字段
type NodeInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Address    string `json:"net_interfaces"` // 简化处理，实际是数组
	Connection string `json:"connection_status"`
}

func main() {
	controllerIP := "127.0.0.1" // 假设在本机运行，或替换为 Controller IP
	apiURL := fmt.Sprintf("http://%s:3370/v1/nodes", controllerIP)

	log.Printf("Querying LINSTOR API at: %s", apiURL)

	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		log.Fatalf("Failed to connect to LINSTOR controller: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("API request failed with status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read API response body: %v", err)
	}

	var nodes []NodeInfo
	if err := json.Unmarshal(body, &nodes); err != nil {
		log.Fatalf("Failed to parse JSON response: %v", err)
	}

	fmt.Println("--- LINSTOR Nodes ---")
	for _, node := range nodes {
		fmt.Printf("Name: %-10s Type: %-12s Status: %s\n", node.Name, node.Type, node.Connection)
	}
}
```

## 🔍 故障排查与优化
- **创建资源失败**: 
  - **排查**: 查看 `linstor resource list` 的 `Reason` 列，它会给出失败的简要原因。更详细的信息需要查看 Controller 的日志 (`journalctl -u linstor-controller`) 和相关 Satellite 的日志。
  - **常见原因**: 存储池空间不足；节点离线；DRBD 无法建立连接。
- **优化**: LINSTOR 支持自动选择节点放置资源。你可以不指定节点名来创建资源，LINSTOR 会根据内置的策略（如可用空间）自动选择最佳节点。
  ```bash
  # 让 LINSTOR 自动选择2个节点放置副本
  linstor resource create --replicas 2 web-data-auto
  ```

## 🏠 课后作业
- **三副本资源**: 基于你的三节点集群，创建一个三副本的资源，并验证其状态。
- **Go API 探索**: 扩展你的 Go 程序，使其不仅能列出节点，还能列出存储池 (`/v1/storage-pools`) 和资源 (`/v1/resources`)。
- **快照恢复**: 完整地练习从快照恢复出一个新卷，并挂载验证其内容与快照创建时一致。
