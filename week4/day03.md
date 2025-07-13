# Day 3: CSI 插件架构与部署

## 🎯 学习目标
- **技能目标**: 深入理解 CSI (Container Storage Interface) 的架构，能够清晰地描述其核心组件（Sidecar Containers）及其各自的职责。
- **具体成果**:
  - 能够成功地在 Kubernetes 集群中部署 LINSTOR CSI 驱动。
  - 能够使用 LINSTOR CSI 动态地创建一个具有3个副本的高可用持久化卷。
  - 能够通过模拟节点故障，验证 LINSTOR CSI 卷的高可用性。
  - 能够解释从 PVC 创建到 Pod 成功挂载卷的完整工作流程。

## 📚 理论基础 (40%)
### 1. 为什么需要 CSI？
在 CSI 出现之前，Kubernetes 的存储驱动代码是直接内置在 Kubelet 和 Controller-Manager 的主代码库中的（称为 "in-tree" 驱动）。这种方式带来了几个严重的问题：
- **更新困难**: 存储厂商想要修复一个 bug 或增加一个新功能，必须等待 Kubernetes 的下一个版本发布，周期长达数月。
- **代码臃肿**: Kubernetes 核心代码库中包含了大量第三方存储的代码，难以维护和测试。
- **安全风险**: 第三方驱动的 bug 可能会直接影响到 Kubernetes 核心组件的稳定性。

CSI 的诞生就是为了解决这些问题。它定义了一套标准的 gRPC 接口，将存储驱动的实现与 Kubernetes 完全解耦（称为 "out-of-tree" 驱动）。存储厂商现在可以独立开发、测试和发布他们的驱动，而无需修改任何 Kubernetes 核心代码。

### 2. CSI 架构深度解析
一个完整的 CSI 驱动通常由两部分组成：一组由 Kubernetes 官方维护的 **Sidecar 辅助容器** 和一组由存储厂商实现的 **CSI 驱动容器**。

#### a. Sidecar 辅助容器 (The Sidecars)
这些是标准的、可重用的容器，它们负责与 Kubernetes API Server 交互，并将 K8s 的存储操作（如创建/删除卷）转换为对 CSI 驱动的 gRPC 调用。

- **`external-provisioner`**:
  - **职责**: 监听（Watch）`PersistentVolumeClaim` 对象的创建。
  - **工作流程**: 当一个 PVC 请求的 `StorageClass` 指向本 CSI 驱动时，它会调用 CSI 驱动的 `CreateVolume` gRPC 接口来创建后端存储和对应的 PV。
- **`external-attacher`**:
  - **职责**: 监听 `VolumeAttachment` 对象的创建。
  - **工作流程**: 当 K8s 调度器决定将一个 Pod 调度到某个节点时，它会创建一个 `VolumeAttachment` 对象。`external-attacher` 监听到后，会调用 CSI 驱动的 `ControllerPublishVolume` gRPC 接口，将卷“附加”（Attach）到目标节点上（例如，在云环境中将 EBS 盘挂载到 EC2 实例上）。
- **`external-resizer`**:
  - **职责**: 监听 PVC 对象中存储容量的变化。
  - **工作流程**: 当用户修改 PVC 的 `spec.resources.requests.storage` 请求扩容时，它会调用 CSI 驱动的 `ControllerExpandVolume` gRPC 接口来扩展后端卷的容量。
- **`external-snapshotter`**:
  - **职责**: 监听 `VolumeSnapshot` CRD 对象的创建。
  - **工作流程**: 调用 CSI 驱动的 `CreateSnapshot` gRPC 接口来为卷创建快照。
- **`node-driver-registrar`**:
  - **职责**: 在每个节点上，将 CSI 驱动注册到该节点的 Kubelet。
  - **工作流程**: 它通过 Kubelet 的插件注册机制，告诉 Kubelet 本节点上有一个 CSI 驱动，并提供了驱动的 gRPC socket 地址。

#### b. CSI 驱动容器 (The Driver)
这部分由存储厂商实现，它包含两个核心组件，通常部署在不同的 Pod 中：

- **Controller Plugin**:
  - **部署方式**: 通常以 `Deployment` 或 `StatefulSet` 的形式部署，在集群中运行一个或多个实例。
  - **职责**: 实现那些不依赖于特定节点的操作，是“控制面”逻辑。
  - **实现的 gRPC 接口**:
    - `CreateVolume` / `DeleteVolume`: 创建和删除卷。
    - `ControllerPublishVolume` / `ControllerUnpublishVolume`: 将卷附加/分离到节点。
    - `ControllerExpandVolume`: 扩容卷。
    - `CreateSnapshot` / `DeleteSnapshot`: 创建和删除快照。
- **Node Plugin**:
  - **部署方式**: 必须以 `DaemonSet` 的形式部署，确保在每个需要挂载存储的节点上都运行一个实例。
  - **职责**: 实现那些必须在特定节点上执行的操作，是“数据面”逻辑。
  - **实现的 gRPC 接口**:
    - `NodeStageVolume`: 对卷进行格式化和预挂载（如果需要）。
    - `NodePublishVolume`: 将卷真正挂载到 Pod 的指定目录。
    - `NodeGetVolumeStats`: 获取卷的使用统计信息。

### 3. 从 PVC 创建到 Pod 挂载的完整流程
1.  用户创建一个 PVC，指定了使用 LINSTOR CSI 的 StorageClass。
2.  `external-provisioner` 监听到新的 PVC，调用 LINSTOR **Controller Plugin** 的 `CreateVolume` 接口。
3.  LINSTOR Controller Plugin 与 LINSTOR Controller 通信，在后端存储池中创建 DRBD 资源，并返回卷的详细信息。`external-provisioner` 收到响应后，在 K8s 中创建出对应的 PV 对象。
4.  PV 和 PVC 成功绑定。
5.  用户创建一个 Pod，并引用了该 PVC。
6.  K8s 调度器将 Pod 分配到一个节点（例如 `node-1`）。
7.  `external-attacher` 监听到 `VolumeAttachment` 对象，调用 LINSTOR **Controller Plugin** 的 `ControllerPublishVolume` 接口，通知 LINSTOR 该卷将在 `node-1` 上使用。
8.  Kubelet 在 `node-1` 上准备挂载卷，它通过 `node-driver-registrar` 注册的 socket 地址，调用 LINSTOR **Node Plugin** (`node-1` 上的实例) 的 `NodeStageVolume` 接口。
9.  LINSTOR **Node Plugin** 在 `node-1` 上执行 `drbdadm attach` 等命令，准备好块设备。
10. Kubelet 再次调用 LINSTOR **Node Plugin** 的 `NodePublishVolume` 接口。
11. LINSTOR **Node Plugin** 执行 `mount` 命令，将 DRBD 设备挂载到 Pod 的目标路径。
12. Pod 成功启动，卷挂载完成。

## 🛠️ 实践操作 (50%)
### 部署 LINSTOR CSI 并验证高可用性

**1. 环境准备**
- 一个至少有3个 Worker 节点的 K8s 集群。
- LINSTOR 已在集群外部署完成，并且 K8s 节点已作为 LINSTOR 的 Satellite 节点加入。（参考 `week-LINSTOR` 学习周内容）

**2. 使用 Helm 部署 LINSTOR CSI 驱动**
这是部署 CSI 驱动最简单、最推荐的方式。
```bash
# 添加 Piraeus (LINSTOR 项目) 的 Helm 仓库
helm repo add piraeus https://piraeus.io/helm-charts/
helm repo update

# 安装 CSI 驱动
# 注意：这里的 `linstorCSIApi.endpoint` 必须指向你的 LINSTOR Controller 的 REST API 地址
helm install linstor-csi piraeus/linstor-csi \
    --set linstorCSIApi.endpoint=http://<YOUR_LINSTOR_CONTROLLER_IP>:3370 \
    --set namespace=kube-system
```
安装完成后，检查 CSI 相关 Pod 是否都正常运行：
```bash
kubectl get pod -n kube-system -l app.kubernetes.io/name=linstor-csi

# 预期输出 (csi-node 是 DaemonSet, csi-controller 是 Deployment)
# NAME                              READY   STATUS    RESTARTS   AGE
# linstor-csi-controller-xxxx...    6/6     Running   0          5m
# linstor-csi-node-abcde            3/3     Running   0          5m
# linstor-csi-node-fghij            3/3     Running   0          5m
# linstor-csi-node-klmno            3/3     Running   0          5m
```
> 注意 `linstor-csi-controller` Pod 中包含了多个 sidecar 容器 (provisioner, attacher, resizer 等)，而 `linstor-csi-node` Pod 中包含了 `node-driver-registrar` 和 LINSTOR 的 Node Plugin。

**3. 创建一个3副本的 StorageClass**
创建一个文件 `sc-linstor-r3.yaml`:
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: linstor-r3-ha
provisioner: linstor.csi.linbit.com
allowVolumeExpansion: true
reclaimPolicy: Retain # 生产环境建议使用 Retain
volumeBindingMode: WaitForFirstConsumer
parameters:
  placementCount: "3" # 关键：指定创建3个副本
  storagePool: "DfltStorPool"
  # allowRemoteVolumeAccess: "false" # 确保 Pod 只会被调度到有数据副本的节点上
```
部署它: `kubectl apply -f sc-linstor-r3.yaml`

**4. 创建 PVC 和 Pod**
创建一个文件 `app-with-ha-storage.yaml`:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-ha-app
spec:
  storageClassName: linstor-r3-ha
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
---
apiVersion: v1
kind: Pod
metadata:
  name: ha-app-pod
spec:
  containers:
  - name: busybox
    image: busybox
    command: ["/bin/sh", "-c", "while true; do echo $(date) >> /data/log.txt; sleep 5; done"]
    volumeMounts:
    - mountPath: "/data"
      name: my-storage
  volumes:
  - name: my-storage
    persistentVolumeClaim:
      claimName: pvc-ha-app
```
部署它: `kubectl apply -f app-with-ha-storage.yaml`

**5. 在 LINSTOR 后端验证**
Pod 启动后，在 LINSTOR 控制节点上验证资源是否已按要��创建。
```bash
linstor resource list

# 预期输出 (可以看到 pvc-xxx 资源在3个不同的节点上都是 InUse 或 Ok 状态)
# ╭──────────────────────────────────────────────────────────────────────────────────╮
# │ ResourceName   ┊ Node         ┊ StoragePool  ┊ VolID ┊ Size(MiB) ┊ Tie-Breaker ┊ State │
# ╞==================================================================================╡
# │ pvc-xxx        ┊ k8s-worker-1 ┊ DfltStorPool ┊ 0     ┊ 1024      ┊             ┊ InUse │
# │ pvc-xxx        ┊ k8s-worker-2 ┊ DfltStorPool ┊ 0     ┊ 1024      ┊             ┊ Ok    │
# │ pvc-xxx        ┊ k8s-worker-3 ┊ DfltStorPool ┊ 0     ┊ 1024      ┊             ┊ Ok    │
# ╰──────────────────────────────────────────────────────────────────────────────────╯
```

**6. 模拟节点故障，验证高可用性**
找出 Pod 当前所在的节点。
```bash
kubectl get pod ha-app-pod -o wide
# NAME         READY   STATUS    RESTARTS   AGE   IP           NODE           ...
# ha-app-pod   1/1     Running   0          2m    10.244.1.8   k8s-worker-1   ...
```
假设 Pod 在 `k8s-worker-1`。现在我们模拟该节点宕机（可以通过 `drain` 和 `cordon` 模拟）。
```bash
kubectl drain k8s-worker-1 --ignore-daemonsets --delete-emptydir-data
```
观察 Pod 的行为。由于 `ha-app-pod` 是一个独立的 Pod（不是由 Deployment 或 StatefulSet 管理），它会被驱逐并进入 `Terminating` 状态，但 K8s 不会自动在别处重建它。我们需要手动删除它。
```bash
kubectl delete pod ha-app-pod --force --grace-period=0
```
现在，重新部署 Pod。
```bash
kubectl apply -f app-with-ha-storage.yaml
```
观察新 Pod 会被调度到哪个节点。
```bash
kubectl get pod ha-app-pod -o wide
# NAME         READY   STATUS    RESTARTS   AGE   IP           NODE           ...
# ha-app-pod   1/1     Running   0          30s   10.244.2.9   k8s-worker-2   ...
```
你会发现新的 Pod 被调度到了一个健康的、并且存有数据副本的节点上（例如 `k8s-worker-2`）。
进入新的 Pod，检查之前写入的数据是否依然存在。
```bash
kubectl exec ha-app-pod -- cat /data/log.txt
# 数据应该完整无缺
```
这个实验证明了，即使一个节点宕机，数据由于在其他节点有副本而没有丢失，应用可以快��在健康的节点上恢复。

## 💻 Go 编程实现 (10%)
今天的 Go 编程任务是理解 CSI 驱动的源码结构。
- **任务**: 克隆 LINSTOR CSI 驱动的源码，并找到 Controller 和 Node 插件实现 gRPC 接口的关键代码。
```bash
git clone https://github.com/piraeusdatastore/linstor-csi.git
cd linstor-csi
```
- **Controller Plugin**: 在 `pkg/linstor-csi/controller.go` 文件中，寻找 `CreateVolume`, `ControllerPublishVolume` 等函数的实现。
- **Node Plugin**: 在 `pkg/linstor-csi/node.go` 文件中，寻找 `NodeStageVolume`, `NodePublishVolume` 等函数的实现。
- **目标**: 不需要修改代码，只需阅读并理解这些函数是如何调用 `linstor-csi` 包中的其他辅助函数来与 LINSTOR API 交互，或者在节点上执行 shell 命令的。

## 🔍 故障排查与优化
- **问题**: CSI Pod (controller 或 node) 无法启动或处于 `CrashLoopBackOff` 状态。
  - **排查**: `kubectl logs <csi-pod-name> -n kube-system -c <container-name>`。
    - 检查 `csi-provisioner` 容器日志，看是否有权限问题或连接 CSI 驱动 gRPC socket 的错误。
    - 检查 CSI 驱动容器 (如 `linstor-csi`) 的日志，看是否有连接后端存储（如 LINSTOR Controller）的错误。
- **问题**: Pod 挂载卷时卡住，`describe pod` 显示 `FailedAttach` 或 `FailedMount`。
  - **排查**:
    1. 查看 `kubelet` 日志: `journalctl -u kubelet` 在对应节点上执行。
    2. 查看 CSI Node Plugin 的日志: `kubectl logs <csi-node-pod-on-that-node> -n kube-system -c linstor-csi`。
    3. 检查 LINSTOR Satellite 日志，看是否有 DRBD 相关的错误。

## 🏠 课后作业
1.  **研究 CSI 驱动对象**: 使用 `kubectl get csinode` 和 `kubectl describe csinode <node-name>` 查看 CSI 驱动在每个节点上的注册信息。
2.  **研究 VolumeSnapshot**: 阅读 LINSTOR CSI 关于卷快照的文档。尝试安装 `snapshot-controller`，并创建一个 `VolumeSnapshotClass` 和 `VolumeSnapshot` 对象，然后从快照恢复一个新的 PVC。
