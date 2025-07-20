package model

import "time"

// KeyVersion 代表一个物理密钥版本
type KeyVersion struct {
	ID                   uint     `gorm:"primaryKey"`
	KeyID                string   `gorm:"type:varchar(36);index"` // 外键，关联到 Key.ID
	Version              int      `gorm:"not null"`
	State                KeyState `gorm:"type:varchar(20)"`
	EncryptedKeyMaterial []byte   `gorm:"type:blob"` // 加密后的密钥材料
	CreatedAt            time.Time
}
