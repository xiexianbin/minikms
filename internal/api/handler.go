package api

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"go.xiexianbin.cn/minikms/internal/model"
	"go.xiexianbin.cn/minikms/internal/service"
)

type KeyHandler struct {
	keyService *service.KeyService
}

func NewKeyHandler(ks *service.KeyService) *KeyHandler {
	return &KeyHandler{keyService: ks}
}

func (h *KeyHandler) CreateKey(c *gin.Context) {
	var req CreateKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key, err := h.keyService.CreateKey(req.Alias, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, KeyResponse{
		ID:          key.ID,
		Alias:       key.Alias,
		Description: key.Description,
		State:       key.State,
		CreatedAt:   key.CreatedAt.Format(time.RFC3339),
	})
}

func (h *KeyHandler) Encrypt(c *gin.Context) {
	var req EncryptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	plaintext, err := base64.StdEncoding.DecodeString(req.Plaintext)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 for plaintext"})
		return
	}

	keyVersion, encryptedDEK, ciphertext, err := h.keyService.EncryptData(req.KeyID, plaintext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, EncryptResponse{
		KeyID:        req.KeyID,
		KeyVersion:   keyVersion,
		EncryptedDEK: base64.StdEncoding.EncodeToString(encryptedDEK),
		Ciphertext:   base64.StdEncoding.EncodeToString(ciphertext),
	})
}

func (h *KeyHandler) Decrypt(c *gin.Context) {
	var req DecryptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	encryptedDEK, err := base64.StdEncoding.DecodeString(req.EncryptedDEK)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 for encrypted_dek"})
		return
	}

	ciphertext, err := base64.StdEncoding.DecodeString(req.Ciphertext)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 for ciphertext"})
		return
	}

	plaintext, err := h.keyService.DecryptData(req.KeyID, req.KeyVersion, encryptedDEK, ciphertext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, DecryptResponse{
		KeyID:     req.KeyID,
		Plaintext: base64.StdEncoding.EncodeToString(plaintext),
	})
}

func (h *KeyHandler) RotateKey(c *gin.Context) {
	keyID := c.Param("key_id")
	if err := h.keyService.RotateKey(keyID); err != nil {
		// 根据错误类型可以返回不同状态码
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Key rotated successfully"})
}

func (h *KeyHandler) ListKeys(c *gin.Context) {
	keys, err := h.keyService.ListKeys()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var resp []KeyResponse
	for _, key := range keys {
		// 注意：列表接口不应返回敏感信息，如 EncryptedKeyMaterial
		resp = append(resp, KeyResponse{
			ID:          key.ID,
			Alias:       key.Alias,
			Description: key.Description,
			State:       key.State,
			CreatedAt:   key.CreatedAt.Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, resp)
}

func (h *KeyHandler) GetKey(c *gin.Context) {
	keyID := c.Param("key_id")
	key, err := h.keyService.GetKey(keyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, KeyResponse{
		ID:               key.ID,
		Alias:            key.Alias,
		Description:      key.Description,
		State:            key.State,
		CreatedAt:        key.CreatedAt.Format(time.RFC3339),
		LatestKeyVersion: key.LatestVersion,
	})
}

func (h *KeyHandler) DisableKey(c *gin.Context) {
	keyID := c.Param("key_id")
	if err := h.keyService.UpdateKeyState(keyID, model.StateDisabled); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *KeyHandler) EnableKey(c *gin.Context) {
	keyID := c.Param("key_id")
	if err := h.keyService.UpdateKeyState(keyID, model.StateEnabled); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
