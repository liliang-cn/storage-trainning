# 第 0 周：Kubernetes 核心基础

## 整体目标
- **技能目标**: 掌握 Kubernetes 的核心概念，理解其架构和基本工作原理。
- **实践目标**: 能够使用 `kubectl` 命令行工具与集群进行交互，管理应用生命周期。
- **核心概念**: 深入理解 Pod, Deployment, Service, ConfigMap, Secret 等核心资源对象。
- **Go编程目标**: 初步接触 `client-go`，能够编写简单的 Go 程序连接 K8s 集群并列出资源。
- **架构目标**: 理解容器编排思想，为后续学习 K8s 存储、网络和高级特性打下坚实基础。
- **运维能力**: 能够查看 K8s 资源状态、日志，进行简单的故障诊断。

---

## Day 1: Kubernetes 核心架构与基本概念

### 🎯 学习目标
- 理解容器编排的必要性以及 Kubernetes 的历史。
- 掌握 K8s 的核心架构：Master (Control Plane) 和 Worker 节点及其组件。
- 理解最核心的资源对象：Cluster, Node, Pod。
- 熟练使用 `kubectl` 进行基本的集群信息查看和 Pod 操作。

### 📚 理论学习
1.  **K8s 简介**: 什么是 Kubernetes？它解决了什么问题？
2.  **控制平面组件 (Control Plane Components)**:
    - `kube-apiserver`: 集群的统一入口，所有操作的唯一前端。
    - `etcd`: 可靠的键值存储，保存了整个集群的状态。
    - `kube-scheduler`: 负责 Pod 的调度，决定将 Pod 放在哪个 Node 上运行。
    - `kube-controller-manager`: 负责维护集群的状态，例如故障检测、自动扩展等。
3.  **工作节点组件 (Worker Node Components)**:
    - `kubelet`: 在每个节点上运行的代理，负责管理 Pod 和容器的生命周期。
    - `kube-proxy`: 负责为 Service 提供网络代理和负载均衡。
    - `Container Runtime`: 容器运行时，如 Docker, containerd。
4.  **核心对象**:
    - `Pod`: K8s 中最小的部署单元，可以包含一个或多个容器。

### 🛠️ 实践操作
- 安装 `minikube` 或 `kind` 搭建本地 K8s 环境。
- 使用 `kubectl cluster-info` 和 `kubectl get nodes` 查看集群状态。
- 编写一个最简单的 Pod YAML 文件，并使用 `kubectl apply` 创建它。
- 使用 `kubectl get pods`, `kubectl describe pod`, `kubectl logs` 查看 Pod 状态。
- 使用 `kubectl exec` 进入 Pod 内部执行命令。
- 使用 `kubectl delete` 删除 Pod。

### 🏠 作业
- 研究 Pod 的生命周期（Pending, Running, Succeeded, Failed, Unknown）。
- 尝试在一个 Pod 中运行两个容器，并理解它���如何共享网络和存储。

---

## Day 2: 无状态应用管理 (Deployment, ReplicaSet)

### 🎯 学习目标
- 理解声明式 API 和控制器模式。
- 掌握 `Deployment` 和 `ReplicaSet` 的作用，以及它们之间的关系。
- 能够部署、更新和回滚一个无状态应用。

### 📚 理论学习
1.  **控制器模式**: K8s 如何通过控制器将集群的当前状态调整为期望状态。
2.  **ReplicaSet**: 确保指定数量的 Pod 副本在任何时候都处于运行状态。
3.  **Deployment**: 为 Pod 和 ReplicaSet 提供了一个声明式的、更高级的管理接口。支持滚动更新和回滚。

### 🛠️ 实践操作
- 编写一个 `Deployment` YAML 文件来部署一个 Nginx 应用。
- 使用 `kubectl get deployment`, `kubectl get rs`, `kubectl get pods` 观察创建的资源。
- 使用 `kubectl scale deployment` 对应用进行扩容和缩容。
- 修改 Deployment 的容器镜像版本，触发一次滚动更新 (`rolling update`)。
- 使用 `kubectl rollout status` 和 `kubectl rollout history` 查看更新状态和历史。
- 使用 `kubectl rollout undo` 将应用回滚到上一个版本。

### 🏠 作业
- 研究 Deployment 的不同更新策略 (`RollingUpdate` 和 `Recreate`)。
- 尝试设置 `maxSurge` 和 `maxUnavailable` 参数，观察滚动更新的行为变化��

---

## Day 3: K8s 服务与网络

### 🎯 学习目标
- 理解 Pod IP 的非持久性问题以及 `Service` 的作用。
- 掌握 `Service` 的几种类型：`ClusterIP`, `NodePort`, `LoadBalancer`。
- 了解 K8s 内部的 DNS 解析机制。

### 📚 理论学习
1.  **K8s 网络模型**: 每个 Pod 都有自己独立的 IP 地址，并且所有 Pod 都在一个可以直接连通的扁平网络空间中。
2.  **Service**: 为一组功能相同的 Pod 提供一个统一的、稳定的访问入口和负载均衡。
3.  **Service 类型**:
    - `ClusterIP`: (默认) 为 Service 分配一个集群内部的虚拟 IP，只能在集群内部访问。
    - `NodePort`: 在每个节点的同一个端口上暴露服务，可以通过 `<NodeIP>:<NodePort>` 从集群外部访问。
    - `LoadBalancer`: (云环境) 使用云厂商提供的负载均衡器来对外暴露服务。
4.  **服务发现**: K8s 如何通过 DNS 将 Service 名称解析到其虚拟 IP。

### 🛠️ 实践操作
- 为前一天创建的 Nginx Deployment 创建一个 `ClusterIP` 类型的 Service。
- 启动一个临时的 Pod，尝试通过 Service 的名称 (`http://<service-name>`) 访问 Nginx。
- 将 Service 类型改为 `NodePort`，并尝试从本地机器通过 `http://<NodeIP>:<NodePort>` 访问 Nginx。
- 使用 `kubectl get svc` 和 `kubectl describe svc` 查看 Service 的信息和 Endpoints。

### 🏠 作业
- 研究 `Endpoint` 对象是如何与 Service 和 Pod 关联的。
- （选做）如果环境允许，了解并尝试配置 `Ingress` 资源，实现基于 HTTP 路径的路由。

---

## Day 4: 配置与密钥管理 (ConfigMap, Secret)

### 🎯 学习目标
- 理解将配置与应用镜像解耦的重要性。
- 掌握使用 `ConfigMap` 来管理非敏感配置数据。
- 掌握使用 `Secret` 来管理敏感数据（如密码、API 密钥）。
- 学会以环境变量和文件挂载两种方式将它们注入到 Pod 中。

### 📚 理论学习
1.  **ConfigMap**: 用于存储键值对形式的非敏感配置数据。
2.  **Secret**: 用于存储敏感数据，数据会以 Base64 编码的形式存储在 etcd 中。
3.  **注入方式**:
    - **环境变量**: 将 ConfigMap 或 Secret 的键值作为环境变量注入容器。
    - **卷挂载**: 将 ConfigMap 或 Secret 作为文件挂载到 Pod 的指定路径下。

### 🛠️ 实践操作
- 使用 `kubectl create configmap` 从文件或字面值创建 ConfigMap。
- 编写一个新的 Pod，通过 `env` 和 `envFrom` 字段将 ConfigMap 的值作为环境变量注入。
- 编写另一个 Pod，通过 `volumes` 和 `volumeMounts` 将 ConfigMap 挂载为��件，并验证文件内容。
- 使用 `kubectl create secret generic` 创建 Secret。
- 实践将 Secret 以环境变量和文件挂载的方式注入 Pod。

### 🏠 作业
- 比较通过环境变量和文件挂载注入配置的优缺点。
- 研究 `Secret` 的不同类型（如 `kubernetes.io/dockerconfigjson`）。

---

## Day 5: 应用健康检查与资源限制

### 🎯 学习目标
- 理解为应用设置健康检查的必要性。
- 掌握配置 `Liveness Probe` (存活探针) 和 `Readiness Probe` (就绪探针)。
- 理解为容器设置资源请求 (`requests`) 和限制 (`limits`) 的重要性。

### 📚 理论学习
1.  **健康探针 (Probes)**:
    - `Liveness Probe`: 用于判断容器是否还在运行。如果探测失败，`kubelet` 会杀死并重启容器。
    - `Readiness Probe`: 用于判断容器是否准备好接收流量。如果探测失败，`kubelet` 会将该 Pod 从 Service 的 Endpoints 中移除。
    - `Startup Probe`: 用于判断容器内的应用是否已经启动成功，适用于启动时间较长的应用。
2.  **探针类型**: `exec` (执行命令), `httpGet` (HTTP 请求), `tcpSocket` (TCP 端口检查)。
3.  **资源管理**:
    - `requests`: K8s 调度时保证 Pod 能获得的最小资源量。
    - `limits`: 容器能使用的资源上限，超出限制可能会被终���。
4.  **QoS 等级**: `Guaranteed`, `Burstable`, `BestEffort`。

### 🛠️ 实践操作
- 在 Nginx Deployment 中增加一个 `httpGet` 类型的 `livenessProbe` 和 `readinessProbe`。
- 故意让健康检查失败（例如，进入 Pod 删除首页文件），观察 K8s 的行为。
- 为容器设置合理的 `requests` 和 `limits` (CPU 和 Memory)。
- 使用 `kubectl describe node` 查看节点上的资源分配情况。

### 🏠 作业
- 思考在什么场景下应该只设置 `livenessProbe`，什么场景下两者都需要？
- 研究 `Burstable` QoS 等级是如何工作的。
