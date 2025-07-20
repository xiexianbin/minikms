# MiniKMS / 迷你密钥管理服务

密钥管理服务（KMS）的核心职责是安全地生成、存储、管理和使用加密密钥。本项目实现一个“迷你”版本的 KMS 应该实现其最核心的功能，同时保持架构的简洁和可扩展性。

## 核心功能需求

1.  **主密钥管理 (Master Key Management):**
      * 创建新的主密钥（在 KMS 中通常称为客户主密钥 - Customer Master Key, CMK，我们这里称之为 `Key`）。每个 `Key` 都有唯一的 ID。
      * 查询密钥列表和单个密钥的元数据（如 ID, 别名, 创建时间, 状态等）。
      * 管理密钥生命周期：启用（Enable）、禁用（Disable）密钥。
2.  **加密与解密 (Cryptographic Operations):**
      * 提供 `/encrypt` 接口，接收`密钥ID`和`明文`，返回`密文`。
      * 提供 `/decrypt` 接口，接收`密钥ID`和`密文`，返回`明文`。
3.  **安全性 (Security):**
      * 主密钥本身**绝不能**以明文形式存储在数据库中。我们需要一个更高层次的密钥（我们称之为“根密钥”或 `MasterEncryptionKey`）来加密所有存储在数据库中的主密钥。这个根密钥不存入数据库，而是通过环境变量等方式在应用启动时加载。
      * API 应该处理二进制数据，使用 Base64 编码在 JSON 中传输是一种标准做法。

## 技术选型

* **语言:** Golang - 高性能、并发友好、静态类型，非常适合构建后端服务。
* **Web 框架:** Gin - 轻量级、高性能的 Web 框架，拥有强大的路由和中间件生态。
* **ORM:** Gorm - 功能齐全的 ORM 库，简化与数据库的交互。
* **数据库:** SQLite - 为了简化部署和演示，我们使用 SQLite。在生产环境中可以轻松替换为 MySQL 或 PostgreSQL。
* **加密算法:** AES-GCM - 带有关联数据的认证加密（AEAD）模式。它同时提供了加密（保密性）和认证（完整性），是现代加密应用的首选。

## 核心概念

### 信封加密 (Envelope Encryption)

信封加密是一个专业 KMS 的核心工作模式，我们的 MiniKMS 也将采用它，这对理解整个架构至关重要。

1.  **主密钥 (Key / CMK):** 存储在 KMS 内部，受根密钥保护。它本身不直接用于加密大量数据。
2.  **数据密钥 (Data Key / DEK):** 临时生成的、用于加密实际业务数据的密钥。
3.  **加密过程 (`/encrypt`):**
    a. 用户请求使用某个 `Key` (通过 `KeyID`) 加密`明文数据 (Plaintext)`。
    b. MiniKMS 内部生成一个全新的、一次性的**数据密钥 (DEK)**。
    c. 使用这个 DEK 加密用户的`明文数据`，得到`密文数据 (Ciphertext)`。
    d. 使用用户指定的 `Key` 来加密这个 DEK，得到**加密后的数据密钥 (Encrypted DEK)**。
    e. 将 `Ciphertext` 和 `Encrypted DEK` 一同返回给用户。用户需要自己存储这两部分。
4.  **解密过程 (`/decrypt`):**
    a. 用户提供 `Ciphertext` 和 `Encrypted DEK`。
    b. MiniKMS 使用 `KeyID` 找到对应的主密钥 `Key`。
    c. 使用 `Key` 解密 `Encrypted DEK`，得到明文的 DEK。
    d. 使用明文 DEK 解密 `Ciphertext`，得到原始的`明文数据 (Plaintext)`。
    e. 将 `Plaintext` 返回给用户。
5.  **原因:**
    a. **安全性:** 主密钥 `Key` 永远不离开 KMS，降低了泄露风险。
    b. **性能:** 对大量数据进行加解密时，可以直接使用 DEK 在应用层完成，无需每次都通过网络请求 KMS，性能更高。
    c. **密钥轮换:** 当主密钥 `Key` 需要轮换时，只需重新加密所有它加密过的 `Encrypted DEK` 即可，而无需重新加密海量的业务数据。

## 密钥轮换设计与分析

密钥轮换是指定期更换密钥的做法，是密码学安全的核心最佳实践。它能有效限制单个密钥被泄露时所造成的损害范围（即“爆炸半径”）。

### 设计思路：版本化管理

我们不能简单地用新密钥替换旧密钥，因为这样做会导致之前用旧密钥加密的数据无法解密。正确的做法是**对密钥进行版本化**。

1.  **逻辑密钥 (Logical Key) vs 物理密钥版本 (Physical Key Version):**

      * 我们之前创建的 `Key` (如 `alias: my-first-key`) 应该被视为一个**逻辑密钥**。它是一个抽象的实体。
      * 每次轮换时，我们为这个逻辑密钥创建一个新的**物理密钥版本**。每个版本都有自己独立的、加密过的密钥材料。

2.  **数据库模型变更:**

      * `keys` 表：现在只存储逻辑密钥的元数据，如别名、描述、状态，以及一个指向**最新版本号**的指针。
      * 新建 `key_versions` 表：存储每个物理版本的具体信息，包括版本号、状态和加密后的密钥材料。

3.  **加解密流程变更:**

      * **加密 (`/encrypt`):**
          * 总是使用逻辑密钥的**最新版本**进行加密。
          * 加密操作的响应中必须包含所使用的**密钥版本号**，以便将来解密时使用。
      * **解密 (`/decrypt`):**
          * 请求中必须提供当初加密时所使用的**密钥版本号**。
          * 服务根据 `key_id` 和 `key_version` 找到唯一的物理密钥版本，用它来解密数据。

4.  **新的 API:**

      * `POST /v1/keys/:key_id/rotate`: 触发指定逻辑密钥的轮换操作。

### 轮换操作流程 (`RotateKey`):

1.  找到 `key_id` 对应的逻辑密钥。
2.  获取其当前的 `LatestVersion`。
3.  生成一份新的密钥材料，并用根密钥 `MASTER_KEY` 加密。
4.  在 `key_versions` 表中插入一条新记录，`Version` 号为 `LatestVersion + 1`。
5.  更新 `keys` 表，将其 `LatestVersion` 字段加一。
6.  **关键点:** 第 4 步和第 5 步必须在一个数据库**事务**中完成，以保证数据的一致性。

## 项目架构 (Project Architecture)

我们将采用经典的分层架构，将职责清晰地分离。

### 目录结构

```
minikms/
├── cmd/
│   └── main.go                 # 程序入口, 初始化和路由设置
├── internal/
│   ├── api/                    # API 层 (Gin Handlers)
│   │   ├── handler.go          # 封装所有 API 逻辑
│   │   ├── request.go          # 定义请求体结构
│   │   └── response.go         # 定义响应体结构
│   ├── config/
│   │   └── config.go           # 配置加载 (环境变量)
│   ├── core/
│   │   └── crypto.go           # 核心加密/解密逻辑
│   ├── model/
│   │   └── key.go              # GORM 数据库模型
│   └── service/
│       └── key_service.go      # 业务逻辑层, 协调 core 和 model
├── go.mod
├── go.sum
└── README.md
```

说明：
* `cmd/main.go`: 项目的启动器。负责加载配置、初始化数据库连接、实例化 Service 和 Handler，并设置 Gin 路由。
* `config/config.go`: 从环境变量中读取 `DSN` (数据库连接字符串) 和 `MASTER_KEY` (根密钥)。
* `model/key.go`: 定义 `Key` 结构体，对应数据库中的 `keys` 表。
* `core/crypto.go`: 实现底层的、纯粹的加密算法（如 AES-GCM 加解密），它不关心业务逻辑。
* `service/key_service.go`: 业务逻辑的核心。它处理如何创建密钥、如何执行信封加密等。它会调用 `core` 层来执行实际的加密操作，并调用 `model` 层来与数据库交互。
* `api/*.go`: Gin 的 Handler 层。负责解析 HTTP 请求、验证输入、调用 `service` 层处理业务，并将结果格式化为 JSON 返回给客户端。它不应包含任何业务逻辑。

## 如何运行项目 (How to Run the Project)

1.  **设置根密钥环境变量**
    这是最重要的一步。根密钥必须是一个 32 字节（256位）的字符串。

    在 Linux/macOS:

    ```bash
    # 生成一个随机的 32 字节密钥
    # 注意：请务必保存好你生成的这个密钥，如果丢失，所有数据都将无法解密！
    export MASTER_KEY=$(openssl rand -base64 32 | head -c 32)
    echo "Your master key is: $MASTER_KEY"
    ```

    在 Windows (PowerShell):

    ```powershell
    # 生成一个随机的 32 字节密钥
    $bytes = New-Object byte[] 32
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    $rng.GetBytes($bytes)
    $masterKey = [System.Text.Encoding]::Default.GetString($bytes)
    $env:MASTER_KEY = $masterKey
    Write-Output "Your master key is: $env:MASTER_KEY"
    ```

    **警告**: 请妥善保管此 `MASTER_KEY`。在生产环境中，应使用更安全的机制（如 HashiCorp Vault, AWS KMS/Secrets Manager, GCP Secret Manager）来注入此密钥。

2.  **运行程序**
    在 `minikms` 目录下执行：

    ```bash
    go run ./cmd/main.go
    ```

    你将看到服务在 `localhost:8080` 上启动。同时，目录下会生成一个 `kms.db` 的 SQLite 文件。

## API 使用示例 (API Usage Examples)

使用 `curl` 和 `jq` (一个 JSON 处理工具) 来测试 API。

1.  **创建新密钥**

    ```bash
    curl -X POST http://localhost:8080/v1/keys \
    -H "Content-Type: application/json" \
    -d '{
        "alias": "my-first-key",
        "description": "Key for general purpose encryption"
    }' | jq
    ```

    **记下返回的 `id`**，我们后面会用到。假设 ID 是 `your_key_id`。

2.  **查看密钥详情**

    ```bash
    curl http://localhost:8080/v1/keys/your_key_id | jq
    ```

    你会看到 `"latest_version": 1`。

3.  **使用版本 1 加密数据**

    明文是 "hello world"。首先用 base64 编码。

    ```bash
    PLAINTEXT_B64=$(echo -n "hello world" | base64)
    KEY_ID="your_key_id"

    curl -X POST http://localhost:8080/v1/encrypt \
    -d '{"key_id": "'$KEY_ID'", "plaintext": "'$PLAINTEXT_B64'"}' \
    -H "Content-Type: application/json" | jq
    ```

    **注意观察响应**:

      * `"key_version": 1`。
      * 记下 `encrypted_dek` 和 `ciphertext` 的值，我们称之为 `DEK_V1` 和 `CIPHER_V1`。

4.  **解密数据**
    使用上一步返回的结果进行解密。

    ```bash
    # 从上一步的响应中复制这两个值
    ENCRYPTED_DEK_B64="..._base64_string_of_encrypted_dek_..."
    CIPHERTEXT_B64="..._base64_string_of_ciphertext_..."
    KEY_ID="your_key_id"

    curl -X POST http://localhost:8080/v1/decrypt \
    -H "Content-Type: application/json" \
    -d '{
        "key_id": "'$KEY_ID'",
        "encrypted_dek": "'$ENCRYPTED_DEK_B64'",
        "ciphertext": "'$CIPHERTEXT_B64'"
    }' | jq
    ```

    在响应的 `plaintext` 字段，你会看到 `aGVsbG8gd29ybGQ=`，这是 "hello world" 的 base64 编码。你可以用 `echo 'aGVsbG8gd29ybGQ=' | base64 --decode` 来验证。

5.  **列出所有密钥**

    ```bash
    curl http://localhost:8080/v1/keys | jq
    ```

6.  **禁用密钥**

    ```bash
    curl -X POST http://localhost:8080/v1/keys/your_key_id/disable
    ```

    禁用后，再尝试使用此密钥加密将会失败。

7.  **轮换密钥**

    ```bash
    curl -X POST http://localhost:8080/v1/keys/your_key_id/rotate | jq
    ```

    你会收到成功消息。

8.  **再次查看密钥详情**

    ```bash
    curl http://localhost:8080/v1/keys/your_key_id | jq
    ```

    现在 `"latest_version"` 应该变成了 `2`。

9.  **使用新版本 (版本 2) 加密数据**

    ```bash
    PLAINTEXT_B64_NEW=$(echo -n "data after rotation" | base64)

    curl -X POST http://localhost:8080/v1/encrypt \
    -d '{"key_id": "'$your_key_id'", "plaintext": "'$PLAINTEXT_B64_NEW'"}' \
    -H "Content-Type: application/json" | jq
    ```

    **再次观察响应**:

      * `"key_version": 2`。
      * 记下新的 `encrypted_dek` 和 `ciphertext`，我们称之为 `DEK_V2` 和 `CIPHER_V2`。

10.  **解密测试**

      * **解密旧数据 (使用版本 1)**:

        ```bash
        # 使用第 3 步得到的 DEK_V1 和 CIPHER_V1
        curl -X POST http://localhost:8080/v1/decrypt \
        -d '{
            "key_id": "key-abc-123",
            "key_version": 1,
            "encrypted_dek": "DEK_V1...",
            "ciphertext": "CIPHER_V1..."
        }' -H "Content-Type: application/json" | jq
        ```

        你会得到 "data before rotation" 的 base64 编码。

      * **解密新数据 (使用版本 2)**:

        ```bash
        # 使用第 6 步得到的 DEK_V2 和 CIPHER_V2
        curl -X POST http://localhost:8080/v1/decrypt \
        -d '{
            "key_id": "key-abc-123",
            "key_version": 2,
            "encrypted_dek": "DEK_V2...",
            "ciphertext": "CIPHER_V2..."
        }' -H "Content-Type: application/json" | jq
        ```

        你会得到 "data after rotation" 的 base64 编码。

## 安全考量、局限性与未来改进

这个 MiniKMS 是一个很好的起点，但距离生产级系统还有差距。

  * **根密钥安全**: 这是整个系统的安全基石。如前所述，`MASTER_KEY` 的管理是重中之重。绝不能硬编码或提交到代码仓库。
  * **访问控制**: 谁可以创建密钥？谁可以使用哪个密钥进行加密/解密？目前系统没有任何认证和授权。可以引入 JWT、OAuth2 或 API Key 等机制，并实现基于策略的访问控制（IAM）。
  * **审计日志**: 所有的 API 调用，特别是创建、禁用、使用密钥的操作，都应该被详细记录下来，形成审计日志，以便追踪和分析。
  * **高可用与容灾**: 单点的 SQLite 数据库是演示性的。生产环境需要使用如 PostgreSQL/MySQL 的集群，并考虑多地域部署。
  * **更健壮的错误处理**: 目前的错误处理比较粗糙，可以将业务错误（如 Key Not Found）和系统内部错误（如 DB connection failed）区分开，返回更精确的 HTTP 状态码和错误信息。
  * **传输层安全**: 生产环境必须使用 HTTPS (TLS) 来保护 API 通信。
