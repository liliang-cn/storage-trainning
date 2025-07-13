# Day 3: LV 创建与复杂卷类型实践

## 🎯 学习目标

- **明确的技能目标**:
  - 掌握标准线性逻辑卷（Linear LV）的创建、格式化、挂载和使用全流程。
  - 深入理解并能亲手创建两种核心的高级卷类型：条带卷（Striped LV）以提升性能，镜像卷（Mirrored LV）以保证数据冗余。
  - 学会使用 `fio` 工具对不同类型的卷进行基础性能测试，用数据验证理论。
- **具体的成果产出**:
  - 在卷组中成功创建并挂载了线性、条带、镜像三种类型的逻辑卷。
  - **扩展我们的 Go 语言 `lvm-manager` 工具，使其具备查询逻辑卷（LV）详细信息的能力，并封装一个用于创建标准逻辑卷的 Go 函数。**

## 📚 理论基础 (30-40%)

- **核心概念深度解析**:
  1.  **逻辑卷 (Logical Volume - LV)**: 这是我们与 LVM 交互的最终产物。LV 从 VG 中“借用”PEs 来构成自己的存储空间。对操作系统而言，一个 LV（如 `/dev/vg_data_01/lv_web`）看起来就像一个普通的块设备（如硬盘分区 `/dev/sda1`），我们可以对其进行格式化和挂载。

- **系统原理和架构设计**:
  1.  **线性卷 (Linear LV)**: 这是最基础、最常用的 LV 类型。当你创建一个线性卷时，LVM 会按顺序从 VG 内的一个或多个 PV 中分配所需数量的 PE。它会优先用完一个 PV 上的空闲 PE，然后再从下一个 PV 分配。这是实现“将多个小硬盘合并成一个大分区”这一功能的基础。
  2.  **条带卷 (Striped LV)**:
     - **原理**: 数据不是连续写入单个 PV，而是被分割成数据块（Chunk），以“条带”（Stripe）的方式**并行写入**到多个指定的 PV 上。例如，一个 1MB 的文件，在双路条带卷上，可能会将前 64KB 写入 PV1，第二个 64KB 写入 PV2，第三个 64KB 再写入 PV1，以此类推。
     - **优势**: **极大地提升顺序读写性能**。因为 I/O 操作被分散到多个物理磁盘上同时进行，突破了单个磁盘的带宽瓶颈。
     - **劣势**: **毫无冗余性**。条带卷中的任何一块 PV 损坏，都会导致整个 LV 的数据全部丢失，其可靠性低于任何单个成员磁盘。
  3.  **镜像卷 (Mirrored LV)**:
     - **原理**: 数据会被**同时写入**到多个（通常是2个）PV 上，形成完全相同的副本。LVM 会确保两个副本的数据一致性。
     - **优势**: **极高的数据冗余**。当其中一个 PV 发生故障时，数据不会丢失，系统可以无缝地从另一个正常的 PV 副本上读取数据，保证了业务的连续性。
     - **劣势**: **写入性能下降**（因为需要同时写多份），以及 **50% 的空间成本**（1TB 的可用空间需要 2TB 的物理磁盘）。

- **企业级应用场景分析**:
  - **线性卷**: 通用场景，如用户主目录、应用软件安装目录等对性能和冗余没有极端要求的场合。
  - **条带卷**: **高性能计算、视频编辑、数据仓库**等需要处理大量连续大文件的场景，追求极致的读写速度。
  - **镜像卷**: **核心数据库、关键业务应用**等对数据可靠性要求极高，不容许因单盘故障而中断服务的场景。

## 🛠️ 实践操作 (40-50%)

我们将使用 Day 2 创建的 `vg_data_01`（包含 `/dev/sdb`, `/dev/sdc`）和另外两块裸盘 `/dev/sdd`, `/dev/sde`。

**准备工作**: 将 `/dev/sdd` 和 `/dev/sde` 也初始化为 PV，并创建一个新的 VG 用于镜像实验。
```bash
sudo pvcreate /dev/sdd /dev/sde
# 为镜像卷创建一个专用的 VG，更符合生产规范
sudo vgcreate vg_safe_01 /dev/sdd /dev/sde
```

### 1. 创建并使用标准线性卷

```bash
# 1. 从 vg_data_01 中创建一个 2GB 大小的线性 LV，命名为 lv_linear_data
sudo lvcreate -L 2G -n lv_linear_data vg_data_01

# 2. 格式化为 ext4 文件系统
sudo mkfs.ext4 /dev/vg_data_01/lv_linear_data

# 3. 创建挂载点并挂载
sudo mkdir -p /mnt/linear_data
sudo mount /dev/vg_data_01/lv_linear_data /mnt/linear_data

# 4. 验证
df -hT /mnt/linear_data
# 预期能看到挂载信息，大小约为 2G，类型为 ext4
```

### 2. 创建并测试条带卷

`vg_data_01` 有两个 PV，正好可以用来创建双路条带。

```bash
# 1. 创建一个 4GB 的双路条带卷，条带大小(Stripe Size)为 64KB
# -i 2: 指定使用 2 个 PV 做条带 (stripes)
# -I 64: 指定条带大小为 64KB (Stripe Size)
sudo lvcreate -L 4G -i 2 -I 64 -n lv_striped_data vg_data_01

# 2. 格式化并挂载
sudo mkfs.ext4 /dev/vg_data_01/lv_striped_data
sudo mkdir -p /mnt/striped_data
sudo mount /dev/vg_data_01/lv_striped_data /mnt/striped_data

# 3. 验证
df -hT /mnt/striped_data
```

### 3. 创建并验证镜像卷

我们将使用 `vg_safe_01` 来创建镜像卷。

```bash
# 1. 创建一个 2GB 的镜像卷
# -m 1: 指定需要 1 个镜像副本，加上原始数据，共需要 2 个 PV
# --mirrorlog core: 指定镜像日志在内存中，性能较高但重启后需完全同步
sudo lvcreate -L 2G -m 1 --mirrorlog core -n lv_mirrored_data vg_safe_01

# 2. 格式化并挂载
sudo mkfs.ext4 /dev/vg_safe_01/lv_mirrored_data
sudo mkdir -p /mnt/mirrored_data
sudo mount /dev/vg_safe_01/lv_mirrored_data /mnt/mirrored_data

# 3. 验证
df -hT /mnt/mirrored_data
# 查看 LV 状态，可以看到其布局
sudo lvs -o +devices vg_safe_01
# 预期输出会显示 lv_mirrored_data 使用了两个设备
```

### 4. 基础性能对比测试

我们将使用 `fio` 工具来简单对比一下线性卷和条带卷的顺序写性能。

```bash
# 1. 安装 fio
# CentOS/RHEL: sudo dnf install -y fio
# Ubuntu/Debian: sudo apt-get install -y fio

# 2. 测试线性卷
sudo fio --name=linear_write --directory=/mnt/linear_data --size=500M --direct=1 --rw=write --bs=1M --ioengine=libaio --runtime=20 --group_reporting

# 3. 测试条带卷
sudo fio --name=striped_write --directory=/mnt/striped_data --size=500M --direct=1 --rw=write --bs=1M --ioengine=libaio --runtime=20 --group_reporting

# 4. 观察结果
# 重点关注 fio 输出中的 bw (Bandwidth) 一项，你会发现条带卷的写入带宽明显高于线性卷。
```

## 💻 Go 编程实现 (20-30%)

**任务**: 扩展我们的 `lvm-manager`，增加查询 LV 的功能，并封装一个创建线性 LV 的函数。

**项目准备**:
```bash
# 确保在 lvm-manager 目录下
mkdir -p cmd/day03
cd cmd/day03
```

**代码 (`main.go`)**:
```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// --- 复用 Day 2 的结构体和函数 ---
type LVMReport struct {
	Report []map[string][]map[string]string `json:"report"`
}
// ... (此处省略 Day 2 的 PhysicalVolume, VolumeGroup 结构体及 runLVMCommand, GetPhysicalVolumes, GetVolumeGroups 函数，实际编码时应将它们放在公共包中)

// --- 新增 LogicalVolume 结构体 ---
type LogicalVolume struct {
	Name   string `json:"lv_name"`
	VG     string `json:"vg_name"`
	Attr   string `json:"lv_attr"`
	Size   string `json:"lv_size"`
	Origin string `json:"origin"` // For snapshots
	Path   string `json:"lv_path"`
}

// --- 新增 GetLogicalVolumes 函数 ---
func GetLogicalVolumes() ([]LogicalVolume, error) {
	// -o 添加 lv_attr,lv_path 等字段
	output, err := runLVMCommand("lvs", "-o", "lv_name,vg_name,lv_attr,lv_size,origin,lv_path")
	if err != nil {
		return nil, err
	}

	var report LVMReport
	if err := json.Unmarshal(output, &report); err != nil {
		return nil, fmt.Errorf("failed to parse lvs JSON: %v", err)
	}

	var lvs []LogicalVolume
	if len(report.Report) > 0 && report.Report[0]["lv"] != nil {
		for _, lvMap := range report.Report[0]["lv"] {
			lvs = append(lvs, LogicalVolume{
				Name:   lvMap["lv_name"],
				VG:     lvMap["vg_name"],
				Attr:   lvMap["lv_attr"],
				Size:   lvMap["lv_size"],
				Origin: lvMap["origin"],
				Path:   lvMap["lv_path"],
			})
		}
	}
	return lvs, nil
}

// --- 新增 CreateLinearLV 函数 ---
// CreateLinearLV creates a standard linear logical volume.
// size is in Gigabytes (G).
func CreateLinearLV(vgName, lvName string, sizeG int) error {
	sizeStr := fmt.Sprintf("%dG", sizeG)
	log.Printf("Attempting to create LV: Name=%s, VG=%s, Size=%s", lvName, vgName, sizeStr)
	
	// 使用 -L 指定大小，-n 指定名称
	_, err := runLVMCommand("lvcreate", "-L", sizeStr, "-n", lvName, vgName)
	if err != nil {
		return fmt.Errorf("failed to create linear LV %s in VG %s: %w", lvName, vgName, err)
	}
	
	log.Printf("Successfully created LV %s.", lvName)
	return nil
}

// runLVMCommand (从 Day 2 复制过来)
func runLVMCommand(command string, args ...string) ([]byte, error) {
	fullArgs := append([]string{command}, args...)
	// lvcreate 不支持 reportformat json, 所以需要特殊处理
	if command != "lvcreate" {
	    fullArgs = append(fullArgs, "--reportformat", "json")
    }
	cmd := exec.Command("sudo", fullArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("command `sudo %s %s` failed: %v\nStderr: %s", command, strings.Join(args, " "), err, stderr.String())
	}
	return stdout.Bytes(), nil
}


func main() {
	log.Println("--- Phase 1: Creating a new LV with Go ---")
	// 演示创建 LV
	err := CreateLinearLV("vg_data_01", "lv_from_go", 1)
	if err != nil {
		log.Printf("WARN: Could not create lv_from_go: %v. It might already exist.", err)
	}

	log.Println("\n--- Phase 2: Fetching LVM logical volumes ---")
	lvs, err := GetLogicalVolumes()
	if err != nil {
		log.Fatalf("Error getting logical volumes: %v", err)
	}

	fmt.Println("\n--- Logical Volumes (LVs) ---")
	fmt.Printf("%-20s %-15s %-12s %-10s %-s\n", "LV Path", "VG Name", "Attributes", "Size", "Origin")
	fmt.Println(strings.Repeat("-", 80))
	for _, lv := range lvs {
		origin := lv.Origin
		if origin == "" {
			origin = "-"
		}
		fmt.Printf("%-20s %-15s %-12s %-10s %-s\n", lv.Path, lv.VG, lv.Attr, lv.Size, origin)
	}
	log.Println("LVM information fetched successfully.")
}
```

## 🔍 故障排查与优化

- **常见问题诊断**:
  - **问题**: `lvcreate` 报错 "Volume group \"vg_data_01\" has insufficient free space"。
    - **诊断**: 卷组中剩余的 PE 数量不足以创建指定大小的 LV。使用 `vgs` 或 `vgdisplay` 查看 `VFree` 字段。
    - **解决**: 减小要创建的 LV 的大小，或使用 `vgextend` 为该 VG 添加新的 PV 以扩容。
  - **问题**: `lvcreate -i 2` 报错 "Cannot create striped LV with only 1 PVs"。
    - **诊断**: 创建 N 路条带卷，至少需要 N 个 PV。
    - **解决**: 确保你的 VG 中有足够数量的 PV。
- **性能优化建议**:
  - **条带大小 (Stripe Size)**: 这是一个重要的调优参数。对于数据库等小文件、随机 I/O 密集的应用，较小的条带（如 16K 或 32K）可能更好。对于视频存储、备份等大文件、顺序 I/O 为主的应用，较大的条带（如 256K 或 512K）能提供更好的性能。**没有万能的配置，必须根据业务场景测试。**
- **最佳实践总结**:
  - **专卷专用**: 不要在一个 LV 上混合存放多种不同 I/O 特征的应用数据。为数据库、日志、Web 文件等创建各自独立的 LV，便于分别管理、扩容和做性能优化。
  - **对齐 (Alignment)**: 虽然现代 LVM 和文件系统能很好地处理对齐，但在要求极致性能的场景，仍需确保分区、PV、LV、文件系统的块大小都经过精心设计和对齐，避免 I/O 跨越物理扇区边界导致性能下降。

## 📝 实战项目

- **综合应用练习**: 扩展今天的 Go 程序，实现一个更智能的 `CreateLV` 函数。
- **项目目标**: `func CreateLV(vgName, lvName string, sizeG int, lvType string, stripes int) error`
  - `lvType` 可以是 "linear", "striped"。
  - 当 `lvType` 是 "striped" 时，`stripes` 参数生效。
  - 函数内部根据 `lvType` 动态构建 `lvcreate` 命令的参数列表。
  - 在执行创建前，调用 `GetVolumeGroups` 和 `GetPhysicalVolumes` 函数进行预检查：
    - 检查 VG 是否存在。
    - 检查 VG 剩余空间是否足够。
    - 如果是创建条带卷，检查 VG 内的 PV 数量是否满足条带数要求。
  - 预检查失败则返回有意义的错误信息，而不是直接执行命令让它失败。

## 🏠 课后作业

- **扩展练习任务**:
  1.  **镜像管理**: 模拟一次镜像卷的磁盘故障。
      - 使用 `lvchange` 或其他工具让 `vg_safe_01` 中的一个 PV（如 `/dev/sdd`）暂时失效。
      - 运行 `lvs -a -o +devices` 查看镜像状态，你会看到状态变为 "degraded"。
      - 验证此时 `/mnt/mirrored_data` 依然可以读写。
      - 模拟修复磁盘后，使用 `vgchange` 和 `lvconvert --repair` 来恢复镜像的健康状态。
  2.  **Go 工具增强**:
      - 为你的 `lvm-manager` 添加一个 `list` 子命令，`list` 后面可以跟 `pv`, `vg`, `lv`。例如 `go run main.go list lv` 就只显示 LV 信息。
      - **提示**: 使用 `os.Args` 来解析命令行参数，或研究 `flag` 包的子命令功能。
