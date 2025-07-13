# Day 2: LINSTOR 架构与集群部署

## 🎯 学习目标
- **技能目标**: 清晰地理解 LINSTOR 的分层架构（Controller, Satellite）以及它如何解决手动管理 DRBD 的痛点。
- **实践目标**: 能够独立部署一个 LINSTOR Controller 和多个 Satellite 节点，组成一个可用的存储集群。
- **核心概念**: 掌握 LINSTOR 的核心抽象：节点 (Node)、存储池 (Storage Pool)、资源定义 (Resource Definition) 和卷定义 (Volume Definition)。
- **成果产出**: 一个健康运行的三节点 LINSTOR 集群，一份详细的 LINSTOR 架构图和组件说明。

## 📚 理论基础 (40%)

### 1. 为什么需要 LINSTOR？
在 Day 1 中，我们手动配置了一个 DRBD 资源。这个过程对于一两个资源尚可接受，但想象一下在一个有数十台服务器、数百个卷的环境中：
- **配置复杂**: 每个资源的 `.res` 文件都需要手动编写和分发。
- **管理困难**: 需要 SSH 到每个节点上执行 `drbdadm` 命令来查看状态和进行操作。
- **无中心视图**: 无法从一个地方直观地看到整个集群的健康状况、资源分布和容量使用情况。
- **扩展性差**: 添加新节点或新磁盘需要大量手动配置。

**LINSTOR 的诞生就是为了解决这些问题。** 它是一个开源的存储管理工具，专门用于配置和管理大规模的 Linux 块存储，特别是 DRBD 和 LVM。

### 2. LINSTOR 架构
LINSTOR 采用经典的控制器-代理 (Controller-Agent) 模型。

- **LINSTOR Controller (控制器)**:
  - **角色**: 集群的“大脑”。
  - **功能**: 
    1. 维护整个集群的配置和状态数据库。
    2. 提供一个 REST API 供外部工具（如 `linstor` CLI, Kubernetes CSI 驱动）交互。
    3. 接收用户的指令（如“创建一个 10GB 的双副本卷”）。
    4. 做出决策，决定在哪些节点上放置数据、如何配置 DRBD 等。
    5. 将具体的执行任务下发给相应的 Satellite 节点。
  - **部署**: 通常部署在一个或多个专用的管理节点上。为了实现高可用，可以部署一个高可用 Controller 集群。

- **LINSTOR Satellite (卫星/代理)**:
  - **角色**: 集群的“手和脚”。
  - **功能**: 
    1. 运行在每一个存储节点上。
    2. 监听来自 Controller 的指令。
    3. 在本地节点上实际执行存储操作，例如：
       - 调用 `lvcreate` 创建 LVM 逻辑卷。
       - 调用 `drbdadm` 创建和配置 DRBD 资源。
       - 调用 `mkfs` 格式化设备。
    4. 向 Controller 汇报本地的状态和操作结果。

- **LINSTOR Client (`linstor` CLI)**:
  - **角色**: 用户的交互工具。
  - **功能**: 它是一个简单的命令行客户端，将用户的命令转换成对 Controller REST API 的调用。

### 3. LINSTOR 核心抽象
- **Node**: 代表一个物理或虚拟服务器，上面运行着 LINSTOR Satellite。
- **Storage Pool**: 定义在某个节点上的一块可用存储。它可以是 LVM 卷组、LVM Thin Pool、ZFS zpool 或简单的文件目录。这是 LINSTOR 放置数据的物理位置。
- **Resource Definition**: 对一类卷的抽象定义，例如 `mysql-data`。它本身不包含大小信息。
- **Volume Definition**: 在一个资源定义下的具体卷定义，包含大小、加密等属性。
- **Resource**: 一个卷在某个节点上的具体实例（副本）。
- **Volume**: 一个逻辑卷的完整概念，它可能由分布在不同节点上的多个 Resource (副本) 组成。

## 🛠️ 实践操作 (50%)

### 环境准备
- 三台虚拟机: `node1`, `node2`, `node3`。
- `node1` 将作为 Controller 和 Satellite。
- `node2`, `node3` 将作为纯 Satellite。
- 所有节点上都有一个可用的 LVM 卷组，例如 `lvm_vg`。

### 1. 添加 LINBIT 软件源

```bash
# 在所有三个节点上执行
# (以 Ubuntu 为例)
sudo add-apt-repository ppa:linbit/linbit-drbd9-stack
sudo apt-get update
```

### 2. 安装 LINSTOR 组件

```bash
# 在 node1 上安装 Controller 和 Satellite
sudo apt-get install -y linstor-controller linstor-satellite linstor-client

# 在 node2 和 node3 上安装 Satellite
sudo apt-get install -y linstor-satellite linstor-client
```

### 3. 启动服务

```bash
# 在 node1 上启动 Controller
sudo systemctl start linstor-controller
sudo systemctl enable linstor-controller

# 在所有三个节点上启动 Satellite
sudo systemctl start linstor-satellite
sudo systemctl enable linstor-satellite
```

### 4. 验证集群状态

```bash
# 在任何一个节点上执行 (因为都装了 linstor-client)

# 1. 查看节点列表。你应该能看到所有三个节点，并且状态是 Online
linstor node list

# 如果节点未出现或状态不是 Online，请检查网络和防火墙，
# 确保 Satellite 可以访问 Controller 的 3370 和 3371 端口。
```

### 5. 创建存储池
存储池告诉 LINSTOR 在哪里存放数据。我们将使用 LVM Thin Pool，因为它支持快照和精简配置。

```bash
# 在任何一个节点上执行

# 1. 首先，在所有节点上创建 LVM 卷组和 Thin Pool (如果还没有的话)
# (假设底层设备是 /dev/sdb)
# sudo pvcreate /dev/sdb
# sudo vgcreate lvm_vg /dev/sdb
# sudo lvcreate -L 9G -T lvm_vg/thin_pool

# 2. 使用 linstor 命令定义存储池
# 语法: linstor storage-pool create <type> <node_name> <pool_name> <provider_name> [--thin-pool <thin_pool_name>]

# 为 node1 创建存储池
linstor storage-pool create lvm-thin node1 sp1 lvm_vg --thin-pool thin_pool

# 为 node2 创建存储池
linstor storage-pool create lvm-thin node2 sp1 lvm_vg --thin-pool thin_pool

# 为 node3 创建存储池
linstor storage-pool create lvm-thin node3 sp1 lvm_vg --thin-pool thin_pool

# 3. 查看存储池列表
linstor storage-pool list
```

## 🔍 故障排查与优化
- **节点无法连接 (Offline)**: 
  - **排查**: 检查 `linstor-satellite` 服务是否在目标节点上运行。检查 Controller 和 Satellite 之间的网络连接，特别是 Controller 的 3370/3371 端口。查看 Satellite 的日志 `journalctl -u linstor-satellite`。
- **创建存储池失败**: 
  - **排查**: 确认底层的 LVM 卷组或 ZFS 池确实存在于指定的节点上。确认 LINSTOR Satellite 有权限执行 `lvs`, `vgs` 等命令。
- **优化**: 在生产环境中，LINSTOR Controller 应该部署为高可用模式。这通常通过 Pacemaker 或其他集群管理软件，配合一个浮动 IP 和共享的 DRBD 卷来实现，以避免 Controller 自身成为单点故障。

## 📝 实战项目
- **ZFS 存储池**: 如果你的环境支持 ZFS，尝试在一个节点上创建一个基于 ZFS 的存储池，并与 LVM 存储池进行对比。
  ```bash
  # 1. 创建 ZFS 池: sudo zpool create zfs_pool /dev/sdc
  # 2. 在 LINSTOR 中创建: linstor storage-pool create zfs nodeX sp_zfs zfs_pool
  ```

## 🏠 课后作业
- **深入研究**: 阅读 LINSTOR 用户手册中关于“节点属性”和“存储池属性”的部分。尝试为节点或存储池添加自定义属性，并理解它们的作用。
- **CLI 探索**: 使用 `linstor node describe <node_name>` 和 `linstor storage-pool describe <node_name> <pool_name>` 命令，详细查看 LINSTOR 收集了哪些关于节点和存储池的信息。
