# Day 4: ZFS 快照、克隆与数据保护实战

## 
 学习目标
- **核心技能**: 熟练掌握 ZFS 快照 (snapshot) 和克隆 (clone) 的创建、管理和恢复操作。
- **数据保护**: 学会使用 `zfs send` 和 `zfs receive` 功能，实现高效、可靠的数据备份与迁移。
- **运维实战**: 亲手实践 ZFS 存储池的磁盘故障模拟与恢复流程，理解 `DEGRADED` 状态和 `resilver` (再同步) 过程。
- **Go 编程进阶**: 编写 Go 程序来列出和管理 ZFS 快照，为后续开发自动化快照工具打下基础。

## 
 理论基础 (30%)

### 1. ZFS 快照 (Snapshot)
- **定义**: 快照是文件系统或 zvol 在特定时间点的一个只读副本。
- **工作原理**: 得益于 ZFS 的写时复制 (CoW) 机制，创建快照几乎是瞬时完成的。它并不复制任何数据，只是创建了一组指向当前数据块的元数据指针。因此，快照在创建之初几乎不占用任何额外空间。只有当活动文件系统中的数据块被修改或删除时，快照才会“持有”这些旧的数据块，从而开始占用空间，以保证快照内容的一致性。
- **核心价值**:
    - **防误操作**: 提供了对抗“`rm -rf`”等意外删除的终极“后悔药”。
    - **一致性备份**: 可以获得一个文件系统在某个瞬间的、完全一致的冻结视图，非常适合用于备份。
    - **克隆基础**: 是创建可写克隆的前提。

### 2. ZFS 克隆 (Clone)
- **定义**: 克隆是基于一个快照创建的可写的文件系统。
- **工作原理**: 克隆在创建时，其所有数据都指向其父快照。它只在数据被修改或写入新数据时才开始占用新的存储空间。
- **核心价值**:
    - **快速环境搭建**: 能够瞬时创建出多个开发、测试环境的副本，极大地节省了时间和存储空间。
    - **虚拟机部署**: 可以用一个基础镜像的快照，快速克隆出多个虚拟机。
- **重要依赖**: 克隆依赖于其父快照。在销毁一个克隆之前，无法销毁其父快照。

### 3. ZFS Send / Receive
- **定义**: 一种将 ZFS 快照（或快照流）序列化为字节流的机制，该字节流可以被发送到文件、标准输出或通过网络传输到另一台机器。
- **工作原理**:
    - **全量发送 (`zfs send <snapshot>`)**: 将一个快照包含的所有数据和属性打包成一个数据流。
    - **增量发送 (`zfs send -i <old_snapshot> <new_snapshot>`)**: 仅发送两个
照之间发生变化的数据块。这是进行高效增量备份的关键。
- **核心价值**:
    - **高效备份**: 增量发送机制使得日常备份非常快速且节省带宽和存储空间。
    - **数据迁移**: 是在不同 ZFS 池或不同服务器之间迁移数据的标准方法。

## 
 实践操作 (50%)

### 1. 环境准备
```bash
# 清理并准备一个简单的镜像池
sudo zfs destroy -r homeserver || true
sudo zpool destroy tank || true
sudo losetup -d /dev/loop* || true
cd /opt/disks && sudo rm *.img
for i in {1..4}; do sudo truncate -s 1G disk${i}.img && sudo losetup /dev/loop${i} disk${i}.img; done
sudo wipefs -a /dev/loop*

# 创建一个名为 'tank' 的镜像池用于实验
sudo zpool create tank mirror /dev/loop1 /dev/loop2
sudo zfs create tank/data
sudo chown -R $USER:$USER /tank # 方便普通用户操作
echo "Original content" > /tank/data/file.txt
```

### 2. 快照与回滚
```bash
# 1. 为 tank/data 创建一个名为 'tuesday' 的快照
sudo zfs snapshot tank/data@tuesday

# 2. 查看快照列表
sudo zfs list -t snapshot
# NAME                USED  AVAIL     REFER  MOUNTPOINT
# tank/data@tuesday      0      -     1.00M  -

# 3. 模拟数据误操作
echo "Modified content" > /tank/data/file.txt
rm /tank/data/file.txt

# 4. 从快照中恢复单个文件 (快照是隐藏挂载的)
ls /tank/data/.zfs/snapshot/tuesday/
cp /tank/data/.zfs/snapshot/tuesday/file.txt /tank/data/

# 5. 回滚整个文件系统到快照状态
sudo zfs rollback tank/data@tuesday
# 注意: rollback 会丢弃 'tuesday' 快照之后的所有更改
cat /tank/data/file.txt # 输出 "Original content"
```

### 3. 克隆与数据迁移
```bash
# 1. 基于 'tuesday' 快照创建一个名为 'data_dev' 的克隆
sudo zfs clone tank/data@tuesday tank/data_dev

# 2. 验证克隆是可写的
sudo chown -R $USER:$USER /tank/data_dev
echo "This is a dev environment" > /tank/data_dev/new_file.txt
cat /tank/data_dev/new_file.txt

# 3. 尝试删除父快照 (会失败)
sudo zfs destroy tank/data@tuesday
# cannot destroy 'tank/data@tuesday': snapshot has dependent clones
# use '-R' to destroy the snapshot and all its dependents.
```

### 4. 使用 Send/Receive 进行备份演练
```bash
# 1. 创建一个备份池，模拟备份服务器
sudo zpool create backup_pool /dev/loop3

# 2. 全量备份: 将 'tuesday' 快照发送到备份池
sudo zfs send tank/data@tuesday | sudo zfs recv backup_pool/data_backup

# 3. 验证备份
sudo zfs list -r backup_pool

# 4. 创建新数据和新快照
echo "Wednesday's data" >> /tank/data/file.txt
sudo zfs snapshot tank/data@wednesday

# 5. 增量备份: 仅发送两个快照间的差异
sudo zfs send -i tank/data@tuesday tank/data@wednesday | sudo zfs recv backup_pool/data_backup
```

### 5. 故障恢复演练
```bash
# 1. 查看当前健康的镜像池状态
sudo zpool status tank

# 2. 模拟磁盘 /dev/loop1 故障，使其离线
sudo zpool offline tank /dev/loop1

# 3. 再次查看状态，池变为 DEGRADED
sudo zpool status tank
# state: DEGRADED
# ...
# loop1   OFFLINE

# 4. 模拟故障盘被修复，使其重新上线
sudo zpool online tank /dev/loop1
# ZFS 会自动开始 resilver (再同步) 过程，将数据同步回 loop1
watch sudo zpool status tank # 观察 resilver 进度

# 5. 模拟物理替换磁盘
# 将 /dev/loop1 替换为新的磁盘 /dev/loop4
sudo zpool replace tank /dev/loop1 /dev/loop4
watch sudo zpool status tank # 再次观察 resilver 过程
```

## 
 Go 编程实现 (10%)

这个 Go 程序将列出指定数据集的所有快照，并显示其名称、使用空间和创建时间。

**`snapshot_lister.go`**
```go
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// SnapshotInfo holds structured info about a ZFS snapshot.
type SnapshotInfo struct {
	Name      string
	Used      string
	Creation  string
}

// listSnapshotsForDataset lists all snapshots for a given ZFS dataset.
func listSnapshotsForDataset(dataset string) ([]SnapshotInfo, error) {
	cmd := exec.Command("sudo", "zfs", "list", "-t", "snapshot", "-r", dataset, "-o", "name,used,creation", "-s", "creation", "-H")
	out, err := cmd.CombinedOutput()
	if err != nil {
		// If no snapshots, command might return error. Handle gracefully.
		if strings.Contains(string(out), "no datasets available") {
			return []SnapshotInfo{}, nil
		}
		return nil, fmt.Errorf("failed to list snapshots for %s: %w\nOutput: %s", dataset, err, string(out))
	}

	var snapshots []SnapshotInfo
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 5 { // Creation time can have spaces
			info := SnapshotInfo{
				Name:     parts[0],
				Used:     parts[1],
				Creation: strings.Join(parts[2:], " "),
			}
			snapshots = append(snapshots, info)
		}
	}
	return snapshots, nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: go run %s <dataset_name (e.g., tank/data)>", os.Args[0])
	}
	dataset := os.Args[1]

	fm