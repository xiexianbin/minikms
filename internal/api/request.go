package api

type CreateKeyRequest struct {
	Alias       string `json:"alias" binding:"required"`
	Description string `json:"description"`
}

type EncryptRequest struct {
	KeyID     string `json:"key_id" binding:"required"`
	Plaintext string `json:"plaintext" binding:"required"` // base64 encoded
}

type DecryptRequest struct {
	KeyID        string `json:"key_id" binding:"required"`
	KeyVersion   int    `json:"key_version" binding:"required,min=1"`
	EncryptedDEK string `json:"encrypted_dek" binding:"required"` // base64 encoded
	Ciphertext   string `json:"ciphertext" binding:"required"`    // base64 encoded
}
