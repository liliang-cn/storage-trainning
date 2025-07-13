# Day 3: ZFS 核心概念与存储池管理

## 🎯 学习目标
- **理论深度**: 深入理解 ZFS 的核心架构，包括 vdev (虚拟设备)、zpool (存储池) 和 dataset (数据集) 之间的关系。
- **核心技能**: 熟练掌握 `zpool` 和 `zfs` 命令行工具，能够独立创建、管理、监控 ZFS 存储池和数据集。
- **关键特性**: 理解 ZFS 的写时复制 (Copy-on-Write)、端到端数据校验 (Checksum) 和自修复 (Self-Healing) 机制的原理与价值。
- **Go 编程实践**: 开始编写一个 Go 程序，用于调用 ZFS 命令并解析输出，实现对 ZFS 存储池健康状况和容量的基础监控。

## 📚 理论基础 (40%)

### 1. ZFS 架构解析

ZFS 从根本上改变了文件系统和卷管理的传统模式，将二者合为一体。其分层架构是理解 ZFS 的关键。

#### a. vdev (Virtual Device - 虚拟设备)
vdev 是 ZFS 存储池的构建基石，是物理磁盘的抽象层。一个或多个物理磁盘（或分区、文件）可以组成一个 vdev。vdev 决定了其内部数据的冗余级别。常见的 vdev 类型包括：
- **单个磁盘 (disk)**: 无冗余，不推荐用于生产。
- **镜像 (mirror)**: 类似 RAID 1，提供最高的数据冗余。
- **raidz1/raidz2/raidz3**: 类似 RAID 5/6，分别可以容忍 1/2/3 个磁盘故障。它通过可变宽度的条带解决了 RAID 的“写洞”问题。
- **热备 (hot spare)**: 用于自动替换故障磁盘的备用盘。
- **特殊 vdev**:
    - **log (ZIL/SLOG)**: 用于加速同步写入操作的独立设备，通常是高速 SSD。
    - **cache (L2ARC)**: 用作二级读缓存的设备，通常是高速 SSD。

#### b. zpool (Storage Pool - 存储池)
zpool 是由一个或多个 vdev 构成的统一存储资源池。它是 ZFS 中最大的存储单元。
- **特性**:
    - **统一管理**: 一个 zpool 将所有 vdev 的容量整合在一起，对外提供一个巨大的存储空间。
    - **扩展性**: 可以通过向 zpool 中添加新的 vdev 来动态扩容（但不能向已有的 vdev 中添加磁盘）。
    - **冗余性**: zpool 的冗余性由其包含的 vdev 决定。例如，一个由两个 mirror vdev 组成的 zpool，可以容忍每个 mirror vdev 中各坏一块盘。

#### c. dataset (数据集)
dataset 是从 zpool 中划分出来的、可以独立挂载和管理的文件系统。这是用户与 ZFS 交互的主要层面。
- **特性**:
    - **轻量级**: 创建和销毁数据集几乎是瞬时完成的。
    - **精细化管理**: 可以为每个数据集设置独立的属性，如挂载点、配额 (quota)、预留空间 (reservation)、压缩算法 (compression)、记录大小 (recordsize) 等。
    - **属性继承**: 子数据集会自动继承父数据集的属性，便于统一管理。
- **zvol**: 除了文件系统，ZFS 还可以创建块设备，称为 zvol，可用于 iSCSI、虚拟机磁盘等场景。

### 2. ZFS 关键数据保护机制

#### a. 写时复制 (Copy-on-Write, CoW)
这是 ZFS 的核心机制。ZFS 从不覆盖写（Overwrite）旧数据。当数据需要修改时，它会将修改后的新数据写入到一块新的空闲位置，然后更新指向该数据的元数据指针。
- **优点**:
    - **无写洞**: 磁盘上的数据状态永远是一致的。如果在写入过程中断电，旧数据依然完好无损，新数据只是未被引用的垃圾空间。
    - **廉价的快照**: 创建快照只需复制一份元数据指针，几乎不占用空间和时间。

#### b. 端到端数据校验 (End-to-end Checksum)
- **工作原理**: 当数据块写入时，ZFS 会计算其校验和（如 SHA-256）并与数据块的元数据指针一同存储。当数据块被读取时，ZFS 会重新计算校验和并与存储的值进行比对。
- **价值**: 能够检测到“静默数据损坏”（Silent Data Corruption），即数据在磁盘上因介质老化等原因发生位翻转，而硬件并未报��任何错误。

#### c. 数据自修复 (Self-Healing)
- **工作原理**: 在一个冗余的 zpool (mirror 或 raidz) 中，如果读取数据时发现校验和不匹配，ZFS 会判定数据损坏。此时，它会利用其他磁盘上的冗余信息（镜像数据或奇偶校验）来重建正确的数据，修复损坏的块，并将正确的数据返回给应用程序，整个过程对用户透明。

## 🛠️ 实践操作 (40%)

### 1. ZFS 环境准备
```bash
# 在 Debian/Ubuntu 上安装 ZFS
sudo apt-get update
sudo apt-get install -y zfsutils-linux

# 确认 ZFS 内核模块已加载
lsmod | grep zfs

# 使用前一天的 loop 设备进行实验
# 确保它们是干净的
sudo mdadm --stop /dev/md* || true
sudo wipefs -a /dev/loop* # 清除可能存在的旧元数据
ls /dev/loop{1..8}
```

### 2. 创建和管理 ZFS 存储池 (`zpool`)

#### a. 创建镜像池 (Mirror Pool)
```bash
# 使用 /dev/loop1 和 /dev/loop2 创建一个名为 'tank' 的镜像池
# tank 是 ZFS 社区常用的示例池名
sudo zpool create tank mirror /dev/loop1 /dev/loop2

# 查看池状态
sudo zpool status tank
# 重点关注 state: ONLINE, scan: none requested, errors: No known data errors
# config 部分会显示 tank 由一个 mirror-0 vdev 构成
```

#### b. 创建 raidz 池 (raidz1 Pool)
```bash
# 使用 /dev/loop3, loop4, loop5 创建一个名为 'datapool' 的 raidz1 池
sudo zpool create datapool raidz /dev/loop3 /dev/loop4 /dev/loop5

# 查看所有池的列表和基本容量信息
sudo zpool list
# NAME       SIZE  ALLOC   FREE  CKPOINT  EXPANDSZ   FRAG    CAP  DEDUP    HEALTH  ALTROOT
# datapool  2.88G   108K  2.88G        -         -     0%     0%  1.00x    ONLINE  -
# tank      1.94G   108K  1.94G        -         -     0%     0%  1.00x    ONLINE  -
```

#### c. 销毁存储池
```bash
# 销毁池会删除所有数据，请谨慎操作！
sudo zpool destroy datapool

# 验证池已被销毁
sudo zpool list
```

### 3. 创建和管理数据集 (`zfs`)

#### a. 创建数据集
默认情况下，创建 zpool 时会自动创建一个同名的数据集，并挂载到 `/<pool_name>`。
```bash
# 在 tank 池中创建一个名为 'data' 的数据集
sudo zfs create tank/data

# 创建一个嵌套的数据集
sudo zfs create tank/data/project_a
```

#### b. 查看和挂载数据集
```bash
# 查看 ZFS 文件系统列表
sudo zfs list
# NAME                  USED  AVAIL     REFER  MOUNTPOINT
# tank                  156K  1.85G     24.0K  /tank
# tank/data            48.0K  1.85G     24.0K  /tank/data
# tank/data/project_a  24.0K  1.85G     24.0K  /tank/data/project_a

# 验证挂载点
df -h /tank/data
```

#### c. 设置数据集属性
```bash
# 为 project_a 数据集开启 lz4 压缩
sudo zfs set compression=lz4 tank/data/project_a

# 为 project_a 设置 500MB 的空间配额
sudo zfs set quota=500M tank/data/project_a

# 查看特定数据集的所有属性
sudo zfs get all tank/data/project_a | grep -E 'compression|quota'
# NAME                 PROPERTY     VALUE     SOURCE
# tank/data/project_a  compression  lz4       local
# tank/data/project_a  quota        500M      local
```

## 💻 Go 编程实现 (20%)

我们编写一个 Go 程序来检查所有 ZFS 池的健康状况，并列出它们的基本信息。我们将利用 ZFS 命令为脚本设计的可解析输出格式。

**`zfs_checker.go`**
```go
package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// checkPoolsHealth uses `zpool status -x` to get a global health check.
// This command is designed for scripts: it outputs "all pools are healthy" on success.
func checkPoolsHealth() (string, error) {
	cmd := exec.Command("sudo", "zpool", "status", "-x")
	out, err := cmd.CombinedOutput()
	if err != nil {
        // If the command fails, it usually means a pool is unhealthy.
        // The output will contain the details.
		return strings.TrimSpace(string(out)), nil
	}
	return strings.TrimSpace(string(out)), nil
}

// listPoolsInfo uses `zpool list` with script-friendly options.
// -p: gives full numbers for sizes (parsable)
// -H: no headers
// -o: specify columns
func listPoolsInfo() (string, error) {
	cmd := exec.Command("sudo", "zpool", "list", "-p", "-H", "-o", "name,size,alloc,free,health")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to list zpools: %w\nOutput: %s", err, string(out))
	}
	return string(out), nil
}

func main() {
	fmt.Println("--- ZFS Global Health Status ---")
	health, err := checkPoolsHealth()
	if err != nil {
		log.Fatalf("Error checking health: %v", err)
	}
	fmt.Println(health)
	fmt.Println("--------------------------------")

	fmt.Println("\n--- ZFS Pool Details (bytes) ---")
	fmt.Println("Name\tSize\tAlloc\tFree\tHealth")
	details, err := listPoolsInfo()
	if err != nil {
		log.Fatalf("Error listing pools: %v", err)
	}
	fmt.Println(details)
	fmt.Println("--------------------------------")
}
```

**如何运行:**
1. 保存代码为 `zfs_checker.go`。
2. 执行 `go run zfs_checker.go`。
3. 程序将首先给出一个总体的健康结论，然后列出每个池的详细容量和健康状态。

## 🔍 故障排查与优化
- **最佳实践**:
    - **禁用硬件 RAID**: 永远不要在硬件 RAID 卡上创建 ZFS，让 ZFS 直接管理裸盘（JBOD 模式），这样 ZFS 才能完全发挥其数据校验和修复功能。
    - **使用整盘**: 推荐将整个磁盘（如 `/dev/sdb`）而不是分区（如 `/dev/sdb1`）交给 ZFS，这样可以获得最佳性能和可靠性。
    - **池扩容**: ZFS 池只能通过添加新的 vdev 来扩容。例如，向一个已有的 mirror pool 中再添加一个 mirror vdev。你不能向一个已有的 mirror vdev 中添加第三块盘。
- **常见状态**:
    - **`DEGRADED`**: 池中某个 vdev 失去了一部分冗余（如 mirror 中坏了一块盘），池仍在线可用，但应尽快替换故障设备。
    - **`FAULTED`**: 池中某个 vdev 完全损坏（如 raidz1 中坏了两块盘），池已离线，数据无法访问。需要从备份中恢复。

## 📝 实战项目
1. **设计并创建存储布局**:
   - 销毁现有的 `tank` 池。
   - 重新创建一个名为 `homeserver` 的新池。
   - 该池应包含两个 vdev：
     - 一个名为 `vdev_docs` 的 mirror vdev，使用 `/dev/loop1` 和 `/dev/loop2`，用于存放关键文档。
     - 一个名为 `vdev_media` 的 raidz1 vdev，使用 `/dev/loop3`, `/dev/loop4`, `/dev/loop5`，用于存放媒体文件。
   - **命令提示**: `sudo zpool create homeserver mirror /dev/loop1 /dev/loop2 raidz /dev/loop3 /dev/loop4 /dev/loop5`
2. **创建和配置数据集**:
   - 在 `homeserver` 池中创建 `documents` 和 `media` 两个数据集。
   - 为 `documents` 数据集开启 `lz4` 压缩。
   - 为 `media` 数据集设置 `recordsize=1M`（大文件存储的推荐优化）。
   - 为 `documents` 数据集设置 `100M` 的配额。
3. **验证**: 使用 `zpool status`, `zfs list`, `zfs get all` 等命令验证你的配置是否正确。

## 🏠 课后作业
1. **官方文档阅读**: 阅读 OpenZFS 官方文档中关于 `zpool` 和 `zfs` 命令的介绍，了解更多高级选项。
2. **方案设计**: 为一个需要高可用性的小型企业数据库服务器设计一个 ZFS 存储池布局方案。你需要考虑以下几点：
   - 使用哪种 vdev 类型？为什么？
   - 是否需要独立的 log (SLOG) 设备？如果需要，推荐什么硬件？
   - 备份策略如何与 ZFS 的快照功能结合？
   - 将你的设计方案和理由写成一份简要的 Markdown 文档。
3. **环境清理**:
   ```bash
   sudo zfs destroy -r homeserver || true
   sudo zpool destroy tank || true
   sudo losetup -d /dev/loop*
   ```
