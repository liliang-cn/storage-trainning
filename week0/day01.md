# Day 1: Kubernetes 核心架构与基本概念

## 🎯 学习目标
- **技能目标**: 理解容器编排的必要性，掌握 Kubernetes 的核心架构和组件作用。
- **核心概念**: 深入理解 Cluster, Node, Pod 这三个最基本的概念。
- **具体成果**:
  - 能够独立搭建一个本地的 Kubernetes 测试环境 (例如 minikube)。
  - 能够熟练使用 `kubectl` 执行集群信息查看、节点状态查询等基本命令。
  - 能够独立编写一个简单的 Pod YAML 文件，并成功部署到集群中。
  - 能够使用 `kubectl` 对运行中的 Pod 进行查看、检查、交互和删除。

## 📚 理论基础 (40%)
### 1. 从容器到容器编排
我们已经知道，容器（如 Docker）为应用提供了一个轻量级、可移植、自包含的运行环境。但这只解决了单个应用的打包和运行问题。当应用变得复杂，由几十上百个微服务构成时，新的问题出现了：
- **部署**: 如何一次性部署和管理成百上千个容器？
- **伸缩**: 如何根据负载自动增加或减少容器实例？
- **服务发现**: 一个容器如何找到并与另一个容器通信？
- **自愈**: 如果一个容器或它所在的机器宕机了，如何自动恢复服务？
- **升级**: 如何在不中断服务的情况下更新应用版本？

**容器编排 (Container Orchestration)** 正是为解决这些问题而生。Kubernetes 就是目前业界最主流、最强大的容器编排系统。你可以把它想象成一个管理海量容器的“分布式操作系统”。

### 2. Kubernetes 核心架构
Kubernetes 集群由两种主要类型的节点组成：**控制平面节点 (Control Plane Nodes)** 和 **工作节点 (Worker Nodes)**。

![K8s Architecture](https://kubernetes.io/images/docs/components-of-kubernetes.svg)

#### a. 控制平面 (Control Plane) - 集群的大脑
控制平面负责做出全局决策，例如调度 Pod、检测和响应集群事件等。它由以下几个关键组件构成：
- **`kube-apiserver`**: **集群的统一入口**。它暴露 Kubernetes API，是所有组件（包括 `kubectl`）与集群状态交互的唯一途径。它负责处理 REST 请求、验证请求、并更新 `etcd` 中的对象状态。
- **`etcd`**: 一个高可用的键值存储系统。**它保存了整个集群的完整状态数据**，是集群的唯一“事实来源 (Source of Truth)”。所有对集群状态的改变都必须通过 `apiserver` 写入 `etcd`。
- **`kube-scheduler`**: **Pod 的调度器**。它监视新创建的、但尚未分配到节点的 Pod，然后根据一系列复杂的规则（如资源需求、亲和性、策略限制）为其选择一个最合适的工作节点。
- **`kube-controller-manager`**: **集群状态的维护者**。它运行着多个控制器进程（如节点控制器、副本控制器等）。每个控制器负责监视一种特定资源的状态，并努力将当前状态调整为在 `etcd` 中定义的期望状态。

#### b. 工作节点 (Worker Node) - 集群的劳动力
工作节点负责运行用户的应用程序（即容器）。它包含以下组件：
- **`kubelet`**: **节点上的代理**。它直接与容器运行时（如 containerd）交互，确保 Pod 中描述的容器能够正确地启动、运行和停止。它也定时向 `apiserver` 汇报本节点的状态。
- **`kube-proxy`**: **网络代理**。它负责维护节点上的网络规则，实现了 Kubernetes Service 的概念，允许网络流量在 Pod 之间进行路由和负载均衡。
- **`Container Runtime`**: **容器运行时**。这是真正负责运行容器的软件，例如 `containerd`, `CRI-O`，或者早期的 `Docker`。

### 3. 核心概念：Cluster, Node, Pod
- **Cluster (集群)**: 由一个或多个控制平面节点和多个工作节点组成的完整 Kubernetes 环境。
- **Node (节点)**: 一个工作机器，可以是物理机或虚拟机。它是 Pod 运行的载体。
- **Pod**: **Kubernetes 中最小、最基本的部署单元**。一个 Pod 封装了一个或多个紧密关联的容器、存储资源、以及一个唯一的网络 IP。Pod 内的容器共享同一个网络命名空间和存储卷，可以通过 `localhost` 相互通信。

## 🛠️ 实践操作 (50%)
### 1. 安装本地 Kubernetes 环境 (minikube)
Minikube 是一个可以在本地快速启动单节点 Kubernetes 集群的工具，非常适合学习和测试。
```bash
# 根据你的操作系统，参考官方文档安装 minikube
# https://minikube.sigs.k8s.io/docs/start/

# 启动一个 minikube 集群
minikube start --driver=docker
```

### 2. 安装并配置 `kubectl`
`kubectl` 是与 Kubernetes 集群交互的命令行工具。
```bash
# 参考官方文档安装 kubectl
# https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/

# minikube start 会自动配置 kubectl 的上下文
# 验证 kubectl 是否配置正确
kubectl cluster-info
# 输出应显示 Master 和 CoreDNS 的地址

# 查看集群中的节点
kubectl get nodes
# NAME       STATUS   ROLES           AGE   VERSION
# minikube   Ready    control-plane   10m   v1.28.3
```

### 3. 创建你的第一个 Pod
创建一个文件 `my-first-pod.yaml`:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: nginx-pod
  labels:
    app: nginx
spec:
  containers:
  - name: nginx-container
    image: nginx:1.25
    ports:
    - containerPort: 80
```
- `apiVersion`: 定义了使用哪个版本的 K8s API 来创建这个对象。
- `kind`: 定义了要创建的资源类型，这里是 `Pod`。
- `metadata`: 包含了对象的元数据，如名称 (`name`) 和标签 (`labels`)。
- `spec`: 定义了对象的期望状态，例如 Pod 中应该包含哪些容器。

使用 `kubectl` 创建这个 Pod:
```bash
kubectl apply -f my-first-pod.yaml
# pod/nginx-pod created
```

### 4. 观察和检查 Pod
```bash
# 查看所有 Pod 的列表和基本状态
kubectl get pods
# NAME        READY   STATUS    RESTARTS   AGE
# nginx-pod   1/1     Running   0          30s

# 查看更详细的状态，包括被分配的 IP 和所在节点
kubectl get pods -o wide

# 查看 Pod 的详细信息，包括事件日志，这对于排错至关重要
kubectl describe pod nginx-pod

# 查看 Pod 中容器的标准输出日志
kubectl logs nginx-pod

# 在运行中的 Pod 内执行命令 (类似于 docker exec)
kubectl exec -it nginx-pod -- /bin/bash
# root@nginx-pod:/# ls
# root@nginx-pod:/# exit
```

### 5. 删除 Pod
```bash
kubectl delete -f my-first-pod.yaml
# pod "nginx-pod" deleted

# 或者按名称删除
kubectl delete pod nginx-pod
```

## 💻 Go 编程实现 (10%)
### 项目: `k8s-cluster-info`
**目标**: 编写一个简单的 Go 程序，使用 `client-go` 连接到集群并打印出所有节点的名称和版本信息。

**1. 初始化项目**
```bash
mkdir k8s-cluster-info
cd k8s-cluster-info
go mod init cluster.info.dev/me
go get k8s.io/client-go@v0.28.2 k8s.io/api@v0.28.2 k8s.io/apimachinery@v0.28.2
```

**2. 编写代码 (`main.go`)**
```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// 1. 加载 kubeconfig 文件
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("无法获取用户家目录: %v", err)
	}
	kubeconfigPath := filepath.Join(userHomeDir, ".kube", "config")

	// 2. 构建配置
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Fatalf("加载 kubeconfig 失败: %v", err)
	}

	// 3. 创建 clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("创建 clientset 失败: %v", err)
	}

	// 4. 使用 clientset 与 API Server 交互
	fmt.Println("--- Kubernetes Nodes ---")
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("列出节点失败: %v", err)
	}

	for _, node := range nodes.Items {
		fmt.Printf("- Name: %s\n", node.Name)
		fmt.Printf("  Kubelet Version: %s\n", node.Status.NodeInfo.KubeletVersion)
		fmt.Printf("  OS: %s\n", node.Status.NodeInfo.OperatingSystem)
		fmt.Println("--------------------")
	}
}
```

**3. 运行**
```bash
go run main.go
# --- Kubernetes Nodes ---
# - Name: minikube
#   Kubelet Version: v1.28.3
#   OS: linux
# --------------------
```

## 🔍 故障排查与优化
- **`kubectl` 命令无法连接**:
  - 检查 `~/.kube/config` 文件是否存在且内容正确。
  - 运行 `minikube status` 确保集群正在运行。
- **Pod 状态为 `Pending`**:
  - `kubectl describe pod <pod-name>` 查看事件。常见原因：调度器找不到合适的节点（如资源不足）。
- **Pod 状态为 `ImagePullBackOff` 或 `ErrImagePull`**:
  - `kubectl describe pod <pod-name>` 查看事件。常见原因：镜像名称错误、Tag 不存在、或无法访问私有镜像仓库。

## 🏠 课后作业
1.  **研究 Pod 生命周期**: 阅读官方文档，详细了解 Pod 从创建到销毁经历的各个阶段（`Pending`, `Running`, `Succeeded`, `Failed`, `Unknown`）及其含义。
2.  **多容器 Pod**: 修改 `my-first-pod.yaml`，在同一个 Pod 中增加一个 `busybox` 容器（`image: busybox`），让它每5秒打印一次日期 (`command: ["/bin/sh", "-c", "while true; do date; sleep 5; done"]`）。部署后，使用 `kubectl logs nginx-pod -c busybox-container` 查看 busybox 容器的日志。思考这种模式的应用场景。
