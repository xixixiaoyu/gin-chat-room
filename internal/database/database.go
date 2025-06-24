package database

import (
	"fmt"
	"gin-chat-room/config"
	"gin-chat-room/internal/models"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() error {
	var err error
	var dialector gorm.Dialector

	cfg := config.AppConfig.Database

	switch cfg.Type {
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
			cfg.Host, cfg.Username, cfg.Password, cfg.Database, cfg.Port, cfg.SSLMode)
		dialector = postgres.Open(dsn)
	case "sqlite":
		dialector = sqlite.Open(cfg.Database)
	default:
		return fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	// 配置 GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// 如果是生产环境，关闭详细日志
	if config.AppConfig.Server.Mode == "release" {
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	DB, err = gorm.Open(dialector, gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// 自动迁移数据库表
	if err := AutoMigrate(); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// 创建默认数据
	if err := CreateDefaultData(); err != nil {
		return fmt.Errorf("failed to create default data: %w", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

// AutoMigrate 自动迁移数据库表
func AutoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.Room{},
		&models.RoomMember{},
		&models.Message{},
	)
}

// CreateDefaultData 创建默认数据
func CreateDefaultData() error {
	// 创建默认聊天室
	var count int64
	DB.Model(&models.Room{}).Count(&count)
	if count == 0 {
		// 创建系统用户
		systemUser := &models.User{
			Username: "system",
			Email:    "system@chatroom.com",
			Nickname: "系统",
		}
		systemUser.SetPassword("system123")
		if err := DB.Create(systemUser).Error; err != nil {
			return err
		}

		// 创建默认聊天室
		defaultRoom := &models.Room{
			Name:        "大厅",
			Description: "欢迎来到聊天室大厅！",
			IsPrivate:   false,
			MaxMembers:  1000,
			CreatorID:   systemUser.ID,
		}
		if err := DB.Create(defaultRoom).Error; err != nil {
			return err
		}

		// 创建欢迎消息
		welcomeMessage := &models.Message{
			RoomID:  defaultRoom.ID,
			UserID:  systemUser.ID,
			Type:    models.MessageTypeSystem,
			Content: "欢迎来到聊天室！请遵守聊天规则，友好交流。",
		}
		if err := DB.Create(welcomeMessage).Error; err != nil {
			return err
		}

		log.Println("Default data created successfully")
	}

	return nil
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}
