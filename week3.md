# 第 3 周：网络存储协议与实践

## 整体目标
- **技能目标**: 掌握主流网络存储方案 NFS、iSCSI 的协议原理、配置和使用。
- **实践目标**: 能够独立搭建 NFS 和 iSCSI 服务端与客户端，并完成挂载和使用。
- **Go编程目标**: 使用 Go 语言开发网络存储挂载状态的监控和简单管理工具。
- **架构目标**: 理解网络存储在现代数据中心和微服务架构中的应用场景与设计考量。
- **安全目标**: 学习并实践网络存储的安全机制，如 Kerberos 和 CHAP。
- **前瞻目标**: 初步了解 NVMe-oF 的基本概念和优势，为未来学习做准备。

---

## Day 1: NFS 协议与服务器配置

### 🎯 学习目标
- 深入理解 NFS 协议 (v3, v4) 的工作原理和区别。
- 掌握 NFS 服务器的安装、配置 (`/etc/exports`) 和管理。
- 在客户端上挂载并使用 NFS 共享目录。

### 📚 理论学习（上午 2 小时）
1. **NFS 协议深度解析**
   - NFS (Network File System) 架构：客户端-服务器模型。
   - RPC (Remote Procedure Call) 在 NFS 中的核心作用。
   - NFS v3 vs v4：有状态 vs 无状态、文件锁定、安全性等方面的差异。
   - NFS 安全模型：`sec=sys` (默认，基于 UID/GID)、`sec=krb5` (Kerberos 加密认证)。
2. **企业级应用考量**
   - 命名规范：如何规划 exports 路径和命名。
   - 性能调优：`rsize`, `wsize` 等挂载参数的影响。
   - 场景分析：何时选择 NFS？（如：用户家目录、Web 内容、配置文件共享）。

### 🛠️ 实践操作（下午 2 小时）
1. **NFS 服务器端配置**
   ```bash
   # 1. 安装 NFS 服务 (以 Ubuntu/Debian 为例)
   sudo apt update
   sudo apt install -y nfs-kernel-server

   # 2. 创建共享目录
   sudo mkdir -p /srv/share/public
   sudo chown nobody:nogroup /srv/share/public
   sudo chmod 777 /srv/share/public

   # 3. 配置 exports 文件
   # 语法: /path/to/share client(options)
   # 将以下行添加到 /etc/exports
   # 允许 192.168.1.0/24 网段的客户端读写访问
   sudo echo "/srv/share/public 192.168.1.0/24(rw,sync,no_subtree_check)" | sudo tee -a /etc/exports

   # 4. 生效配置并启动服务
   sudo exportfs -a
   sudo systemctl restart nfs-kernel-server
   ```

2. **NFS 客户端配置**
   ```bash
   # 1. 安装客户端工具
   sudo apt update
   sudo apt install -y nfs-common

   # 2. 创建挂载点
   sudo mkdir -p /mnt/nfs/public

   # 3. 手动挂载 (将 192.168.1.100 替换为你的 NFS 服务器 IP)
   sudo mount -t nfs 192.168.1.100:/srv/share/public /mnt/nfs/public

   # 4. 验证挂载和读写
   df -hT /mnt/nfs/public
   touch /mnt/nfs/public/test_from_client.txt
   ls -l /mnt/nfs/public
   ```

### 📝 实践练习
- 配置一个只读的 NFS 共享。
- 配置一个基于特定 IP 的访问控制规则。
- 将 NFS 挂载配置写入 `/etc/fstab` 实现开机自动挂载。

### 🏠 作业
- 研究 NFS 的 `sync` 和 `async` 选项的区别及其对数据安全和性能的影响。
- 编写一个 Shell 脚本，检查指定的 NFS 服务器上有哪些可用的共享目录 (`showmount -e server_ip`)。

---

## Day 2: iSCSI 协议与块存储实践

### 🎯 学习目标
- 理解 iSCSI 协议的架构（Target, Initiator, LUN）。
- 掌握 iSCSI Target 和 Initiator 的配置流程。
- 将 iSCSI 提供的块设备格式化并使用。

### 📚 理论学习（上午 2 小时）
1. **iSCSI 架构解析**
   - iSCSI (Internet Small Computer System Interface)：将 SCSI 命令封装在 IP 包中传输，实现通过网络提供块级存储。
   - Target (目标器)：存储服务器，提供存储资源。
   - Initiator (启动器)：客户端，访问存储资源。
   - LUN (Logical Unit Number)：逻辑单元号，代表一个具体的块设备。
2. **iSCSI 认证与安全**
   - CHAP (Challenge-Handshake Authentication Protocol)：质询握手认证协议，用于验证 Initiator 的身份，防止未经授权的访问。
   - IQN (iSCSI Qualified Name)：iSCSI 节点的全球唯一命名规范。

### 🛠️ 实践操作（下午 2 小时）
使用 LIO (Linux-IO Target) 工具 `targetcli` 配置 Target。

1. **iSCSI Target (服务器) 配置**
   ```bash
   # 1. 安装 targetcli
   sudo apt update
   sudo apt install -y targetcli-fb

   # 2. 准备一个块设备用于后端存储 (这里用文件模拟)
   sudo fallocate -l 10G /var/lib/iscsi_storage.img

   # 3. 启动 targetcli 配置界面
   sudo targetcli

   # 4. 在 targetcli 中执行以下命令
   # 创建后端存储
   /> /backstores/fileio create file1 /var/lib/iscsi_storage.img

   # 创建 Target IQN
   /> /iscsi create iqn.2025-07.com.example:storage.disk1

   # 将后端存储关联到 LUN
   /> /iscsi/iqn.2025-07.com.example:storage.disk1/tpg1/luns create /backstores/fileio/file1

   # 配置访问控制 (ACL)，将 client_iqn 替换为客户端的 IQN
   /> /iscsi/iqn.2025-07.com.example:storage.disk1/tpg1/acls create iqn.2025-07.com.client:host0

   # 保存配置并退出
   /> saveconfig
   /> exit
   ```

2. **iSCSI Initiator (客户端) 配置**
   ```bash
   # 1. 安装 open-iscsi
   sudo apt update
   sudo apt install -y open-iscsi

   # 2. 获取客户端的 IQN (用于配置服务器 ACL)
   cat /etc/iscsi/initiatorname.iscsi

   # 3. 发现 Target (将 192.168.1.100 替换为 Target IP)
   sudo iscsiadm -m discovery -t sendtargets -p 192.168.1.100

   # 4. 登录 Target
   sudo iscsiadm -m node --login

   # 5. 验证块设备是否出现
   lsblk
   # 你应该能看到一个新的磁盘，如 /dev/sdb

   # 6. 格式化并使用该设备
   sudo mkfs.ext4 /dev/sdb
   sudo mount /dev/sdb /mnt
   df -h /mnt
   ```

### 📝 实践练习
- 配置 iSCSI 的 CHAP 认证。
- 创建多个 LUN 并全部挂载到客户端。
- 练习如何安全地登出和删除 iSCSI 会话。

### 🏠 作业
- 研究 iSCSI 和 Fibre Channel (FC) 的区别，分析它们各自的优势和应用场景。
- 编写一个 Shell 脚本，自动发现并登录到指定的 iSCSI Target。

---

## Day 3: 高级网络存储特性与安全

### 🎯 学习目标
- 理解 iSCSI 多路径 (MPIO) 的概念和配置方法。
- 掌握 NFSv4 的 Kerberos 安全认证配置。
- 初步了解 NVMe-oF 的概念。

### 📚 理论学习（上午 2 小时）
1. **高可用性与性能**
   - **iSCSI MPIO (Multipath I/O)**：通过多条网络路径连接同一个 LUN，实现负载均衡和故障转移，提高性能和可用性。
   - **NFS 高可用**: 通常通过 Pacemaker/Corosync 等集群软件或负载均衡器实现。
2. **网络存储安全强化**
   - **Kerberos**: 一种强大的网络认证协议，可为 NFS 提供加密和严格的身份验证，防止 IP 欺骗和数据嗅探。
3. **未来技术：NVMe-oF**
   - **NVMe over Fabrics**: 将 NVMe 命令通过网络（如 RDMA, Fibre Channel, TCP）传输，提供比 iSCSI 更低的延迟和更高的性能，是下一代数据中心存储网络的热点。

### 🛠️ 实践操作（下午 2 小时）
1. **配置 iSCSI MPIO** (概念性，需要多网卡环境)
   - 在 Target 上监听多个 IP 地址。
   - 在 Initiator 上安装 `multipath-tools`。
   - 配置 `/etc/multipath.conf` 文件。
   - `multipath -ll` 命令会显示一个 `/dev/mapper/mpathX` 设备，应用层应使用此设备。

2. **配置 NFS with Kerberos** (流程复杂，重点理解)
   - 搭建 KDC (Key Distribution Center) 服务器。
   - 为 NFS 服务和用户创建 Principal。
   - 在 NFS 服务器和客户端上配置 Kerberos (`/etc/krb5.conf`)。
   - 在 `/etc/exports` 中使用 `sec=krb5` 选项。
   - 客户端使用 `kinit` 获取票据后挂载。

### 📝 实践练习
- 理论研究：画出 iSCSI MPIO 的数据流图。
- 理论研究：画出 NFS with Kerberos 的认证流程图。

### 🏠 作业
- 详细对比 iSCSI MPIO 和 LVM Mirroring 在实现高可用性方面的异同。
- 阅读一篇关于 NVMe-oF vs iSCSI 的技术对比文章并写下总结。

---

## Day 4: Go 编程实现网络存储监控

### 🎯 学习目标
- 使用 Go 语言开发一个网络存储挂载状态的监控工具。
- 实现对 NFS 和 iSCSI 连接状态的自动检测。
- 在检测到问题时，实现简单的日志记录和告警。

### 🔧 Go 编程实践（全天）

**项目: `net-storage-monitor`**

1. **解析挂载点**
   - 编写一个函数，读取 `/proc/mounts` 文件，找出所有类型为 `nfs` 或 `iscsi` 的挂载点。

2. **NFS 状态检测**
   - 对于 NFS 挂载点，最简单的检测方法是尝试对挂载点执行 `stat` 系统调用。如果 NFS 服务中断，`stat` 调用通常会卡住或超时。
   - **优化**: 使用 `os.Stat` 并设置一个超时上下文 `context.WithTimeout` 来防止程序永久阻塞。

3. **iSCSI 状态检测**
   - 调用 `iscsiadm -m session -P 3` 命令，并解析其输出。
   - 检查 `session.state` 是否为 `logged in`，以及 `iface.state` 是否为 `online`。

4. **主监控循环**
   ```go
   package main

   import (
       "context"
       "log"
       "os"
       "os/exec"
       "strings"
       "time"
   )

   // MountInfo 存储挂载点信息
   type MountInfo struct {
       Device     string
       Path       string
       Type       string
   }

   // getNetworkMounts 解析 /proc/mounts 获取网络挂载点
   func getNetworkMounts() ([]MountInfo, error) { /* ... */ }

   // checkNFS 检查 NFS 挂载点状态
   func checkNFS(mountPath string, timeout time.Duration) bool {
       ctx, cancel := context.WithTimeout(context.Background(), timeout)
       defer cancel()

       // 在一个 goroutine 中执行 stat，以便我们可以超时
       ch := make(chan error, 1)
       go func() {
           _, err := os.Stat(mountPath)
           ch <- err
       }()

       select {
       case err := <-ch:
           return err == nil
       case <-ctx.Done():
           return false // 超时
       }
   }

   // checkISCSI 检查 iSCSI 会话状态
   func checkISCSI() bool {
       out, err := exec.Command("iscsiadm", "-m", "session", "-P", "1").Output()
       if err != nil {
           return false // 没有会话或命令失败
       }
       return strings.Contains(string(out), "State: Logged in")
   }

   func main() {
       ticker := time.NewTicker(30 * time.Second)
       for range ticker.C {
           log.Println("Running network storage check...")
           // ... 调用 getNetworkMounts, checkNFS, checkISCSI ...
           // ... 如果检测到问题，打印 WARN 或 ERROR 日志 ...
       }
   }
   ```

### 📝 实践练习
- 完善 `net-storage-monitor`，使其能并发检查多个挂载点。
- 增加一个 `--fix` 标志，当检测到 NFS 挂载点无响应时，尝试执行 `umount -l` 和 `mount -a` 来恢复它。

### 🏠 作业
- 将监控工具的配置（如要监控的挂载点、检查间隔）外部化到 YAML 或 JSON 文件中。
- 研究如何使用 Go 的 `net` 包直接与 NFS 服务器的 RPC 端口进行简单的连接性测试，作为 `os.Stat` 的替代方案。

---

## Day 5: 性能测试、故障排查与总结

### 🎯 学习目标
- 学会使用 `fio` 等工具测试网络存储的性能。
- 掌握网络存储常见问题的排查方法。
- 总结对比不同网络存储方案的适用场景。

### 🛠️ 实践操作（上午 2 小时）
1. **性能基准测试**
   ```bash
   # 在 NFS 挂载点上测试随机写 IOPS
   fio --name=randwrite --ioengine=libaio --iodepth=16 --rw=randwrite --bs=4k --direct=1 --size=512M --numjobs=8 --runtime=60 --group_reporting --directory=/mnt/nfs/public

   # 在 iSCSI 挂载的设备上测试顺序读带宽
   fio --name=readbw --ioengine=libaio --iodepth=64 --rw=read --bs=128k --direct=1 --size=1G --numjobs=1 --runtime=60 --group_reporting --filename=/dev/sdb
   ```

2. **常见故障排查**
   - **NFS `Permission Denied`**: 检查 `/etc/exports` 配置、客户端 IP 是否匹配、目录权限、`squash` 选项。
   - **NFS `Stale File Handle`**: 当服务器上的文件或目录被删除或改变，但客户端仍持有旧的引用时发生。通常需要强制卸载 (`umount -l`) 并重新挂载。
   - **iSCSI 登录失败**: 检查网络连通性、防火墙、Target IQN 和 ACL 配置、CHAP 密码。

### 🤔 架构总结与复盘（下午 2 小时）
1. **技术选型讨论**
   - **文件存储 (NFS)**: 多个客户端需要共享访问同一组文件。简单、易于管理。适用于 Web 服务器、代码仓库、用户主目录。
   - **块存储 (iSCSI)**: 单个客户端需要一个专用的“磁盘”。性能通常优于 NFS。适用于数据库、虚拟机镜像、需要特定文件系统的应用。
   - **对象存储 (如 S3)**: (理论补充) 通过 API 访问非结构化数据。高扩展性、高持久性。适用于备份归档、大数据、静态网站资源。

2. **Go 代码审查**
   - Review `net-storage-monitor` 项目，评估其健壮性、错误处理和日志清晰度。

### 🏠 本周作业交付
- **Go 工具**: 提交功能完善的 `net-storage-monitor` 项目。
- **技术文档**: 撰写一份网络存储故障排查清单 (Cookbook)。
- **分析报告**: 提交一份详细的 NFS vs iSCSI 对比报告，包含性能测试数据和选型建议。

---

## 📊 学习效果评估

### 技能检查清单
- [ ] 能够独立配置 NFSv4 服务器和客户端。
- [ ] 能够独立配置 iSCSI Target 和 Initiator。
- [ ] 理解 CHAP 和 Kerberos 在网络存储中的作用。
- [ ] 能够使用 `fio` 对网络存储进行基础性能测试。
- [ ] 完成 Go 语言监控工具的开发。

### 实战项目评估
- [ ] `net-storage-monitor` 功能完整，代码健壮。
- [ ] 故障排查清单内容实用，覆盖常见问题。
- [ ] 对比报告数据详实，结论清晰。

---

## 🔗 参考资源
1. **官方文档**
   - [NFS ArchWiki](https://wiki.archlinux.org/title/NFS)
   - [LIO (targetcli) ArchWiki](https://wiki.archlinux.org/title/ISCSI/LIO)
   - [open-iscsi README](https://github.com/open-iscsi/open-iscsi/blob/master/README)

2. **Go 开发参考**
   - [Go `context` package](https://pkg.go.dev/context/)
   - [Go `os/exec` package](https://pkg.go.dev/os/exec)

3. **工具和测试**
   - [fio - Flexible I/O Tester](https://fio.readthedocs.io/en/latest/)