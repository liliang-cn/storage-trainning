# Day 4: Go 编程实现网络存储监控

## 🎯 学习目标
- **技能目标**: 掌握使用 Go 语言与操作系统底层交互的核心方法，特别是文件系统操作和执行外部命令。
- **实践目标**: 开发一个功能可用的网络存储监控工具 `net-storage-monitor`，能自动检测 NFS 和 iSCSI 的连接状态。
- **编程思想**: 学会使用 Go 的并发特性（goroutine）和上下文（context）来处理可能阻塞的 I/O 操作，提高程序的健壮性。
- **成果产出**: 一个完整的 `net-storage-monitor` Go 项目源代码，包含对 NFS 和 iSCSI 的并发检测、超时控制和清晰的日志输出。

## 📚 理论基础 (30%)

### 1. Go 与操作系统交互
Go 语言提供了强大的标准库，可以轻松地与操作系统进行交互。
- **`os` 包**: 提供了平台无关的操作系统功能接口。`os.Stat()` 是一个关键函数，它返回一个 `FileInfo` 接口，描述一个文件的元数据。对于一个挂载点，成功调用 `os.Stat()` 意味着该挂载点至少在 VFS (Virtual File System) 层面是可访问的。
- **`os/exec` 包**: 提供了执行外部命令的能力。这是与现有系统管理工具（如 `iscsiadm`）集成的最直接方式。`exec.Command()` 创建一个 `Cmd` 对象，其 `Output()` 或 `CombinedOutput()` 方法可以执行命令并捕获其标准输出和标准错误。

### 2. 处理阻塞 I/O 与超时
网络文件系统的操作（如 `os.Stat` 一个无响应的 NFS 挂载点）可能会无限期阻塞，导致监控程序卡死。必须使用超时机制来处理这种情况。
- **`context` 包**: Go 的上下文包是处理请求范围内的截止日期、取消信号和传递请求范围值的标准方式。
  - `context.WithTimeout(parentContext, duration)`: 创建一个新的上下文，它会在指定的时间后自动被取消。
- **结合 Goroutine**: 将可能阻塞的操作放在一个新的 goroutine 中执行，主 goroutine 则使用 `select` 语句同时等待操作完成或上下文超时。这是 Go 中实现超时控制的经典模式。

### 3. 解析系统文件
Linux 的 `/proc` 文件系统是一个虚拟文件系统，它提供了内核数据结构的接口。通过读取 `/proc` 下的文件，可以获取大量系统状态信息，而无需执行外部命令。
- **`/proc/mounts`**: 这个文件列出了当前系统上所有的挂载点，包括设备、挂载路径、文件系统类型和挂载选项。它的格式是文本，可以逐行读取和解析，是发现需要监控的目标的理想来源。

## 🛠️ 实践操作 (10%)

今天的重点是 Go 编程，实践操作主要是准备一个可供监控的环境。

- **确保环境**: 保证你有一个在前两天配置好的、正常工作的 NFS 挂载点和一个 iSCSI 挂载点。
- **模拟故障**: 为了测试监控工具，你需要能够模拟故障。最简单的方法是在服务器端停止相关服务：
  - **模拟 NFS 故障**: `sudo systemctl stop nfs-kernel-server`
  - **模拟 iSCSI 故障**: `sudo systemctl stop iscsid` 或 `sudo targetcli` 进入后禁用 portal。

## 💻 Go 编程实现 (60%)

**项目: `net-storage-monitor`**

我们将一步步构建这个工具。

### 1. 定义数据结构和主循环

**`main.go`**
```go
package main

import (
	"log"
	"time"
)

// MountInfo 存储一个挂载点的信息
type MountInfo struct {
	Device string
	Path   string
	Type   string
}

func main() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Starting network storage monitor...")

	for range ticker.C {
		log.Println("--- Running check ---")
		// 在这里添加我们的监控逻辑
	}
}
```

### 2. 解析 `/proc/mounts`

**`mounts.go`**
```go
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// getNetworkMounts 解析 /proc/mounts 文件，返回 NFS 和 iSCSI 挂载点
func getNetworkMounts() ([]MountInfo, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, fmt.Errorf("could not open /proc/mounts: %w", err)
	}
	defer file.Close()

	var mounts []MountInfo
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}

		fsType := fields[2]
		if strings.HasPrefix(fsType, "nfs") || fsType == "iscsi" { // iscsi 类型不常见，但作为例子
			mounts = append(mounts, MountInfo{
				Device: fields[0],
				Path:   fields[1],
				Type:   fsType,
			})
		}
	}
	return mounts, scanner.Err()
}
```

### 3. 实现状态检测器

**`checkers.go`**
```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// checkNFS 检查 NFS 挂载点状态，带超时控制
func checkNFS(mountPath string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ch := make(chan error, 1)
	go func() {
		_, err := os.Stat(mountPath)
		ch <- err
	}()

	select {
	case err := <-ch:
		return err == nil
	case <-ctx.Done():
		return false // 超时被视为失败
	}
}

// checkISCSI 检查 iSCSI 会话状态
func checkISCSI() (bool, error) {
	out, err := exec.Command("iscsiadm", "-m", "session").Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if strings.Contains(string(exitErr.Stderr), "No active sessions") {
				return false, nil
			}
		}
		return false, fmt.Errorf("iscsiadm command failed: %w", err)
	}
	return len(out) > 0, nil
}
```

### 4. 组装到 `main.go`

```go
// ... (main.go 的 import 和 struct 定义)

func main() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	log.Println("Starting network storage monitor...")

	for range ticker.C {
		log.Println("--- Running check ---")

		mounts, err := getNetworkMounts()
		if err != nil {
			log.Printf("ERROR: Failed to get mounts: %v", err)
			continue
		}

		// 检查 NFS
		for _, mount := range mounts {
			if strings.HasPrefix(mount.Type, "nfs") {
				log.Printf("Checking NFS mount: %s", mount.Path)
				if ok := checkNFS(mount.Path, 5*time.Second); !ok {
					log.Printf("WARN: NFS mount %s seems to be unresponsive!", mount.Path)
				} else {
					log.Printf("INFO: NFS mount %s is OK.", mount.Path)
				}
			}
		}

		// 检查 iSCSI
		log.Println("Checking iSCSI sessions...")
		if ok, err := checkISCSI(); err != nil {
			log.Printf("ERROR: Failed to check iSCSI: %v", err)
		} else if !ok {
			log.Println("INFO: No active iSCSI sessions found.")
		} else {
			log.Println("INFO: Active iSCSI sessions are present.")
		}
	}
}
```

## 🔍 故障排查与优化
- **程序编译失败**: 检查 Go 环境是否配置正确，所有包是否都已导入。
- **监控结果不准**: 
  - **NFS**: 如果 `checkNFS` 总是超时，尝试增加超时时间。如果服务器确实故障，这是预期行为。
  - **iSCSI**: `iscsiadm` 命令可能需要 `sudo` 权限。在生产环境中，可以配置 `sudoers` 文件，允许运行监控工具的用户无密码执行此特定命令。
- **优化**: 当前的实现是顺序检查所有挂载点。可以使用 Go 的 `sync.WaitGroup` 和 goroutine 来并发地检查所有 NFS 挂载点，从而在挂载点很多时缩短总检查时间。

## 📝 实战项目
- **并发改造**: 使用 `sync.WaitGroup` 和 goroutine 修改 `main` 函数中的 NFS 检查循环，使其能够并发地检查所有 NFS 挂载点。
- **添加修复功能**: 增加一个命令行标志 `--fix`。当程序检测到 NFS 挂载点无响应时，如果此标志被设置，则尝试执行 `umount -l <path>` 和 `mount <path>` 来自动恢复它。注意：执行这些命令需要 root 权限。

## 🏠 课后作业
- **配置化**: 将监控工具的配置（如检查间隔、NFS 超时时间、要监控的特定挂载点列表）外部化到一个 YAML 或 JSON 文件中，让程序启动时读取此配置文件。
- **深入研究**: `iscsiadm` 的输出是人类可读的文本，解析起来比较脆弱。研究一下是否有其他更可靠的方式来检查 iSCSI 会话状态，例如通过 netlink 套接字或 `/sys` 文件系统下的某些文件。
