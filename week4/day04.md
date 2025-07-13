# Day 4: Go client-go 实践：与 K8s API 交互

## 🎯 学习目标
- **技能目标**: 掌握 `client-go` 库的核心用法，能够使用 `clientset` 对 Kubernetes 资源进行 CRUD (创建, 读取, 更新, 删除) 操作。
- **核心概念**: 深入理解 `informer`, `lister`, 和 `reflector` 的工作机制，并能解释为什么在生产级别的控制器中应该使用 `informer` 而不是直接调用 `clientset`。
- **具体成果**:
  - 能够独立编写一个 Go 程序，使用 `clientset` 列出、获取、创建和删除 PV 和 PVC。
  - 能够使用 `informer` 机制来监听 PVC 的变化，并在 PVC 被创建或删除时打印日志。
  - 完成一个实战项目：一个简单的 PVC 自动清理工具。

## 📚 理论基础 (40%)
### 1. `client-go` 简介
`client-go` 是 Kubernetes 官方提供的 Go 语言客户端库。它是构建所有与 Kubernetes API Server 交互的 Go 应用（如 `kubectl`、控制器、Operator）的基础。

### 2. `clientset` vs. `informer`
与 K8s API 交互主要有两种模式：

#### a. `clientset`: 直接的 API 请求
- **是什么**: `clientset` 是一个包含了访问所有 K8s API Group (如 `core/v1`, `apps/v1`, `storage.k8s.io/v1`) 的客户端集合。
- **工作方式**: 每次调用 `clientset.CoreV1().Pods("default").List(...)` 都会发起一次到 API Server 的 REST API 请求。
- **优点**: 简单直接，易于理解。
- **缺点**:
  - **效率低**: 如果需要频繁获取资源状态（例如，在一个循环中），会产生大量的 API 请求，给 API Server 带来巨大压力。
  - **无实时性**: 只能获取到调用那一刻的快照，无法实时感知资源的变化。
- **适用场景**: 一次性的、临时的操作，例如编写一个简单的命令行工具来获取一次信息。

#### b. `informer`: 高效的、基于事件的缓存机制
- **是什么**: `informer` 是 `client-go` 的核心机制，它为一种或多种资源类型提供了一个事件驱动的接口，并维护了一个本地的内存缓存。
- **核心组件**:
  - **Reflector**: 负责“监视”（Watch）指定类型的 K8s 资源。它通过一个 List-Watch 机制与 API Server 通信。首先，它会列出（List）所有对象来填充本地缓存；然后，它会启动一个长连接的监视（Watch），实时接收所有关于该资源的变更事件（Added, Updated, Deleted）。
  - **Indexer (本地缓存)**: 一个线程安全的、存储对象的本地数据库。`Reflector` 获取到的所有对象和变更都会被存入这里。它还支持根据标签、注解等字段为对象建立索引，以便快速查询。
  - **Informer (控制器)**: 将从 `Reflector` 收到的变更事件，分发给注册的事件处理函数（`ResourceEventHandlerFuncs`）。
- **工作方式**:
  1. `informer` 启动，其内部的 `Reflector` 开始 List-Watch API Server。
  2. 资源数据被同步到本地的 `Indexer` 缓存中。
  3. 开发者通过 `Lister` 从本地缓存中高效地读取数据，而**无需访问 API Server**。
  4. 开发者注册事件处理函数，当资源发生变化时，`informer` 会自动调用这些函数。
- **优点**:
  - **高效**: 所有读取操作都来自本地内存缓存，极大地减轻了 API Server 的负载。
  - **实时**: 通过 Watch 机制，可以近乎实时地响应集群中的变化。
  - **可靠**: `informer` 内部处理了网络中断、Watch 重连等复杂问题。
- **适用场景**: 任何需要持续监控集群状态的应用，特别是编写自定义控制器 (Operator) 的标准模式。

![Informer Architecture](https://miro.medium.com/v2/resize:fit:1400/1*eL6A5Zp_2t9a_sV2_v_8_Q.png)
*图：Informer 架构示意图*

## 🛠️ 实践操作 (40%)
### 实践一：使用 `clientset` 进行基本的 CRUD 操作

**1. 项目初始化**
```bash
mkdir k8s-crud-cli
cd k8s-crud-cli
go mod init crud.cli.dev/me
go get k8s.io/client-go@v0.28.2 k8s.io/api@v0.28.2 k8s.io/apimachinery@v0.28.2
```

**2. 编写代码 (`main.go`)**
这个程序将演示如何创建一个 PVC，然后获取它，最后删除它。
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// --- 1. 配置和创建 clientset ---
	userHomeDir, _ := os.UserHomeDir()
	kubeconfig := filepath.Join(userHomeDir, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset: %s", err.Error())
	}
	
	namespace := "default"
	pvcName := "my-test-pvc-from-go"

	// --- 2. 创建 PVC ---
	fmt.Printf("Creating PVC '%s'...\n", pvcName)
	storageClassName := "linstor-r2" // 使用 Day 2 创建的 SC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			StorageClassName: &storageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Mi"),
				},
			},
		},
	}

	createdPvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), pvc, metav1.CreateOptions{})
	if err != nil {
		log.Fatalf("Failed to create PVC: %s", err.Error())
	}
	fmt.Printf("PVC '%s' created successfully. Status: %s\n\n", createdPvc.Name, createdPvc.Status.Phase)

	// --- 3. 获取 PVC ---
	fmt.Printf("Getting PVC '%s'...\n", pvcName)
	retrievedPvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), pvcName, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Failed to get PVC: %s", err.Error())
	}
	fmt.Printf("Found PVC '%s'. Volume Name: %s\n\n", retrievedPvc.Name, retrievedPvc.Spec.VolumeName)

	// --- 4. 删除 PVC ---
	fmt.Printf("Press 'Enter' to delete PVC '%s'...", pvcName)
	fmt.Scanln()
	fmt.Printf("Deleting PVC '%s'...\n", pvcName)
	err = clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), pvcName, metav1.DeleteOptions{})
	if err != nil {
		log.Fatalf("Failed to delete PVC: %s", err.Error())
	}
	fmt.Println("PVC deleted successfully.")
}
```

**3. 运行**
```bash
go run main.go
```
程序会创建一个 10Mi 大小的 PVC，获取并显示其信息，然后等待你按回车键后将其删除。你可以在另一个终端使用 `kubectl get pvc` 观察到这个过程。

### 实践二：使用 `informer` 监听 PVC 变化

**1. 项目初始化**
```bash
mkdir k8s-pvc-watcher
cd k8s-pvc-watcher
go mod init pvc.watcher.dev/me
go get k8s.io/client-go@v0.28.2 k8s.io/api@v0.28.2 k8s.io/apimachinery@v0.28.2
```

**2. 编写代码 (`main.go`)**
```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// --- 1. 配置和创建 clientset ---
	userHomeDir, _ := os.UserHomeDir()
	kubeconfig := filepath.Join(userHomeDir, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset: %s", err.Error())
	}

	// --- 2. 创建 Informer Factory ---
	// 创建一个 Informer 工厂，设置 30 秒重新同步一次
	factory := informers.NewSharedInformerFactory(clientset, 30*time.Second)
	
	// 从工厂中获取一个针对 PVC 的 Informer
	pvcInformer := factory.Core().V1().PersistentVolumeClaims().Informer()

	// --- 3. 注册事件处理函数 ---
	pvcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pvc := obj.(*corev1.PersistentVolumeClaim)
			log.Printf("PVC ADDED: %s/%s", pvc.Namespace, pvc.Name)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			pvc := newObj.(*corev1.PersistentVolumeClaim)
			log.Printf("PVC UPDATED: %s/%s, Status: %s", pvc.Namespace, pvc.Name, pvc.Status.Phase)
		},
		DeleteFunc: func(obj interface{}) {
			pvc := obj.(*corev1.PersistentVolumeClaim)
			log.Printf("PVC DELETED: %s/%s", pvc.Namespace, pvc.Name)
		},
	})

	// --- 4. 启动 Informer ---
	stopCh := make(chan struct{})
	defer close(stopCh)
	
	factory.Start(stopCh)

	// 等待 Informer 的缓存同步完成
	if !cache.WaitForCacheSync(stopCh, pvcInformer.HasSynced) {
		log.Fatal("Failed to sync cache")
	}
	log.Println("Informer cache synced. Watching for PVC changes...")

	// 阻塞主 goroutine，否则程序会直接退出
	<-stopCh
}
```

**3. 运行**
```bash
go run main.go
```
程序启动后会阻塞。现在，打开另一个终端，尝试创建、删除 PVC，你会看到 Go 程序会立即打印出相应的日志。
```bash
# 在另一个终端
kubectl create -f my-pvc.yaml
kubectl delete -f my-pvc.yaml
```

## 💻 Go 编程实现 (20%)
### 实战项目: `pvc-cleaner`
**目标**: 编写一个简单的控制器，自动删除所有处于 `Released` 状态的 PVC。当一个 PVC 的 `reclaimPolicy` 是 `Retain` 时，删除该 PVC 后其底层的 PV 会被保留，但状态变为 `Released`。这种 PV 无法被新的 PVC 绑定，需要手动清理。我们的工具将自动完成这个清理工作。

**思路**:
1.  使用 `informer` 监听 PVC 的变化。
2.  在 `UpdateFunc` 事件处理器中，检查 PVC 的新状态。
3.  如果 `pvc.Status.Phase == corev1.VolumeReleased`，则使用 `clientset` 删除这个 PVC。

**核心代码片段**:
```go
// ... 在 informer 的实践代码基础上修改 ...
pvcInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
    AddFunc: func(obj interface{}) { /* ... */ },
    UpdateFunc: func(oldObj, newObj interface{}) {
        pvc := newObj.(*corev1.PersistentVolumeClaim)
        log.Printf("PVC UPDATED: %s/%s, Status: %s", pvc.Namespace, pvc.Name, pvc.Status.Phase)

        // 检查状态是否为 Released
        if pvc.Status.Phase == corev1.VolumeReleased {
            log.Printf("Found Released PVC '%s'. Deleting...", pvc.Name)
            err := clientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Delete(context.TODO(), pvc.Name, metav1.DeleteOptions{})
            if err != nil {
                log.Printf("Failed to delete Released PVC '%s': %v", pvc.Name, err)
            } else {
                log.Printf("Released PVC '%s' deleted successfully.", pvc.Name)
            }
        }
    },
    DeleteFunc: func(obj interface{}) { /* ... */ },
})
// ...
```
> **注意**: 这是一个简化的示例。生产级的控制器需要更复杂的逻辑，如重试、错误处理、速率限制等。

## 🔍 故障排查与优化
- **问题**: `client-go` 程序无法连接到 K8s 集群。
  - **排查**:
    1. 确认 `~/.kube/config` 文件存在且配置正确。
    2. 如果在 Pod 内部运行，应该使用 `rest.InClusterConfig()` 来获取配置，而不是从文件加载。
    3. 检查 RBAC 权限。程序所使用的 ServiceAccount 是否有权限访问它要操作的资源？
- **优化**:
  - 总是优先使用 `informer` 和 `lister` 来读取数据，只在需要写入（Create, Update, Delete）时才使用 `clientset`。
  - 使用 `SharedInformerFactory` 可以让多个 `informer` 共享同一个底层的 `Reflector` 和缓存，节省资源。

## 🏠 课后作业
1.  **扩展 `k8s-crud-cli`**: 为你的 CRUD 程序增加 `Update` 功能。尝试修改一个已存在 PVC 的标签（`labels`）或注解（`annotations`）。
2.  **扩展 `pvc-cleaner`**: 增加一个命令行参数 `--dry-run`。当启用此参数时，程序只打印将要删除的 `Released` PVC，而不执行真正的删除操作。这在生产环境中是一个非常重要的安全功能。
3.  **思考题**: 为什么 `informer` 的事件处理函数中，不应该执行耗时很长的操作？如果必须执行，应该怎么做？（提示: 考虑工作队列 `workqueue` 模式）
