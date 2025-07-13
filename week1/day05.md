# Day 5: LVM 扩容操作与 Go 自动化实践

## 🎯 学习目标
- **技能目标**: 熟练掌握 LVM 逻辑卷和文件系统的在线扩容技术，理解其底层原理和安全操作规范。
- **实践目标**: 能够独立完成对 ext4 和 xfs 文件系统的在线扩容，并处理扩容过程中的常见问题。
- **Go编程目标**: 开发一个企业级的 Go 自动化工具，实现对逻辑卷使用率的监控，并根据预设阈值自动执行扩容操作。
- **运维能力目标**: 建立一套完整的 LVM 健康检查和报告体系，模拟并处理磁盘满载、元数据损坏等典型故障。
- **成果产出**: 一个功能完整的 Go LVM 管理工具、一份 LVM 故障排查手册、一份 LVM 性能调优文档。

## 📚 理论基础 (30%)

### 1. 在线扩容原理
在线扩容（Online Resizing）是指在不卸载文件系统、不中断业务的情况下，动态增加存储容量。LVM 的分层架构使其天然支持此功能。

- **LVM 层面**: `lvextend` 命令负责扩展逻辑卷（LV）的容量。它会从卷组（VG）的空闲物理区（PE）中分配新的 PE 给目标 LV。这个过程只修改 LVM 的元数据，告诉系统这个 LV 现在拥有更多的块设备空间，但文件系统对此一无所知。
- **文件系统层面**: `resize2fs` (for ext4) 或 `xfs_growfs` (for XFS) 命令负责扩展文件系统。它会读取文件系统自身的元数据，识别到逻辑卷变大了，然后将文件系统的边界扩展到与逻辑卷大小一致，从而让操作系统能够真正使用这些新增的空间。

### 2. 企业级扩容流程与风险
在生产环境中，任何扩容操作都必须谨慎。
- **风险**:
    - **VG 空间不足**: 扩容前未检查 VG 剩余空间，导致扩容失败。
    - **命令误用**: 对错误的 LV 或文件系统执行扩容。
    - **文件系统损坏**: 在扩容过程中遭遇意外断电或系统崩溃，可能导致文件系统元数据不一致。
- **安全流程**:
    1. **备份**: 在任何重要操作前，对数据和 LVM 元数据进行备份 (`vgcfgbackup`)。
    2. **检查**: 确认 VG 中有足够的空闲空间 (`vgdisplay`)。
    3. **扩容LV**: 执行 `lvextend`。
    4. **检查文件系统**: (可选但推荐) `e2fsck -f` (ext4) 检查文件系统一致性。
    5. **扩容文件系统**: 执行 `resize2fs` 或 `xfs_growfs`。
    6. **验证**: 使用 `df -h` 确认容量已更新。

## 🛠️ 实践操作 (40%)

假设我们有一个名为 `data_lv` 的逻辑卷，挂载在 `/data` 目录，其所在的 VG 为 `storage_vg`。

### 1. 检查当前状态
```bash
# 检查卷组剩余空间
sudo vgdisplay storage_vg | grep "Free  PE"

# 检查逻辑卷和文件系统当前大小
df -hT /data
```

### 2. 逻辑卷扩容 (lvextend)
```bash
# 将 data_lv 的容量增加 2GB
sudo lvextend -L +2G /dev/storage_vg/data_lv

# 或者，直接扩容到指定大小，例如 10GB
# sudo lvextend -L 10G /dev/storage_vg/data_lv

# 验证 LV 大小是否已改变
sudo lvdisplay /dev/storage_vg/data_lv
```
**预期输出**: `lvextend` 会提示 "Size of logical volume ... changed from X to Y. Logical volume ... successfully resized."。`lvdisplay` 会显示新的 LV Size。此时 `df -h` 看到的大小**不变**。

### 3. 文件系统扩展

#### 针对 ext4 文件系统
```bash
# 检查文件系统以确保一致性（建议在非高峰期操作）
sudo e2fsck -f /dev/storage_vg/data_lv

# 在线扩展文件系统以使用所有可用空间
sudo resize2fs /dev/storage_vg/data_lv
```
**预期输出**: `resize2fs` 会显示文件系统从多少个块增长到多少个块。

#### 针对 XFS 文件系统
XFS 的扩容工具是 `xfs_growfs`，它不需要事先检查，且只能扩展到挂载点。
```bash
# XFS 扩容非常简单，直接指定挂载点
sudo xfs_growfs /data
```
**预期输出**: `xfs_growfs` 会报告数据块从旧值变为新值。

### 4. 最终验证
```bash
# 再次检查文件系统大小，确认扩容成功
df -hT /data
```
**预期输出**: `df -h` 现在应该显示增加后的总容量。

## 💻 Go 编程实现 (30%)

我们将开发一个 `lvm-autoscaler` 工具，它监控指定 LV 的使用率，并在超过阈值时自动扩容。

### 1. 项目结构
```
lvm-manager/
├── cmd/
│   └── main.go
├── internal/
│   ├── lvm/
│   │   └── lvm.go
│   └── monitor/
│       └── monitor.go
├── pkg/
│   └── utils/
│       └── exec.go
└── configs/
    └── config.yaml
```

### 2. 配置文件 `configs/config.yaml`
```yaml
monitor:
  interval_seconds: 60
  targets:
    - lv_path: "/dev/storage_vg/data_lv"
      mount_point: "/data"
      threshold_percent: 80
      increment_gb: 2
```

### 3. 系统命令执行器 `pkg/utils/exec.go`
```go
package utils

import (
	"bytes"
	"os/exec"
	"strings"
)

// RunCommand 执行一个 shell 命令并返回其输出
func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("command failed: %s
%s", err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}
```

### 4. LVM 核心功能 `internal/lvm/lvm.go`
```go
package lvm

import (
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"lvm-manager/pkg/utils"
)

// GetUsagePercent 获取挂载点的磁盘使用率
func GetUsagePercent(mountPoint string) (int, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs(mountPoint, &stat)
	if err != nil {
		return 0, fmt.Errorf("failed to get fs stats for %s: %w", mountPoint, err)
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	used := total - free
	
	return int(float64(used) / float64(total) * 100), nil
}

// ExtendLV 扩容逻辑卷
func ExtendLV(lvPath string, incrementGB int) error {
	_, err := utils.RunCommand("lvextend", "-L", fmt.Sprintf("+%dG", incrementGB), lvPath)
	return err
}

// ResizeFS 扩容文件系统
func ResizeFS(lvPath string) error {
    // 在实际应用中，这里需要判断文件系统类型 (ext4/xfs)
    // 为简化示例，我们假设是 ext4
	_, err := utils.RunCommand("resize2fs", lvPath)
	return err
}
```

### 5. 监控逻辑 `internal/monitor/monitor.go`
```go
package monitor

import (
	"fmt"
	"log"
	"time"
	"lvm-manager/internal/lvm"
)

type Target struct {
	LVPath           string `yaml:"lv_path"`
	MountPoint       string `yaml:"mount_point"`
	ThresholdPercent int    `yaml:"threshold_percent"`
	IncrementGB      int    `yaml:"increment_gb"`
}

func Start(targets []Target, interval time.Duration) {
	log.Println("Starting LVM auto-scaler...")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, target := range targets {
				checkAndScale(target)
			}
		}
	}
}

func checkAndScale(t Target) {
	usage, err := lvm.GetUsagePercent(t.MountPoint)
	if err != nil {
		log.Printf("ERROR: Failed to get usage for %s: %v", t.MountPoint, err)
		return
	}

	log.Printf("INFO: Usage for %s is %d%%", t.MountPoint, usage)

	if usage > t.ThresholdPercent {
		log.Printf("WARN: Usage %d%% > %d%%. Scaling up %s by %dGB.", usage, t.ThresholdPercent, t.LVPath, t.IncrementGB)
		
		if err := lvm.ExtendLV(t.LVPath, t.IncrementGB); err != nil {
			log.Printf("ERROR: Failed to extend LV %s: %v", t.LVPath, err)
			return
		}
		log.Printf("INFO: LV %s extended successfully.", t.LVPath)

		if err := lvm.ResizeFS(t.LVPath); err != nil {
			log.Printf("ERROR: Failed to resize filesystem for %s: %v", t.LVPath, err)
			return
		}
		log.Printf("INFO: Filesystem for %s resized successfully.", t.LVPath)
	}
}
```

## 🔍 故障排查与优化

### 1. 常见问题
- **`resize2fs: Bad magic number in super-block`**: 文件系统类型错误，可能不是 ext4。或者文件系统已损坏。
- **`lvextend: Insufficient free space`**: 卷组（VG）中没有足够的空闲 PE。需要先使用 `vgextend` 为 VG 添加新的物理卷（PV）。
- **扩容后 `df -h` 容量不变**: 忘记执行文件系统扩容步骤 (`resize2fs` 或 `xfs_growfs`)。

### 2. 优化建议
- **健康检查**: 在 Go 程序中，执行扩容前先调用 `vgdisplay` 检查 VG 剩余空间是否足够。
- **日志记录**: 将所有操作（检查、决策、执行结果）记录到结构化的日志文件（如 JSON 格式），而不仅仅是打印到控制台。
- **告警集成**: 在扩容成功或失败后，通过 Webhook、邮件等方式发送通知。

## 📝 实战项目

**目标**: 完善 `lvm-autoscaler` 工具，使其达到生产可用标准。

1. **完善文件系统识别**: 修改 `ResizeFS` 函数，使其能自动检测文件系统类型（ext4 或 xfs），并调用正确的扩容工具。
   - *提示*: 可以使用 `blkid -o value -s TYPE /dev/path` 命令获取文件系统类型。
2. **添加 Dry-Run 模式**: 增加一个命令行标志 `--dry-run`。在此模式下，程序只打印将要执行的操作，而不实际执行。
3. **编写单元测试**: 为 `internal/lvm` 包中的函数编写单元测试。由于这些函数依赖外部命令，需要使用 Mocking 技术模拟命令执行。
4. **生成报告**: 增加一个功能，定期生成 LVM 状态报告（HTML 或 Markdown），包含所有 VG/LV 的大小、使用率和健康状况。

## 🏠 课后作业

1. **故障模拟与恢复**:
   - **场景一**: 模拟 VG 空间耗尽，手动执行 `vgextend` 添加新磁盘，然后让 `lvm-autoscaler` 成功完成扩容。
   - **场景二**: 手动备份 LVM 元数据 (`vgcfgbackup`)，然后故意执行一次错误操作（如删除一个未使用的 LV），最后练习如何从备份中恢复 (`vgcfgrestore`)。
2. **编写技术文档**:
   - **LVM 故障排查手册**: 总结本周遇到的所有问题及其解决方案。
   - **LVM 性能调优文档**: 总结 Day 3 的性能测试结果，并给出不同场景下的卷类型（线性、条带、镜像）选择建议。
3. **代码交付**: 将完整的 `lvm-manager` Go 项目提交到代码仓库，包含完整的文档、测试用例和部署说明。
