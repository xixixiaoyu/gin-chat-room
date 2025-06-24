package handlers

import (
	"gin-chat-room/internal/middleware"
	"gin-chat-room/internal/services"
	"gin-chat-room/internal/websocket"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HandleWebSocket 处理 WebSocket 连接
func HandleWebSocket(hub *services.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取当前用户
		userID, exists := middleware.GetCurrentUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User not authenticated",
			})
			return
		}

		// 获取房间ID
		roomIDStr := c.Query("room_id")
		if roomIDStr == "" {
			roomIDStr = "1" // 默认房间
		}

		roomID, err := strconv.ParseUint(roomIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid room ID",
			})
			return
		}

		// 升级到 WebSocket 连接
		conn, err := websocket.NewConnection(c.Writer, c.Request)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		// 创建客户端
		client := &services.Client{
			ID:     uuid.New().String(),
			UserID: userID,
			RoomID: uint(roomID),
			Conn:   conn,
			Send:   make(chan []byte, 256),
			Hub:    hub,
		}

		// 注册客户端
		hub.RegisterClient(client)

		// 启动读写协程
		go conn.WritePump(client)
		go conn.ReadPump(client)
	}
}
