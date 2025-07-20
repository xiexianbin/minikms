import requests
import base64
import json

# 1. 准备你的加密数据 (请替换成你自己的值)
kms_endpoint = "http://localhost:8080/v1/decrypt"
payload = {
    "key_id": "your-key-id",
    "key_version": 1,  # 使用加密时返回的版本号
    "encrypted_dek": "your_encrypted_dek_base64_string",
    "ciphertext": "your_ciphertext_base64_string",
}

try:
    # 2. 发送 HTTP POST 请求
    # requests 库会自动处理 JSON 序列化和 Content-Type header
    response = requests.post(kms_endpoint, json=payload)

    # 3. 检查 HTTP 错误 (例如 4xx 或 5xx)
    response.raise_for_status()

    # 4. 解析 JSON 响应
    response_data = response.json()
    base64_plaintext = response_data.get("plaintext")

    if not base64_plaintext:
        print("❌ 解密失败: 响应中未找到 'plaintext' 字段。")
        exit(1)

    # 5. Base64 解码最终的明文
    plaintext_bytes = base64.b64decode(base64_plaintext)
    plaintext = plaintext_bytes.decode('utf-8')

    # 6. 打印结果 ✨
    print("✅ 解密成功!")
    print(f"解密后的明文: {plaintext}")

except requests.exceptions.RequestException as e:
    print(f"❌ 请求失败: {e}")
except (KeyError, TypeError) as e:
    print(f"❌ 解析响应失败: {e}")
