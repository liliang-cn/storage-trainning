# Day 4: 配置与密钥管理 (ConfigMap, Secret)

## 🎯 学习目标
- **技能目标**: 掌握在 Kubernetes 中管理应用配置和敏感数据的核心方法 `ConfigMap` 和 `Secret`。
- **核心概念**: 深刻理解将配置与应用镜像解耦的重要性，以及两种主要的配置注入方式：环境变量和卷挂载。
- **具体成果**:
  - 能够独立地从文件或字面值创建 `ConfigMap` 和 `Secret`。
  - 能够成功地将 `ConfigMap` 的数据作为环境变量注入到容器中。
  - 能够成功地将 `ConfigMap` 和 `Secret` 作为文件挂载到 Pod 的文件系统中。
  - 能够解释 `ConfigMap` 和 `Secret` 的主要区别和适用场景。

## 📚 理论基础 (30%)
### 1. 为什么需要解耦配置？
在软件开发中，一个最佳实践是将**代码**和**配置**分离。如果将数据库地址、API 密钥、功能开关等配置信息硬编码在应用镜像中，会带来很多问题：
- **灵活性差**: 每次修改配置都需要重新构建和发布镜像。
- **复用性低**: 同一个应用镜像无法直接用于开发、测试、生产等不同环境，因为各环境的配置不同。
- **安全性风险**: 将敏感信息（如密码）打包到镜像中，会增���泄露的风险。

Kubernetes 提供了两种核心资源来解决这个问题：`ConfigMap` 用于非敏感配置，`Secret` 用于敏感配置。

### 2. ConfigMap: 管理普通配置
- **定义**: `ConfigMap` 是一个用于存储键值对形式的、非敏感配置数据的 API 对象。它可以存储单个配置项，也可以存储完整的配置文件内容。
- **数据来源**:
  - **字面值 (Literal)**: 直接在命令行或 YAML 中定义键值对。
  - **文件 (File)**: 将一个或多个文件的内容作为 `ConfigMap` 的数据。文件名会成为键 (key)，文件内容会成为值 (value)。
- **大小限制**: `ConfigMap` 的设计目标是存储少量配置数据，其总大小通常被限制在 1MiB 以内。不适合存储大型文件。

### 3. Secret: 管理敏感数据
- **定义**: `Secret` 是一个专门用于存储敏感数据（如密码、OAuth 令牌、SSH 密钥）的 API 对象。
- **与 ConfigMap 的区别**:
  - **自动编码**: `Secret` 中的数据在存储到 `etcd` 之前，会默认进行 **Base64 编码**。**注意：这只是编码，不是加密！** 任何有权限访问 `etcd` 或 API 的人都可以轻松解码。它的主要目的是防止数据以明文形式直接暴露在 YAML 文件或 API 响应中。
  - **额外保护**: Kubernetes 会对 `Secret` 提供一些额外的保护��施，例如默认情况下不将 `Secret` 挂载到临时容器 (`tmpfs`)，以及在未来的版本中可能提供静态加密 (`Encryption at Rest`)。
  - **特定类型**: `Secret` 支持多种类型，用于满足特定场景，例如 `kubernetes.io/dockerconfigjson` 用于存储私有镜像仓库的认证信息。
- **核心原则**: 永远不要将 `Secret` 的 YAML 文件提交到公共的代码仓库中。

### 4. 注入方式：如何让 Pod 使用它们？
将 `ConfigMap` 或 `Secret` 的数据提供给容器主要有两种方式：

#### a. 作为环境变量 (Environment Variables)
- **优点**: 简单直接，大多数应用都支持通过环境变量读取配置。
- **缺点**: 如果注入的环境变量过多，`kubectl describe pod` 的输出会变得非常冗长。更重要的是，**当 `ConfigMap` 或 `Secret` 更新后，已经运行的 Pod 中的环境变量不会自动更新**，必须重启 Pod 才能加载新值。
- **注入方法**:
  - `env`: 逐个地将 `ConfigMap` 或 `Secret` 中的某个键注入为指定的环境变量。
  - `envFrom`: 将 `ConfigMap` 或 `Secret` 中的所有键值对一次性全部注入为环境变量。

#### b. 作为卷挂载 (Volume Mount)
- **优点**:
  - **自动更新**: 这是最关键的优势。当挂载的 `ConfigMap` 或 `Secret` 更新后，Pod 中被挂载的文件内容**会自动地、近乎实时地更新**，无需重启 Pod。这对于需要动态重载配置的应用非常有用。
  - 适合存储完整的配置文件（如 `nginx.conf`, `application.properties`）。
- **缺点**: 应用需要改造以支持从文件系统读取配置，并在文件变更时自动重载。
- **注入方法**: 在 Pod 的 `spec.volumes` 中定义一个 `configMap` 或 `secret` 类型的卷，然后在 `spec.containers.volumeMounts` 中将其挂载到容器的指定路径。

## 🛠️ 实践操作 (50%)
### 1. 创建 ConfigMap
**a. 从字面值创建**
```bash
kubectl create configmap app-config --from-literal=app.color=blue --from-literal=app.environment=development
```
**b. 从文件创建**
先创建一些配置文件：
```bash
echo "user.name=guest" > user.properties
echo "database.url=jdbc:mysql://localhost:3306/mydb" > db.properties
```
创建 ConfigMap:
```bash
kubectl create configmap db-config --from-file=db.properties --from-file=user.properties
```
**c. 查看 ConfigMap**
```bash
kubectl get configmap db-config -o yaml
# apiVersion: v1
# data:
#   db.properties: |
#     database.url=jdbc:mysql://localhost:3306/mydb
#   user.properties: |
#     user.name=guest
# kind: ConfigMap
# ...
```

### 2. 将 ConfigMap 注入为环境变量
创建一个文件 `pod-env-demo.yaml`:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-env-demo
spec:
  containers:
  - name: test-container
    image: busybox
    command: [ "/bin/sh", "-c", "env && sleep 3600" ]
    env: # 逐个注入
      - name: APP_COLOR
        valueFrom:
          configMapKeyRef:
            name: app-config # ConfigMap 名称
            key: app.color   # Key 名称
    envFrom: # 批量注入
      - configMapRef:
          name: db-config # ConfigMap 名称
  restartPolicy: Never
```
部署并查看日志：
```bash
kubectl apply -f pod-env-demo.yaml
kubectl logs pod-env-demo
# ...
# APP_COLOR=blue
# db.properties=database.url=jdbc:mysql://localhost:3306/mydb
# user.properties=user.name=guest
# ...
```

### 3. 将 ConfigMap 挂载为卷
创建一个文件 `pod-volume-demo.yaml`:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: pod-volume-demo
spec:
  containers:
  - name: test-container
    image: busybox
    command: [ "/bin/sh", "-c", "ls -l /etc/config && sleep 3600" ]
    volumeMounts:
    - name: config-volume # 对应下面的 volume 名称
      mountPath: /etc/config # 挂载到容器的路径
  volumes:
  - name: config-volume
    configMap:
      name: db-config # 使用哪个 ConfigMap
```
部署并验证：
```bash
kubectl apply -f pod-volume-demo.yaml
kubectl logs pod-volume-demo
# total 8
# lrwxrwxrwx ... db.properties -> ..data/db.properties
# lrwxrwxrwx ... user.properties -> ..data/user.properties

# 进入 Pod 查看文件内容
kubectl exec -it pod-volume-demo -- cat /etc/config/db.properties
# database.url=jdbc:mysql://localhost:3306/mydb
```

### 4. 创建和使用 Secret
**a. 创建 Secret**
```bash
# Base64 编码是自动完成的
kubectl create secret generic db-secret --from-literal=username=admin --from-literal=password='S3cr3tP@ssw0rd'
```
**b. 查看 Secret**
```bash
# 直接 get 不会显示数据
kubectl get secret db-secret

# 使用 -o yaml 查看，数据是 Base64 编码的
kubectl get secret db-secret -o yaml
# data:
#   password: UzNjcjN0UA==c3cwcmQ=
#   username: YWRtaW4=

# 解码验证
echo 'UzNjcjN0UA==c3cwcmQ=' | base64 --decode
# S3cr3tP@ssw0rd
```
**c. 注入 Secret**
注入 `Secret` 的方式与 `ConfigMap` **完全相同**，只需将 `configMapKeyRef` 替换为 `secretKeyRef`，将 `configMapRef` 替换为 `secretRef`，将 `volumes` 中的 `configMap` 替换为 `secret` 即可。

## 💻 Go 编程实现 (20%)
### 项目: `k8s-config-creator`
**目标**: 编写一个 Go 程序，以编程方式创建一个 `ConfigMap`。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// --- 配置和创建 clientset ---
	userHomeDir, _ := os.UserHomeDir()
	kubeconfig := filepath.Join(userHomeDir, ".kube", "config")
	config, _ := clientcmd.BuildConfigFromFlags("", kubeconfig)
	clientset, _ := kubernetes.NewForConfig(config)

	namespace := "default"
	cmName := "go-created-cm"

	fmt.Printf("在命名空间 '%s' 中创建 ConfigMap '%s'...
", namespace, cmName)

	// 定义 ConfigMap 对象
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"message": "Hello from Go client!",
			"author":  "Gemini",
		},
	}

	// 使用 clientset 创建 ConfigMap
	createdCM, err := clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), configMap, metav1.CreateOptions{})
	if err != nil {
		log.Fatalf("创建 ConfigMap 失败: %v", err)
	}

	fmt.Printf("ConfigMap '%s' 创建成功!
", createdCM.Name)
	fmt.Printf("Data: %v
", createdCM.Data)

	// 清理
	fmt.Println("按回车键删除创建的 ConfigMap...")
	fmt.Scanln()
	clientset.CoreV1().ConfigMaps(namespace).Delete(context.TODO(), cmName, metav1.DeleteOptions{})
	fmt.Println("清理完成。")
}
```

## 🔍 故障排查与优化
- **Pod 状态为 `CreateContainerConfigError`**:
  - `kubectl describe pod <pod-name>` 查看事件。
  - 常见原因：引用的 `ConfigMap` 或 `Secret` 不存在，或者引用的 `key` 在 `ConfigMap`/`Secret` 中不存在。
- **自动更新不生效**:
  - 只有通过**卷挂载**的方式注入，文件内容才会自动更新。环境变量方式不会更新。
  - 某些应用（如 Java 程序）启动时会将配置加载到内存中，即使文件更新了，应用自身也需要有热重载机制才能生效。

## 🏠 课后作业
1.  **比较注入方式**: 总结一下使用环境变量和卷挂载注入配置的优缺点，分别说明它们最适合的应用场景。
2.  **Secret 类型**: 阅读官方文档，研究 `Secret` 的其他类型，特别是 `kubernetes.io/service-account-token` 和 `kubernetes.io/tls`，了解它们的用途。
3.  **卷挂载特定路径**: 实践一下如何将 `ConfigMap` 中的某个特定 `key` 挂载为卷中的一个文件名，而不是将所有 `key` 都作为文件名。（提示: `volumes.configMap.items`）
