package service

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"go.xiexianbin.cn/minikms/internal/core"
	"go.xiexianbin.cn/minikms/internal/model"
)

const (
	// 内部使用 AES-256 作为主密钥和数据密钥
	KeySize = 32
)

// KeyService 封装了与密钥相关的业务逻辑
type KeyService struct {
	db        *gorm.DB
	masterKey []byte // 用来加密数据库中所有密钥的根密钥
}

func NewKeyService(db *gorm.DB, masterKey []byte) *KeyService {
	return &KeyService{
		db:        db,
		masterKey: masterKey,
	}
}

// CreateKey 创建一个新的逻辑密钥及其第一个版本
func (s *KeyService) CreateKey(alias, description string) (*model.Key, error) {
	// 1. 生成一个新的主密钥材料 (明文)
	keyMaterial, err := core.GenerateKey(KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key material: %w", err)
	}

	// 2. 使用根密钥加密这个新的主密钥材料
	encryptedKeyMaterial, err := core.EncryptAESGCM(s.masterKey, keyMaterial)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt key material: %w", err)
	}

	key := &model.Key{
		ID:            uuid.NewString(),
		Alias:         alias,
		Description:   description,
		State:         model.StateEnabled,
		LatestVersion: 1,
	}

	// 3. 构造模型并存入数据库
	keyVersion := &model.KeyVersion{
		KeyID:                key.ID,
		Version:              1,
		State:                model.StateEnabled,
		EncryptedKeyMaterial: encryptedKeyMaterial,
	}

	// 使用事务确保逻辑密钥和它的第一个版本同时创建成功
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(key).Error; err != nil {
			return err
		}
		if err := tx.Create(keyVersion).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to save key and version to database: %w", err)
	}

	return key, nil
}

// findAndDecryptKeyVersion 查找并解密一个指定的物理密钥版本
func (s *KeyService) findAndDecryptKeyVersion(keyID string, version int) ([]byte, error) {
	// 首先检查逻辑密钥的状态
	var key model.Key
	if err := s.db.First(&key, "id = ?", keyID).Error; err != nil {
		return nil, fmt.Errorf("key with id '%s' not found", keyID)
	}
	if key.State != model.StateEnabled {
		return nil, fmt.Errorf("logical key '%s' is not enabled", keyID)
	}

	// 查找指定的物理版本
	var keyVersion model.KeyVersion
	if err := s.db.First(&keyVersion, "key_id = ? AND version = ?", keyID, version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("key version '%d' for key '%s' not found", version, keyID)
		}
		return nil, err
	}
	if keyVersion.State != model.StateEnabled {
		return nil, fmt.Errorf("key version '%d' for key '%s' is not enabled", version, keyID)
	}

	// 解密获得明文密钥材料
	decryptedKeyMaterial, err := core.DecryptAESGCM(s.masterKey, keyVersion.EncryptedKeyMaterial)
	if err != nil {
		return nil, fmt.Errorf("critical: failed to decrypt key material for key '%s' version '%d': %w", keyID, version, err)
	}
	return decryptedKeyMaterial, nil
}

// EncryptData 使用逻辑密钥的最新版本执行信封加密
func (s *KeyService) EncryptData(keyID string, plaintext []byte) (keyVersion int, encryptedDEK, ciphertext []byte, err error) {
	var key model.Key
	if err = s.db.First(&key, "id = ?", keyID).Error; err != nil {
		err = fmt.Errorf("key with id '%s' not found", keyID)
		return
	}

	// 1. 使用最新版本进行加密
	latestVersion := key.LatestVersion
	cmk, err := s.findAndDecryptKeyVersion(keyID, latestVersion)
	if err != nil {
		return
	}

	// 2. 生成一个新的数据密钥 (DEK)
	dek, err := core.GenerateKey(KeySize)
	if err != nil {
		err = fmt.Errorf("failed to generate data key: %w", err)
		return
	}

	// 3. 使用 CMK 加密 DEK
	encryptedDEK, err = core.EncryptAESGCM(cmk, dek)
	if err != nil {
		err = fmt.Errorf("failed to encrypt data key: %w", err)
		return
	}

	// 4. 使用 DEK 加密明文数据
	ciphertext, err = core.EncryptAESGCM(dek, plaintext)
	if err != nil {
		err = fmt.Errorf("failed to encrypt plaintext: %w", err)
		return
	}

	keyVersion = latestVersion
	return
}

// DecryptData 使用指定的密钥版本执行信封解密
func (s *KeyService) DecryptData(keyID string, version int, encryptedDEK, ciphertext []byte) (plaintext []byte, err error) {
	// 1. 使用最新版本进行解密
	cmk, err := s.findAndDecryptKeyVersion(keyID, version)
	if err != nil {
		return nil, err
	}

	// 2. 使用 CMK 解密 Encrypted DEK 得到明文 DEK
	dek, err := core.DecryptAESGCM(cmk, encryptedDEK)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data key: %w", err)
	}

	// 3. 使用 DEK 解密密文数据
	plaintext, err = core.DecryptAESGCM(dek, ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt ciphertext: %w", err)
	}

	return plaintext, nil
}

// RotateKey 轮换密钥：创建新版本并更新逻辑密钥
func (s *KeyService) RotateKey(keyID string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var key model.Key
		// 使用 FOR UPDATE 行锁来防止并发轮换
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&key, "id = ?", keyID).Error; err != nil {
			return fmt.Errorf("key with id '%s' not found", keyID)
		}

		if key.State != model.StateEnabled {
			return fmt.Errorf("cannot rotate a disabled key: %s", keyID)
		}

		newVersionNum := key.LatestVersion + 1

		// 创建新的密钥版本
		keyMaterial, err := core.GenerateKey(KeySize)
		if err != nil {
			return fmt.Errorf("failed to generate new key material: %w", err)
		}
		encryptedKeyMaterial, err := core.EncryptAESGCM(s.masterKey, keyMaterial)
		if err != nil {
			return fmt.Errorf("failed to encrypt new key material: %w", err)
		}

		newKeyVersion := model.KeyVersion{
			KeyID:                keyID,
			Version:              newVersionNum,
			State:                model.StateEnabled,
			EncryptedKeyMaterial: encryptedKeyMaterial,
		}

		if err := tx.Create(&newKeyVersion).Error; err != nil {
			return err
		}

		// 更新逻辑密钥的最新版本号
		key.LatestVersion = newVersionNum
		if err := tx.Save(&key).Error; err != nil {
			return err
		}

		return nil
	})
}

// ListKeys 列出所有密钥
func (s *KeyService) ListKeys() ([]model.Key, error) {
	var keys []model.Key
	if err := s.db.Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

// GetKey 现在可以额外加载版本信息
func (s *KeyService) GetKey(keyID string) (*model.Key, error) {
	var key model.Key
	// Preload 会把关联的 Versions 一起查出来
	err := s.db.Preload("Versions").First(&key, "id = ?", keyID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("key with id '%s' not found", keyID)
		}
		return nil, err
	}
	return &key, nil
}

// UpdateKeyState 更新密钥状态
func (s *KeyService) UpdateKeyState(keyID string, state model.KeyState) error {
	var key model.Key
	if err := s.db.First(&key, "id = ?", keyID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("key with id '%s' not found", keyID)
		}
		return err
	}

	key.State = state
	return s.db.Save(&key).Error
}
