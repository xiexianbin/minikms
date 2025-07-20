package config

import (
	"errors"
	"os"
)

type Config struct {
	DSN       string
	MasterKey []byte
}

// LoadConfig 从环境变量加载配置
func LoadConfig() (*Config, error) {
	dsn := os.Getenv("DSN")
	if dsn == "" {
		// 为方便演示，默认使用本地 sqlite 文件
		dsn = "kms.db"
	}

	masterKey := os.Getenv("MASTER_KEY")
	if masterKey == "" {
		return nil, errors.New("MASTER_KEY environment variable not set")
	}
	// 主密钥必须是 32 字节 (AES-256)
	if len(masterKey) != 32 {
		return nil, errors.New("MASTER_KEY must be 32 bytes long for AES-256")
	}

	return &Config{
		DSN:       dsn,
		MasterKey: []byte(masterKey),
	}, nil
}
