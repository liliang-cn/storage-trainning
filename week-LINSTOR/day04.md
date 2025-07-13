# Day 4: 故障处理与高可用

## 🎯 学习目标
- **技能目标**: 深入理解分布式存储中最危险的问题——“裂脑”(Split-Brain)，并掌握其成因、预防和恢复机制。
- **实践目标**: 能够模拟节点网络故障和宕机，并练习使用 LINSTOR 进行故障切换和裂脑恢复。
- **核心概念**: 掌握 LINSTOR 的自动裂脑恢复策略和高可用 Controller 的配置思想。
- **成果产出**: 一份详细的 DRBD 裂脑问题分析及恢复手册，一个经过故障演练、更加健壮的 LINSTOR 集群。

## 📚 理论基础 (50%)

### 1. 裂脑 (Split-Brain) 深度解析
裂脑是任何高可用集群（无论是数据库、文件系统还是 DRBD）都需要面对的终极问题。

- **什么是裂脑？**
  在一个双主（或主从）集群中，当两个节点之间的心跳网络中断时，如果缺乏有效的仲裁机制，每个节点都可能认为对方已经死亡，从而都尝试接管服务，即都将自己提升为 Primary 角色。此时，集群中出现了两个“大脑”，它们各自独立地接受写操作。当网络恢复时，这两个“大脑”上的数据已经发生了分歧，无法自动合并。这种情况就是“裂脑”。

- **DRBD 中的裂脑场景**: 
  1. `node1` (Primary) 和 `node2` (Secondary) 正常运行。
  2. 两者之间的网络连接中断。
  3. `node1` 没有检测到问题，继续作为 Primary 运行。
  4. 一个外部的集群管理器（如 Pacemaker）检测到 `node1` “无响应”，于是决定将 `node2` 提升为 Primary。
  5. 此时，`node1` 和 `node2` 都是 Primary，都在接受写请求，数据开始分叉。

- **后果**: 数据永久性不一致，是灾难性的。恢复的唯一方法是**选择一个节点作为“胜利者”，并完全丢弃另一个节点在此期间的所有数据更改**。

### 2. 预防和处理裂脑
预防永远胜于治疗。

- **Quorum (仲裁机制)**: 在一个大于两节点的集群中，可以设置“多数派”规则。只有当一个节点能联系到集群中超过半数的节点时，它才能成为 Primary。这可以有效防止裂脑。对于双节点集群，需要一个第三方的仲裁设备或节点（Arbiter）。
- **STONITH (Shoot The Other Node In The Head)**: 这是最可靠的裂脑防止机制。当集群管理器决定提升一个节点时，它会先通过一个独立的带外通道（如服务器的 IPMI/BMC 管理口）去强制关闭或重启另一个节点，确保旧的 Primary 绝对“死亡”后，再提升新的 Primary。

- **LINSTOR/DRBD 的裂脑自动恢复策略**: 
  DRBD 自身可以检测到裂脑的发生。当网络恢复时，它会拒绝连接，并将资源标记为 `SplitBrain` 状态。LINSTOR 提供了多种自动处理策略，可以在 `resource-definition` 中设置：
  - `discard-zero-changes`: 如果一个节点没有收到任何新的写操作，自动丢弃它。
  - `discard-least-changes`: 自动丢弃写操作较少的那个节点的数据。
  - `discard-younger-primary` / `discard-older-primary`: 根据成为 Primary 的时间来决定。
  - **强烈建议**: 仔细配置这些策略，否则默认情况下需要手动干预。

### 3. LINSTOR Controller 高可用
LINSTOR Satellite 是无状态的，但 Controller 存储了整个集群的配置，它本身可能成为单点故障。生产环境必须部署高可用的 Controller。
- **实现方式**: 通常使用 Pacemaker + Corosync 集群软件，配合一个浮动 IP (Virtual IP) 和一个由 DRBD 保护的共享磁盘。这个 DRBD 磁盘专门用于存放 LINSTOR Controller 的数据库。无论 Controller 在哪个物理节点上运行，它都通过浮动 IP 提供服务，并读写同一个 DRBD 设备上的数据。

## 🛠️ 实践操作 (40%)

### 1. 模拟网络故障并制造裂脑

```bash
# 假设 web-data 资源在 node1 (Primary) 和 node2 (Secondary) 上

# 1. 在 node1 上，使用 iptables 阻止与 node2 的 DRBD 通信
sudo iptables -A INPUT -p tcp --dport 7789 -s 192.168.1.102 -j DROP

# 2. 查看状态，会看到 node1 上的资源进入 WFConnection (等待连接) 状态
linstor resource list

# 3. 在 node2 上，强制将其提升为 Primary (模拟集群管理器的错误决策)
sudo drbdadm primary web-data

# 4. 此时，node1 和 node2 都认为自己是 Primary。可以各自挂载并写入不同数据
# 在 node1 上: sudo mount /dev/drbd/by-res/web-data/0 /mnt; echo "from node1" > /mnt/test.txt; sudo umount /mnt
# 在 node2 上: sudo mount /dev/drbd/by-res/web-data/0 /mnt; echo "from node2" > /mnt/test.txt; sudo umount /mnt

# 5. 恢复网络，移除 iptables 规则
sudo iptables -D INPUT -p tcp --dport 7789 -s 192.168.1.102 -j DROP

# 6. 查看状态，你会看到资源状态变为 SplitBrain
linstor resource list
```

### 2. 手动恢复裂脑

```bash
# 决策：我们选择保留 node1 上的数据，丢弃 node2 上的数据。

# 1. 在要被丢弃的节点 (node2) 上，执行 discard 命令
sudo drbdadm disconnect web-data
sudo drbdadm secondary web-data
sudo drbdadm connect --discard-my-data web-data

# 2. 在要保留的节点 (node1) 上，重新连接
sudo drbdadm disconnect web-data
sudo drbdadm connect web-data

# 3. 观察状态，node2 会开始作为同步目标，从 node1 重新同步数据
watch -n1 cat /proc/drbd

# 4. 验证数据。同步完成后，在 node1 上挂载，应该只能看到 "from node1" 的内容。
```

### 3. 模拟节点宕机与故障切换

```bash
# 假设 node1 是 Primary

# 1. 模拟 node1 宕机
sudo systemctl stop linstor-satellite # 或者直接重启虚拟机

# 2. 在 node2 上，将资源提升为 Primary
linstor resource promote web-data node2
# 或者使用 drbdadm
# sudo drbdadm primary web-data

# 3. 在 node2 上挂载并提供服务
sudo mount /dev/drbd/by-res/web-data/0 /mnt
# ... 服务正常运行 ...

# 4. 恢复 node1
sudo systemctl start linstor-satellite

# 5. LINSTOR/DRBD 会自动检测到 node1 回归，并将其作为 Secondary 从 node2 同步数据。
```

## 💻 Go 编程实现 (10%)

**目标**: 扩展昨天的 Go 程序，解析 LINSTOR API 返回的资源信息，并高亮显示状态不正常的资源（如 `SplitBrain`, `Inconsistent`）。

```go
// ... (引入新的 struct)
type ResourceState struct {
	NodeName string `json:"node_name"`
	State    string `json:"state"`
}

type ResourceInfo struct {
	Name   string          `json:"name"`
	States []ResourceState `json:"states"`
}

// ... (在 main 函数中添加新的 API 调用)
// resp, err := client.Get("http://.../v1/resources")
// ... (解析 JSON 到 []ResourceInfo)

fmt.Println("--- LINSTOR Resources ---")
for _, res := range resources {
	fmt.Printf("Resource: %s\n", res.Name)
	for _, state := range res.States {
		isHealthy := state.State == "UpToDate"
		if !isHealthy {
			fmt.Printf("  - Node: %-10s State: \033[91m%s\033[0m\n", state.NodeName, state.State) // 标红
		} else {
			fmt.Printf("  - Node: %-10s State: %s\n", state.NodeName, state.State)
		}
	}
}
```

## 🔍 故障排查与优化
- **无法自动恢复裂脑**: 检查资源定义中是否配置了自动恢复策略。`linstor resource-definition get-property <res_name> DrbdOptions/auto-recover-target-role`。
- **优化**: 在关键业务中，永远不要依赖手动的故障切换。必须使用像 Pacemaker 这样的成熟的集群管理器来自动化故障检测和资源转移，并配合 STONITH 机制来彻底杜绝裂脑风险。

## 🏠 课后作业
- **深入研究**: 详细阅读 LINSTOR 用户手册中关于“Quorum”和裂脑恢复策略的所有选项，并理解每个选项的适用场景。
- **高可用 Controller**: 画出一张 LINSTOR 高可用 Controller 的架构图，包含 Pacemaker、Corosync、Virtual IP 和 DRBD 等组件，并描述其工作流程。
