package api

import "go.xiexianbin.cn/minikms/internal/model"

type KeyResponse struct {
	ID               string         `json:"id"`
	Alias            string         `json:"alias"`
	Description      string         `json:"description"`
	State            model.KeyState `json:"state"`
	CreatedAt        string         `json:"created_at"`
	LatestKeyVersion int            `json:"latest_key_version"`
}

type EncryptResponse struct {
	KeyID        string `json:"key_id"`
	KeyVersion   int    `json:"key_version"`
	EncryptedDEK string `json:"encrypted_dek"` // base64 encoded
	Ciphertext   string `json:"ciphertext"`    // base64 encoded
}

type DecryptResponse struct {
	KeyID     string `json:"key_id"`
	Plaintext string `json:"plaintext"` // base64 encoded
}
