# Day 2: 高级 RAID 模式与故障恢复

## 🎯 学习目标
- **核心技能**: 掌握 RAID 5 (分布式奇偶校验) 和 RAID 10 (镜像条带) 的创建、配置与管理方法。
- **运维实战**: 能够熟练模拟 RAID 阵列的磁盘故障，并完成添加热备、自动重建、替换坏盘等一系列恢复流程。
- **性能分析**: 学会使用专业的 I/O 测试工具 `fio`，对不同 RAID 级别进行性能基准测试，并能解读测试结果。
- **Go 编程进阶**: 开发一个更智能的 Go 程序，用于解析 `mdadm` 的输出，并以结构化的方式报告阵列的健康状况。

## 📚 理论基础 (30%)

### 1. 高级 RAID 模式

#### RAID 5: 分布式奇偶校验 (Distributed Parity)
- **工作原理**: 数据以条带化方式写入多个磁盘，同时将校验信息（Parity）分布存储在所有成员磁盘上。对于 N 块盘的 RAID 5，每次写入 N-1 个数据块和 1 个校验块。校验块是通过对其他 N-1 个数据块进行异或（XOR）运算得到的。
- **优点**:
    - **良好的平衡**: 在性能、容量和冗余之间取得了很好的平衡。
    - **高容量利用率**: 容量为 (N-1) * 单盘容量，利用率较高。
- **缺点**:
    - **写性能惩罚 (Write Penalty)**: 每次写入都需要“读取旧数据 -> 读取旧校验 -> 计算新校验 -> 写入新数据 -> 写入新校验”这几个步骤，写操作相对复杂，性能较低。
    - **重建速度慢且有风险**: 当一块盘故障后，重建过程需要读取所有其他磁盘的数据来计算恢复，对系统 I/O 压力大，耗时长。在重建期间如果再坏一块盘，数据将全部丢失。
- **企业级应用场景**:
    - 读多写少的应用，如文件服务器、Web 服务器、数据归档等。

#### RAID 10 (RAID 1+0): 镜像与条带的结合 (A Stripe of Mirrors)
- **工作原理**: 先将磁盘两两配对做成 RAID 1 镜像组，然后再将这些镜像组做成一个 RAID 0 条带。至少需要 4 块磁盘。
- **优点**:
    - **高性能与高冗余**: 兼具 RAID 0 的高读写性能和 RAID 1 的高数据安全性。
    - **快速重建**: 坏盘后，只需从同一镜像组的另一块好盘上复制数据即可，重建速度快，对系统性能影响小。
- **缺点**:
    - **磁盘利用率低**: 只有 50%，与 RAID 1 相同，成本较高。
- **企业级应用场景**:
    - 对性能和可靠性都有极高要求的场景，如数据库、虚拟机存储、高负载应用服务器等。

### 2. 热备盘 (Hot Spare)
- **定义**: 一块或多块处于待命状态的备用磁盘，它已连接到系统中，但不参与正常的数据读写。
- **工作原理**: 当 RAID 阵列中某块磁盘发生故障时，RAID 控制器（或 `mdadm`）会自动激活热备盘，将其加入阵列，并开始进行数据重建，从而实现故障的快速、自动恢复。
- **价值**: 极大地缩短了从故障发生到开始恢复之间的时间窗口，降低了在降级（degraded）状态下运行的风险，是提升运维自动化和系统可用性的重要手段。

## 🛠️ 实践操作 (50%)

### 1. 环境准备
我们需要更多的虚拟磁盘。我们将创建 8 个 loop 设备用于接下来的实验。
```bash
# 清理并准备新环境
sudo umount /mnt/raid* || true
sudo mdadm --stop /dev/md* || true
sudo losetup -d /dev/loop* || true
cd /opt/disks
sudo rm *.img

# 创建 8 个 1GB 的虚拟磁盘文件
for i in {1..8}; do sudo truncate -s 1G disk${i}.img; done

# 将文件映射为块设备
for i in {1..8}; do sudo losetup /dev/loop${i} disk${i}.img; done

# 验证
ls /dev/loop*
```

### 2. 创建 RAID 5 和 RAID 10

#### 创建 RAID 5
```bash
# 使用 3 块盘创建 RAID 5 (/dev/loop1, loop2, loop3)
# 至少需要 3 块盘
echo "yes" | sudo mdadm --create /dev/md2 --level=5 --raid-devices=3 /dev/loop1 /dev/loop2 /dev/loop3
```

#### 创建 RAID 10
```bash
# 使用 4 块盘创建 RAID 10 (/dev/loop4, loop5, loop6, loop7)
# 至少需要 4 块盘，且为偶数
echo "yes" | sudo mdadm --create /dev/md3 --level=10 --raid-devices=4 /dev/loop4 /dev/loop5 /dev/loop6 /dev/loop7
```

### 3. 故障模拟与恢复 (以 RAID 5 为例)

#### a. 添加热备盘
```bash
# 将 /dev/loop8 添加为 /dev/md2 的热备盘
sudo mdadm /dev/md2 --add /dev/loop8

# 查看详细信息，确认热备盘状态
sudo mdadm --detail /dev/md2
# 在末尾会看到 /dev/loop8 的状态为 spare
```

#### b. 模拟磁盘故障
```bash
# 将 /dev/loop1 标记为故障 (faulty)
sudo mdadm /dev/md2 --fail /dev/loop1

# 观察自动重建过程
# 立即查看状态，会看到热备盘被激活，开始重建 (recovering)
watch cat /proc/mdstat
# 示例输出:
# md2 : active raid5 loop8[3](S) loop3[2] loop2[1] loop1[0](F)
# ...
# [UU_]
# recovery = 1.2% (12345/1047552) finish=1.0min speed=16460K/sec
```

#### c. 替换故障盘
重建完成后，阵列会恢复 `active` 状态，但故障盘 `(F)` 依然在阵列信息中。我们需要手动移除它。
```bash
# 移除故障盘
sudo mdadm /dev/md2 --remove /dev/loop1

# 假设 /dev/loop1 已经被物理替换或修复，我们可以把它重新加回阵列
# 它会成为新的热备盘
sudo mdadm /dev/md2 --add /dev/loop1

# 最终查看状态
sudo mdadm --detail /dev/md2
# 此时阵列由 loop2, loop3, loop8 组成，loop1 成为新的热备盘
```

### 4. 性能基准测试
`fio` 是一个功能强大的 I/O 压力测试工具。
```bash
# 安装 fio
sudo apt-get install fio -y || sudo yum install fio -y

# 格式化并挂载阵列
sudo mkfs.ext4 /dev/md2
sudo mkfs.ext4 /dev/md3
sudo mkdir -p /mnt/raid5 /mnt/raid10
sudo mount /dev/md2 /mnt/raid5
sudo mount /dev/md3 /mnt/raid10

# 测试 RAID 5 顺序写性能
sudo fio --name=seqwrite --ioengine=libaio --direct=1 --bs=1M --size=256M --rw=write --directory=/mnt/raid5 --output=raid5-seqwrite.log

# 测试 RAID 5 随机读性能
sudo fio --name=randread --ioengine=libaio --direct=1 --bs=4k --size=256M --rw=randread --directory=/mnt/raid5 --output=raid5-randread.log

# 测试 RAID 10 顺序写性能
sudo fio --name=seqwrite --ioengine=libaio --direct=1 --bs=1M --size=256M --rw=write --directory=/mnt/raid10 --output=raid10-seqwrite.log

# 测试 RAID 10 随机读性能
sudo fio --name=randread --ioengine=libaio --direct=1 --bs=4k --size=256M --rw=randread --directory=/mnt/raid10 --output=raid10-randread.log

# 查看结果
cat raid5-seqwrite.log | grep "bw="
cat raid10-seqwrite.log | grep "bw="
# 对比不同 RAID 级别的带宽 (bw) 和 IOPS
```

## 💻 Go 编程实现 (10%)

我们来升级昨天的脚本。这个新版本将调用 `mdadm --detail` 并用正则表达式解析出关键信息，如阵列状态、设备总数、活动设备数等。

**`raid_parser.go`**
```go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// RaidInfo holds structured information about a RAID array.
type RaidInfo struct {
	Device          string
	State           string
	TotalDevices    int
	ActiveDevices   int
	WorkingDevices  int
	FailedDevices   int
	SpareDevices    int
	IsHealthy       bool
}

// parseMdadmDetail parses the output of `mdadm --detail [device]`.
func parseMdadmDetail(output string) (*RaidInfo, error) {
	info := &RaidInfo{}
	
	// Regex to find key-value pairs
	re := regexp.MustCompile(`\s*(?P<Key>[^:]+?)\s*:\s*(?P<Value>.+)`)
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) < 3 {
			continue
		}
		key := strings.TrimSpace(matches[1])
		value := strings.TrimSpace(matches[2])

		switch key {
		case "State":
			info.State = value
		case "Total Devices":
			info.TotalDevices, _ = strconv.Atoi(value)
		case "Active Devices":
			info.ActiveDevices, _ = strconv.Atoi(value)
		case "Working Devices":
			info.WorkingDevices, _ = strconv.Atoi(value)
		case "Failed Devices":
			info.FailedDevices, _ = strconv.Atoi(value)
		case "Spare Devices":
			info.SpareDevices, _ = strconv.Atoi(value)
		}
	}
    
    // Determine health
    info.IsHealthy = strings.Contains(info.State, "active") && info.FailedDevices == 0 && info.ActiveDevices == info.WorkingDevices

	return info, nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: go run %s <raid_device_path (e.g., /dev/md2)>", os.Args[0])
	}
	raidDevice := os.Args[1]
	info.Device = raidDevice

	cmd := exec.Command("sudo", "mdadm", "--detail", raidDevice)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to execute mdadm for %s: %v\nOutput: %s", raidDevice, err, string(out))
	}

	raidInfo, err := parseMdadmDetail(string(out))
	if err != nil {
		log.Fatalf("Failed to parse mdadm output: %v", err)
	}
    raidInfo.Device = raidDevice

	fmt.Printf("--- RAID Status for %s ---\n", raidInfo.Device)
	fmt.Printf("State: %s\n", raidInfo.State)
	fmt.Printf("Is Healthy: %t\n", raidInfo.IsHealthy)
	fmt.Printf("Total/Active/Working/Failed/Spare: %d/%d/%d/%d/%d\n",
		raidInfo.TotalDevices, raidInfo.ActiveDevices, raidInfo.WorkingDevices, raidInfo.FailedDevices, raidInfo.SpareDevices)
	fmt.Println("-----------------------------")
}
```

**如何运行:**
1. 保存代码为 `raid_parser.go`。
2. 在终端中执行 `go run raid_parser.go /dev/md2`。
3. 程序将输出对 `/dev/md2` 状态的结构化分析。尝试在模拟故障的不同阶段运行此脚本，观察输出变化。

## 🔍 故障排查与优化
- **重建速度调优**: Linux 内核允许调整 RAID 的重建速度，以平衡业务 I/O 和恢复速度。
  ```bash
  # 查看当前速度限制 (min/max)
  cat /proc/sys/dev/raid/speed_limit_min
  cat /proc/sys/dev/raid/speed_limit_max

  # 临时提高最低重建速度 (例如到 50MB/s)
  echo 50000 | sudo tee /proc/sys/dev/raid/speed_limit_min
  ```
- **阵列降级 (`degraded`)**: 当阵列处于 `active, degraded` 状态时，意味着它仍在工作，但已失去冗余能力。此时应尽快替换故障盘，因为再有一次磁盘故障就可能导致数据全失。

## 📝 实战项目
1. **创建 RAID 10 阵列**: 使用 6 块虚拟磁盘创建一个 `/dev/md4` 的 RAID 10 阵列。
2. **故障演练**:
   - 为 `/dev/md4` 添加一个热备盘。
   - 模拟其中一块磁盘故障。
   - 验证热备盘是否自动接管并开始重建。
   - 使用你的 Go 程序 `raid_parser.go` 在故障前、故障中、重建后三个时间点检查阵列状态，并记录输出。
3. **文档记录**: 将上述过程的每一步命令、`mdadm --detail` 的关键输出以及 Go 程序的输出整理成一份操作报告。

## 🏠 课后作业
1. **Shell 脚本挑战**: 编写一个 Shell 脚本，功能与今天的 Go 程序类似，即接收一个 RAID 设备名作为参数，然后解析 `mdadm --detail` 的输出，最后以 "HEALTHY" 或 "DEGRADED" 或 "FAULTY" 的形式报告阵列的总体健康状况。
2. **性能对比报告**: 整理今天使用 `fio` 测试 RAID 5 和 RAID 10 的性能数据，并加入昨天 RAID 0 和 RAID 1 的测试结果。创建一个 Markdown 表格，从顺序读、顺序写、随机读、随机写四个维度对比四个 RAID 级别的性能，并简要分析数据差异的原因。
3. **清理环境**: 记得清理所有虚拟设备。
   ```bash
   sudo umount /mnt/* || true
   sudo mdadm --stop /dev/md* || true
   sudo losetup -d /dev/loop* || true
   ```
