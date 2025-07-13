# Day 5: Kubernetes 集成与总结

## 🎯 学习目标
- **技能目标**: 理解 CSI (Container Storage Interface) 作为 Kubernetes 存储标准的重要性。
- **实践目标**: 能够在一个测试 Kubernetes 集群中成功部署 LINSTOR CSI 驱动，并通过 PVC 动态创建由 LINSTOR 管理的持久化存储。
- **架构目标**: 掌握 LINSTOR 作为 Kubernetes 有状态应用（如数据库）后端存储的架构模式和优势。
- **成果产出**: 一个在 K8s 中运行、使用 LINSTOR 持久化存储的 Pod，一份 LINSTOR vs Ceph RBD 的对比分析报告。

## 📚 理论基础 (40%)

### 1. CSI (Container Storage Interface) 简介
在 CSI 出现之前，Kubernetes 的存储驱动是“in-tree”（内置在 Kubernetes 代码中）的。这导致了几个问题：
- **更新缓慢**: 存储厂商必须等待 Kubernetes 的发布周期才能更新他们的驱动。
- **代码臃肿**: Kubernetes 的核心代码库中包含了大量第三方存储的代码。
- **稳定性风险**: 存储驱动的 bug 可能会影响到整个 Kubelet 的稳定性。

CSI 的出现解决了这些问题。它是一个标准化的 API 规范，旨在将存储系统的实现与 Kubernetes 的核心逻辑解耦。

- **CSI 工作原理**: 
  1.  **CSI 驱动**: 由存储厂商（如 LINBIT）提供的一组服务（通常是 Pod），它们实现了 CSI 规范中定义的 gRPC 接口。
  2.  **Sidecar 容器**: Kubernetes 提供了一组标准的 sidecar 容器（如 `external-provisioner`, `external-attacher`），它们负责监听 Kubernetes 的 API 事件（如 PVC 创建），并将这些事件转换成对 CSI 驱动的 gRPC 调用。
  3.  **流程**: 
      -   用户创建一个 `PersistentVolumeClaim` (PVC)。
      -   `external-provisioner` sidecar 监听到这个事件，调用 CSI 驱动的 `CreateVolume` 接口。
      -   CSI 驱动（在我们的例子中是 LINSTOR CSI 驱动）接收到调用，然后通过 LINSTOR API 在后端创建一个 DRBD 卷。
      -   卷创建成功后，CSI 驱动返回卷的信息，`external-provisioner` 据此创建一个 `PersistentVolume` (PV) 对象并与 PVC 绑定。
      -   当用户创建一个使用该 PVC 的 Pod 时，Kubelet 会调用 CSI 驱动的 `NodePublishVolume` 接口，将卷挂载到 Pod 内部。

### 2. LINSTOR 在 Kubernetes 中的优势
相比于其他分布式存储方案（如 Ceph），LINSTOR 在某些场景下具有独特优势：
- **高性能本地读**: 当 Pod 调度到数据所在的节点时，读操作是纯本地的，不经过网络，延迟极低。
- **数据局部性 (Data Locality)**: LINSTOR CSI 驱动会告知 Kubernetes 调度器数据副本存在于哪些节点上。调度器会倾向于将 Pod 调度到这些节点，从而最大化本地读的概率。
- **简单高效**: LINSTOR 的架构相对 Ceph 更简单，部署和维护开销较低，非常适合中小型集群或对性能要求极高的场景。
- **同步复制**: DRBD 的同步复制协议 (Protocol C) 可以为数据库等关键应用提供零数据丢失 (RPO=0) 的保障。

## 🛠️ 实践操作 (50%)

### 环境准备
- 一个可用的 Kubernetes 集群（如 Minikube, Kind, k3s, or a cloud-based one）。
- 一个已部署好的 LINSTOR 集群，并且 K8s 的 worker 节点就是 LINSTOR 的 Satellite 节点。
- `kubectl` 和 `helm` 命令行工具已配置好。

### 1. 部署 LINSTOR CSI 驱动
使用 Helm 是最简单的方式。

```bash
# 1. 添加 Piraeus (LINSTOR 项目的社区名称) Helm 仓库
helm repo add piraeus-charts https://piraeus.io/helm-charts/
helm repo update

# 2. 安装 CSI 驱动
# 需要将 linstorEndpoint 指向你的 LINSTOR Controller 的 IP 地址和端口
helm install piraeus-csi piraeus-charts/piraeus \
    --namespace kube-system \
    --set csi.controller.linstorEndpoint=http://192.168.1.101:3370

# 3. 验证 CSI Pod 是否正常运行
kubectl get pods -n kube-system -l app.kubernetes.io/name=piraeus
# 你应该能看到 csi-linstor-controller 和 csi-linstor-node 等 Pods
```

### 2. 创建 StorageClass
StorageClass 告诉 Kubernetes 如何动态创建存储。我们来创建一个使用 LINSTOR 的双副本存储类。

**`linstor-sc.yaml`**
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: linstor-replicated-sc
provisioner: linstor.csi.linbit.com
allowVolumeExpansion: true
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer # 推荐！这会延迟 PV 创建，直到 Pod 被调度，从而让调度器做出更好的数据局部性决策
parameters:
  # === LINSTOR Parameters ===
  linstor.csi.linbit.com/placementCount: "2" # 创建2个副本
  linstor.csi.linbit.com/storagePool: "sp1"   # 使用我们之前创建的存储池
  # 可选：linstor.csi.linbit.com/allowRemoteVolumeAccess: "false" # 强制Pod只能调度到数据所在节点

  # === Filesystem Parameter ===
  csi.storage.k8s.io/fstype: ext4
```

```bash
# 应用 StorageClass
kubectl apply -f linstor-sc.yaml
```

### 3. 创建 PVC 和 Pod

**`test-pvc-pod.yaml`**
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-linstor-pvc
spec:
  accessModes:
    - ReadWriteOnce # DRBD 是块设备，通常用于 ReadWriteOnce
  storageClassName: linstor-replicated-sc
  resources:
    requests:
      storage: 5Gi
---
apiVersion: v1
kind: Pod
metadata:
  name: my-test-pod
spec:
  containers:
  - name: app
    image: busybox
    command: ["sh", "-c", "while true; do echo $(date) >> /data/outfile.txt; sleep 5; done"]
    volumeMounts:
    - name: my-data
      mountPath: /data
  volumes:
  - name: my-data
    persistentVolumeClaim:
      claimName: my-linstor-pvc
```

```bash
# 1. 应用 PVC 和 Pod
kubectl apply -f test-pvc-pod.yaml

# 2. 观察资源创建过程
watch kubectl get pvc,pv,pod
# 等待 Pod 变为 Running

# 3. 在 LINSTOR 中验证
linstor resource list
# 你应该能看到一个由 K8s 创建的新资源

# 4. 验证数据写入
kubectl exec my-test-pod -- tail /data/outfile.txt
```

## 🤔 架构总结与复盘 (10%)

### LINSTOR vs Ceph RBD 对比

| 特性 | LINSTOR/DRBD | Ceph RBD |
| :--- | :--- | :--- |
| **架构** | 主从复制 (RAID-1 like) | 分布式对象存储 (CRUSH 算法) |
| **最小节点数** | 2 | 3 (推荐 5+) |
| **性能模型** | 读写路径短，本地读性能极高 | 所有 I/O 都经过网络，延迟相对较高 |
| **数据一致性** | 支持强同步复制 (RPO=0) | 最终一致性 (副本间) |
| **扩展性** | 线性扩展，但节点数不宜过多 | 极强的水平扩展能力，可达 PB 级 |
| **复杂度** | 相对简单 | 复杂，组件多，运维门槛高 |
| **适用场景** | 数据库、虚拟机、对延迟敏感的应用 | 大规模云平台、对象存储、需要海量扩展的场景 |

## 🏠 本周作业交付
- **Go 工具**: 提交你的 LINSTOR 监控 Go 程序的最终版本，应至少能列出节点、存储池和资源，并高亮显示异常状态。
- **技术文档**: 提交一份图文并茂的 DRBD 裂脑问题分析及恢复手册，详细描述你如何模拟和解决裂脑问题。
- **K8s 实践报告**: 提交一份完整的报告，记录你在 Kubernetes 中部署和使用 LINSTOR CSI 的所有步骤、遇到的问题和解决方案，并附上关键的 `kubectl` 和 `linstor` 命令输出截图。
