# Day 1: LVM 基础理论与环境准备

## 🎯 学习目标

- **明确的技能目标**:
  - 深入理解 LVM 的三层架构（PV/VG/LV）及其内在联系。
  - 掌握 LVM 元数据的工作机制、冗余策略及其在灾备中的核心地位。
  - 熟悉企业级 LVM 实践中的命名规范、容量规划原则和风险评估方法。
  - 能够独立搭建并配置一个用于 LVM 实验的标准虚拟环境。

- **具体的成果产出**:
  - 一个配置完毕的虚拟机，包含4块用于后续实验的虚拟裸盘。
  - 一个可执行的 Go 程序，能够调用系统命令，以结构化（JSON）格式获取并展示系统所有块设备的信息。

## 📚 理论基础 (30-40%)

- **核心概念深度解析**:
  1.  **物理卷 (Physical Volume - PV)**: LVM 的基础构建块。它并非一种文件系统，而是对物理存储设备（整个硬盘、分区、RAID设备等）的标准化封装。通过 `pvcreate` 命令，LVM 在设备头部写入元数据，并将其划分为统一大小的 **物理扩展块 (Physical Extents - PE)**。PE 是 LVM 进行所有空间分配的最小原子单位，默认通常为 4MB。
  2.  **卷组 (Volume Group - VG)**: 一个或多个 PV 组成的统一存储池。VG 的创建 (`vgcreate`) 将多个物理设备聚合为一个逻辑上的大资源池，从而屏蔽了底层物理磁盘的差异和边界。其核心优势在于弹性，可以随时通过 `vgextend` 添加新 PV 来扩容。
  3.  **逻辑卷 (Logical Volume - LV)**: 从 VG 中按需划分出的逻辑分区。LV (`lvcreate`) 是最终用户可以格式化并挂载使用的设备。它可以灵活地扩容和缩容，只要其所在的 VG 有足够空间。

- **系统原理和架构设计**:
  - **LVM 元数据机制**: LVM 的“配置总账”，记录了 PV、VG、LV 之间的全部映射关系。这份元数据以**冗余方式**存储在 VG 内**每一个 PV 的头部**。这个设计是 LVM 可靠性的基石：只要 VG 中有一块 PV 幸存，其上的元数据副本就可以用来恢复整个 VG 的结构信息。因此，理解和定期备份元数据是至关重要的运维纪律。

- **企业级应用场景分析**:
  - **数据库存储**: 为 MySQL、PostgreSQL 等数据库的数据文件和日志文件创建独立的 LV，可以根据业务增长情况，在不中断服务的情况下在线扩容磁盘空间。
  - **虚拟化平台**: 在 KVM 或 Xen 等虚拟化宿主机上，可以将整个 VG 作为存储池，为虚拟机动态创建和分配 LV 作为其虚拟磁盘，实现灵活的资源调配。
  - **文件服务器**: 对于需要大容量且持续增长的文件服务，使用 LVM 可以轻松地将多块物理硬盘合并为一个巨大的逻辑卷，简化管理。

## 🛠️ 实践操作 (40-50%)

- **详细的命令行操作步骤**:

  1.  **虚拟机环境准备**:
      - 使用 VirtualBox 或 VMware 创建一台虚拟机。
      - **系统**: CentOS 9 Stream 或 Ubuntu Server 22.04 LTS。
      - **配置**: 2 CPU, 2GB RAM, 20GB 系统盘。
      - **关键步骤**: 在虚拟机设置中，额外添加 **4 块** 10GB 的虚拟裸盘，用于后续创建 PV。

  2.  **LVM 工具安装与验证**:
      - 登录虚拟机，打开终端。
      - **CentOS/RHEL**: `sudo dnf install -y lvm2`
      - **Ubuntu/Debian**: `sudo apt-get update && sudo apt-get install -y lvm2`
      - **验证**: `lvm version`。看到版本信息即表示安装成功。

  3.  **查看初始磁盘状态**:
      - **命令**: `lsblk`
      - **预期输出**: 你应该能看到系统盘（如 `sda`）及其分区，以及新添加的四块无分区、无挂载点的裸盘（如 `sdb`, `sdc`, `sdd`, `sde`）。这是我们后续操作的对象。
      ```
      NAME   MAJ:MIN RM SIZE RO TYPE MOUNTPOINT
      sda      8:0    0  20G  0 disk 
      ├─sda1   8:1    0   1G  0 part /boot
      └─sda2   8:2    0  19G  0 part /
      sdb      8:16   0  10G  0 disk 
      sdc      8:32   0  10G  0 disk 
      sdd      8:48   0  10G  0 disk 
      sde      8:64   0  10G  0 disk 
      ```

## 💻 Go 编程实现 (20-30%)

- **系统调用封装**: 我们将封装对 `lsblk` 命令的调用，并利用其 JSON 输出能力，避免手动解析字符串。

- **自动化脚本开发**: 创建一个 Go 程序，以结构化方式获取系统块设备信息。

  1.  **项目结构**: `mkdir -p lvm-manager/cmd/day01 && cd lvm-manager/cmd/day01`
  2.  **代码 (`main.go`)**:
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

      // BlockDevice 对应 lsblk JSON 输出中的一个设备条目
      type BlockDevice struct {
      	Name       string `json:"name"`
      	Size       string `json:"size"`
      	Type       string `json:"type"`
      	MountPoint string `json:"mountpoint"`
      }

      // LsblkOutput 是 lsblk 完整 JSON 输出的顶层结构
      type LsblkOutput struct {
      	BlockDevices []BlockDevice `json:"blockdevices"`
      }

      // getBlockDevices 执行 lsblk 命令并返回解析后的设备列表
      func getBlockDevices() ([]BlockDevice, error) {
      	// -J 表示 JSON 输出, -o 指定我们需要的字段
      	cmd := exec.Command("lsblk", "-J", "-o", "NAME,SIZE,TYPE,MOUNTPOINT")

      	var out, stderr bytes.Buffer
      	cmd.Stdout = &out
      	cmd.Stderr = &stderr

      	log.Println("Executing command:", cmd.String())
      	err := cmd.Run()
      	if err != nil {
      		return nil, fmt.Errorf("lsblk command failed: %v\nStderr: %s", err, stderr.String())
      	}

      	var report LsblkOutput
      	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
      		return nil, fmt.Errorf("failed to unmarshal lsblk JSON output: %v", err)
      	}

      	return report.BlockDevices, nil
      }

      func main() {
      	log.Println("Starting block device discovery...")

      	devices, err := getBlockDevices()
      	if err != nil {
      		log.Fatalf("FATAL: Could not retrieve block device information: %v", err)
      	}

      	fmt.Println("\n--- System Block Devices ---")
      	fmt.Printf("%-15s %-10s %-10s %-s\n", "DEVICE", "SIZE", "TYPE", "MOUNTPOINT")
      	fmt.Println(strings.Repeat("-", 60))
      	for _, device := range devices {
      		mountPoint := device.MountPoint
      		if mountPoint == "" {
      			mountPoint = "(none)"
      		}
      		fmt.Printf("%-15s %-10s %-10s %-s\n", device.Name, device.Size, device.Type, mountPoint)
      	}
      	fmt.Println(strings.Repeat("-", 60))

      	log.Println("Discovery finished successfully.")
      }
      ```

## 🔍 故障排查与优化

- **常见问题诊断**:
  - **问题**: `lsblk` 看不到新添加的虚拟磁盘。
    - **诊断**: 1. 确认虚拟机已**完全关闭**后再添加硬盘，而非处于挂起状态。2. 部分老系统可能需要重启或执行 `rescan-scsi-bus.sh` (如已安装) 来识别新硬件。
  - **问题**: `go run main.go` 报错 `exec: "lsblk": executable file not found in $PATH`。
    - **诊断**: 你的系统环境可能非常精简，缺少 `util-linux` 包。使用 `sudo dnf install util-linux` 或 `sudo apt-get install util-linux` 安装即可。

- **最佳实践总结**:
  - **日志先行**: 在 Go 代码中，像我们在 `main` 函数和 `getBlockDevices` 中做的那样，在关键步骤（如执行命令、解析数据）前后打印日志。这在调试复杂脚本时能救命。
  - **明确错误**: `fmt.Errorf` 是你的好朋友。当封装的函数返回错误时，附加上下文信息（如哪个命令失败了，stderr是什么），能让你更快地定位问题根源。

## 📝 实战项目

- **综合应用练习**: 将今天的 Go 程序作为我们 `lvm-manager` 工具的第一个子命令。
- **项目目标**: 创建一个名为 `discover` 的基础工具，它能清晰地列出系统中的所有块设备，为后续识别可用的 PV 裸盘做准备。
- **代码质量要求**:
  - **封装**: 将核心逻辑（执行命令、解析JSON）封装在独立的函数中，`main` 函数只负责流程控制和输出。
  - **错误处理**: 必须处理 `exec.Command` 和 `json.Unmarshal` 可能返回的所有错误，并提供有意义的错误信息。
  - **可读性**: 代码注释清晰，变量命名规范，输出格式整洁。

## 🏠 课后作业

- **扩展练习任务**:
  1.  **理论题**: 请用自己的话解释，为什么说“PE 是 LVM 进行空间分配的原子单位”？这有什么好处？
  2.  **编码题**: 修改今天的 Go 程序。增加一个 `-u` 或 `--unmounted` 的命令行标志。当用户提供这个标志时，程序只打印出那些 `MountPoint` 为空（即未挂载）的块设备。这对于快速找到可用裸盘非常有用。
     - **提示**: 你需要使用 Go 的 `flag` 包来处理命令行参数。

- **深入研究方向**: 阅读 `lvm.conf` 的 man page (`man lvm.conf`)，了解 LVM 的主要配置文件结构，特别是 `devices` 部分，看看它是如何通过 `filter` 规则来控制 LVM 能看到哪些设备的。这是企业环境中进行设备隔离的重要配置。
