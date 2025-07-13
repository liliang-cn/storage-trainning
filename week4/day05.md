# Day 5: K8s 存储运维与高级主题

## 🎯 学习目标
- **技能目标**: 掌握 Kubernetes 存储运维中的核心技能，包括问题诊断、卷扩容和卷快照。
- **具体成果**:
  - 能够独立诊断并解决 PVC `Pending` 和 Pod `FailedMount` 等常见存储问题。
  - 能够成功地对一个由 CSI 驱动创建的 PVC 完成在线扩容。
  - 能够使用 CSI 的快照功能，为一个 PVC 创建快照，并从该快照恢复出一个新的 PVC。
  - 能够撰写一份简洁明了的 K8s 存储问题排查手册。

## 📚 理论基础 (30%)
### 1. 存储运维核心场景
在生产环境中，存储系统最常遇到的问题和需求包括：
- **故障诊断**: 应用无法获取存储，我该从哪里查起？
- **容量管理**: 应用的存储空间快用完了，如何平滑地扩容？
- **数据保护**: 如何为我的数据库创建备份和快照，以便在数据损坏时进行恢复？

Kubernetes 联合 CSI 生态为这些场景提供了标准的解决方案。

### 2. 卷扩容 (Volume Expansion)
- **工作原理**:
  1. 管理员在 `StorageClass` 中设置 `allowVolumeExpansion: true`。
  2. 用户（或自动化脚本）修改 PVC 对象的 `spec.resources.requests.storage` 字段，将其调大。
  3. `external-resizer` sidecar 监听到 PVC 的变化。
  4. `external-resizer` 调用 CSI 驱动的 `ControllerExpandVolume` 接口，扩展后端存储卷的容量。
  5. 如果扩容成功，`external-resizer` 会更新 PV 对象的 `spec.capacity`。
  6. Kubelet 发现 PVC 和 PV 的容量不一致，它会调用 CSI 驱动的 `NodeExpandVolume` 接口。
  7. CSI Node 插件在节点上执行文件系统扩展命令（如 `resize2fs` 或 `xfs_growfs`），使容器内的文件系统能够识别到新的空间。
- **在线 vs. 离线扩容**:
  - **在线扩容**: Pod 正在运行时进行扩容，应用无感知。大部分现代 CSI 驱动和文件系统都支持。
  - **离线扩容**: 需要先将 Pod 停止，扩容完成后再启动。

### 3. 卷快照 (Volume Snapshot)
- **背景**: 为了提供一套标准的、与存储厂商无��的快照接口，社区引入了一组新的 API 资源。
- **核心 CRDs (Custom Resource Definitions)**:
  - **`VolumeSnapshotClass`**: 类似于 `StorageClass`，它定义了创建快照的“类别”。由管理员创建，指定了使用哪个 CSI 驱动以及其他快照相关的参数（如快照的保留策略）。
  - **`VolumeSnapshot`**: 类似于 `PVC`，它是由用户创建的、对某个特定 PVC 的“快照请求”。
  - **`VolumeSnapshotContent`**: 类似于 `PV`，它代表了一个实际存在于存储系统上的快照。它可以由 CSI 驱动动态创建，或由管理员手动创建来导入一个已有的快照。
- **工作流程 (动态创建)**:
  1. 管理员部署 `snapshot-controller` 和相关的 CRDs。
  2. 管理员创建一个 `VolumeSnapshotClass`，指向某个 CSI 驱动。
  3. 用户创建一个 `VolumeSnapshot` 对象，在 `spec.source` 中指定要快照的 PVC 名称。
  4. `snapshot-controller` 监听到 `VolumeSnapshot` 的创建，调用 CSI 驱动的 `CreateSnapshot` 接口。
  5. CSI 驱动在后端存储上创建快照，并返回快照信息。
  6. `snapshot-controller` 根据返回信息，创建一个 `VolumeSnapshotContent` 对象，并将其与 `VolumeSnapshot` 绑定。
- **从快照恢复**:
  - 要从快照恢复数据，只需创建一个新的 PVC，并在其 `spec.dataSource` 字段中引用之前创建的 `VolumeSnapshot` 对象。CSI 驱动就会根据快照创建一个包含相同数据的新卷。

## 🛠️ 实践操作 (50%)
### 实践一：诊断存储问题

**1. 模拟 PVC `Pending` 状态**
创建一个 `pvc-pending-demo.yaml`，故意使用一个不存在的 StorageClass。
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-pending-demo
spec:
  storageClassName: "non-existent-sc" # 故意写错
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```
部署: `kubectl apply -f pvc-pending-demo.yaml`

**2. 诊断过程**
```bash
# 1. 查看 PVC 状态，发现是 Pending
kubectl get pvc pvc-pending-demo
# NAME               STATUS    VOLUME   CAPACITY   ACCESS MODES   STORAGECLASS        AGE
# pvc-pending-demo   Pending                                      non-existent-sc     5s

# 2. 使用 describe 查看详细事件，找到根本原因
kubectl describe pvc pvc-pending-demo
# ...
# Events:
#   Type    Reason                Age   From                         Message
#   ----    ------                ---   ----                         -------
#   Normal  ProvisioningFailed    10s   persistentvolume-controller  storageclass.storage.k8s.io "non-existent-sc" not found
# ...
```
**结论**: `describe` 命令的 `Events` 部��是排查 K8s 对象问题的首选工具。

### 实践二：在线卷扩容

**1. 准备工作**
- 确保你的 `StorageClass` (例如 `linstor-r3-ha`) 已经设置了 `allowVolumeExpansion: true`。
- 部署一个使用该 SC 的应用。我们可以复用 Day 3 的 `app-with-ha-storage.yaml`。

**2. 部署并检查初始大小**
```bash
kubectl apply -f app-with-ha-storage.yaml
# ... Pod 启动后 ...

# 进入 Pod 内部，使用 df -h 查看文件系统大小
kubectl exec ha-app-pod -- df -h /data
# Filesystem                Size      Used Available Use% Mounted on
# /dev/drbd1000             1007.9M     1.0M   1006.9M   0% /data
```
可以看到，初始大小约为 1Gi。

**3. 执行扩容**
直接编辑 PVC 对象，修改存储请求的大小。
```bash
kubectl edit pvc pvc-ha-app
```
将 `spec.resources.requests.storage` 从 `1Gi` 修改为 `2Gi`，保存退出。

**4. 验证扩容结果**
```bash
# 1. 观察 PVC 事件，可以看到扩容相关的事件
kubectl describe pvc pvc-ha-app
# ...
# Events:
#   Type    Reason                      Age   From                         Message
#   ----    ------                      ---   ----                         -------
#   Normal  Resizing                    2m    external-resizer linstor...  External resizer is resizing volume pvc-xxx
#   Normal  FileSystemResizeSuccessful  1m    kubelet                      MountVolume.NodeExpandVolume succeeded for volume "pvc-xxx"

# 2. 再次进入 Pod 内部，检查文件系统大小
kubectl exec ha-app-pod -- df -h /data
# Filesystem                Size      Used Available Use% Mounted on
# /dev/drbd1000               2.0G      1.0M      2.0G   0% /data
```
**结论**: 文件系统已成功在线扩展到 2Gi，应用全程没有中断。

### 实践三：卷快照与恢复

**1. 安装 Snapshot Controller 和 CRDs**
```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.2/client/config/crd/snapshot.storage.k8s.io_volumesnapshotclasses.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.2/client/config/crd/snapshot.storage.k8s.io_volumesnapshotcontents.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.2/client/config/crd/snapshot.storage.k8s.io_volumesnapshots.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.2/deploy/kubernetes/snapshot-controller/rbac-snapshot-controller.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-csi/external-snapshotter/v6.2.2/deploy/kubernetes/snapshot-controller/setup-snapshot-controller.yaml
```

**2. 创建 VolumeSnapshotClass**
创建一个文件 `snapclass-linstor.yaml`。
```yaml
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshotClass
metadata:
  name: linstor-snapshot-class
driver: linstor.csi.linbit.com # 必须是 CSI 驱动的名称
deletionPolicy: Delete # 当 VolumeSnapshot 对象被删除时，也删除后端的快照
```
部署: `kubectl apply -f snapclass-linstor.yaml`

**3. 为现有 PVC 创建快照**
假设我们想为 `pvc-ha-app` 创建一个快照。先向里面写入一些独特的数据。
```bash
kubectl exec ha-app-pod -- sh -c "echo 'data-before-snapshot' > /data/snapshot_test.txt"
```
创建一个文件 `snapshot-demo.yaml`:
```yaml
apiVersion: snapshot.storage.k8s.io/v1
kind: VolumeSnapshot
metadata:
  name: pvc-ha-app-snapshot-1
spec:
  volumeSnapshotClassName: linstor-snapshot-class
  source:
    persistentVolumeClaimName: pvc-ha-app
```
部署: `kubectl apply -f snapshot-demo.yaml`

**4. 检查快照状态**
```bash
kubectl get volumesnapshot
# NAME                      READYTOUSE   SOURCEPVC      SOURCESNAPSHOTCONTENT   RESTORESIZE   SNAPSHOTCLASS            SNAPSHOTCONTENT                                    CREATIONTIME   AGE
# pvc-ha-app-snapshot-1   true         pvc-ha-app                             1Gi           linstor-snapshot-class   snapcontent-c8e...   2m             2m

# READYTOUSE 为 true 表示快照已成功创建并可以使用
```

**5. 从快照恢复一个新的 PVC**
创建一个文件 `pvc-restore-demo.yaml`:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-restored-from-snapshot
spec:
  storageClassName: linstor-r3-ha
  dataSource:
    name: pvc-ha-app-snapshot-1
    kind: VolumeSnapshot
    apiGroup: snapshot.storage.k8s.io
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi # 大小必须和快照源卷一致或更大
```
部署: `kubectl apply -f pvc-restore-demo.yaml`

**6. 验证恢复的数据**
创建一个新的 Pod `restore-test-pod`，挂载这个新的 PVC `pvc-restored-from-snapshot`。
Pod 启动后，检查文件内容：
```bash
kubectl exec restore-test-pod -- cat /data/snapshot_test.txt
# 预期输出: data-before-snapshot
```
**结论**: 我们成功地从快照中恢复了数据到一个全新的卷。

## 💻 Go 编程实现 (20%)
### 项目: `k8s-storage-reporter`
**目标**: 编写一个 Go 程序，生成一个关于集群存储使用情况的简单报告。
- 列出所有 StorageClasses。
- 列出所有 PV，显示其容量、状态和所属的 StorageClass。
- 列出所有 PVC，显示其请求容量、状态和绑定的 PV。

这个项目可以作为 Day 1 `k8s-storage-lister` 的一个功能增强��。

**核心代码片段**:
```go
// ... clientset 初始化 ...

// 列出 StorageClasses
scList, err := clientset.StorageV1().StorageClasses().List(context.TODO(), metav1.ListOptions{})
// ... 遍历并打印 ...

// 列出 PVs
pvList, err := clientset.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
// ... 遍历并打印 pv.Spec.StorageClassName ...

// 列出 PVCs
pvcList, err := clientset.CoreV1().PersistentVolumeClaims("").List(context.TODO(), metav1.ListOptions{})
// ... 遍历并打印 pvc.Spec.StorageClassName 和 pvc.Spec.VolumeName ...
```

## 🔍 故障排查与优化
### K8s 存储问题排查手册 (精简版)

**1. PVC 处于 `Pending` 状态**
   - `kubectl describe pvc <pvc-name>`
   - **检查点**:
     - **事件(Events)**: 是否有 `ProvisioningFailed` 或类似错误？
     - **StorageClass**: 名称是否正确？`kubectl get sc <sc-name>` 是否存在？
     - **静态绑定**: 是否有匹配容量和 `accessModes` 的 `Available` PV？`storageClassName` 是否匹配？
     - **CSI驱动**: `external-provisioner` Pod 日志是否有错误？

**2. Pod 启动失败，`describe pod` 显示 `FailedMount` 或 `FailedAttach`**
   - `kubectl describe pod <pod-name>`
   - **检查点**:
     - **事件(Events)**: 查看详细错误信息。
     - **节点排查**:
       - 登录 Pod 所在节点。
       - 查看 `kubelet` 日志: `journalctl -u kubelet`。
       - 查看 CSI Node 插件 Pod 的日志: `kubectl logs <csi-node-pod> -n kube-system -c <driver-container>`。
     - **后端存储**: 检查 LINSTOR、Ceph 等后端存储本身是否健康。

**3. 卷扩容失败**
   - `kubectl describe pvc <pvc-name>`
   - **检查点**:
     - **StorageClass**: `allowVolumeExpansion` 是否为 `true`？
     - **CSI驱动**: `external-resizer` Pod 日志是否有错误？
     - **文件系统**: `kubelet` 日志中是否有 `FileSystemResizeFailed` 事件？

**4. 卷快照失败**
   - `kubectl describe volumesnapshot <snapshot-name>`
   - **检查点**:
     - **Snapshot Controller**: `snapshot-controller` Pod 日志是否有错误？
     - **VolumeSnapshotClass**: `driver` 名称是否正确？
     - **CSI驱动**: CSI Controller 插件日志是否有 `CreateSnapshot` 相关的错误？

## 🏠 本周作业交付
1.  **Go 工具**: 提交你的 `k8s-storage-reporter` 或其他本周完成的 Go 项目代码。
2.  **技术文档**: 提交你撰写的《Kubernetes 存储问题排查手册》。
3.  **实践报告**: 提交一份完整的实践报告，记录你使用 StatefulSet 和 LINSTOR CSI 部署一个高可用 Redis ��群的全过程，包括故障模拟和恢复的步骤与截图。
