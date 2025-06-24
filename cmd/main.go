package main

import (
	"gin-chat-room/config"
	"gin-chat-room/internal/database"
	"gin-chat-room/internal/handlers"
	"gin-chat-room/internal/middleware"
	"gin-chat-room/internal/services"
	"gin-chat-room/pkg/logger"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	config.LoadConfig()

	// 初始化日志
	logger.InitLogger()

	// 初始化数据库
	if err := database.InitDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// 初始化 Redis（可选）
	if err := services.InitRedis(); err != nil {
		log.Printf("Warning: Failed to initialize Redis: %v", err)
		log.Println("Redis features will be disabled")
	}

	// 初始化 WebSocket Hub
	hub := services.NewHub()
	go hub.Run()

	// 设置 Gin 模式
	gin.SetMode(config.AppConfig.Server.Mode)

	// 创建路由
	router := gin.Default()

	// 配置 CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// 静态文件服务
	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")

	// 首页路由
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "聊天室",
		})
	})

	// API 路由组
	api := router.Group("/api/v1")
	{
		// 认证相关路由
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
		}

		// 需要认证的路由
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			// 用户相关
			protected.GET("/profile", handlers.GetProfile)
			protected.PUT("/profile", handlers.UpdateProfile)

			// 聊天室相关
			protected.GET("/rooms", handlers.GetRooms)
			protected.POST("/rooms", handlers.CreateRoom)
			protected.GET("/rooms/:id", handlers.GetRoom)
			protected.POST("/rooms/:id/join", handlers.JoinRoom)
			protected.POST("/rooms/:id/leave", handlers.LeaveRoom)

			// 消息相关
			protected.GET("/rooms/:id/messages", handlers.GetMessages)
		}

		// WebSocket 连接
		api.GET("/ws", middleware.AuthMiddleware(), handlers.HandleWebSocket(hub))
	}

	// 启动服务器
	log.Printf("Server starting on port %s", config.AppConfig.Server.Port)
	if err := router.Run(":" + config.AppConfig.Server.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
