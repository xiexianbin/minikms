package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"go.xiexianbin.cn/minikms/internal/api"
	"go.xiexianbin.cn/minikms/internal/config"
	"go.xiexianbin.cn/minikms/internal/model"
	"go.xiexianbin.cn/minikms/internal/service"
)

func main() {
	// 1. 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 2. 初始化数据库
	db, err := gorm.Open(sqlite.Open(cfg.DSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 自动迁移 schema
	if err := db.AutoMigrate(&model.Key{}, &model.KeyVersion{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 3. 初始化服务
	keyService := service.NewKeyService(db, cfg.MasterKey)
	keyHandler := api.NewKeyHandler(keyService)

	// 4. 设置 Gin 路由器
	r := gin.Default()

	// API V1 路由组
	v1 := r.Group("/v1")
	{
		// 密钥管理
		v1.POST("/keys", keyHandler.CreateKey)
		v1.GET("/keys", keyHandler.ListKeys)
		v1.GET("/keys/:key_id", keyHandler.GetKey)
		v1.POST("/keys/:key_id/disable", keyHandler.DisableKey)
		v1.POST("/keys/:key_id/enable", keyHandler.EnableKey)
		v1.POST("/keys/:key_id/rotate", keyHandler.RotateKey) // 新增路由

		// 加密操作
		v1.POST("/encrypt", keyHandler.Encrypt)
		v1.POST("/decrypt", keyHandler.Decrypt)
	}

	// 5. 启动服务器
	log.Println("Starting MiniKMS server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
