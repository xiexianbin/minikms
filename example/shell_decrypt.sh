#!/bin/bash

# 确保脚本在出错时立即退出
set -e

# 1. 准备你的加密数据 (请替换成你自己的值)
KMS_ENDPOINT="http://localhost:8080/v1/decrypt"
KEY_ID="your-key-id"
KEY_VERSION=1 # 使用加密时返回的版本号
ENCRYPTED_DEK="your_encrypted_dek_base64_string"
CIPHERTEXT="your_ciphertext_base64_string"

echo "🚀 正在向 MiniKMS 发送解密请求..."

# 2. 使用 curl 发送请求，通过 jq 解析响应，并提取 base64 编码的明文
# -s: 静默模式，不显示进度条
# -X POST: 指定请求方法
# -H: 设置请求头
# -d: 设置请求体
BASE64_PLAINTEXT=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "key_id": "'"$KEY_ID"'",
    "key_version": '$KEY_VERSION',
    "encrypted_dek": "'"$ENCRYPTED_DEK"'",
    "ciphertext": "'"$CIPHERTEXT"'"
  }' \
  "$KMS_ENDPOINT" | jq -r '.plaintext')

# 3. 检查是否成功获取到 base64 明文
if [ -z "$BASE64_PLAINTEXT" ] || [ "$BASE64_PLAINTEXT" == "null" ]; then
  echo "❌ 解密失败! 无法从 KMS 响应中获取明文。"
  exit 1
fi

# 4. Base64 解码最终的明文
# 注意：在 macOS 上使用 `base64 -d`，在 Linux 上使用 `base64 -d` 或 `base64 --decode`
PLAINTEXT=$(echo "$BASE64_PLAINTEXT" | base64 --decode)

# 5. 打印结果 ✨
echo "✅ 解密成功!"
echo "解密后的明文: $PLAINTEXT"
