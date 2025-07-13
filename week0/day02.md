# Day 2: 无状态应用管理 (Deployment, ReplicaSet)

## 🎯 学习目标
- **技能目标**: 理解并掌握 Kubernetes 中管理无状态应用的核心控制器 `Deployment` 和 `ReplicaSet`。
- **核心概念**: 深入理解声明式 API、控制器模式，以及 `Deployment` 如何实现应用的滚动更新和回滚。
- **具体成果**:
  - 能够独立编写一个 `Deployment` YAML 文件来部署、扩容和缩容一个无状态应用（如 Nginx）。
  - 能够成功地对一个已部署的应用执行滚动更新，将其升级到新版本。
  - 能够查看更新历史，并在需要时将应用回滚到指定的旧版本。
  - 能够解释 `Deployment`, `ReplicaSet`, `Pod` 三者之间的关系。

## 📚 理论基础 (30%)
### 1. 声明式 API 与控制器模式
Kubernetes 的工作模式是**声明式 (Declarative)** 的，而非命令式 (Imperative)。
- **命令式**: 你告诉系统“做什么”，例如 `运行一个容器`、`停止那个容器`。
- **声明式**: 你告诉系统“我想要什么状态”，例如 `我想要一直有3个Nginx容器在运行`。

你通过 YAML 文件向 API Server 声明你的“期望状态”。而 Kubernetes 内部的各种**控制器 (Controllers)** 则会不停地工作，持续地将集群的“当前状态”调整为你的“期望状态”。这正是 Kubernetes 强大自愈能力的来源。

### 2. ReplicaSet: 副本的守护者
- **职责**: `ReplicaSet` 的唯一职责就是确保在任何时候都有指定数量的、符合特定模板的 Pod 副本在运行。
- **工作原理**: 它通过一个**标签选择器 (Label Selector)** 来识别它应该管理的 Pod。如果发现运行中的 Pod 数量少于期望值，它就会根据 **Pod 模板 (Pod Template)** 创建新的 Pod。如果数量多于期望值，它就会随机删除多余的 Pod。
- **使用**: 你通常不会直接创建 `ReplicaSet`，而是通过 `Deployment` 来间接管理它。

### 3. Deployment: 更高级的应用管理器
`Deployment` 是一个比 `ReplicaSet` 更高阶的控制器，它提供了更多管理应用所需的功能，是部署无状态应用的首选方式。
- **核心功能**:
  - **管理 ReplicaSet 和 Pod**: 你创建一个 Deployment，它会自动为你创建一个 ReplicaSet，然后由 ReplicaSet 来创建 Pod。
  - **滚动更新 (Rolling Update)**: 这是 Deployment 最核心的功能之一。当你更新应用的镜像或配置时，Deployment 会以一种受控的方式，逐步地用新版本的 Pod 替换旧版本的 Pod，从而实现平滑升级，避免服务中断。
  - **版本回滚 (Rollback)**: Deployment 会记录下每次更新的历史版本。如果发现新版本有问题，你可以轻松地将应用一键回滚到之前的某个稳定版本。

### 4. 三者关系
**Deployment → ReplicaSet → Pod**
- 你定义一个 `Deployment`。
- `Deployment` 根据自己的定义，创建一个 `ReplicaSet`。
- `ReplicaSet` 根据自己的定义，创建出指定数量的 `Pod`。
- 当你更新 `Deployment` 时，它会创建一个**新的** `ReplicaSet`，然后逐步地将 Pod 从旧 `ReplicaSet` 的管理下转移到新 `ReplicaSet`，从而实现滚动更新。旧的 `ReplicaSet` 不会被立即删除，以便支持回滚。

![Deployment Relationship](https://i.stack.imgur.com/kflbS.png)

## 🛠️ 实践操作 (50%)
### 1. 创建一个 Deployment
创建一个文件 `nginx-deployment.yaml`:
```yaml
apiVersion: apps/v1 # 注意这里的 apiVersion 是 apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 3 # 声明期望状态：需要 3 个副本
  selector:
    matchLabels:
      app: nginx # 标签选择器：管理那些带有 app=nginx 标签的 Pod
  template: # Pod 模板：如何创建 Pod
    metadata:
      labels:
        app: nginx # Pod 的标签，必须与上面的 selector 匹配
    spec:
      containers:
      - name: nginx
        image: nginx:1.24 # 使用 1.24 版本
        ports:
        - containerPort: 80
```
部署它:
```bash
kubectl apply -f nginx-deployment.yaml
```

### 2. 观察创建的资源
```bash
# 查看 Deployment 状态
kubectl get deployment nginx-deployment
# NAME               READY   UP-TO-DATE   AVAILABLE   AGE
# nginx-deployment   3/3     3            3           30s

# 查看 ReplicaSet，注意它的名字是由 Deployment 名称加一个 hash 构成的
kubectl get rs
# NAME                          DESIRED   CURRENT   READY   AGE
# nginx-deployment-6b6c47b5b6   3         3         3       45s

# 查看 Pods，注意它们都带有 app=nginx 标签
kubectl get pods --show-labels
# NAME                                READY   STATUS    RESTARTS   AGE   LABELS
# nginx-deployment-6b6c47b5b6-abcde   1/1     Running   0          60s   app=nginx,pod-template-hash=6b6c47b5b6
# nginx-deployment-6b6c47b5b6-fghij   1/1     Running   0          60s   app=nginx,pod-template-hash=6b6c47b5b6
# nginx-deployment-6b6c47b5b6-klmno   1/1     Running   0          60s   app=nginx,pod-template-hash=6b6c47b5b6
```

### 3. 扩容和缩容
```bash
# 使用 scale 命令将副本数扩展到 5
kubectl scale deployment nginx-deployment --replicas=5
# deployment.apps/nginx-deployment scaled

# 观察 Pod 数量变化
kubectl get pods -l app=nginx # 使用标签选择器来查看

# 缩容回 2 个
kubectl scale deployment nginx-deployment --replicas=2
```

### 4. 执行滚动更新
现在，我们将 Nginx 的版本从 `1.24` 升级到 `1.25`。
最简单的方式是使用 `kubectl set image` 命令：
```bash
kubectl set image deployment/nginx-deployment nginx=nginx:1.25
# deployment.apps/nginx-deployment image updated
```
> 你也可以直接修改 YAML 文件中的 `image` 字段，然后再次执行 `kubectl apply -f nginx-deployment.yaml`，效果是一样的。

观察滚动更新的过程：
```bash
# 使用 -w 参数持续观察 Pod 的变化
kubectl get pods -l app=nginx -w
# 你会看到新的 Pod 被创建 (terminating 旧的，creating 新的)

# 查看更新状态
kubectl rollout status deployment/nginx-deployment
# Waiting for deployment "nginx-deployment" rollout to finish: 2 of 3 updated pods are available...
# deployment "nginx-deployment" successfully rolled out
```
更新完成后，查看 ReplicaSet，你会发现多了一个新的 RS，而旧的 RS 的副本数变为了 0。
```bash
kubectl get rs
# NAME                          DESIRED   CURRENT   READY   AGE
# nginx-deployment-6b6c47b5b6   0         0         0       10m  <-- 旧的 RS
# nginx-deployment-7d7c58c6c7   3         3         3       2m   <-- 新的 RS
```

### 5. 回滚应用
假设新版本 `1.25` 有 bug，我们需要回滚。
```bash
# 查看更新历史
kubectl rollout history deployment/nginx-deployment
# REVISION  CHANGE-CAUSE
# 1         <none>
# 2         <none>

# 执行回滚，回到上一个版本 (REVISION 1)
kubectl rollout undo deployment/nginx-deployment
# deployment.apps/nginx-deployment rolled back

# 再次观察 Pod 变化，它们会回滚到使用 1.24 镜像
kubectl get pods -l app=nginx -w
```

## 💻 Go 编程实现 (20%)
### 项目: `k8s-deployment-manager`
**目标**: 编写一个 Go 程序，使用 `client-go` 来获取指定 Deployment 的信息，并提供扩容/缩容的功能。

**1. 初始化项目**
```bash
mkdir k8s-deployment-manager
cd k8s-deployment-manager
go mod init deployment.manager.dev/me
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
	"strconv"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// 用法: go run main.go <namespace> <deployment-name> [replicas]
	if len(os.Args) < 3 {
		fmt.Println("用法: go run main.go <namespace> <deployment-name> [replicas]")
		os.Exit(1)
	}
	namespace := os.Args[1]
	deploymentName := os.Args[2]

	// --- 配置和创建 clientset ---
	userHomeDir, _ := os.UserHomeDir()
	kubeconfig := filepath.Join(userHomeDir, ".kube", "config")
	config, _ := clientcmd.BuildConfigFromFlags("", kubeconfig)
	clientset, _ := kubernetes.NewForConfig(config)

	// --- 如果没有提供副本数参数，则只获取信息 ---
	if len(os.Args) < 4 {
		fmt.Printf("获取 Deployment '%s' 信息...\n", deploymentName)
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf(" - Replicas: %d\n", *deployment.Spec.Replicas)
		fmt.Printf(" - Image: %s\n", deployment.Spec.Template.Spec.Containers[0].Image)
		return
	}

	// --- 如果提供了副本数参数，则执行扩/缩容 ---
	replicas, err := strconv.Atoi(os.Args[3])
	if err != nil {
		log.Fatalf("副本数必须是整数: %v", err)
	}

	fmt.Printf("将 Deployment '%s' 的副本数调整为 %d...\n", deploymentName, replicas)
	
	// 使用 Get-Update 的方式来更新对象
	retryErr := clientcmd.RetryOnConflict(clientcmd.DefaultRetry, func() error {
		// 1. Get: 获取最新版本的 Deployment 对象
		deployment, getErr := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
		if getErr != nil {
			return getErr
		}

		// 2. Update: 修改副本数
		*deployment.Spec.Replicas = int32(replicas)

		// 3. Commit: 提交更新
		_, updateErr := clientset.AppsV1().Deployments(namespace).Update(context.TODO(), deployment, metav1.UpdateOptions{})
		return updateErr
	})

	if retryErr != nil {
		log.Fatalf("更新失败: %v", retryErr)
	}
	fmt.Println("更新成功!")
}
```

**3. 运行**
```bash
# 获取 nginx-deployment 的信息
go run main.go default nginx-deployment

# 将副本数调整为 5
go run main.go default nginx-deployment 5

# 将副本数调整为 1
go run main.go default nginx-deployment 1
```

## 🔍 故障排查与优化
- **滚动更新卡住**:
  - `kubectl rollout status deployment/<name>` 查看状态。
  - `kubectl describe deployment <name>` 查看详细信息。
  - `kubectl describe rs <new-rs-name>` 查看新 ReplicaSet 的事件。
  - `kubectl describe pod <new-pod-name>` 查看新 Pod 的事件。常见原因：新镜像拉取失败 (`ImagePullBackOff`)、健康检查失败导致 Pod 不 Ready、资源不足无法创建新 Pod。
- **优化**: 在 Deployment 的 `spec.strategy.rollingUpdate` 中可以设置 `maxSurge` 和 `maxUnavailable` 参数来精细控制滚动更新的过程。
  - `maxSurge`: 更新过程中，允许比期望副本数多出的 Pod 数量。
  - `maxUnavailable`: 更新过程中，允许的不可用 Pod 的最大数量。

## 🏠 课后作业
1.  **研究 Deployment 更新策略**: 创建一个 Deployment，将其 `spec.strategy.type` 设置为 `Recreate`。然后尝试更新镜像，观察其行为与 `RollingUpdate` 有何不同。
2.  **带命令的更新历史**: 在执行 `kubectl set image` 或 `kubectl apply` 时，使用 `--record` 标志 (虽然已废弃，但为了解其功能可以一试) 或者在 YAML 中使用 `annotations` 来记录每次变更的原因。然后执行 `kubectl rollout history` 查看 `CHANGE-CAUSE` 列。
