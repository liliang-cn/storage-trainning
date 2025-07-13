# Day 1: DRBD 核心原理与手动配置

## 🎯 学习目标
- **技能目标**: 深入理解 DRBD 作为网络RAID-1的实时块设备复制原理。
- **实践目标**: 能够独立、手动地配置一个双节点的 DRBD 资源，并完成主从切换。
- **运维能力**: 学会使用 `drbdadm` 和 `/proc/drbd` 对 DRBD 资源进行状态监控和基础管理。
- **成果产出**: 一个正常工作的双节点 DRBD 集群，一份详细的 DRBD 同步协议 (A, B, C) 对比分析笔记。

## 📚 理论基础 (40%)

### 1. DRBD (Distributed Replicated Block Device) 深度解析
DRBD 是一种软件定义的存储解决方案，用于在网络上的多台服务器之间实现块设备的实时、同步复制。它工作在 Linux 内核的块设备层，虚拟出一个 `/dev/drbdX` 设备。对这个设备的任何写入操作，都会被 DRBD 驱动拦截，一份写入本地磁盘，另一份通过网络发送给对端节点，再由对端节点的 DRBD 驱动写入其本地磁盘。

- **网络 RAID-1**: 这是理解 DRBD 最简单的方式。传统的 RAID-1 在一块主机的两块硬盘间镜像数据，而 DRBD 在两台主机之间通过网络镜像数据。
- **主/从 (Primary/Secondary) 角色**: 为了保证数据一致性，任何时候只有一个节点能以 Primary 角色挂载并读写 DRBD 设备。另一个节点必须处于 Secondary 角色，它接收所有更新，但不能直接被上层应用访问。
- **元数据 (Metadata)**: DRBD 在底层物理设备的末尾保留一小块区域来存储自己的元数据，包括配置信息、活动日志、位图等，用于跟踪数据同步状态和快速恢复。

### 2. DRBD 同步协议
DRBD 提供了三种不同的同步协议，用于在性能和数据安全性之间做权衡。

- **Protocol C (同步复制)**: 
  - **流程**: 应用程序的写操作 -> 本地磁盘写入完成 -> 数据包通过网络发送 -> **对端节点确认写入完成** -> 应用程序收到写成功的回应。
  - **优点**: **零数据丢失 (RPO=0)**。主节点故障时，可以确信所有已确认的写操作都已同步到从节点。
  - **缺点**: 性能最低，因为每次写操作的延迟都包含了网络往返时间和对端磁盘写入时间。
  - **适用场景**: 数据库、交易系统等对数据一致性要求最高的场景。

- **Protocol B (半同步复制)**: 
  - **流程**: 应用程序的写操作 -> 本地磁盘写入完成 -> 数据包通过网络发送 -> **对端节点确认已接收到数据包** -> 应用程序收到写成功的回应。
  - **优点**: 性能优于 Protocol C，因为它不等待对端磁盘写入完成。
  - **缺点**: 主节点突然宕机时，最后一次传输中的数据可能在从节点上还未写入磁盘，存在微小的数据丢失风险。
  - **适用场景**: 需要较高性能，但能容忍极少量数据丢失的场景。

- **Protocol A (异步复制)**: 
  - **流程**: 应用程序的写操作 -> **本地磁盘写入完成** -> 应用程序收到写成功的回应。数据包在后台异步发送给对端节点。
  - **优点**: **性能最高**，几乎不受网络延迟影响。
  - **缺点**: 数据丢失风险最大。如果主节点宕机，所有在网络缓冲区中还未发送的数据都会丢失。
  - **适用场景**: 广域网 (WAN) 灾备、对性能要求极高但对数据同步要求不高的场景。

## 🛠️ 实践操作 (50%)

### 环境准备
- 两台虚拟机: `node1` (192.168.1.101), `node2` (192.168.1.102)
- 每台机器添加一块同样大小的裸盘，如 `/dev/sdb` (例如 10GB)。确保它未被分区和使用。
- 确保两台机器主机名配置正确，并且可以通过主机名互相 ping 通。

### 1. 安装与配置

```bash
# 在 node1 和 node2 上同时执行

# 1. 安装 DRBD 工具
sudo apt-get update
sudo apt-get install -y drbd-utils

# 2. 加载 DRBD 内核模块
sudo modprobe drbd

# 3. 创建 DRBD 资源配置文件。文件名必须是 .res 结尾
# 在 /etc/drbd.d/r0.res 文件中写入以下内容 (两台机器配置相同)
sudo tee /etc/drbd.d/r0.res > /dev/null <<'EOF'
resource r0 {
  protocol C; # 使用最安全的同步协议

  on node1 {
    device    /dev/drbd0;      # DRBD 虚拟设备名
    disk      /dev/sdb;        # 底层物理设备
    address   192.168.1.101:7789; # 本机监听地址和端口
    meta-disk internal;        # 元数据存储在物理设备内部
  }

  on node2 {
    device    /dev/drbd0;
    disk      /dev/sdb;
    address   192.168.1.102:7789;
    meta-disk internal;
  }
}
EOF
```

### 2. 初始化与启动

```bash
# 在 node1 和 node2 上同时执行

# 1. 初始化元数据区域。这会向底层设备写入 DRBD 元数据。
# 这个操作只需要在第一次创建资源时执行一次。
sudo drbdadm create-md r0

# 2. 启动 DRBD 资源。这会使 DRBD 驱动关联上层和下层设备。
sudo drbdadm up r0

# 3. 查看当前状态。此时两个节点都应该是 Secondary/Secondary, Inconsistent/Inconsistent
sudo drbdadm status r0
# 或者更详细的
cat /proc/drbd
```

### 3. 首次同步

```bash
# 只在 node1 上执行

# 1. 将 node1 强制设置为主节点。--force 选项只在首次同步时需要，
# 它告诉 DRBD “以我为准，覆盖对端的数据”。
sudo drbdadm primary --force r0

# 2. 监控同步状态。你会看到同步进度。
watch -n1 cat /proc/drbd
# 等待直到状态变为 Primary/Secondary, UpToDate/UpToDate
```

### 4. 使用 DRBD 设备

```bash
# 只在 node1 (当前的主节点) 上执行

# 1. 在 DRBD 设备上创建文件系统
sudo mkfs.ext4 /dev/drbd0

# 2. 挂载并使用
sudo mount /dev/drbd0 /mnt
df -hT /mnt

# 3. 写入测试数据
sudo touch /mnt/testfile1.txt
ls /mnt

# 4. 使用完毕后卸载
sudo umount /mnt
```

## 🔍 故障排查与优化
- **`drbdadm up` 失败**: 
  - **排查**: 检查防火墙是否阻止了 7789 端口的通信。检查 `/etc/drbd.d/r0.res` 中的 IP 地址和主机名是否正确。
- **节点无法连接 (WFConnection)**: 
  - **排查**: 检查网络连通性。确认对端节点的 `drbd` 服务正在运行。
- **优化**: 在生产环境中，DRBD 的网络流量应该跑在专用的高速、低延迟网络链路上（例如两台服务器网卡直连），以最小化对业务网络的影响和复制延迟。

## 📝 实战项目
- **主/从角色切换 (Failover)**: 
  1. 在当前主节点 `node1` 上，卸载文件系统 `sudo umount /mnt`。
  2. 将 `node1` 降级为从节点: `sudo drbdadm secondary r0`。
  3. 在 `node2` 上，将 `node2` 升级为主节点: `sudo drbdadm primary r0`。
  4. 在 `node2` 上，挂载设备: `sudo mount /dev/drbd0 /mnt`。
  5. 验证之前在 `node1` 上创建的文件 `testfile1.txt` 是否存在。

## 🏠 课后作业
- **深入研究**: 详细阅读 `drbd.conf` 的 man page，特别是 `net` 和 `disk` 配置块中的高级选项，如 `timeout`, `c-plan-ahead`, `on-io-error` 等，并理解它们的含义。
- **协议对比**: 编辑 `r0.res` 文件，将 `protocol C` 改为 `protocol A`，重启 DRBD 服务后，使用 `fio` 或 `dd` 测试写性能，直观感受不同协议带来的性能差异。
