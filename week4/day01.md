# Day 1: Kubernetes 存储核心概念 (PV, PVC, StorageClass)

## 🎯 学习目标
- **技能目标**: 深入理解 PersistentVolume (PV), PersistentVolumeClaim (PVC), 和 StorageClass 的作用，以及它们之间如何通过动态和静态绑定进行解耦。
- **具体成果**:
  - 能够使用 YAML 文件手动创建并绑定一个 PV 和 PVC。
  - 能够成功部署一个 Pod，并挂载 PVC，验证数据的持久化。
  - 能够配置一个 StorageClass 并通过它动态创建一个 PV。
  - 完成一个 Go 程序，用于列出集群中的 PV 和 PVC。

## 📚 理论基础 (40%)
### 1. 为什么 K8s 需要新的存储抽象？
在容器化的世界里，Pod 的生命周期是短暂的、易逝的。当一个 Pod 崩溃或被销毁后，其内部的数据也会随之丢失。对于数据库、消息队列等有状态应用，这是不可接受的。因此，Kubernetes 设计了一套存储机制，将存储的生命周期与 Pod 的生命周期解耦，实现了数据的持久化。

### 2. 核心概念深度解析
#### a. PersistentVolume (PV) - “集群的存储资源”
- **定义**: PV 是由集群管理员（或存储插件）创建和配置的一块网络存储。它不是属于任何特定节点的资源，而是属于整个集群的资源。可以把它想象成数据中心里一个可供使用的、已经插好线的网络硬盘。
- **关键属性**:
  - `capacity`: 存储容量，例如 `storage: 5Gi`。
  - `accessModes`: 访问模式，定义了 PV 能如何被挂载。
    - `ReadWriteOnce` (RWO): 只能被**单个节点**以读写模式挂载。适用于大多数块存储。
    - `ReadOnlyMany` (ROX): 可以被**多个节点**以只读模式挂载。适用于共享配置文件等场景。
    - `ReadWriteMany` (RWX): 可以被**多个节点**以读写模式挂载。适用于共享文件系统，如 NFS。
  - `persistentVolumeReclaimPolicy`: 回收策略，定义了当绑定的 PVC 被删除后，这个 PV 何去何从。
    - `Retain` (保留): PV 不会被删除，数据得以保留。管理员需要手动清理和回收。**生产环境推荐**。
    - `Delete` (删除): PV 和后端的存储会一起被删除。适用于测试环境。
    - `Recycle` (回收): (已废弃) 会执行 `rm -rf /thevolume/*` 清理数据。
  - `storageClassName`: 关联的 StorageClass 名称。
  - `volumeMode`: 卷模式，可以是 `Filesystem` (默认) 或 `Block` (作为裸块设备)。

#### b. PersistentVolumeClaim (PVC) - “用户的存储请求”
- **定义**: PVC 是由用户（开发者）创建的，对存储资源的一个“申请”。它描述了应用需要什么样的存储，而无需关心存储到底从哪里来。
- **关键属性**:
  - `accessModes`: 期望的访问模式，必须是 PV 所支持的模式的子集。
  - `resources.requests.storage`: 期望的存储容量。
  - `storageClassName`: 想要使用的 StorageClass 名称。如果指定，K8s 会尝试动态创建 PV。

#### c. StorageClass - “存储的模板”
- **定义**: StorageClass 是由管理员定义的“存储类别”或“存储模板”。它将存储的实现细节（用什么插件、什么参数）封装起来，为用户提供不同性能、不同特性的存储选项（如 `fast-ssd`, `slow-hdd`, `backup-storage`）。
- **关键属性**:
  - `provisioner`: 存储分配器，指定了使用哪个存储插件来创建 PV，例如 `linstor.csi.linbit.com` 或 `kubernetes.io/nfs`。
  - `parameters`: 传递给 `provisioner` 的参数，例如副本数、加密选项等。
  - `reclaimPolicy`: 该 StorageClass 创建的 PV 默认的回收策略。
  - `allowVolumeExpansion`: 是否允许扩容。

### 3. 静态供给 vs. 动态供给
- **静态供给 (Static Provisioning)**:
  1. 管理员预先创建好一批 PV。
  2. 用户创建 PVC。
  3. K8s 在现有的 PV 中寻找一个满足 PVC 要求（容量、访问模式等）的并进行绑定。
  - **场景**: 适用于已经存在的、需要手动管理的存储设备。

- **动态供给 (Dynamic Provisioning)**:
  1. 管理员创建好 StorageClass。
  2. 用户创建 PVC，并在其中指定 `storageClassName`。
  3. K8s 发现没有现成的 PV 可满足，但 PVC 指定了 StorageClass，于是触发该 StorageClass 关联的 `provisioner`。
  4. 存储插件（Provisioner）根据 StorageClass 的参数自动创建后端存储，并为其创建一个对应的 PV。
  5. 新创建的 PV 自动与用户的 PVC 绑定。
  - **场景**: 云原生环境下的主流模式，实现了存储的按需、自动化供给。

## 🛠️ 实践操作 (40%)
### 环境准备
- 一个可用的 Kubernetes 集群 (minikube, kind, or a cloud provider's K8s)。
- `kubectl` 命令行工具配置完成。

### 实践一：静态供给 (Static Provisioning)
我们将使用 `hostPath` 来模拟一个预先存在的本地存储。

**1. 创建 PV**
创建一个文件 `pv-manual.yaml`:
```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-manual-hostpath
  labels:
    type: local
spec:
  storageClassName: manual # 使用一个自定义的类名
  capacity:
    storage: 1Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: "/mnt/data" # 确保这个目录在你的 K8s 节点上存在
```
> **注意**: `hostPath` 仅用于单节点集群测试。在多节点集群中，Pod 可能会被调度到没有该路径的节点上而导致挂载失败。

执行创建:
```bash
# 如果使用 minikube, 先 ssh 进去创建目录
minikube ssh -- sudo mkdir -p /mnt/data
minikube ssh -- sudo chmod 777 /mnt/data

# 应用 PV 定义
kubectl apply -f pv-manual.yaml
```

查看 PV 状态，此时应为 `Available`。
```bash
kubectl get pv pv-manual-hostpath

# 预期输出
# NAME                 CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS      CLAIM   STORAGECLASS   REASON   AGE
# pv-manual-hostpath   1Gi        RWO            Retain           Available           manual                  10s
```

**2. 创建 PVC 来请求存储**
创建一个文件 `pvc-manual.yaml`:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-manual-request
spec:
  storageClassName: manual # 必须与 PV 的 storageClassName 匹配
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 500Mi # 请求大小小于等于 PV 容量
```
执行创建:
```bash
kubectl apply -f pvc-manual.yaml
```

**3. 验证绑定**
再次查看 PV 和 PVC，它们的状态都应该变为 `Bound`。
```bash
kubectl get pv pv-manual-hostpath
# STATUS: Bound

kubectl get pvc pvc-manual-request
# STATUS: Bound
# VOLUME: pv-manual-hostpath
```

**4. 在 Pod 中使用 PVC**
创建一个文件 `pod-with-pvc.yaml`:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: storage-test-pod
spec:
  volumes:
    - name: my-storage
      persistentVolumeClaim:
        claimName: pvc-manual-request # 引用上面创建的 PVC
  containers:
    - name: busybox
      image: busybox
      command: ["/bin/sh", "-c", "sleep 3600"]
      volumeMounts:
        - mountPath: "/data" # 将存储挂载到容器内的 /data 目录
          name: my-storage
```
部署 Pod:
```bash
kubectl apply -f pod-with-pvc.yaml
```

**5. 验证数据持久性**
向挂载点写入数据，然后删除并重建 Pod，检查数据是否依然存在。
```bash
# 向容器内写入文件
kubectl exec storage-test-pod -- sh -c "echo 'Hello from static PV!' > /data/test.txt"

# 验证文件内容
kubectl exec storage-test-pod -- cat /data/test.txt
# 输出: Hello from static PV!

# 删除 Pod
kubectl delete pod storage-test-pod

# 重新创建 Pod
kubectl apply -f pod-with-pvc.yaml

# 再次验证文件内容，数据应该依然存在
kubectl exec storage-test-pod -- cat /data/test.txt
# 输出: Hello from static PV!
```

### 实践二：动态供给 (Dynamic Provisioning)
大多数 K8s 环境会自带一个默认的 StorageClass。我们可以用 `kubectl get sc` 查看。这里我们创建一个新的。

**1. 创建 StorageClass**
我们将使用 LINSTOR CSI 驱动。如果未安装，请先参考 `week-LINSTOR/day05.md` 进行安装。
创建一个文件 `sc-linstor.yaml`:
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: linstor-r2 # 2副本存储
provisioner: linstor.csi.linbit.com
allowVolumeExpansion: true
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
parameters:
  placementCount: "2"
  storagePool: "DfltStorPool"
```
> `volumeBindingMode: WaitForFirstConsumer` 是一个重要优化。它会延迟 PV 的创建和绑定，直到第一个使用该 PVC 的 Pod 被调度。这样可以确保 PV 创建在 Pod 所在的区域，避免跨区访问。

执行创建:
```bash
kubectl apply -f sc-linstor.yaml
```

**2. 创建 PVC 请求动态存储**
创建一个文件 `pvc-dynamic.yaml`:
```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-dynamic-request
spec:
  storageClassName: linstor-r2 # 指定我们刚创建的 StorageClass
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```
执行创建:
```bash
kubectl apply -f pvc-dynamic.yaml
```
查看 PVC 状态，它会是 `Pending`，因为 `volumeBindingMode` 设置为 `WaitForFirstConsumer`。

**3. 部署 Pod 触发供给**
修改 `pod-with-pvc.yaml`，将其指向新的 PVC `pvc-dynamic-request`，然后部署。
一旦 Pod 开始创建，LINSTOR CSI 插件就会被触发，自动创建 DRBD 设备和 PV，并完成绑定。
查看 PV 和 PVC，它们的状态会很快变为 `Bound`。

## 💻 Go 编程实现 (20%)
### 项目: `k8s-storage-lister`
这个工具将使用 `client-go` 库来列出集群中的 PV 和 PVC。

**1. 初始化项目**
```bash
mkdir k8s-storage-lister
cd k8s-storage-lister
go mod init storage.lister.dev/me
```

**2. 添加依赖**
```bash
go get k8s.io/client-go/tools/clientcmd
go get k8s.io/client-go/kubernetes
go get k8s.io/apimachinery/pkg/apis/meta/v1
```

**3. 编写代码 (`main.go`)**
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	clientcmd "k8s.io/client-go/tools/clientcmd"
)

func main() {
	// 1. 加载 kubeconfig 文件
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("获取家目录失败: %v", err)
	}
	kubeconfigPath := filepath.Join(userHomeDir, ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Fatalf("加载 kubeconfig 失败: %v", err)
	}

	// 2. 创建 clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("创建 clientset 失败: %v", err)
	}

	// 3. 列出所有 PV
	fmt.Println("--- PersistentVolumes ---")
	pvList, err := clientset.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("列出 PV 失败: %v", err)
	}

	for _, pv := range pvList.Items {
		fmt.Printf("  - Name: %s\n", pv.Name)
		fmt.Printf("    Status: %s\n", pv.Status.Phase)
		fmt.Printf("    Capacity: %s\n", pv.Spec.Capacity.Storage().String())
		if pv.Spec.ClaimRef != nil {
			fmt.Printf("    Claim: %s\n", pv.Spec.ClaimRef.Name)
		}
		fmt.Println("-------------------------")
	}

	// 4. 列出所有命名空间中的 PVC
	fmt.Println("\n--- PersistentVolumeClaims (all namespaces) ---")
	pvcList, err := clientset.CoreV1().PersistentVolumeClaims("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("列出 PVC 失败: %v", err)
	}

	for _, pvc := range pvcList.Items {
		fmt.Printf("  - Namespace: %s\n", pvc.Namespace)
		fmt.Printf("    Name: %s\n", pvc.Name)
		fmt.Printf("    Status: %s\n", pvc.Status.Phase)
		fmt.Printf("    Volume: %s\n", pvc.Spec.VolumeName)
		fmt.Println("-------------------------")
	}
}
```

**4. 运行**
```bash
go run main.go
```
程序将输出当前 K8s 集群中所有 PV 和 PVC 的信息。

## 🔍 故障排查与优化
- **问题**: PVC 长时间处于 `Pending` 状态。
  - **排查思路**:
    1. `kubectl describe pvc <pvc-name>`: 查看 Events 部分，通常有最直接的错误信息。
    2. **静态供给**: 是否有满足容量和 `accessModes` 的 `Available` 状态的 PV？PV 和 PVC 的 `storageClassName` 是否完全匹配？
    3. **动态供给**: PVC 指定的 `storageClassName` 是否存在 (`kubectl get sc`)？CSI 驱动的 Pod (如 `csi-provisioner`) 是否正常运行？查看 CSI 驱动日志。
    4. 资源配额 (`ResourceQuota`) 是否已用尽？

## 📝 实战项目
- 结合今天的学习，为你的团队编写一份简短的 "K8s 存储申请指南"，说明如何通过提交一个 PVC 的 YAML 文件来申请存储资源。

## 🏠 课后作业
1.  **研究 `accessModes`**: 找一个支持 `ReadWriteMany` 的存储方案（如 NFS），并部署一个 CSI 驱动（如 `nfs-subdir-external-provisioner`）。尝试创建一个 RWX 的 PVC，并同时挂载到两个 Pod 上，验证两个 Pod 可以同时读写同一个文件。
2.  **研究 `reclaimPolicy`**:
    - 创建一个 `reclaimPolicy: Retain` 的 PV 和 PVC。删除 PVC 后，验证 PV 依然存在且状态变为 `Released`。思考如何手动恢复这个 PV 给新的 PVC 使用。
    - 创建一个 `reclaimPolicy: Delete` 的 StorageClass 和 PVC。删除 PVC 后，验证 PV 和后端存储是否都被自动删除了。

```