# Day 3: K8s 服务与网络 (Service)

## 🎯 学习目标
- **技能目标**: 理解并掌握 Kubernetes 中实现服务发现和负载均衡的核心资源 `Service`。
- **核心概念**: 深入理解 Pod IP 的非持久性问题，以及 `Service` 如何通过一个稳定的虚拟 IP 和 DNS 名称来解决这个问题。
- **具体成果**:
  - 能够独立为一组 Pod 创建一个 `ClusterIP` 类型的 `Service`，并实现集群内部的访问。
  - 能够使用 `NodePort` 类型的 `Service`，将应用端口暴露到集群外部进行访问。
  - 能够解释 `Service`、`EndpointSlice` (或 `Endpoints`) 和 `Pod` 之间的关联关系。
  - 能够解释 K8s 内部的 DNS 是如何工作的。

## 📚 理论基础 (40%)
### 1. 为什么需要 Service？
在 Day 2 我们学习了 Deployment，它可以动态地创建和销毁 Pod 来维持期望的副本数。这带来一个新问题：**Pod 的 IP 地址是不固定的**。当一个 Pod 挂掉并被重建后，它会获得一个新的 IP 地址。

这就意味着，如果一个“前端” Pod 想访问一个“后端” Pod，它不能硬编码后端的 IP 地址。我们需���一种机制，能够：
1.  为一组提供相同服务的 Pod 提供一个**稳定、不变的**访问入口。
2.  自动追踪这组 Pod 的 IP 地址变化，更新路由信息。
3.  在多个 Pod 副本之间进行**负载均衡**。

`Service` 就是 Kubernetes 为解决这个问题而设计的核心资源。

### 2. Service 的工作原理
`Service` 的核心思想是在客户端和 Pod 之间增加一个抽象层。它通过**标签选择器 (Label Selector)** 来找到它要代理的一组 Pod。

当一个 `Service` 被创建时，会发生两件主要的事情：
1.  **分配虚拟 IP (ClusterIP)**: Kubernetes 会为这个 Service 分配一个**虚拟的、仅在集群内部有效的 IP 地址**。这个 IP 地址是稳定的，只要 Service 存在，它就不会改变。
2.  **创建 Endpoints (或 EndpointSlice)**: Kubernetes 会自动创建一个 `EndpointSlice` 对象。这个对象会持续地、自动地列出所有被 Service 的标签选择器匹配到的、并且处于 `Ready` 状态的 Pod 的真实 IP 地址和端口。

当集群内的任何一个客户端（例如另一个 Pod）尝试访问 Service 的 ClusterIP 时，节点上的 `kube-proxy` 组件会拦截这个请求，并根据 `EndpointSlice` 中的列表，从后端的健康 Pod 中选择一个，然后将流量转发过去，从而实现了负载均衡。

![Service Architecture](https://miro.medium.com/v2/resize:fit:1200/1*OBWhC0b_n6xG_a_msH2uFw.png)

### 3. Service 的类型
`Service` 有多种类型，用于满足不同的暴露需求：

- **`ClusterIP`**:
  - **默认类型**。
  - 为 Service 分配一个集群内部的虚拟 IP。
  - **只能在集群内部访问**。
  - **适用场景**: 大多数集群内部服务之间的通信，例如前端服务访问后端服务、API 网关访问微服务。

- **`NodePort`**:
  - 在 `ClusterIP` 的基础上，额外在**每一个工作节点**上都打开一个相同的、固定的端口（范围通常是 30000-32767）。
  - 任何发送到 ` <NodeIP>:<NodePort>` 的流量都会被转发到该 Service 的 ClusterIP，进而转发到后端的 Pod。
  - **可以从集群外部访问**。
  - **适用场景**: 用于临时暴露服务或在开发环境中快速测试，不建议在生产环境中直接用于关键业务，因为它绕过了云提供商的负载均衡器。

- **`LoadBalancer`**:
  - 在 `NodePort` 的基础上，额外请求云提供商（如 AWS, GCP, Azure）创建一个**外部负载均衡器**。
  - 这个外部负载均衡器会有一个公网 IP，并将流量导向所有节点的 `NodePort`。
  - **是向公网暴露服务的标准方式**。
  - **适用场景**: 需要从互联网公开访问的应用，如网站、对外 API。此类型仅在云 K8s 环境中有效。

- **`Headless`**:
  - 通过将 `spec.clusterIP` 设置为 `None` 来创建。
  - Kubernetes 不会为它分配 ClusterIP。
  - 当查询这个 Service 的 DNS 名称时，它不会返回一个虚拟 IP，而是直接返回**所有后端 Pod 的 IP 地址列表**。
  - **适用场景**: 用于 StatefulSet，为每个有状态的 Pod 提供独立的、稳定的 DNS 记录；或者当客户端希望自己来决定连接哪个 Pod 实例时。

### 4. 服务发现与 DNS
Kubernetes 集群内部有一个 DNS 服务（通常是 CoreDNS）。当一个 Service 被创建时，DNS 服务会自动为其创建一条 DNS A 记录：
` <service-name>.<namespace>.svc.cluster.local`
这条记录会解析到该 Service 的 ClusterIP。

这意味着，在同一个 `namespace` 下的 Pod，可以直接通过 `<service-name>` 来访问另一个服务，无需关心其 IP 地址。例如，`order-service` 可以直接通过 `http://user-service` 来访问 `user-service`。

## 🛠️ 实践操作 (50%)
### 1. 为 Deployment 创建 ClusterIP Service
我们将为 Day 2 创建的 `nginx-deployment` 创建一个 Service。
创建一个文件 `nginx-service.yaml`:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  type: ClusterIP # 可以省略，因为是默认值
  selector:
    app: nginx # 关键：选择所有带有 app=nginx 标签的 Pod
  ports:
    - protocol: TCP
      port: 80 # Service 自身暴露的端口
      targetPort: 80 # 流量要转发到 Pod 的哪个端口
```
部署它:
```bash
kubectl apply -f nginx-service.yaml
```

### 2. 验证 ClusterIP 访问
```bash
# 查看 Service，注意它获得了一个 CLUSTER-IP
kubectl get svc nginx-service
# NAME            TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
# nginx-service   ClusterIP   10.108.111.222   <none>        80/TCP    15s

# 启动一个临时的 busybox Pod，用于在集群内部测试访问
kubectl run -it --rm busybox --image=busybox -- /bin/sh

# 在 busybox 的 shell 中，使用 wget 访问 Service 的 DNS 名称
# wget -q -O - http://nginx-service
# <!DOCTYPE html>
# <html>
# <head>
# <title>Welcome to nginx!</title>
# ...
# </html>

# 多次访问，流量会被负载均衡到不同的 Nginx Pod
```

### 3. 查看 Endpoints
`kube-proxy` 是如何知道要将流量转发到哪些 Pod 的呢？答案是 `EndpointSlice`。
```bash
# 查看与 Service 关联的 EndpointSlice
kubectl get endpointslice -l kubernetes.io/service-name=nginx-service
# NAME                      ADDRESSTYPE   PORTS   ENDPOINTS                           AGE
# nginx-service-abcde       IPv4          80      10.244.1.10,10.244.2.8,10.244.3.9   5m

# 可以看到，它列出了所有后端 Pod 的真实 IP 地址。
```

### 4. 将 Service 暴露到集群外 (NodePort)
修改 `nginx-service.yaml`，将 `type` 改为 `NodePort`。
```yaml
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
spec:
  type: NodePort # 修改类型
  selector:
    app: nginx
  ports:
    - protocol: TCP
      port: 80
      targetPort: 80
      # nodePort: 30080 # 可以指定一个端口，但通常让 K8s 自动分配
```
重新应用: `kubectl apply -f nginx-service.yaml`

查看 Service，注意 `PORT(S)` 列的变化：
```bash
kubectl get svc nginx-service
# NAME            TYPE       CLUSTER-IP       EXTERNAL-IP   PORT(S)        AGE
# nginx-service   NodePort   10.108.111.222   <none>        80:31234/TCP   10m
# 80:31234 的意思是，Service 的 80 端口被映射到了所有节点的 31234 端口。
```

获取 minikube 节点的 IP 地址，并从你的电脑上访问它：
```bash
minikube ip
# 192.168.49.2

# 在你的浏览器或使用 curl 访问
curl http://192.168.49.2:31234
# <!DOCTYPE html> ...
```

## 💻 Go 编程实现 (10%)
### 项目: `k8s-service-lister`
**目标**: 编写一个 Go 程序，列出指定命名空间下的所有 Service 及其类型和 ClusterIP。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s.io/client-go/kubernetes"
	k8s.io/client-go/tools/clientcmd"
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

	fmt.Printf("--- Services in namespace '%s' ---\n", namespace)
	serviceList, err := clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for _, svc := range serviceList.Items {
		fmt.Printf("- Name: %s\n", svc.Name)
		fmt.Printf("  Type: %s\n", svc.Spec.Type)
		fmt.Printf("  ClusterIP: %s\n", svc.Spec.ClusterIP)
		fmt.Println("--------------------")
	}
}
```
**运行**:
```bash
go run main.go default
# --- Services in namespace 'default' ---
# - Name: kubernetes
#   Type: ClusterIP
#   ClusterIP: 10.96.0.1
# --------------------
# - Name: nginx-service
#   Type: NodePort
#   ClusterIP: 10.108.111.222
# --------------------
```

## 🔍 故障排查与优化
- **无法通过 Service 名称访问**:
  - 检查 DNS 是否正常: `kubectl exec -it <pod-name> -- nslookup <service-name>`。
  - 检查 Service 的 `selector` 是否正确，是否能匹配到 Pod 的 `labels`。
- **无法通过 Service IP 访问**:
  - `kubectl describe svc <service-name>` 查看 `Endpoints` 是否为空。
  - 如果 `Endpoints` 为空，说明没有健康的、`Ready` 状态的 Pod 被选中。检查后端 Pod 的状态和健康探针（Day 5 内容）。
- **无法通过 NodePort 访问**:
  - 检查防火墙规则，确保节点上的端口是开放的。
  - 检查 `kube-proxy` Pod 是否在所有节点上都正常运行。

## 🏠 课后作业
1.  **研究 `EndpointSlice`**: 使用 `kubectl get endpointslice` 和 `kubectl describe endpointslice <name>` 命令，详细查看 `EndpointSlice` 对象的内容，理解它是如何将 Service 与一组 Pod IP 地址关联起来的。
2.  **Headless Service 实践**: 创建一个 `Headless` Service (设置 `clusterIP: None`)，并为它关联 Nginx Deployment。然后在一个临时 Pod 中使用 `nslookup <headless-service-name>`，观察返回的结果与��通 Service 有何不同。
