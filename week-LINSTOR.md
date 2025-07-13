# DRBD/LINSTOR 专题周：构建高可用复制存储

## 整体目标
- **技能目标**: 深入理解 DRBD 的实时块设备复制原理，掌握其在构建高可用 (HA) 系统中的核心作用。
- **实践目标**: 能够独立部署和管理 LINSTOR 集群，使用 LINSTOR 自动化 DRBD 资源的创建、管理和快照。
- **Go编程目标**: 学会使用 Go 语言与 LINSTOR REST API 交互，开发自定义的监控和管理小工具。
- **架构目标**: 掌握使用 DRBD/LINSTOR 为虚拟机和容器提供高可用持久化存储的架构设计。
- **运维能力**: 学会诊断和处理 DRBD 的常见问题，特别是“裂脑”(Split-Brain) 场景的应对策略。
- **K8s集成**: 为后续在 Kubernetes 中使用 LINSTOR CSI 驱动提供持久化存储打下坚实基础。

---

## Day 1: DRBD 核心原理与手动配置

### 🎯 学习目标
- 理解 DRBD 的工作模式（主/从）和数据同步协议（A, B, C）。
- 手动配置一个双节点的 DRBD 资源。
- 在 DRBD 设备上创建文件系统并进行读写测试。

### 📚 理论学习（上午 2 小时）
1. **DRBD 深度解析**
   - DRBD (Distributed Replicated Block Device) 原理：通过网络复制整个块设备。
   - 主/从 (Primary/Secondary) 角色：任何时候只有一个节点可以挂载读写。
   - 同步协议：
     - **Protocol C (同步)**: 写操作在本地和远程磁盘都确认后才返回。最安全，性能有损耗。
     - **Protocol B (半同步)**: 写操作在本地写入且数据包被对方接收后即返回。
     - **Protocol A (异步)**: 写操作在本地写入后立即返回。性能最高，但主节点故障可能丢数据。

### 🛠️ 实践操作（下午 2 小时）
1. **环境准备**
   - 准备两台虚拟机（node1, node2），每台添加一个同样大小的裸盘（如 `/dev/sdb`）。
2. **手动配置 DRBD**
   ```bash
   # 1. 安装 DRBD 工具
   sudo apt-get install -y drbd-utils

   # 2. 创建 DRBD 资源配置文件 (e.g., /etc/drbd.d/r0.res)
   # resource r0 {
   #   on node1 { device /dev/drbd0; disk /dev/sdb; address 192.168.1.101:7789; }
   #   on node2 { device /dev/drbd0; disk /dev/sdb; address 192.168.1.102:7789; }
   # }

   # 3. 初始化元数据并启动资源
   sudo drbdadm create-md r0
   sudo drbdadm up r0

   # 4. 在一个节点上设置为主节点并进行初次同步
   sudo drbdadm primary --force r0

   # 5. 监控同步状态
   watch -n1 cat /proc/drbd

   # 6. 创建文件系统并使用
   sudo mkfs.ext4 /dev/drbd0
   sudo mount /dev/drbd0 /mnt
   ```

### 🏠 作业
- 练习主/从角色的切换 (`drbdadm secondary r0` 和 `drbdadm primary r0`)。
- 研究 DRBD 配置文件中的 `net` 和 `disk` 选项。

---

## Day 2: LINSTOR 架构与集群部署

### 🎯 学习目标
- 理解 LINSTOR 的架构（Controller, Satellite, Storage Pool）。
- 部署一个 LINSTOR Controller 和多个 Satellite。
- 使用 `linstor` 命令行工具创建存储池。

### 📚 理论学习（上午 2 小时）
1. **为什么需要 LINSTOR?**
   - 手动管理大量 DRBD 资源非常繁琐且容易出错。LINSTOR 提供了集中式的管理和自动化能力。
2. **LINSTOR 架构**
   - **Controller**: 集群的大脑，存储配置和状态，响应 API 请求。通常需要高可用（HA Controller）。
   - **Satellite**: 运行在每个存储节点上，负责执行 Controller 的指令（如创建 LVM 卷、配置 DRBD）。
   - **Storage Pool**: 定义在每个 Satellite 上的可用存储，通常基于 LVM Thin Pool 或 ZFS zpool。
   - **Resource/Volume Definition**: 定义卷的抽象属性（如大小）。
   - **Resource/Volume**: 具体的卷实例，在节点上有具体的设备。

### 🛠️ 实践操作（下午 2 小时）
1. **安装 LINSTOR**
   - 按照官方文档，为你的发行版添加 PPA 或 YUM/DNF 源。
   - 在一个节点上安装 `linstor-controller`。
   - 在所有存储节点上安装 `linstor-satellite` 和 `drbd-utils`。
2. **配置和启动集群**
   ```bash
   # 1. 启动 Controller
   sudo systemctl start linstor-controller

   # 2. 启动 Satellites
   sudo systemctl start linstor-satellite

   # 3. 验证节点是否已连接到 Controller
   linstor node list
   ```
3. **创建存储池**
   - 假设所有节点上都有一个名为 `lvm_vg` 的 LVM 卷组。
   ```bash
   # 在所有节点上创建一个基于 LVM Thin Pool 的存储池
   linstor storage-pool create lvm-thin node1 lvm_vg --thin-pool lvm_thin_pool
   linstor storage-pool create lvm-thin node2 lvm_vg --thin-pool lvm_thin_pool
   ```

### 🏠 作业
- 练习使用 `linstor` CLI 的其他命令，如 `node`, `storage-pool` 的 `list`, `describe` 等。
- 阅读 LINSTOR 用户手册中关于不同存储后端（LVM, ZFS）的部分。

---

## Day 3: 使用 LINSTOR 管理 DRBD 资源

### 🎯 学习目标
- 使用 LINSTOR 创建、删除、查询 DRBD 资源。
- 理解 LINSTOR 如何自动化资源放置和设备创建。
- 实现对 LINSTOR 管理的卷进行快照。

### 🛠️ 实践操作（全天）
1. **创建资源定义 (Resource Definition)**
   ```bash
   # 创建一个名为 'web-storage' 的资源定义
   linstor resource-definition create web-storage
   ```
2. **创建卷定义 (Volume Definition)**
   ```bash
   # 在 'web-storage' 资源定义下创建一个大小为 10G 的卷定义
   linstor volume-definition create web-storage 10G
   ```
3. **部署资源 (Deploy Resource)**
   ```bash
   # 在两个节点上部署该资源，实现双副本复制
   linstor resource create node1 web-storage --storage-pool lvm_thin_pool
   linstor resource create node2 web-storage --storage-pool lvm_thin_pool

   # LINSTOR 会自动完成 DRBD 设备的创建和配置
   # 查看资源状态
   linstor resource list
   ```
4. **使用资源**
   - LINSTOR 创建的设备路径通常是 `/dev/drbd/by-res/web-storage/0`。
   - 在一个节点上挂载并使用它。

5. **创建快照**
   ```bash
   # 为资源创建一个快照
   linstor snapshot create node1 web-storage my_snapshot

   # 查看快照
   linstor snapshot list
   ```

### 💻 Go 编程实践
- **目标**: 编写一个 Go 程序，连接到 LINSTOR API 并列出所有节点的状态。
- **提示**: LINSTOR Controller 默认在端口 3370/3371 上提供 REST API。你可以使用 Go 的 `net/http` 包来发出 GET 请求到 `http://<controller_ip>:3370/v1/nodes`。

### 🏠 作业
- 练习创建一个三副本的资源（需要三个节点）。
- 尝试从一个快照恢复（创建一个新卷）。

---

## Day 4: 故障处理与高可用

### 🎯 学习目标
- 理解 DRBD 的“裂脑”(Split-Brain) 问题及其成因。
- 学习 LINSTOR 的裂脑自动恢复策略。
- 模拟节点故障，并练习手动或自动的角色切换。

### 📚 理论学习（上午 2 小时）
1. **裂脑 (Split-Brain)**
   - **成因**: 当集群节点间的通信中断，且没有外部仲裁机制时，两个节点可能都会认为自己是 Primary，并各自接受写操作。当通信恢复时，数据无法合并，就形成了“裂脑”。
   - **后果**: 数据不一致，是分布式系统中最严重的问题之一。
   - **恢复策略**: 必须放弃其中一个节点的数据。LINSTOR 提供了自动恢复策略，如 `discard-zero-changes`, `discard-least-changes` 等。

### 🛠️ 实践操作（下午 2 小时）
1. **模拟网络故障**
   - 使用 `iptables` 或其他防火墙工具断开两个节点之间的 DRBD 通信端口 (7789)。
   - 尝试将两个节点都提升为 Primary (会失败一个)，观察 `drbd-overview` 或 `linstor resource list` 中的状态变化。
2. **恢复裂脑**
   - 恢复网络连接。
   - 使用 `linstor` 命令手动解决裂脑，选择一个节点作为“胜利者”。
   ```bash
   # 放弃 node2 上的更改，以 node1 为准
   linstor resource-definition set-property web-storage DrbdOptions/auto-recover-target-role --discard-secondary
   ```
3. **模拟节点宕机**
   - 强制关闭一个持有 Primary 角色的节点。
   - 在另一个节点上，将资源提升为 Primary，挂载并继续提供服务。

### 🏠 作业
- 详细阅读 LINSTOR 用户手册中关于高可用 Controller 的配置方法。
- 研究 DRBD Proxy，了解其在广域网复制场景中的作用。

---

## Day 5: Kubernetes 集成与总结

### 🎯 学习目标
- 理解 CSI (Container Storage Interface) 的基本概念。
- 在测试 Kubernetes 集群中部署 LINSTOR CSI 驱动。
- 通过 PVC 动态创建 LINSTOR 支持的持久化存储。

### 📚 理论学习（上午 2 小时）
1. **CSI 简介**
   - CSI 是一个标准接口，旨在将任意存储系统暴露给容器编排系统（如 Kubernetes）。
   - 它将存储插件的开发与 Kubernetes 本身的发布周期解耦。
   - 主要组件：`external-provisioner`, `external-attacher`, `external-resizer`, `node-driver-registrar` 和存储厂商自己实现的 CSI 驱动。

### 🛠️ 实践操作（下午 2 小时）
1. **环境准备**
   - 一个测试 Kubernetes 集群（如 Minikube, Kind, k3s）。
   - 一个已部署好的 LINSTOR 集群。
2. **部署 LINSTOR CSI 驱动**
   - 通常使用 Helm 或 Kustomize/kubectl 来部署。
   ```bash
   # 使用 Helm
   helm repo add piraeus-charts https://piraeus.io/helm-charts/
   helm install piraeus-csi piraeus-charts/piraeus --set csi.controller.linstorEndpoint=http://<controller_ip>:3370
   ```
3. **创建 StorageClass**
   - 创建一个 `StorageClass` YAML 文件，`provisioner` 指向 `linstor.csi.linbit.com`。
   ```yaml
   apiVersion: storage.k8s.io/v1
   kind: StorageClass
   metadata:
     name: linstor-replicated-sc
   provisioner: linstor.csi.linbit.com
   parameters:
     csi.storage.k8s.io/fstype: ext4
     linstor.csi.linbit.com/placementCount: "2"
     linstor.csi.linbit.com/storagePool: "lvm_thin_pool"
   ```
4. **创建 PVC 和 Pod**
   - 创建一个 `PersistentVolumeClaim` (PVC) 来请求存储。
   - 创建一个 Pod 来使用这个 PVC。
   - 验证 Pod 可以成功读写，并且 `linstor resource list` 中出现了新的资源。

### 🤔 架构总结与复盘
- 对比 DRBD/LINSTOR 与 Ceph RBD 在提供块存储方面的优缺点。
- 总结 LINSTOR 在 Kubernetes 环境下作为持久化存储方案的优势（如高性能、本地读写、跨节点复制）。

### 🏠 本周作业交付
- **Go 工具**: 提交一个可以列出 LINSTOR 节点、存储池、资源的 Go 小工具。
- **技术文档**: 撰写一份 DRBD 裂脑问题的分析及恢复手册。
- **K8s 实践报告**: 记录在 K8s 中部署和使用 LINSTOR CSI 的完整步骤和截图。
