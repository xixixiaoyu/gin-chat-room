package websocket

import (
	"encoding/json"
	"gin-chat-room/internal/database"
	"gin-chat-room/internal/models"
	"gin-chat-room/internal/services"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// 写入等待时间
	writeWait = 10 * time.Second

	// Pong 等待时间
	pongWait = 60 * time.Second

	// Ping 发送周期，必须小于 pongWait
	pingPeriod = (pongWait * 9) / 10

	// 最大消息大小
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 在生产环境中应该检查 Origin
		return true
	},
}

// Connection WebSocket 连接包装
type Connection struct {
	ws   *websocket.Conn
	send chan []byte
}

// WebSocketConnection WebSocket 连接接口
type WebSocketConnection interface {
	ReadPump(client *services.Client)
	WritePump(client *services.Client)
	Close() error
}

// NewConnection 创建新的 WebSocket 连接
func NewConnection(w http.ResponseWriter, r *http.Request) (*Connection, error) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	conn := &Connection{
		ws:   ws,
		send: make(chan []byte, 256),
	}

	return conn, nil
}

// ReadPump 处理从 WebSocket 连接读取消息
func (c *Connection) ReadPump(client *services.Client) {
	defer func() {
		client.Hub.UnregisterClient(client)
		c.ws.Close()
	}()

	c.ws.SetReadLimit(maxMessageSize)
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, messageData, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// 解析消息
		var wsMessage services.WebSocketMessage
		if err := json.Unmarshal(messageData, &wsMessage); err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		// 处理消息
		c.handleMessage(client, &wsMessage)
	}
}

// WritePump 处理向 WebSocket 连接写入消息
func (c *Connection) WritePump(client *services.Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.ws.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 添加排队的消息到当前消息
			n := len(client.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Close 关闭连接
func (c *Connection) Close() error {
	return c.ws.Close()
}

// handleMessage 处理接收到的消息
func (c *Connection) handleMessage(client *services.Client, wsMessage *services.WebSocketMessage) {
	switch wsMessage.Type {
	case "message":
		c.handleChatMessage(client, wsMessage)
	case "join_room":
		c.handleJoinRoom(client, wsMessage)
	case "leave_room":
		c.handleLeaveRoom(client, wsMessage)
	default:
		log.Printf("Unknown message type: %s", wsMessage.Type)
	}
}

// handleChatMessage 处理聊天消息
func (c *Connection) handleChatMessage(client *services.Client, wsMessage *services.WebSocketMessage) {
	// 验证用户是否在房间中
	var roomMember models.RoomMember
	if err := database.DB.Where("room_id = ? AND user_id = ?", client.RoomID, client.UserID).First(&roomMember).Error; err != nil {
		log.Printf("User %d is not a member of room %d", client.UserID, client.RoomID)
		return
	}

	// 创建消息记录
	message := models.Message{
		RoomID:  client.RoomID,
		UserID:  client.UserID,
		Type:    models.MessageTypeText,
		Content: wsMessage.Content,
	}

	// 保存到数据库
	if err := database.DB.Create(&message).Error; err != nil {
		log.Printf("Error saving message: %v", err)
		return
	}

	// 预加载用户信息
	database.DB.Preload("User").First(&message, message.ID)

	// 缓存消息到 Redis
	services.CacheMessage(client.RoomID, message.ToJSON())

	// 广播消息
	broadcastMessage := services.WebSocketMessage{
		Type:   "message",
		RoomID: client.RoomID,
		Data:   message.ToJSON(),
	}

	client.Hub.BroadcastMessage(client.RoomID, broadcastMessage)
}

// handleJoinRoom 处理加入房间
func (c *Connection) handleJoinRoom(client *services.Client, wsMessage *services.WebSocketMessage) {
	if wsMessage.RoomID == 0 {
		return
	}

	// 验证房间是否存在
	var room models.Room
	if err := database.DB.First(&room, wsMessage.RoomID).Error; err != nil {
		log.Printf("Room %d not found", wsMessage.RoomID)
		return
	}

	// 检查用户是否已经是房间成员
	var roomMember models.RoomMember
	if err := database.DB.Where("room_id = ? AND user_id = ?", wsMessage.RoomID, client.UserID).First(&roomMember).Error; err != nil {
		// 如果不是成员，添加为成员
		roomMember = models.RoomMember{
			RoomID:   wsMessage.RoomID,
			UserID:   client.UserID,
			Role:     "member",
			JoinedAt: time.Now(),
		}
		database.DB.Create(&roomMember)
	}

	// 更新客户端房间ID
	client.RoomID = wsMessage.RoomID

	// 设置用户在线状态
	services.SetUserOnline(client.UserID, client.RoomID)
}

// handleLeaveRoom 处理离开房间
func (c *Connection) handleLeaveRoom(client *services.Client, wsMessage *services.WebSocketMessage) {
	// 设置用户离线状态
	services.SetUserOffline(client.UserID)

	// 注销客户端
	client.Hub.UnregisterClient(client)
}
