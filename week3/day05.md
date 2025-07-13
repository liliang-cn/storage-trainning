# Day 5: 性能测试、故障排查与总结

## 🎯 学习目标
- **技能目标**: 学会使用 `fio` 等专业工具对网络存储进行量化的性能基准测试。
- **实践目标**: 掌握网络存储（NFS, iSCSI）常见问题的诊断思路和排查方法。
- **架构目标**: 能够根据业务需求，清晰地分析和总结不同网络存储方案（文件、块、对象）的适用场景。
- **成果产出**: 一份包含 `fio` 测试结果的性能对比报告，一份网络存储故障排查清单 (Cookbook)，一份文件 vs 块 vs 对象存储的选型指南。

## 📚 理论基础 (30%)

### 1. 存储性能核心指标
理解存储性能不能只看“快”或“慢”，需要量化的指标：
- **IOPS (Input/Output Operations Per Second)**: 每秒输入输出操作次数。衡量的是存储系统处理**高频小 I/O**的能力，对数据库、VDI 等随机读写密集的应用至关重要。
- **吞吐量/带宽 (Throughput/Bandwidth)**: 每秒传输的数据量（如 MB/s 或 GB/s）。衡量的是存储系统处理**大文件连续 I/O**的能力，对视频编辑、大数据分析、备份等场景至关重要。
- **延迟 (Latency)**: 完成一次 I/O 操作所需的时间（如毫秒 ms 或微秒 μs）。延迟越低，应用响应越快。这是对用户体验影响最直接的指标。

### 2. `fio` - 灵活的 I/O 测试工具
`fio` 是一个功能极其强大的开源 I/O 测试工具，可以模拟各种复杂的 I/O 负载。
- **核心参数**:
  - `name`: 测试任务的名称。
  - `ioengine`: I/O 引擎。`libaio` 是 Linux 上常用的异步 I/O 引擎。
  - `rw`: 读写模式。如 `read`, `write`, `randread`, `randwrite`, `randrw`。
  - `bs`: 块大小 (Block Size)。如 `4k`, `128k`。小块 `bs` 侧重测试 IOPS，大块 `bs` 侧重测试吞吐量。
  - `direct=1`: 绕过操作系统缓存，直接测试存储设备本身的性能。
  - `size`: 测试文件的总大小。
  - `numjobs`: 并发任务数。
  - `runtime`: 测试持续时间。
  - `group_reporting`: 将多个 job 的结果汇总报告。

## 🛠️ 实践操作 (40%)

### 1. 使用 `fio` 进行性能基准测试

#### a. 测试 NFS 的随机写 IOPS
这个测试模拟数据库或大量小文件的写入场景。
```bash
# 在 NFS 客户端的挂载点上执行
# --directory 指定测试将在哪个目录下生成文件
fio --name=nfs_randwrite_iops \
    --ioengine=libaio \
    --iodepth=16 \
    --rw=randwrite \
    --bs=4k \
    --direct=1 \
    --size=512M \
    --numjobs=8 \
    --runtime=60 \
    --group_reporting \
    --directory=/mnt/nfs/public
```
**关注结果**: `write: IOPS=...`

#### b. 测试 iSCSI 的顺序读吞吐量
这个测试模拟大文件读取或视频流场景。
```bash
# 在 iSCSI 客户端上执行
# --filename 指定直接在哪个块设备上测试
fio --name=iscsi_read_bw \
    --ioengine=libaio \
    --iodepth=64 \
    --rw=read \
    --bs=128k \
    --direct=1 \
    --size=1G \
    --numjobs=1 \
    --runtime=60 \
    --group_reporting \
    --filename=/dev/sdb # 替换为你的 iSCSI 设备
```
**关注结果**: `read: BW=...` (带宽)

### 2. 常见故障排查 (Cookbook)

#### NFS 问题
- **故障现象**: `mount.nfs: access denied by server while mounting ...`
  - **排查思路**:
    1. **服务器端检查**: 查看 `/etc/exports`。客户端 IP 是否在允许列表中？共享目录路径是否正确？
    2. **权限检查**: 查看服务器上共享目录本身的权限 (`ls -ld /path/to/share`)。
    3. **防火墙**: 检查服务器防火墙是否允许来自客户端的 NFS 请求 (TCP/UDP 2049)。
    4. **`exportfs -v`**: 在服务器上执行此命令，查看当前生效的导出规则。

- **故障现象**: 挂载点无响应，`ls` 卡住，`df` 卡住。
  - **排查思路**:
    1. **网络检查**: 从客户端 `ping` 服务器 IP，检查网络是否通畅。
    2. **服务器服务检查**: 在服务器上 `systemctl status nfs-kernel-server`，确认服务正在运行。
    3. **强制卸载**: 这是最后的手段。`sudo umount -l /path/to/mount` (`-l` 表示 lazy unmount)，然后尝试重新挂载。

- **故障现象**: `mount.nfs: Stale file handle`
  - **原因**: 服务器上的文件或目录在客户端仍有缓存引用时被重命名或删除。
  - **解决方案**: 强制卸载并重新挂载。

#### iSCSI 问题
- **故障现象**: `iscsiadm -m discovery` 失败或超时。
  - **排查思路**:
    1. **网络/防火墙**: 检查客户端到服务器的 TCP 3260 端口是否可达。
    2. **Target 服务**: 在服务器上 `systemctl status targetcli` 或 `iscsid`，确认服务正在运行。

- **故障现象**: `iscsiadm -m node --login` 失败，提示 `Login failed` 或 `Authentication failure`。
  - **排查思路**:
    1. **ACL**: 检查服务器 `targetcli` 中的 ACL 配置，确保客户端的 InitiatorName (IQN) 被正确添加。
    2. **CHAP**: 如果启用了 CHAP，请仔细核对客户端 `/etc/iscsi/iscsid.conf` 中的用户名和密码是否与服务器端配置的完全一致。

## 🤔 架构总结与复盘 (30%)

### 技术选型指南: 文件 vs 块 vs 对象存储

| 特性 | 文件存储 (File Storage) | 块存储 (Block Storage) | 对象存储 (Object Storage) |
| :--- | :--- | :--- | :--- |
| **访问方式** | 文件路径 (e.g., `/mnt/nfs/file.txt`) | 本地块设备 (e.g., `/dev/sdb`) | HTTP API (GET, PUT, DELETE) |
| **协议** | NFS, SMB/CIFS | iSCSI, Fibre Channel, NVMe-oF | S3, Swift |
| **共享性** | **多客户端**可同时读写共享 | 通常**单客户端**挂载使用 | **高并发**，多客户端通过 API 访问 |
| **性能** | 延迟较高，适合共享文件 | 延迟较低，性能接近本地磁盘 | 延迟最高，但吞吐量可极高扩展 |
| **用例** | - 用户家目录<br>- Web 服务器内容<br>- 配置文件共享 | - 数据库存储<br>- 虚拟机磁盘镜像 (VMDK, VHD)<br>- 需要特定文件系统的应用 | - 备份和归档<br>- 大数据湖<br>- 网站静态资源 (图片, 视频)<br>- 云原生应用 |
| **一句话总结** | 像一个网络U盘，大家都能用 | 像一根很长的SATA线，给你一块专用的网络硬盘 | 像一个无限容量的仓库，用钥匙(API)存取货物 |

### Go 代码审查
- **项目**: `net-storage-monitor`
- **审查要点**:
  - **健壮性**: 超时处理是否在所有可能阻塞的地方都已实现？
  - **错误处理**: 当 `os/exec` 命令失败时，错误信息是否被清晰地记录下来了？
  - **日志清晰度**: 日志是提供 INFO, WARN, ERROR 等级别，还是混在一起？清晰的日志是运维的关键。
  - **可配置性**: 硬编码的字符串（如路径、超时时间）是否过多？一个好的工具应该允许用户通过配置文件或命令行参数进行调整。

## 🏠 本周作业交付
- **Go 工具**: 提交功能完善、有并发检查、可配置的 `net-storage-monitor` 项目的最终版本。
- **技术文档**: 撰写一份图文并茂的网络存储故障排查清单 (Cookbook)，包含本周遇到的所有问题及其解决方案。
- **分析报告**: 提交一份详细的 NFS vs iSCSI 性能对比报告，应包含你用 `fio` 测出的 IOPS 和吞吐量数据，并结合数据分析两者的差异和适用场景。

```