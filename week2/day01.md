# Day 1: 软件 RAID 基础与 `mdadm` 实战

## 🎯 学习目标
- **理论理解**: 深入理解 RAID 0 (条带化) 和 RAID 1 (镜像化) 的工作原理、性能特点及各自的优缺点。
- **核心技能**: 熟练掌握 `mdadm` 命令行工具，能够独立创建、管理、监控和停止 RAID 0/1 阵列。
- **风险认知**: 理解 RAID 写洞 (Write Hole) 问题的成因、影响，并了解其常见的解决方案。
- **工程思维**: 能够从成本、性能和数据安全角度，为不同的业务场景选择合适的 RAID 级别。

## 📚 理论基础 (30%)

### 1. RAID 核心概念

RAID (Redundant Array of Independent Disks) 是一种将多个独立的物理磁盘组合成一个或多个逻辑单元的存储技术，旨在提升性能、增加冗余或两者兼备。

#### RAID 0: 条带化 (Striping)
- **工作原理**: 数据被分割成块（Chunk），依次写入阵列中的各个磁盘。例如，数据块 1 写入磁盘 A，数据块 2 写入磁盘 B，数据块 3 写入磁盘 C，以此类推。
- **优点**:
    - **高性能**: 读写操作可以并行在多个磁盘上进行，理论读写速度是单个磁盘的 N 倍（N 为磁盘数量）。
- **缺点**:
    - **无冗余**: 阵列中任何一块磁盘损坏，将导致所有数据丢失。
- **企业级应用场景**:
    - 对性能要求极高，但对数据可靠性要求不高的场景，如视频剪辑的临时交换空间、科学计算的临时数据存储、数据库的索引分区等。

#### RAID 1: 镜像化 (Mirroring)
- **工作原理**: 数据被完全相同地写入两块或更多磁盘。所有磁盘互为镜像，存储完全相同的数据。
- **优点**:
    - **高冗余**: 只要阵列中还有一块磁盘正常，数据就不会丢失。可靠性是所有 RAID 级别中最高的。
    - **读性能好**: 读请求可以由阵列中任意一块磁盘来处理，理论上读性能可以翻倍。
- **缺点**:
    - **写性能略有下降**: 数据需要同时写入所有磁盘，写性能取决于最慢的那块盘。
    - **磁盘利用率低**: N 块磁盘组成的 RAID 1 阵列，可用容量只有单块磁盘的容量 (1/N)。
- **企业级应用场景**:
    - 对数据可靠性要求极高的场景，如操作系统盘、数据库日志文件、关键业务应用数据等。

### 2. RAID 写洞 (Write Hole) 问题
- **定义**: 在需要进行“读-改-写”操作的 RAID 级别（如 RAID 5/6）中，如果在更新数据块和其对应的校验块之间发生系统崩溃或断电，会导致数据和校验位不一致。当系统重启后，无法判断是数据块出错还是校验块出错，这就是“写洞”。
- **影响**: 如果此时恰好有另一块磁盘发生故障，进行数据重建时，RAID 控制器可能会使用错误的校验信息来计算并恢复数据，导致数据永久性损坏。
- **解决方案**:
    - **Journaling (日志)**: 在更新数据前，将操作意图写入日志区域，即使发生断电，重启后可通过日志恢复一致性。`mdadm` 支持此功能。
    - **BBU (Battery Backup Unit)**: 为 RAID 卡配备备用电池，断电后可将缓存中的数据写回磁盘。
    - **CoW (Copy-on-Write)**: 不直接修改旧数据，而是将新数据写入新位置，然后更新元数据指针。ZFS 等现代文件系统采用此机制，从根本上避免了写洞问题。

### 3. 企业级思考
- **TCO (总体拥有成本)**:
    - **RAID 0**: 硬件成本最低（100% 容量利用率），但数据丢失风险带来的潜在损失成本最高。
    - **RAID 1**: 硬件成本最高（50% 容量利用率），但提供了最高的数据保护，降低了因故障导致的业务中断成本。
- **故障域分析**:
    - **RAID 0**: 故障域是整个阵列。任何单点故障都会摧毁所有数据。
    - **RAID 1**: 故障域是单个磁盘。在 N 盘镜像中，可以容忍 N-1 块磁盘同时故障。

## 🛠️ ���践操作 (50%)

### 1. 环境准备
在开始前，请确保系统已安装 `mdadm`。如果没有，请使用包管理器安装。
```bash
# 对于 Debian/Ubuntu 系统
sudo apt-get update && sudo apt-get install mdadm -y

# 对于 CentOS/RHEL 系统
sudo yum install mdadm -y
```
为了不影响真实磁盘，我们使用 `truncate` 和 `losetup` 创建 4 个 1GB 的虚拟磁盘文件作为练习。
```bash
# 创建挂载点和虚拟磁盘存放目录
sudo mkdir -p /mnt/raid0 /mnt/raid1
sudo mkdir /opt/disks
cd /opt/disks

# 创建 4 个稀疏文件作为虚拟磁盘
sudo truncate -s 1G disk1.img
sudo truncate -s 1G disk2.img
sudo truncate -s 1G disk3.img
sudo truncate -s 1G disk4.img

# 将文件映射为块设备
sudo losetup /dev/loop1 disk1.img
sudo losetup /dev/loop2 disk2.img
sudo losetup /dev/loop3 disk3.img
sudo losetup /dev/loop4 disk4.img

# 验证块设备是否创建成功
ls /dev/loop*
```

### 2. 创建和管理 RAID 阵列

#### 创建 RAID 0
```bash
# 使用 /dev/loop1 和 /dev/loop2 创建一个 RAID 0 阵列 /dev/md0
# --create: 创建阵列
# --level=0: 指定 RAID 级别为 0
# --raid-devices=2: 指定使用 2 个设备
echo "yes" | sudo mdadm --create /dev/md0 --level=0 --raid-devices=2 /dev/loop1 /dev/loop2
```

#### 创建 RAID 1
```bash
# 使用 /dev/loop3 和 /dev/loop4 创建一个 RAID 1 阵列 /dev/md1
echo "yes" | sudo mdadm --create /dev/md1 --level=1 --raid-devices=2 /dev/loop3 /dev/loop4
```

### 3. 查看与监控阵列状态
```bash
# 查看所有活动 RAID 阵列的简要状态
cat /proc/mdstat
# 预期输出 (md1 可能正在同步 resyncing):
# Personalities : [raid1] [raid0]
# md1 : active raid1 loop4[1] loop3[0]
#       1047552 blocks super 1.2 [2/2] [UU]
# md0 : active raid0 loop2[1] loop1[0]
#       2097152 blocks super 1.2 512k chunks

# 查看特定阵列的详细信息
sudo mdadm --detail /dev/md0
# 重点关注 State: active, Active Devices, Working Devices

sudo mdadm --detail /dev/md1
# 重点关注 State: active, clean (同步完成后)
```

### 4. 格式化、挂载与测试
```bash
# 格式化为 ext4 文件系统
sudo mkfs.ext4 /dev/md0
sudo mkfs.ext4 /dev/md1

# 挂载到指定目录
sudo mount /dev/md0 /mnt/raid0
sudo mount /dev/md1 /mnt/raid1

# 验证挂载和读写
df -h /mnt/raid*
# 可以看到 /mnt/raid0 的容量约 2G, /mnt/raid1 的容量约 1G

# 写入测试文件
sudo sh -c "echo 'Hello RAID 0' > /mnt/raid0/test.txt"
sudo sh -c "echo 'Hello RAID 1' > /mnt/raid1/test.txt"

# 读取验证
cat /mnt/raid0/test.txt
cat /mnt/raid1/test.txt
```

### 5. 停止与重组阵列
```bash
# 停止阵列前必须先卸载
sudo umount /mnt/raid0
sudo umount /mnt/raid1

# 停止阵列
sudo mdadm --stop /dev/md0
sudo mdadm --stop /dev/md1

# 验证阵列已停止
cat /proc/mdstat

# 自动重组 (assemble) 阵列
# mdadm 会扫描设备并根据元数据自动重组
sudo mdadm --assemble --scan

# 验证阵列已恢复
cat /proc/mdstat
sudo mdadm --detail /dev/md0
```

## 💻 Go 编程实现 (10%)

作为入门，我们编写一个简单的 Go 程序来读取并显示 `/proc/mdstat` 的内容。这是所有监控脚本的基础。

**`raid_status_checker.go`**
```go
package main

import (
	"fmt"
	"io/ioutil"
	"log"
)

// getMDStatContent reads the content of /proc/mdstat file.
// This file provides real-time status of Linux software RAID arrays.
func getMDStatContent() (string, error) {
	content, err := ioutil.ReadFile("/proc/mdstat")
	if err != nil {
		return "", fmt.Errorf("failed to read /proc/mdstat: %w", err)
	}
	return string(content), nil
}

func main() {
	fmt.Println("--- Linux Software RAID Status ---")
	status, err := getMDStatContent()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	fmt.Println(status)
	fmt.Println("----------------------------------")
	fmt.Println("Tip: 'UU' indicates all devices are up. '[U_]' indicates a degraded array.")
}
```

**如何运行:**
1. 保存代码为 `raid_status_checker.go`。
2. 在终端中执行 `go run raid_status_checker.go`。
3. 程序将输出当前系统中所有 RAID 阵列的状态。

这个简单的脚本展示了如何用 Go 与系统底层文件交互来获取关键信息，是后续开发更复杂监控工具的第一步。

## 🔍 故障排查与优化
- **常见问题**:
    - **`mdadm: device /dev/loopX busy.`**: 设备可能已被挂载或用于其他 RAID。请使用 `lsblk` 或 `df -h` 检查。
    - **阵列无法重组**: 确保所有成员盘都存在且未被占用。使用 `mdadm --examine /dev/loopX` 查看磁盘元数据。
- **最佳实践**:
    - **保存配置**: 创建阵列后，将配置保存到 `mdadm.conf`，以便系统启动时自动加载。
      ```bash
      sudo mdadm --detail --scan | sudo tee -a /etc/mdadm/mdadm.conf
      # 在 Debian/Ubuntu 上，可能还需要更新 initramfs
      sudo update-initramfs -u
      ```
    - **使用 UUID 挂载**: 在 `/etc/fstab` 中使用设备的 UUID 而不是 `/dev/mdX` 名称来挂载，因为设备名可能在重启后改变。
      ```bash
      # 获取 UUID
      sudo blkid /dev/md0
      # 添加到 /etc/fstab
      # UUID=... /mnt/raid0 ext4 defaults 0 2
      ```

## 📝 实战项目
1. **创建新阵列**: 使用 3 个新的虚拟磁盘创建一个名为 `/dev/md2` 的 RAID 0 ���列。
2. **格式化与挂载**: 将其格式化为 `xfs` 文件系统，并挂载到 `/mnt/raid0_extra`。
3. **持久化配置**: 将新阵列的配置追加到 `/etc/mdadm/mdadm.conf` 并更新 `/etc/fstab` 以实现开机自动挂载。
4. **验证**: 重启系统（或模拟重启：停止阵列后执行 `mdadm -A --scan`），验证阵列和挂载点是否都自动恢复正常。

## 🏠 课后作业
1. **深入阅读**: 使用 `man mdadm` 和 `man mdadm.conf` 详细阅读 `mdadm` 的手册页，特别是关于 `CREATE`, `MANAGE`, `MISC` 模式的选项。
2. **文档总结**: 撰写一份 Markdown 文档，详细对比 RAID 0 和 RAID 1 的性能测试数据（可以使用 `fio` 或 `dd` 进行简单测试）、成本和各自最适合的 3 种业务场景，并解释原因。
3. **清理环境**: 练习结束后，记得停止并移除阵列，然后卸除 loop 设备。
   ```bash
   sudo mdadm --stop /dev/md0 /dev/md1
   sudo mdadm --remove /dev/md0 /dev/md1 # 移除元数据
   sudo losetup -d /dev/loop1 /dev/loop2 /dev/loop3 /dev/loop4
   ```
