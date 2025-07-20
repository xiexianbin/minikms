# example

## 准备工作：获取加密数据

在运行任何客户端之前，你需要先从 MiniKMS 服务获取一份加密数据。请确保你的 MiniKMS 服务正在运行。

1.  **创建密钥** (如果还没有的话):

    ```bash
    curl -X POST http://localhost:8080/v1/keys \
    -d '{"alias": "client-test-key"}' \
    -H "Content-Type: application/json"
    ```

    记下返回的 `id` (例如: `"a1b2c3d4-..."`)。

2.  **加密一段数据**:
    我们将使用这段数据来测试所有客户端。

    ```bash
    # 明文 base64 编码
    PLAINTEXT_B64=$(echo -n "Data to be decrypted by clients" | base64)
    KEY_ID="your-key-id" # 替换成你上一步得到的 ID

    # 发送加密请求，并将结果保存到变量中
    RESPONSE=$(curl -s -X POST http://localhost:8080/v1/encrypt \
    -d '{"key_id": "'$KEY_ID'", "plaintext": "'$PLAINTEXT_B64'"}' \
    -H "Content-Type: application/json")

    # 提取需要的数据
    ENCRYPTED_DEK=$(echo $RESPONSE | jq -r .encrypted_dek)
    CIPHERTEXT=$(echo $RESPONSE | jq -r .ciphertext)
    KEY_VERSION=$(echo $RESPONSE | jq -r .key_version)

    # 显示这些值，以便你复制到客户端代码中
    echo "请将以下值用于你的客户端代码中:"
    echo "KEY_ID: $KEY_ID"
    echo "KEY_VERSION: $KEY_VERSION"
    echo "ENCRYPTED_DEK: $ENCRYPTED_DEK"
    echo "CIPHERTEXT: $CIPHERTEXT"
    ```

现在，你可以使用上面生成的 `KEY_ID`, `KEY_VERSION`, `ENCRYPTED_DEK`, 和 `CIPHERTEXT` 来运行下面的客户端了。

## 运行客户端解密数据

### Go 客户端

```bash
# 将加密数据填入 golang_decrypt_client.go 文件，然后运行
go run golang_decrypt_client.go
```

**预期输出:**

```
✅ 解密成功!
解密后的明文: Data to be decrypted by clients
```

### Python 客户端 🐍

**依赖安装:**

```bash
pip install requests
```
**运行 Python 客户端**

```bash
# 将你的加密数据填入 python_decrypt_client.py 文件
# 然后运行
python python_decrypt_client.py
```

**预期输出:**

```
✅ 解密成功!
解密后的明文: Data to be decrypted by clients
```

### Shell 脚本客户端 🐚

该例子使用 `curl` 来发送请求, `jq` 来解析 JSON, 以及 `base64` 来解码。这是自动化和 CI/CD 流程中非常常见的做法。

**依赖安装:**

  * `curl`: 大多数系统自带。
  * `jq`: (一个轻量级的命令行 JSON 处理器)
      * `sudo apt-get install jq` (Debian/Ubuntu)
      * `sudo yum install jq` (CentOS/RHEL)
      * `brew install jq` (macOS)

**运行 Shell 客户端**

```bash
# 将你的加密数据填入 shell_decrypt.sh 文件
# 赋予脚本执行权限
chmod +x shell_decrypt.sh
# 运行脚本
./shell_decrypt.sh
```

**预期输出:**

```
🚀 正在向 MiniKMS 发送解密请求...
✅ 解密成功!
解密后的明文: Data to be decrypted by clients
```
