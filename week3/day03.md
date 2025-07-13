# Day 3: 高级网络存储特性与安全

## 🎯 学习目标
- **技能目标**: 理解并能理论上规划 iSCSI 多路径 (MPIO) 以实现高可用和负载均衡。
- **安全目标**: 掌握为 NFSv4 配置 Kerberos 安全认证的核心概念和流程。
- **前瞻目标**: 初步了解 NVMe-oF (NVMe over Fabrics) 的基本概念、优势和应用场景。
- **成果产出**: 一份详细的 iSCSI MPIO 架构图和数据流说明，一份 NFS with Kerberos 的认证流程图，一份 NVMe-oF vs iSCSI 的技术对比分析报告。

## 📚 理论基础 (50%)

### 1. iSCSI 多路径 (MPIO - Multipath I/O)
在生产环境中，单点故障是不可接受的。如果 Initiator 和 Target 之间只有一条网络路径，那么交换机、网卡或网线的任何故障都会导致存储中断。MPIO 就是为了解决这个问题。

- **核心思想**: 为 Initiator 和 Target 之间的 LUN 提供多条独立的网络路径。这些路径可以分布在不同的交换机、网卡上。
- **两大优势**:
  - **高可用性 (High Availability)**: 当一条路径发生故障时，I/O 会自动切换到其他健康的路径上，业务无感知。
  - **负载均衡 (Load Balancing)**: I/O 请求可以同时在多条路径上传输，从而聚合多条链路的带宽，提升整体性能。
- **工作模式**:
  - **Failover**: 平时只有一条路径是激活（Active）状态，其他路径处于备用（Standby）状态。当激活路径故障，备用路径才会被启用。
  - **Round-Robin**: 所有路径轮流处理 I/O 请求，实现简单的负载均衡。
  - **Least Queue Depth**: 将新的 I/O 请求发送到当前排队请求最少的路径，是一种更智能的负载均衡。

### 2. NFSv4 with Kerberos 安全认证
标准的 NFS (`sec=sys`) 依赖于客户端声称的 UID/GID，这是极不安全的，因为任何有 root 权限的客户端都可以伪造用户身份。Kerberos 提供了一种基于密码学的强认证机制。

- **核心组件**:
  - **KDC (Key Distribution Center)**: 密钥分发中心，是 Kerberos 的核心。它包含认证服务器 (AS) 和票据授予服务器 (TGS)。
  - **Principal**: Kerberos 系统中的一个唯一身份标识，可以是用户（如 `user@REALM`）或服务（如 `nfs/server.example.com@REALM`）。
  - **Ticket (票据)**: 客户端从 KDC 获取的加密凭证，用于向特定的服务证明自己的身份。
- **认证流程 (简化版)**:
  1. **用户认证**: 用户使用 `kinit` 命令，用自己的密码向 KDC 的 AS 请求一个初始票据——票据授予票据 (TGT)。
  2. **请求服务票据**: 当客户端（代表用户）想访问 NFS 服务时，它会拿着 TGT 去向 KDC 的 TGS 请求一个用于访问 NFS 服务的特定服务票据。
  3. **访问服务**: 客户端将服务票据呈现给 NFS 服务器。NFS 服务器可以解密该票据（因为它也在 KDC 注册过并拥有密钥），从而验证客户端的身份。
- **安全级别 (`sec=` 选项)**:
  - `krb5`: 只进行身份验证。
  - `krb5i`: 在 `krb5` 的基础上，对每个请求进行加密校验和，防止数据包被篡改 (完整性保护)。
  - `krb5p`: 在 `krb5i` 的基础上，对客户端和服务器之间的所有通信进行加密 (隐私保护)，性能开销最大。

### 3. 未来技术: NVMe-oF (NVMe over Fabrics)
NVMe 是专为 SSD 设计的高性能存储协议，它取代了为机械硬盘设计的传统 AHCI/SATA 协议。NVMe-oF 则是将 NVMe 的能力扩展到了网络上。

- **核心思想**: 将 NVMe 命令直接封装在网络协议中进行传输，绕过了传统 iSCSI 的 SCSI-to-IP 转换层，从而极大地降低了延迟。
- **底层传输协议 (Fabrics)**:
  - **RDMA (Remote Direct Memory Access)**: 如 RoCE 或 iWARP，允许一台计算机的内存直接读写另一台计算机的内存，延迟极低。
  - **Fibre Channel**: 可以利用现有的 FC 网络设施。
  - **TCP**: 最新的 NVMe/TCP 方案，使其可以在标准以太网上运行，降低了部署门槛。
- **NVMe-oF vs iSCSI**:
  - **性能**: NVMe-oF 的延迟通常在微秒级，远低于 iSCSI 的毫秒级。
  - **效率**: NVMe-oF 的 CPU 开销更低，因为它更接近硬件。
  - **应用场景**: 对延迟和性能要求极高的场景，如高性能计算 (HPC)、大规模数据库、AI/ML 训练等。

## 🛠️ 实践操作 (20%)

由于 MPIO 和 Kerberos 的配置相当复杂，需要特定的多网卡和 KDC 环境，本节的实践以**理论规划和流程梳理**为主。

### 1. 规划 iSCSI MPIO 架构
- **任务**: 画出一张包含以下组件的 iSCSI MPIO 架构图：
  - 1 个 iSCSI Initiator (Client)
  - 1 个 iSCSI Target (Server)
  - 2 个独立的网络交换机 (Switch A, Switch B)
  - Initiator 和 Target 均有两张网卡，分别连接到两个交换机。
- **要求**: 在图中标明每条网络路径，并用文字说明当其中一条路径（如 Switch A 故障）时，数据流如何切换。

### 2. 梳理 NFS with Kerberos 配置流程
- **任务**: 以文档形式，按顺序写出配置 NFS with Kerberos 的主要步骤。
- **要求**: 不需要写出具体命令，但要清晰地描述每一步的目标。
  - **示例步骤**: 
    1. 安装和配置 KDC 服务器。
    2. 在 KDC 中为 NFS 服务器和测试用户创建 Principal。
    3. 为 NFS 服务生成 keytab 文件并分发到 NFS 服务器。
    4. 配置 NFS 服务器，在 `/etc/exports` 中启用 `sec=krb5i`。
    5. 配置 NFS 客户端，确保 `krb5.conf` 正确。
    6. 在客户端上，用户使用 `kinit` 获取票据。
    7. 客户端尝试挂载 NFS 共享。
    8. 验证挂载是否成功，以及是否使用了 Kerberos 认证。

## 💻 Go 编程实现 (0%)

今天没有 Go 编程任务，重点是消化高级理论知识。

## 🔍 故障排查与优化
- **MPIO 不工作**: 
  - **问题**: `multipath -ll` 没有显示聚合设备，或者路径没有按预期工作。
  - **排查**: 检查 `multipath-tools` 是否安装并启动；检查 `/etc/multipath.conf` 配置是否正确，特别是 `defaults` 和 `devices` 部分；确认所有网络路径在物理和 IP 层都是通的。
- **Kerberos 认证失败**: 
  - **问题**: `kinit` 失败或挂载时提示 `Access denied by server`。
  - **排查**: 检查 KDC、服务器和客户端的时间是否同步 (Kerberos 对时间敏感)；检查 `/etc/krb5.conf` 中的 REALM 和 KDC 地址是否正确；确认服务器上的 keytab 文件权限和内容是否正确。

## 📝 实战项目
- **架构设计**: 为一个需要 99.99% 可用性的数据库设计其存储方案。方案应至少包含 RAID 级别选择、iSCSI MPIO 路径设计，并说明为什么这样设计。
- **安全评估**: 分析在一个中型企业环境中，从 `sec=sys` 升级到 `sec=krb5i` 的成本（人力、复杂度）和收益（安全性提升），并给出一个是否值得升级的建议。

## 🏠 课后作业
- **深入研究**: 详细对比 iSCSI MPIO 和 LVM Mirroring (在两个不同的 PV 上创建镜像) 在实现高可用性方面的异同点。从故障检测机制、切换速度、性能影响、管理复杂度等角度进行分析。
- **技术报告**: 阅读至少两篇关于 NVMe/TCP 的技术文章，撰写一份不少于 500 字的总结报告，阐述你认为 NVMe/TCP 是否会在未来 5 年内大规模取代 iSCSI，并说明理由。
