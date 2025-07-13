# Day 5: 应用健康检查与资源限制

## 🎯 学习目标
- **技能目标**: 掌握为 Kubernetes 应用配置健康检查和资源配额的核心方法。
- **核心概念**: 深刻理解 `Liveness Probe`, `Readiness Probe`, `Startup Probe` 的区别和作用，以及 `requests` 和 `limits` 对 Pod 调度和稳定性的影响。
- **具体成果**:
  - 能够为一个 Deployment 配置 `httpGet` 类型的存活探针和就绪探针。
  - 能够通过模拟健康检查失败，观察并解释 Kubernetes 的自愈行为（重启 Pod 或将其移出 Service 端点）。
  - 能够为一个容器设置合理的 CPU 和内存 `requests` 与 `limits`。
  - 能够解释 Kubernetes 的三种 QoS (Quality of Service) 等级。

## 📚 理论基础 (40%)
### 1. 为什么需要健康检查？
一个容器进程在运行，不代表它提供的服务就一定正常。例如：
- 应用程序可能发生死锁，进程仍在但无法响应请求。
- 应用可能因为依赖的后端服务（如数据库）无法连接而暂时无法提供服务。
- 应用启动过程较长，需要一段时间来加载数据或预热缓存，期间无法处理流量。

如果 Kubernetes 无法感知到这些内部状态，它可能会将流量发送给一个无法处理请求的 Pod，或者无法从一个已经“僵死”的应用中恢复。**健康探针 (Probes)** 就是 Kubelet 用来检测容器内部健康状况的机制。

### 2. 三种探针 (Probes)
Kubelet 可以配置三种探针来检查容器：

- **`Liveness Probe` (存活探针)**:
  - **作用**: 判断容器是否**存活**。
  - **行为**: 如果存活探针**失败**，Kubelet 会认为容器已经死亡，会**杀死并重启**该容器。
  - **适用场景**: 用于检测应用是否发生死锁或进入不可恢复的故障状态，通过重启来尝试恢复服务。

- **`Readiness Probe` (就绪探针)**:
  - **作用**: 判断容器是否**准备好接收流量**。
  - **行为**: 如果就绪探针**失败**，Kubelet 不会杀死容器，而是将该 Pod 从 Service 的 Endpoints 列表中**移除**。这样，新的网络流量就不会再被转发到这个 Pod。直到就绪探针再次成功，Pod 才会被重新加回 Endpoints 列表。
  - **适用场景**: 用于处理应用启动慢、依赖外部服务、或需要进行临时维护的场景。

- **`Startup Probe` (启动探针)**:
  - **作用**: 判断容器内的应用是否已经**启动成功**。它在其他两种探针之前执行。
  - **行为**: 只有当启动探针**成功**后，存活探针和就绪探针才会开始��作。如果启动探针在设定的 `failureThreshold` * `periodSeconds` 时间内一直不成功，Kubelet 就会杀死并重启容器。
  - **适用场景**: 专门用于启动时间非常长的应用，可以给应用足够的启动时间，避免被存活探针过早地杀死。

### 3. 探针的配置方式
每种探针都可以通过以下三种方式之一来配置：
- **`httpGet`**: 向容器的指定端口和路径发送一个 HTTP GET 请求。如果返回的 HTTP 状态码在 200-399 之间，则认为探测成功。
- **`exec`**: 在容器内执行一个指定的命令。如果命令的退出码为 0，则认为探测成功。
- **`tcpSocket`**: 尝试与容器的指定 TCP 端口建立连接。如果连接能够成功建立，则认为探测成功。

### 4. 资源请求 (Requests) 与限制 (Limits)
在定义 Pod 时，你可以为每个容器指定它需要的 CPU 和内存资源。

- **`requests` (资源请求)**:
  - **作用**: 告诉调度器 (Scheduler)，这个容器**至少需要**多少资源才能正常运行。
  - **行为**: 调度器在调度 Pod 时，会确保目标节点上有足够的可用资源来满足 Pod 所有容器的 `requests` 总和。`requests` 是一个**有保证的**资源量。
  - **单位**: CPU 的单位是 `cores` (核心数)，可以写成 `0.5` 或 `500m` (500 millicores)。内存的���位是字节，通常使用 `Mi` (Mebibytes) 或 `Gi` (Gibibytes)。

- **`limits` (资源限制)**:
  - **作用**: 定义一个容器**最多可以**使用多少资源。
  - **行为**:
    - **CPU**: 如果容器的 CPU 使用试图超过 `limits`，它的 CPU 时间会被**节流 (throttled)**，导致性能下降。
    - **内存**: 如果容器的内存使用超过 `limits`，它会被系统**杀死**（OOMKilled, Out of Memory Killed）。
  - **核心价值**: 防止单个有问题的容器（如内存泄漏）耗尽整个节点的资源，从而影响到节点上其他所有 Pod 的稳定性。

### 5. QoS (Quality of Service) 等级
根据容器设置的 `requests` 和 `limits`，Kubernetes 会为 Pod 分配三种不同的 QoS 等级：

- **`Guaranteed` (有保证的)**:
  - **条件**: Pod 中的**每一个**容器都必须同时设置了 CPU 和内存的 `requests` 和 `limits`，并且 `requests` 值必须**等于** `limits` 值。
  - **待遇**: 最高优先级。这种 Pod 最不可能在节点资源紧张时被杀死。

- **`Burstable` (可突发的)**:
  - **条件**: Pod 中至少有一个容器设置了 CPU 或内存的 `requests`，但不满足 `Guaranteed` 的条件（例如，`limits` 大于 `requests`，或只设置了 `requests`）。
  - **待遇**: 中等优先级。

- **`BestEffort` (尽力而为的)**:
  - **条件**: Pod 中的所有容器都没有设置任何 `requests` 或 `limits`。
  - **待遇**: 最低优先级。当节点资源不足时，这种 Pod 是**最先被驱逐或杀死**的。

**最佳实践**: 总是为你的生产应用设置 `requests` 和 `limits`，至少让它们成为 `Burstable`，以保证基本的运行资源和稳定性。

## 🛠️ 实践操作 (50%)
### 1. 为 Deployment 添加健康探针
修改 Day 2 的 `nginx-deployment.yaml`，为其添加 `livenessProbe` 和 `readinessProbe`。
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  replicas: 2
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.25
        ports:
        - containerPort: 80
        livenessProbe:
          httpGet:
            path: / # 检查根路径
            port: 80
          initialDelaySeconds: 5 # Pod 启动后 5 秒开始第一次探测
          periodSeconds: 10    # 每 10 秒探测一次
        readinessProbe:
          httpGet:
            path: /
            port: 80
          initialDelaySeconds: 3
          periodSeconds: 5
```
部署: `kubectl apply -f nginx-deployment.yaml`

### 2. 模拟 Liveness Probe 失败
```bash
# 找��一个 Nginx Pod 的名字
kubectl get pods -l app=nginx

# 进入 Pod，手动删除首页文件，让 httpGet / 返回 404
kubectl exec -it <nginx-pod-name> -- rm /usr/share/nginx/html/index.html

# 观察 Pod 状态
kubectl get pods -l app=nginx -w
# 你会看到该 Pod 的 RESTARTS 次数从 0 变为 1，因为它被 Kubelet 重启了。

# 查看 Pod 事件，可以看到 Liveness probe failed 的记录
kubectl describe pod <nginx-pod-name>
```

### 3. 模拟 Readiness Probe 失败
为了方便观察，我们先创建一个 Service 指向这个 Deployment。
```bash
kubectl expose deployment nginx-deployment --port=80 --type=ClusterIP
```
现在，再次删除一个 Pod 的首页文件。
```bash
# 进入另一个 Pod，删除首页文件
kubectl exec -it <another-nginx-pod-name> -- rm /usr/share/nginx/html/index.html

# 观察 Pod 状态，READY 列会从 1/1 变为 0/1
kubectl get pods -l app=nginx
# NAME                                READY   STATUS    RESTARTS   AGE
# nginx-deployment-xxxx-abcde         1/1     Running   0          10m
# nginx-deployment-xxxx-fghij         0/1     Running   0          5m  <-- 就绪探针失败

# 查看 Service 的 Endpoints，会发现失败的 Pod 的 IP 已经被移除了
kubectl describe svc nginx-deployment
# Endpoints:         10.244.1.12:80  <-- 只剩下一个健康的 Pod
```
这证明了就绪探针失败后，流量将不再被发送到有问题的 Pod。

### 4. 设置资源请求和限制
修改 `nginx-deployment.yaml`，为容器添加资源配置。
```yaml
# ...
        ports:
        - containerPort: 80
        resources:
          requests:
            memory: "64Mi"
            cpu: "250m" # 1/4 核
          limits:
            memory: "128Mi"
            cpu: "500m" # 1/2 核
# ...
```
重新部署: `kubectl apply -f nginx-deployment.yaml`

查看 Pod 的 QoS 等级：
```bash
kubectl get pod <nginx-pod-name> -o yaml
# ...
# status:
#   qosClass: Burstable
```
查看节点上的资源分配情况：
```bash
kubectl describe node minikube
# ...
# Allocated resources:
#   (Total limits may be over 100 percent, i.e., overcommitted.)
#   Resource           Requests      Limits
#   --------           --------      ------
#   cpu                500m (25%)    1 (50%)
#   memory             128Mi (1%)    256Mi (3%)
# ...
```
可以看到，两个 Pod 的 `requests` 和 `limits` 都被统计进去了。

## 💻 Go 编程实现 (10%)
### 项目: `k8s-pod-resource-viewer`
**目标**: 编写一个 Go 程序，列出指定命名空间下所有 Pod 及其容器的资源 `requests` 和 `limits`。

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
	if len(os.Args) < 2 {
		fmt.Println("用法: go run main.go <namespace>")
		os.Exit(1)
	}
	namespace := os.Args[1]

	// --- 配置和创建 clientset ---
	userHomeDir, _ := os.UserHomeDir()
	kubeconfig := filepath.Join(userHomeDir, ".kube", "config")
	config, _ := clientcmd.BuildConfigFromFlags("", kubeconfig)
	clientset, _ := kubernetes.NewForConfig(config)

	fmt.Printf("--- Pod Resources in namespace '%s' ---\n", namespace)
	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for _, pod := range podList.Items {
		fmt.Printf("- Pod: %s\n", pod.Name)
		for _, container := range pod.Spec.Containers {
			fmt.Printf("  - Container: %s\n", container.Name)
			fmt.Printf("    Requests:\n")
			fmt.Printf("      CPU: %s\n", container.Resources.Requests.Cpu().String())
			fmt.Printf("      Memory: %s\n", container.Resources.Requests.Memory().String())
			fmt.Printf("    Limits:\n")
			fmt.Printf("      CPU: %s\n", container.Resources.Limits.Cpu().String())
			fmt.Printf("      Memory: %s\n", container.Resources.Limits.Memory().String())
		}
		fmt.Println("--------------------")
	}
}
```

## 🔍 故障排查与优化
- **Pod 因 Liveness Probe 失败被反复重启**:
  - `kubectl describe pod` 查看事件，确认是存活探针失败。
  - 检查探针的配置是否正确（路径、端口）。
  - 可能是应用本身有问题，`kubectl logs --previous <pod-name>` 查看上一个被杀死的容器的日志。
  - 可能是 `initialDelaySeconds` 设置太短，应用还没启动好就被探测了。
- **Pod 无法达到 Ready 状态**:
  - `kubectl describe pod` 查看事件，确认是就绪探针失败。
  - 检查应用是否能正常响应探测请求。
- **Pod 因 OOMKilled 被重启**:
  - `kubectl describe pod` 查看 `Reason: OOMKilled`。
  - 说明内存 `limits` 设置太小，需要调大。

## 🏠 课后作业
1.  **研究 `exec` 探针**: 创建一个 Pod，使用 `exec` 类型的探针。例如，`command: ["cat", "/tmp/healthy"]`。然后通过 `kubectl exec` 进入 Pod 创建或删除 `/tmp/healthy` 文件，观察探针状态的变化。
2.  **研究 `Guaranteed` QoS**: 修改你的 Deployment，让 CPU 和内存的 `requests` 和 `limits` 完全相等。部署后，使用 `kubectl get pod <name> -o yaml` 验证其 `qosClass` 是否变为了 `Guaranteed`。
3.  **思考**: 在什么情况下，你应该只设置 `readinessProbe` 而不设置 `livenessProbe`？（提示：考虑一个需要从队列中处理任务，但处理一个任务可能耗时很长的 Worker 应用）。
