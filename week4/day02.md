# Day 2: 有状态应用与 StatefulSet

## 🎯 学习目标
- **技能目标**: 深刻理解无状态应用 (Deployment) 与有状态应用 (StatefulSet) 的核心区别，并能阐述 StatefulSet 的三大特性。
- **具体成果**:
  - 能够独立编写并部署一个带有持久化存储的 StatefulSet 应用（例如 Redis）。
  - 能够验证 StatefulSet 的稳定网络标识和稳定存储。
  - 能够演示 StatefulSet 的有序部署、扩容和缩容过程。
  - 能够解释 Headless Service 在 StatefulSet 中的作用。

## 📚 理论基础 (30%)
### 1. 无状态 (Stateless) vs. 有状态 (Stateful)
在深入 StatefulSet 之前，必须理解这两种应用模式的根本区别：

- **无状态应用 (Stateless Application)**:
  - **特点**: 所有实例都是完全一样的，它们不保存任何本地数据。可以将它们看作是可任意替换的“计算单元”。
  - **例子**: Web 前端服务器 (Nginx, Apache), 无状态的 API 网关。
  - **K8s 管理器**: `Deployment` 或 `ReplicaSet`。
  - **核心优势**: 易于水平扩展、替换和升级。任何一个实例挂掉，K8s 都可以随意启动一个新的来替代，无需关心数据一致性问题。

- **有状态应用 (Stateful Application)**:
  - **特点**: 每个实例都有其独特的“身份”，并需要持久化地保存自己的状态（数据）。实例之间通常不是对等的，可能有主从、分片等关系。
  - **例子**: 数据库 (MySQL, PostgreSQL, MongoDB), 消息队列 (Kafka, RabbitMQ), 分布式协调服务 (Zookeeper, etcd)。
  - **K8s 管理器**: `StatefulSet`。
  - **核心挑战**: 实例不能被随意替换。如果一个数据库主节点挂了，必须确保新的主节点能访问到原来的数据，并且集群中的其他节点知道它的新地址。

### 2. StatefulSet 的三大核心特性
StatefulSet 正是为了解决有状态应用的挑战而设计的，它提供了三大保证：

#### a. 稳定的、唯一的网络标识 (Stable, Unique Network Identifiers)
- **Pod 名称**: StatefulSet 管理的 Pod 名称是固定的、有序的，格式为 `<StatefulSet名称>-<序号>`，例如 `redis-0`, `redis-1`, `redis-2`。
- **DNS 域名**: 配合 **Headless Service**，每个 Pod 会获得一个唯一的、可预测的 DNS A 记录，格式为 `<Pod名称>.<Headless Service名称>.<命名空间>.svc.cluster.local`。
  - 例如，`redis-0.redis-headless.default.svc.cluster.local` 会稳定地解析到 `redis-0` 这个 Pod 的 IP 地址。
- **价值**: 应用内部的节点之间可以通过固定的 DNS 名称相互发现和通信，无需关心 Pod IP 的变化。

#### b. 稳定的、持久的存储 (Stable, Persistent Storage)
- **机制**: StatefulSet 使用 `volumeClaimTemplates` 字段为每个 Pod 动态地、自动地创建一个专属的 PVC。
- **命名**: PVC 的名称也是固定的，格式为 `<volumeClaimTemplate名称>-<StatefulSet名称>-<序号>`，例如 `data-redis-0`, `data-redis-1`。
- **绑定关系**: Pod `redis-0` 会永远绑定到 PVC `data-redis-0`。即使 `redis-0` 被删除或重启，新创建的 `redis-0` 依然会挂载回原来的 `data-redis-0`，从而保证了数据的持久性和连续性。

#### c. 有序的、优雅的部署和伸缩 (Ordered, Graceful Deployment and Scaling)
- **部署 (Scaling Up)**: 按照序号从小到大（0, 1, 2...）依次创建 Pod。K8s 会等待前一个 Pod (`n`) 进入 `Running and Ready` 状态后，才会开始创建下一个 (`n+1`)。
- **销毁 (Scaling Down)**: 按照序号从大到小（...2, 1, 0）依次删除 Pod。
- **价值**: 这种有序性对于需要依赖关系和启动顺序的集群应用至关重要。例如，在部署一个数据库集群时，通常需要先启动主节点，然后才能启动从节点。

### 3. Headless Service 的作用
- **定义**: Headless Service 是一种特殊的 Service，它通过将 `spec.clusterIP` 设置为 `None` 来创建。
- **功能**: 它不像普通 Service 那样提供一个负载均衡的虚拟 IP，而是直接将 Service 的 DNS 名称解析到其背后所有 Pod 的 IP 地址列表。
- **与 StatefulSet 的关系**: 当与 StatefulSet 结合使用时，它为每个 Pod 提供了上文所述的、独一无二的 DNS 记录，这是实现稳定网络标识的关键。

## 🛠️ 实践操作 (50%)
### 部署一个带持久化存储的 Redis StatefulSet

**1. 创建 Headless Service**
创建一个文件 `redis-headless-svc.yaml`:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: redis-headless # Service 名称
  labels:
    app: redis
spec:
  ports:
  - port: 6379
    name: redis
  clusterIP: None # 关键：设置为 Headless
  selector:
    app: redis # 匹配 StatefulSet 的 Pod 标签
```

**2. 创建 StatefulSet**
创建一个文件 `redis-statefulset.yaml`:
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis
spec:
  serviceName: "redis-headless" # 必须匹配 Headless Service 的名称
  replicas: 3 # 创建 3 个实例
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:6.2-alpine
        ports:
        - containerPort: 6379
          name: redis
        volumeMounts:
        - name: data # 对应下面的 volumeClaimTemplates
          mountPath: /data
  volumeClaimTemplates: # 关键：PVC 模板
  - metadata:
      name: data # PVC 名称的前缀
    spec:
      accessModes: [ "ReadWriteOnce" ]
      storageClassName: "linstor-r2" # 使用 Day 1 创建的 StorageClass
      resources:
        requests:
          storage: 1Gi
```

**3. 部署并观察**
```bash
# 部署 Service 和 StatefulSet
kubectl apply -f redis-headless-svc.yaml
kubectl apply -f redis-statefulset.yaml

# 观察 Pod 的有序创建过程
kubectl get pod -w -l app=redis
# 你会看到 redis-0, redis-1, redis-2 依次被创建

# 观察 PVC 的创建
kubectl get pvc -l app=redis
# NAME           STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
# data-redis-0   Bound    pvc-c8a...   1Gi        RWO            linstor-r2     2m
# data-redis-1   Bound    pvc-f9b...   1Gi        RWO            linstor-r2     1m
# data-redis-2   Bound    pvc-a1c...   1Gi        RWO            linstor-r2     30s
```

**4. 验证稳定的网络标识**
启动一个临时的 Pod，使用 `nslookup` 查询 DNS。
```bash
kubectl run -it --rm --image=busybox dns-test -- /bin/sh

# 在 dns-test Pod 的 shell 中执行:
# nslookup redis-headless
# Server:    10.96.0.10
# Address 1: 10.96.0.10 kube-dns.kube-system.svc.cluster.local
#
# Name:      redis-headless
# Address 1: 10.244.1.5 redis-2.redis-headless.default.svc.cluster.local
# Address 2: 10.244.2.4 redis-0.redis-headless.default.svc.cluster.local
# Address 3: 10.244.3.3 redis-1.redis-headless.default.svc.cluster.local

# 查询单个 Pod
# nslookup redis-0.redis-headless
# ...
# Name:      redis-0.redis-headless
# Address 1: 10.244.2.4
```

**5. 验证稳定的存储**
向 `redis-0` 写入数据，然后模拟其故障。
```bash
# 向 redis-0 写入数据
kubectl exec redis-0 -- redis-cli SET mykey "Hello Stateful World"

# 验证数据
kubectl exec redis-0 -- redis-cli GET mykey
# "Hello Stateful World"

# 手动删除 redis-0 Pod 来模拟故障
kubectl delete pod redis-0

# 观察 Pod 重建
kubectl get pod -w -l app=redis
# 你会看到一个新的 redis-0 Pod 被创建出来

# 在新的 redis-0 Pod 中验证数据是否依然存在
kubectl exec redis-0 -- redis-cli GET mykey
# "Hello Stateful World"  <-- 数据依然存在！
```
这个实验证明了，即使 Pod 实例被替换，它也会被重新挂载到原来的 PVC 上，保证了数据的连续性。

**6. 演示扩容和缩容**
```bash
# 扩容到 5 个实例
kubectl scale statefulset redis --replicas=5
# 观察到 redis-3, redis-4 依次被创建

# 缩容回 3 个实例
kubectl scale statefulset redis --replicas=3
# 观察到 redis-4, redis-3 依次被终止
```

## 💻 Go 编程实现 (20%)
### 项目: `k8s-statefulset-checker`
这个工具将使用 `client-go` 检查指定 StatefulSet 的状态，并列出其管理的 Pod 和对应的 PVC。

**1. 初始化项目**
```bash
mkdir k8s-statefulset-checker
cd k8s-statefulset-checker
go mod init statefulset.checker.dev/me
go get k8s.io/client-go k8s.io/apimachinery k8s.io/api
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

	"k8s.io/apimachinery/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// 检查命令行参数
	if len(os.Args) < 3 {
		fmt.Println("用法: go run main.go <namespace> <statefulset-name>")
		os.Exit(1)
	}
	namespace := os.Args[1]
	stsName := os.Args[2]

	// 加载 kubeconfig
	userHomeDir, _ := os.UserHomeDir()
	kubeconfigPath := filepath.Join(userHomeDir, ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		log.Fatalf("加载 kubeconfig 失败: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("创建 clientset 失败: %v", err)
	}

	// 获取 StatefulSet
	sts, err := clientset.AppsV1().StatefulSets(namespace).Get(context.TODO(), stsName, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("获取 StatefulSet '%s' 失败: %v", stsName, err)
	}

	fmt.Printf("--- StatefulSet: %s ---\n", sts.Name)
	fmt.Printf("  - Replicas: %d/%d\n", sts.Status.ReadyReplicas, *sts.Spec.Replicas)
	fmt.Printf("  - ServiceName: %s\n", sts.Spec.ServiceName)
	fmt.Println("\n--- Managed Pods and PVCs ---")

	// 根据 StatefulSet 的 selector 查找关联的 Pods
	selector := labels.Set(sts.Spec.Selector.MatchLabels).AsSelector()
	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		log.Fatalf("列出 Pods 失败: %v", err)
	}

	for _, pod := range podList.Items {
		fmt.Printf("  - Pod: %s (IP: %s)\n", pod.Name, pod.Status.PodIP)
		// 查找与 Pod 关联的 PVC
		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				pvcName := vol.PersistentVolumeClaim.ClaimName
				fmt.Printf("    - PVC: %s\n", pvcName)
			}
		}
	}
}
```

**3. 运行**
```bash
# 假设 StatefulSet 'redis' 在 'default' 命名空间
go run main.go default redis

# 预期输出
# --- StatefulSet: redis ---
#   - Replicas: 3/3
#   - ServiceName: redis-headless
#
# --- Managed Pods and PVCs ---
#   - Pod: redis-0 (IP: 10.244.2.4)
#     - PVC: data-redis-0
#   - Pod: redis-1 (IP: 10.244.3.3)
#     - PVC: data-redis-1
#   - Pod: redis-2 (IP: 10.244.1.5)
#     - PVC: data-redis-2
```

## 🔍 故障排查与优化
- **问题**: StatefulSet 的 Pod 创建卡在 `pending` 状态。
  - **排查**:
    1. `kubectl describe pod <pod-name>`: 查看事件。
    2. 很大可能是 PVC 无法绑定。参考 Day 1 的 PVC `Pending` 状态排查方法。
    3. Headless Service 是否已创建并且 `selector` 正确？
- **问题**: Pod 状态为 `CrashLoopBackOff`。
  - **排查**: `kubectl logs <pod-name>` 查看应用日志，通常是应用自身配置问题。
- **优化**: 对于需要大量磁盘 I/O 的应用，选择高性能的 StorageClass (如基于本地 SSD 的) 至关重要。

## 📝 实战项目
- 尝试部署一个比 Redis 更复杂的有状态应用集群，例如 Zookeeper 或 etcd。它们对节点的启动顺序和网络发现有更严格的要求，是练习 StatefulSet 的绝佳案例。

## 🏠 课后作业
1.  **研究 Headless Service**: 删除我们创建的 `redis-headless` Service，然后删除 `redis-2` Pod。观察会发生什么？（提示: StatefulSet 将无法重建 `redis-2`，因为它依赖 Headless Service 来提供网络标识）。
2.  **研究 Pod 管理策略 (`podManagementPolicy`)**: StatefulSet 默认的策略是 `OrderedReady`。阅读文档，了解并测试 `Parallel` 策略，观察 Pod 的创建行为有何不同。
