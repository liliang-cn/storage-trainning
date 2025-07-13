# 第 4 周：Kubernetes 存储模型与 CSI 实践

## 整体目标
- **技能目标**: 深入理解 Kubernetes 的存储模型，掌握 StorageClass, PersistentVolume (PV), PersistentVolumeClaim (PVC) 的关系和生命周期。
- **实践目标**: 能够熟练地通过 YAML 创建和管理 K8s 存储资源，并为 Pod 提供持久化存储。
- **核心概念**: 掌握 CSI (Container Storage Interface) 插件的架构原理，并能成功部署和使用一个 CSI 驱动（如 LINSTOR CSI）。
- **Go编程目标**: 学会使用 Go 的 `client-go` 库与 Kubernetes API Server 交互，编写程序来查询和管理 PV, PVC 等存储资源。
- **架构目标**: 理解有状态应用 (StatefulSet) 如何利用持久化存储，并掌握其部署模式。
- **运维能力**: 学会诊断 K8s 存储相关的常见问题，如 PVC Pending、挂载失败等。

---

## Day 1: Kubernetes 存储核心概念 (PV, PVC, StorageClass)

### 🎯 学习目标
- 理解 PV, PVC, StorageClass 的作用和它们之间的解耦关系。
- 掌握静态和动态两种 PV 공급模式。
- 手动创建 PV 和 PVC，并将其绑定给一个 Pod 使用。

### 📚 理论学习
1. **PV (PersistentVolume)**: 由管理员创建的、集群中的一块存储资源。它像一个可用的“磁盘”，拥有独立的生命周期。
2. **PVC (PersistentVolumeClaim)**: 由用户（开发者）创建的、对存储的“请求”。它声明了需要多大的空间、需要什么样的访问模式（如 ReadWriteOnce）。
3. **绑定过程**: Kubernetes 会寻找能满足 PVC 要求的 PV，并将它们绑定在一起。
4. **StorageClass**: 为管理员提供了一种描述“存储类别”的方法。用户可以在 PVC 中指定一个 StorageClass，从而触发**动态供给 (Dynamic Provisioning)**，由存储插件自动创建一个匹配的 PV。

### 🛠️ 实践操作
- 使用 `hostPath` 或本地 NFS 服务器作为后端，手动创建一个 PV。
- 创建一个 PVC，观察它如何与手动创建的 PV 绑定。
- 创建一个 Pod，挂载该 PVC，并验证数据持久性。
- 创建一个简单的 StorageClass，然后创建一个新的 PVC，观察动态供给的过程。

### 🏠 作业
- 研究不同的 `accessModes` (`ReadWriteOnce`, `ReadOnlyMany`, `ReadWriteMany`) 的含义和支持它们的存储类型。
- 研究不同的 `reclaimPolicy` (`Retain`, `Delete`, `Recycle`) 的作用。

---

## Day 2: 有状态应用与 StatefulSet

### 🎯 学习目标
- 理解无状态应用 (Deployment) 和有状态应用 (StatefulSet) 的核心区别。
- 掌握 StatefulSet 的关键特性：稳定的网络标识和稳定的持久化存储。
- 部署一个简单的有状态应用（如 Zookeeper 或 Redis），并为其提供稳定的存储。

### 📚 理论学习
1. **StatefulSet 特性**: 
   - **稳定的网络 ID**: Pod 的名称是可预测且有序的（如 `web-0`, `web-1`）。
   - **稳定的存储**: 每个 Pod 都会根据 `volumeClaimTemplates` 获得一个独一无二的、稳定绑定的 PVC。
   - **有序的部署和伸缩**: Pod 会按照顺序创建和销毁。

### 🛠️ 实践操作
- 编写一个简单的 `StatefulSet` YAML 文件。
- 在 `volumeClaimTemplates` 部分定义 PVC 模板。
- 部署 StatefulSet，并观察 Pod 和 PVC 是如何按顺序创建的 (`web-0`, `my-pvc-web-0`)。
- 模拟 Pod 故障，观察 K8s 如何在重建 Pod 后，将其重新绑定到原来的 PVC 上，保证数据不丢失。

### 🏠 作业
- 尝试对 StatefulSet 进行扩容 (`kubectl scale`) 和缩容，观察 Pod 和 PVC 的行为。
- 研究 Headless Service 如何与 StatefulSet 配合，为每个 Pod 提供一个唯一的 DNS 域名。

---

## Day 3: CSI 插件架构与部署

### 🎯 学习目标
- 深入理解 CSI 的架构和其主要组件的作用。
- 在 Kubernetes 集群中成功部署 LINSTOR CSI 驱动。
- 使用 LINSTOR CSI 驱动动态创建一个多副本的持久化卷。

### 📚 理论学习
1. **CSI 组件回顾**: `external-provisioner`, `external-attacher`, `external-resizer`, `csi-driver` (Node 和 Controller 部分)。
2. **CSI 工作流程**: 从 PVC 创建到 Pod 挂载的全过程，理解每个组件在其中扮演的角色。

### 🛠️ 实践操作
- （复习 LINSTOR 专题周 Day 5 的内容）
- 使用 Helm 在 K8s 集群中部署 LINSTOR CSI 驱动。
- 创建一个基于 LINSTOR 的 `StorageClass`，并明确指定副本数量（如 `placementCount: "3"`）。
- 创建一个 PVC，并使用 `linstor resource list` 在后端验证 DRBD 资源是否已按要求创建了三个副本。
- 部署一个 Pod 使用该 PVC，并验证其高可用特性（模拟节点故障）。

### 🏠 作业
- 研究 LINSTOR StorageClass 中的其他高级参数，如 `allowRemoteVolumeAccess`，并测试其效果。
- 阅读 CSI 官方文档，了解 `VolumeSnapshot` 功能。

---

## Day 4: Go client-go 实践：与 K8s API 交互

### 🎯 学习目标
- 掌握 `client-go` 库的基本用法，学会如何配置 `clientset` 来连接 K8s 集群。
- 使用 `client-go` 编写程序来列出、获取和创建 K8s 资源（特别是 PV 和 PVC）。
- 理解 `informer` 和 `lister` 的概念，学会如何高效地从本地缓存中查询资源。

### 📚 理论学习
1. **`client-go` 简介**: Kubernetes 官方提供的 Go 语言客户端库，用于与 API Server 进行交互。
2. **`clientset`**: 包含了访问所有 K8s API Group（如 `core/v1`, `apps/v1`）的客户端集合。
3. **Informer 机制**: `client-go` 的核心。它会高效地 watch API Server 的资源变化，并将其同步到一个本地的内存缓存中。直接查询这个缓存可以大大减轻 API Server 的压力。

### 🛠️ 实践操作
- **项目: `k8s-storage-lister`**
- **步骤**:
  1. 初始化 Go 项目，并引入 `k8s.io/client-go` 等依赖。
  2. 配置 `kubeconfig`，创建一个 `clientset`。
  3. 使用 `clientset.CoreV1().PersistentVolumes().List()` 来列出集群中所有的 PV。
  4. 格式化输出 PV 的名称、容量、状态等信息。
  5. （进阶）创建一个 `PVCInformer`，并使用其 `Lister()` 从本地缓存中查询 PVC。

### 🏠 作业
- 扩展你的 Go 程序，使其能够接受一个 PVC 名称作为参数，并打印出该 PVC 的详细信息（`Get()` 方法）。
- 尝试使用 `client-go` 动态创建一个新的 PVC（需要编写 YAML 的 Go 结构体表示）。

---

## Day 5: K8s 存储运维与高级主题

### 🎯 学习目标
- 学会诊断 K8s 存储相关的常见问题。
- 掌握 PVC 的在线和离线扩容操作。
- 了解 K8s 的存储快照和备份恢复机制。

### 📚 理论学习
1. **存储运维**: 
   - **PVC Pending**: PVC 一直处于 `Pending` 状态的常见原因（没有匹配的 PV、StorageClass 不存在、资源配额不足）。
   - **Pod 挂载失败**: `FailedMount` 或 `FailedAttach` 事件的排查思路（CSI 驱动日志、Kubelet 日志）。
2. **高级功能**: 
   - **卷扩容**: 如果 StorageClass 设置了 `allowVolumeExpansion: true`，可以直接编辑 PVC 的 `spec.resources.requests.storage` 来扩容。
   - **卷快照**: 通过 `VolumeSnapshot` 和 `VolumeSnapshotClass` CRD，可以为 PVC 创建时间点快照，并从中恢复出新的 PVC。

### 🛠️ 实践操作
- **故障模拟**: 故意创建一个请求不存在的 StorageClass 的 PVC，观察其 `Pending` 状态和相关事件。
- **卷扩容**: 对一个已有的 PVC（由支持扩容的 CSI 驱动创建）进行在线扩容，并验证 Pod 内的文件系统也已扩展。
- **卷快照与恢复** (如果 CSI 驱动支持):
  1. 安装 `snapshot-controller`。
  2. 创建一个 `VolumeSnapshotClass`。
  3. 为一个 PVC 创建一个 `VolumeSnapshot` 对象。
  4. 从该快照创建一个新的 PVC，并验证数据。

### 🤔 架构总结与复盘
- 总结在 Kubernetes 中为有状态应用提供存储的最佳实践。
- 梳理 Go 与 Kubernetes API 的交互模式，为开发自定义控制器 (Operator) 打下基础。

### 🏠 本周作业交付
- **Go 工具**: 提交 `k8s-storage-lister` 项目，能列出 PV 和 PVC。
- **技术文档**: 撰写一份 Kubernetes 存储问题排查手册。
- **实践报告**: 记录使用 StatefulSet 和 LINSTOR CSI 部署一个高可用应用的完整过程。
