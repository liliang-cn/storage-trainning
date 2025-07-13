# 第 2 周：RAID 与 ZFS 高级存储技术

## 整体目标
- 深入理解 RAID 0/1/5/10 等模式的原理与应用场景
- 掌握 ZFS 存储池、数据集、快照的核心管理技能
- 学会使用 `mdadm` 和 `zfs` 命令进行实战操作
- 通过 Go 语言实现对 RAID 和 ZFS 的状态监控与基础管理
- 重点关注数据保护策略和业务连续性设计
- 为第三周学习网络存储奠定数据层基础

---

## Day 1: 软件 RAID 基础与 `mdadm` 实战

### 🎯 学习目标
- 理解 RAID 0 和 RAID 1 的工作原理、优缺点
- 掌握 `mdadm` 工具创建和管理 RAID 0/1 阵列
- 理解 RAID 写洞问题及其影响

### 📚 理论学习（上午 2 小时）
1. **RAID 核心概念**
   - RAID 0 (条带化)：为性能而生，无冗余
   - RAID 1 (镜像化)：为冗余而生，写性能有损耗
   - RAID 写洞 (Write Hole) 问题解析与解决方案
2. **企业级思考**
   - TCO (总体拥有成本) 分析：对比不同 RAID 级别的硬件成本、性能收益和维护开销
   - 故障域分析：评估单点故障对业务的影响

### 🛠️ 实践操作（下午 2 小时）
1. **环境准备与 `mdadm` 安装**
   - 确认系统已安装 `mdadm` 工具
   - 准备至少 4 块未使用的虚拟磁盘

2. **创建和管理 RAID 阵列**
   ```bash
   # 创建 RAID 0 阵列
   mdadm --create /dev/md0 --level=0 --raid-devices=2 /dev/sdb /dev/sdc

   # 创建 RAID 1 阵列
   mdadm --create /dev/md1 --level=1 --raid-devices=2 /dev/sdd /dev/sde

   # 查看阵列详细状态
   mdadm --detail /dev/md0
   cat /proc/mdstat
   ```
3. **格式化、挂载与测试**
   - `mkfs.ext4 /dev/md0`
   - `mount /dev/md0 /mnt/raid0`
   - 验证读写功能

### 📝 实践练习
- 创建一个包含 3 块磁盘的 RAID 0 阵列
- 停止并重新组装一个 RAID 1 阵列

### 🏠 作业
- 详细阅读 `mdadm` 的 man page
- 撰写文档，总结 RAID 0 和 RAID 1 的性能特点及适用业务场景

---

## Day 2: 高级 RAID 模式与故障恢复

### 🎯 学习目标
- 掌握 RAID 5 和 RAID 10 的配置与管理
- 练习 RAID 阵列的故障模拟与恢复流程
- 学会使用 `fio` 对不同 RAID 级别进行性能基准测试

### 📚 理论学习（上午 1.5 小时）
1. **高级 RAID 模式**
   - RAID 5 (分布式奇偶校验)：性能、容量和冗余的平衡
   - RAID 10 (镜像条带)：高性能与高冗余的结合
   - 热备盘 (Hot Spare) 概念与自动重建机制

### 🛠️ 实践操作（下午 2.5 小时）
1. **创建 RAID 5 和 RAID 10**
   ```bash
   # 创建 RAID 5 阵列
   mdadm --create /dev/md2 --level=5 --raid-devices=3 /dev/sdf /dev/sdg /dev/sdh

   # 创建 RAID 10 阵列
   mdadm --create /dev/md3 --level=10 --raid-devices=4 /dev/sdi /dev/sdj /dev/sdk /dev/sdl
   ```
2. **故障模拟与恢复**
   ```bash
   # 为 RAID 5 添加热备盘
   mdadm /dev/md2 --add /dev/sdm

   # 模拟磁盘故障
   mdadm /dev/md2 --fail /dev/sdf

   # 观察自动重建过程
   watch cat /proc/mdstat

   # 移除故障盘并添加新盘
   mdadm /dev/md2 --remove /dev/sdf
   mdadm /dev/md2 --add /dev/sdf  # 假设 sdf 已修复或被新盘替换
   ```

### 📊 性能基线测试
- 使用 `fio` 对 RAID 0, 1, 5, 10 进行顺序和随机读写测试
- 分析并记录不同 RAID 级别的性能数据，建立性能基线

### 🏠 作业
- 编写一个 Shell 脚本，用于解析 `mdadm --detail` 的输出，并判断阵列健康状态

---

## Day 3: ZFS 核心概念与存储池管理

### 🎯 学习目标
- 深入理解 ZFS 的核心架构（vdev, zpool, dataset）
- 掌握 ZFS 存储池的创建、管理和监控
- 理解 ZFS 的数据校验和自修复机制

### 📚 理论学习（上午 2 小时）
1. **ZFS 架构解析**
   - vdev (虚拟设备)：物理设备的抽象层
   - zpool (存储池)：由 vdev 构成的存储资源池
   - dataset (数据集)：类似文件系统，可精细化管理
   - 写时复制 (Copy-on-Write) 原理及其对数据一致性的保障
2. **ZFS 数据保护机制**
   - 端到端的数据校验 (Checksum)
   - 数据自修复 (Self-Healing) 原理
   - ZFS 缓存机制：ARC, L2ARC, ZIL/SLOG 的作用与调优方向

### 🛠️ 实践操作（下午 2 小时）
1. **ZFS 环境准备**
   - 安装 ZFS on Linux 相关软件包
   - 准备至少 4 块裸盘用于实验
2. **创建和管理 ZFS 存储池**
   ```bash
   # 创建镜像池 (类似 RAID 1)
   zpool create tank mirror /dev/sdb /dev/sdc

   # 创建 raidz 池 (类似 RAID 5)
   zpool create datapool raidz /dev/sdd /dev/sde /dev/sdf

   # 查看池状态和健康状况
   zpool status -v
   zpool list
   ```
3. **创建和管理数据集**
   - `zfs create tank/data`
   - `zfs create -o compression=lz4 -o quota=10G tank/project`
   - `zfs list`

### 📝 实践练习
- 销毁并重建一个 zpool
- 在数据集中设置不同的属性（如 `recordsize`, `atime`）并观察效果

### 🏠 作业
- 阅读 ZFS 官方文档，特别是关于 `zpool` 和 `zfs` 命令的部分
- 设计一个 ZFS 存储池布局方案，用于高可用数据库存储，并说明理由

---

## Day 4: ZFS 快照、克隆与数据保护实战

### 🎯 学习目标
- 掌握 ZFS 快照和克隆的创建与使用
- 学会使用 ZFS send/receive 功能进行数据备份与迁移
- 实践 ZFS 存储池的故障恢复

### 📚 理论学习（上午 1.5 小时）
1. **ZFS 高级功能**
   - ZFS 快照：瞬时、只读、空间高效的时间点副本
   - ZFS 克隆：基于快照的可写副本，用于快速创建开发测试环境
   - ZFS send/receive：高效的增量数据流复制
   - ZFS 压缩与去重：原理、效果及性能开销分析

### 🛠️ 实践操作（下午 2.5 小时）
1. **快照与回滚**
   ```bash
   # 创建快照
   zfs snapshot tank/data@tuesday

   # 模拟数据误删除后，从快照回滚
   zfs rollback tank/data@tuesday
   ```
2. **克隆与数据迁移**
   ```bash
   # 创建克隆
   zfs clone tank/data@tuesday tank/data_dev_env

   # 增量备份
   zfs send -i tank/data@tuesday tank/data@wednesday | zfs recv backup/data
   ```
3. **故障恢复演练**
   ```bash
   # 模拟磁盘故障并让其离线
   zpool offline tank /dev/sdb

   # 替换故障磁盘
   zpool replace tank /dev/sdb /dev/sdn # sdn 是新磁盘

   # 观察 resilvering (再同步) 过程
   zpool status -v
   ```

### 📝 实践练习
- 创建一个递归快照（包含所有子数据集）
- 测试 ZFS 压缩效果：在一个数据集中存入大文本文件，对比开启/关闭压缩后的空间占用

### 🏠 作业
- 编写一个 Shell 脚本，实现 ZFS 数据集的每日自动快照，并保留最近 7 天的快照

---

## Day 5: Go 编程集成与综合对比

### 🎯 学习目标
- 使用 Go 语言开发 RAID 和 ZFS 的状态监控工具
- 深入比较 LVM、mdadm RAID 和 ZFS 的技术优劣
- 总结生产环境存储方案选型的最佳实践

### 🔧 Go 编程实践（上午 2.5 小时）
1. **项目规划：`storage-monitor`**
   - 定义项目结构，区分 `raid` 和 `zfs` 监控模块
2. **RAID 监控模块开发**
   - 使用 `os/exec` 调用 `mdadm --detail` 命令
   - 编写解析器，提取阵列状态、磁盘角色（active, spare, faulty）
   - 实现当阵列状态为 `degraded` 或 `inactive` 时发送告警
3. **ZFS 监控模块开发**
   - 调用 `zpool status -x` (简洁可解析模式) 判断池健康状况
   - 调用 `zfs list -p -H -o name,used,avail` 获取数据集空间使用情况
   - 实现当池状态不为 `ONLINE` 或数据集使用率超阈值时发送告警

### 🤔 架构总结与复盘（下午 1.5 小时）
1. **技术对比分析**
   - 创建一个详细的 Markdown 表格，从以下维度对比 LVM、mdadm RAID、ZFS：
     - 数据保护能力
     - 性能特点
     - 灵活性与扩展性
     - 管理复杂度
     - 典型应用场景
2. **代码审查与总结**
   - Review `storage-monitor` 项目代码，重点关注错误处理、代码复用和可读性
   - 总结本周学习的核心要点和难点

### 🏠 本周作业交付
1. **Go 工具代码**
   - 提交完整的 `storage-monitor` 项目，包含 README 和使用说明
2. **技术文档**
   - 提交 LVM vs. mdadm RAID vs. ZFS 的详细对比分析报告
   - 提交 RAID 和 ZFS 在生产环境中的运维最佳实践总结

---

## 📊 学习效果评估

### 技能检查清单
- [ ] 能够独立使用 `mdadm` 创建、管理和维护 RAID 0/1/5/10 阵列
- [ ] 能够处理 `mdadm` 阵列中的磁盘故障和替换
- [ ] 能够独立使用 `zfs`/`zpool` 创建和管理 ZFS 存储池与数据集
- [ ] 熟练掌握 ZFS 快照、克隆、send/receive 的使用场景和操作
- [ ] 能够处理 ZFS 存储池中的磁盘故障和替换
- [ ] 完成 Go 语言监控工具的开发，并能正确解析和告警

### 实战项目评估
- [ ] `storage-monitor` 工具功能完整，代码符合基本规范
- [ ] 技术对比报告内容详实、分析到位
- [ ] 运维最佳实践文档具有实际指导意义

---

## 🔗 参考资源
1. **官方文档**
   - [Linux RAID Wiki (mdadm)](https://raid.wiki.kernel.org/index.php/Main_Page)
   - [OpenZFS Documentation](https://openzfs.github.io/openzfs-docs/)

2. **Go 开发参考**
   - [Go `os/exec` package](https://pkg.go.dev/os/exec)
   - [Go `regexp` package for parsing](https://pkg.go.dev/regexp)

3. **工具和测试**
   - [fio - Flexible I/O Tester](https://fio.readthedocs.io/en/latest/)
   - [ZFS Best Practices Guide](https://www.truenas.com/docs/core/gettingstarted/storage/zfsbestpractices/)
