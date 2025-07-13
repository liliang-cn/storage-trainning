# Day 2: PV 和 VG 核心操作与 Go 语言实践

## 🎯 学习目标

- **技能目标**:
  - 精通物理卷（PV）的创建、显示和检查 (`pvcreate`, `pvs`, `pvdisplay`)。
  - 精通卷组（VG）的创建、显示、扩展和重命名 (`vgcreate`, `vgs`, `vgdisplay`, `vgextend`)。
  - 理解物理扩展块（PE）在 LVM 空间管理中的核心作用。
- **具体成果产出**:
  - 成功将至少两块物理磁盘初始化为 LVM 物理卷。
  - 创建一个包含上述物理卷的卷组，并为其设置符合企业规范的名称。
  - **编写一个 Go 语言工具，该工具能以结构化（JSON）格式获取并展示系统中所有 PV 和 VG 的详细信息，为后续的自动化监控奠定基础。**

## 📚 理论基础

### 物理卷（PV）深度解析：一切的基石

昨天我们学习到，PV 是 LVM 的基础。今天我们深入一层：`pvcreate` 命令究竟做了什么？

它并非格式化，而是在块设备（如 `/dev/sdb`）的起始位置，写入一个被称为 **LVM 标签（LVM Label）** 的头部信息。这个标签包含了:
1.  **LVM 标识符**: 一个魔术字符串，声明“我是 LVM 设备”。
2.  **设备 UUID**: 一个为此 PV 生成的、全球唯一的标识符。
3.  **设备大小**: PV 的精确尺寸。
4.  **元数据区（Metadata Area）**: 最关键的部分，一块为存储整个卷组（VG）的配置信息而预留的空间。

**物理扩展块 (Physical Extent - PE)**:
`pvcreate` 完成后，LVM 在概念上将整个 PV 划分为大小均等的 PE。**PE 是 LVM 进行所有空间分配的原子单位**。默认大小为 4MB。这意味着你创建的任何逻辑卷（LV），其大小都必须是 PE 大小的整数倍。这个设计大大简化了空间管理和数据块的寻址映射。

### 卷组（VG）深度解析：资源池的管理者

如果说 PV 是“建筑材料”，那么 VG 就是“材料仓库”。`vgcreate` 命令的本质是一个**元数据操作**。

当你执行 `vgcreate vg_data_01 /dev/sdb /dev/sdc` 时，会发生以下事情:
1.  **生成 VG UUID**: 为这个新的卷组创建一个唯一的标识符。
2.  **构建元数据**: 创建一份描述 `vg_data_01` 的“账本”，记录它包含了 `/dev/sdb` 和 `/dev/sdc` 这两个 PV，并详细列出这两个 PV 上所有 PE 的状态（空闲或已分配）。
3.  **元数据同步**: **最关键的一步**，LVM 会将这份完整的“账本”（元数据）写入到 `/dev/sdb` 和 `/dev/sdc` 的元数据区。**是的，每个 PV 都拥有整个 VG 的完整配置信息副本。** 这就是 LVM 的元数据冗余机制，也是其高可靠性的体现。只要 VG 中有一块 PV 存活，就不会丢失整个存储池的结构信息。

## 🛠️ 实践操作

我们将使用昨天准备好的环境，假设我们有 `/dev/sdb`, `/dev/sdc`, `/dev/sdd`, `/dev/sde` 四块 10GB 的空闲磁盘。

### 1. Physical Volume (PV) 创建与检视

**步骤 1: 创建 PV**
我们将 `/dev/sdb` 和 `/dev/sdc` 初始化为物理卷。

```bash
# 使用 sudo 获取 root 权限执行
sudo pvcreate /dev/sdb /dev/sdc
```

**预期输出**:
```
  Physical volume "/dev/sdb" successfully created.
  Physical volume "/dev/sdc" successfully created.
```

**步骤 2: 简略查看 PV 信息**
使用 `pvs` 命令可以快速列出系统上所有的 PV。

```bash
sudo pvs
```

**预期输出及解读**:
```
  PV         VG Fmt  Attr PSize   PFree  
  /dev/sdb      lvm2 ---  <10.00g <10.00g
  /dev/sdc      lvm2 ---  <10.00g <10.00g
```
- `PV`: 物理卷的设备名。
- `VG`: 所属卷组名。此刻为空，因为我们还没创建 VG。
- `Fmt`: 文件格式，`lvm2` 是当前标准。
- `Attr`: 属性。`---` 表示标准的可写属性。
- `PSize`: PV 的总大小。
- `PFree`: PV 上的空闲空间。此刻等于总大小。

**步骤 3: 详细查看单个 PV 信息**
使用 `pvdisplay` 可以深入了解一个 PV 的所有属性。

```bash
sudo pvdisplay /dev/sdb
```

**预期输出及解读**:
```
  "/dev/sdb" is a new physical volume of "<10.00 GiB"
  --- NEW Physical volume ---
  PV Name               /dev/sdb
  VG Name               
  PV Size               <10.00 GiB
  Allocatable           NO
  PE Size               0   
  Total PE              0
  Free PE               0
  Allocated PE          0
  PV UUID               xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx 
```
- `PV Name`: 设备名。
- `VG Name`: 所属卷组，仍为空。
- `PV UUID`: **注意这个全球唯一的ID**，LVM 内部通过它而不是设备名来识别 PV。
- `PE Size`, `Total PE`: 此刻为 0，因为只有将 PV 加入 VG 后，PE 的大小和数量才会被确定。

### 2. Volume Group (VG) 创建与管理

**步骤 1: 创建 VG**
现在，我们将刚才创建的两个 PV 组成一个名为 `vg_data_01` 的卷组，遵循我们昨天制定的企业命名规范。

```bash
sudo vgcreate vg_data_01 /dev/sdb /dev/sdc
```

**预期输出**:
```
  Volume group "vg_data_01" successfully created
```

**步骤 2: 简略查看 VG 信息**
使用 `vgs` 命令快速查看。

```bash
sudo vgs
```

**预期输出及解读**:
```
  VG         #PV #LV #SN Attr   VSize   VFree  
  vg_data_01   2   0   0 wz--n- <19.99g <19.99g
```
- `VG`: 卷组名。
- `#PV`: 包含的 PV 数量，这里是 2。
- `#LV`, `#SN`: 逻辑卷和快照数量，目前为 0。
- `Attr`: 属性。`wz--n-` 表示可写、可调整大小。
- `VSize`: VG 的总大小，约等于两个 PV 大小之和。
- `VFree`: VG 的空闲空间。

**步骤 3: 详细查看 VG 信息**
使用 `vgdisplay` 查看 `vg_data_01` 的所有细节。

```bash
sudo vgdisplay vg_data_01
```

**预期输出及解读**:
```
  --- Volume group ---
  VG Name               vg_data_01
  System ID             
  Format                lvm2
  Metadata Areas        2
  Metadata Sequence No  1
  VG Access             read/write
  VG Status             resizable
  ...
  VG Size               <19.99 GiB
  PE Size               4.00 MiB
  Total PE              5118
  Alloc PE / Size       0 / 0   
  Free  PE / Size       5118 / <19.99 GiB
  VG UUID               yyyyyyyy-yyyy-yyyy-yyyy-yyyyyyyyyyyy
```
- `Metadata Areas`: 2，表示元数据有 2 份副本（在 `/dev/sdb` 和 `/dev/sdc` 上各一份）。
- `PE Size`: **现在被确定为 4.00 MiB**。这是整个 VG 的标准 PE 大小。
- `Total PE`: VG 中 PE 的总数。
- `Free PE / Size`: 当前所有 PE 均为空闲。

**步骤 4: 再次检查 PV**
现在再运行 `pvs`，你会发现变化。

```bash
sudo pvs
```

**预期输出**:
```
  PV         VG         Fmt  Attr PSize   PFree  
  /dev/sdb   vg_data_01 lvm2 a--  <10.00g <10.00g
  /dev/sdc   vg_data_01 lvm2 a--  <10.00g <10.00g
```
`VG` 列现在已经正确地显示为 `vg_data_01`。

## 💻 Go 编程实现

**任务**: 命令行工具的输出是给人看的，但对程序不友好。我们将编写一个 Go 工具，它调用 `pvs` 和 `vgs` 命令，但获取其 JSON 格式的输出，然后解析为 Go 结构体。这是实现自动化的第一步。

**项目准备**:
```bash
# 确保在昨天的 lvm-manager 目录下
mkdir -p cmd/day02
cd cmd/day02
```

**代码 (`main.go`):**
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

// LVMReport is the top-level structure for LVM JSON reports.
type LVMReport struct {
	Report []map[string][]map[string]string `json:"report"`
}

// PhysicalVolume defines the structure for a PV's attributes.
type PhysicalVolume struct {
	Name  string `json:"pv_name"`
	VG    string `json:"vg_name"`
	Size  string `json:"pv_size"`
	Free  string `json:"pv_free"`
	UUID  string `json:"pv_uuid"`
}

// VolumeGroup defines the structure for a VG's attributes.
type VolumeGroup struct {
	Name    string `json:"vg_name"`
	PVCount string `json:"pv_count"`
	LVCount string `json:"lv_count"`
	Size    string `json:"vg_size"`
	Free    string `json:"vg_free"`
	UUID    string `json:"vg_uuid"`
}

// runLVMCommand executes an LVM command with JSON reporting options.
func runLVMCommand(command string, args ...string) ([]byte, error) {
	// Prepend sudo to run with root privileges
	fullArgs := append([]string{command}, args...)
	fullArgs = append(fullArgs, "--reportformat", "json")
	
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

// GetPhysicalVolumes fetches and parses PV information.
func GetPhysicalVolumes() ([]PhysicalVolume, error) {
	output, err := runLVMCommand("pvs", "-o", "pv_name,vg_name,pv_size,pv_free,pv_uuid")
	if err != nil {
		return nil, err
	}

	var report LVMReport
	if err := json.Unmarshal(output, &report); err != nil {
		return nil, fmt.Errorf("failed to parse pvs JSON: %v", err)
	}

	var pvs []PhysicalVolume
	if len(report.Report) > 0 && report.Report[0]["pv"] != nil {
		for _, pvMap := range report.Report[0]["pv"] {
			pvs = append(pvs, PhysicalVolume{
				Name: pvMap["pv_name"],
				VG:   pvMap["vg_name"],
				Size: pvMap["pv_size"],
				Free: pvMap["pv_free"],
				UUID: pvMap["pv_uuid"],
			})
		}
	}
	return pvs, nil
}

// GetVolumeGroups fetches and parses VG information.
func GetVolumeGroups() ([]VolumeGroup, error) {
	output, err := runLVMCommand("vgs", "-o", "vg_name,pv_count,lv_count,vg_size,vg_free,vg_uuid")
	if err != nil {
		return nil, err
	}

	var report LVMReport
	if err := json.Unmarshal(output, &report); err != nil {
		return nil, fmt.Errorf("failed to parse vgs JSON: %v", err)
	}

	var vgs []VolumeGroup
	if len(report.Report) > 0 && report.Report[0]["vg"] != nil {
		for _, vgMap := range report.Report[0]["vg"] {
			vgs = append(vgs, VolumeGroup{
				Name:    vgMap["vg_name"],
				PVCount: vgMap["pv_count"],
				LVCount: vgMap["lv_count"],
				Size:    vgMap["vg_size"],
				Free:    vgMap["vg_free"],
				UUID:    vgMap["vg_uuid"],
			})
		}
	}
	return vgs, nil
}

func main() {
	log.Println("Fetching LVM information...")

	pvs, err := GetPhysicalVolumes()
	if err != nil {
		log.Fatalf("Error getting physical volumes: %v", err)
	}

	vgs, err := GetVolumeGroups()
	if err != nil {
		log.Fatalf("Error getting volume groups: %v", err)
	}

	fmt.Println("\n--- Physical Volumes (PVs) ---")
	fmt.Printf("%-15s %-15s %-12s %-12s %-s\n", "PV Name", "VG Name", "Size", "Free", "UUID")
	fmt.Println(strings.Repeat("-", 80))
	for _, pv := range pvs {
		fmt.Printf("%-15s %-15s %-12s %-12s %-s\n", pv.Name, pv.VG, pv.Size, pv.Free, pv.UUID)
	}

	fmt.Println("\n--- Volume Groups (VGs) ---")
	fmt.Printf("%-15s %-5s %-5s %-12s %-12s %-s\n", "VG Name", "#PV", "#LV", "Size", "Free", "UUID")
	fmt.Println(strings.Repeat("-", 80))
	for _, vg := range vgs {
		fmt.Printf("%-15s %-5s %-5s %-12s %-12s %-s\n", vg.Name, vg.PVCount, vg.LVCount, vg.Size, vg.Free, vg.UUID)
	}
	
	log.Println("LVM information fetched successfully.")
}
```

**运行与分析**:
1.  在 `cmd/day02` 目录下执行 `go run main.go`。
2.  **代码分析**:
    - 我们使用了 LVM 命令的一个强大特性：`--reportformat json`。这让输出直接变成机器可读的格式，避免了用正则表达式去解析易变的文本输出，这是编写健壮的自动化脚本的关键。
    - `runLVMCommand` 函数封装了执行命令的逻辑，包括了错误处理和 `stderr` 的捕获，这对于调试至关重要。
    - `GetPhysicalVolumes` 和 `GetVolumeGroups` 函数分别负责获取和解析特定资源的信息，实现了逻辑分离。
    - `main` 函数协调了整个流程，并以清晰的表格格式化输出，展示了处理结构化数据的好处。

## 🔍 故障排查与优化

- **常见问题**: `pvcreate` 报错 "Device /dev/sdb is already in use"。
  - **原因**: 该设备可能已经被挂载，或者是一个分区并且该分区正在被使用。
  - **诊断**: 使用 `lsblk -f` 或 `mount` 命令检查设备是否被挂载或已有文件系统。
  - **解决**: 确保你操作的是一块完全干净、未被使用的裸盘或分区。
- **最佳实践**: **备份元数据！** 每当对 VG 结构进行更改后（如创建、扩容），都应备份元数据。
  ```bash
  # 备份指定 VG 的元数据到 /tmp
  sudo vgcfgbackup -f /tmp/vg_data_01.backup vg_data_01
  ```
  这个备份文件是纯文本，包含了恢复整个 VG 结构所需的所有信息，是灾难恢复的救命稻草。

## 📝 实战项目

**任务**: 巩固今天的知识，创建一个新的 VG，并用 Go 程序展示 VG 和 PV 的归属关系。

1.  **创建新 VG**: 使用 `/dev/sdd` 创建一个新的、名为 `vg_archive_01` 的卷组。
    ```bash
    sudo pvcreate /dev/sdd
    sudo vgcreate vg_archive_01 /dev/sdd
    ```
2.  **Go 程序扩展**: 修改今天的 `main.go`。在打印完所有 VG 列表后，增加一个部分，遍历所有 VG，然后在其下缩进打印出属于该 VG 的所有 PV。
    - **提示**: 你需要嵌套循环。外层循环遍历 `vgs` 切片，内层循环遍历 `pvs` 切片，通过 `pv.VG == vg.Name` 来判断归属关系。

## 🏠 课后作业

1.  **VG 管理实践**: 
    - 使用 `sudo pvcreate /dev/sde` 将 `/dev/sde` 初始化为 PV。
    - 使用 `sudo vgextend vg_archive_01 /dev/sde` 命令，将这个新的 PV 添加到 `vg_archive_01` 中。
    - 使用 `vgs` 和 `vgdisplay` 验证 `vg_archive_01` 的容量是否增加。
    - 练习 `vgreduce vg_archive_01 /dev/sde` 将其移除。记录每一步的验证结果。
2.  **Go 工具增强**:
    - **挑战**: 当前我们的 Go 程序输出的 `Size` 和 `Free` 是带单位的字符串（如 `<19.99g`）。这不利于进行计算。请你编写一个 Go 函数 `parseSize(sizeStr string) (float64, error)`，该函数可以解析 LVM 的大小字符串（忽略 `<`, `>` 等符号，并处理 `g`, `m`, `t` 等单位），统一将其转换为以 `GB` 为单位的 `float64` 值。
    - **目标**: 修改你的 Go 程序，使用这个函数来计算并显示每个 VG 的**剩余空间百分比**。
