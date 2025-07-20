package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// 定义请求和响应的结构体，与服务器的 API 定义匹配
type DecryptRequest struct {
	KeyID        string `json:"key_id"`
	KeyVersion   int    `json:"key_version"`
	EncryptedDEK string `json:"encrypted_dek"`
	Ciphertext   string `json:"ciphertext"`
}

type DecryptResponse struct {
	KeyID     string `json:"key_id"`
	Plaintext string `json:"plaintext"` // Base64 编码的明文
}

func main() {
	// 1. 准备你的加密数据 (请替换成你自己的值)
	reqData := DecryptRequest{
		KeyID:        "your-key-id",
		KeyVersion:   1, // 使用加密时返回的版本号
		EncryptedDEK: "your_encrypted_dek_base64_string",
		Ciphertext:   "your_ciphertext_base64_string",
	}

	kmsEndpoint := "http://localhost:8080/v1/decrypt"

	// 2. 将请求数据序列化为 JSON
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		log.Fatalf("无法序列化请求: %v", err)
	}

	// 3. 创建 HTTP POST 请求
	request, err := http.NewRequest("POST", kmsEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("无法创建 HTTP 请求: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	// 4. 发送请求
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Fatalf("发送请求失败: %v", err)
	}
	defer response.Body.Close()

	// 5. 读取并检查响应
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("无法读取响应体: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		log.Fatalf("解密失败，状态码: %d, 响应: %s", response.StatusCode, string(body))
	}

	// 6. 将响应 JSON 反序列化到结构体中
	var decryptResp DecryptResponse
	if err := json.Unmarshal(body, &decryptResp); err != nil {
		log.Fatalf("无法解析响应 JSON: %v", err)
	}

	// 7. Base64 解码最终的明文
	plaintext, err := base64.StdEncoding.DecodeString(decryptResp.Plaintext)
	if err != nil {
		log.Fatalf("无法 Base64 解码明文: %v", err)
	}

	// 8. 打印结果 ✨
	fmt.Println("✅ 解密成功!")
	fmt.Printf("解密后的明文: %s\n", string(plaintext))
}
