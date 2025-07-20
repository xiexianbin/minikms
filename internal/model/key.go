package model

import (
	"time"

	"gorm.io/gorm"
)

type KeyState string

const (
	StateEnabled  KeyState = "ENABLED"
	StateDisabled KeyState = "DISABLED"
)

// Key 代表一个逻辑密钥，它可以有多个版本
type Key struct {
	ID            string `gorm:"type:varchar(36);primaryKey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	Alias         string         `gorm:"type:varchar(255);uniqueIndex"`
	Description   string         `gorm:"type:text"`
	State         KeyState       `gorm:"type:varchar(20)"`
	LatestVersion int            `gorm:"default:1"`        // 指向最新的版本号
	Versions      []KeyVersion   `gorm:"foreignKey:KeyID"` // 关联关系
}
