# 04 - WebSocket 实时通信

## 📋 概述

本章节将详细介绍聊天室应用的 WebSocket 实时通信系统，包括连接管理、消息广播、在线用户管理以及 Redis 集成等核心功能。

## 🎯 学习目标

- 理解 WebSocket 协议和实时通信原理
- 掌握 WebSocket 连接的管理和维护
- 学会实现消息广播和房间管理
- 了解 Redis 在实时系统中的应用

## 🌐 WebSocket 架构设计

### 整体架构图

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Client 1  │    │   Client 2  │    │   Client 3  │
│ (WebSocket) │    │ (WebSocket) │    │ (WebSocket) │
└──────┬──────┘    └──────┬──────┘    └──────┬──────┘
       │                  │                  │
       └──────────────────┼──────────────────┘
                          │
                   ┌──────▼──────┐
                   │     Hub     │
                   │ (连接管理器) │
                   └──────┬──────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
   ┌────▼────┐      ┌────▼────┐      ┌────▼────┐
   │ Room 1  │      │ Room 2  │      │ Room 3  │
   │ 3 users │      │ 2 users │      │ 1 user  │
   └─────────┘      └─────────┘      └─────────┘
```

### 核心组件

1. **Hub**: 中央连接管理器，负责客户端注册/注销和消息分发
2. **Client**: 代表一个 WebSocket 连接的客户端
3. **Room**: 聊天室，管理房间内的用户和消息
4. **Message**: 消息结构，定义不同类型的消息格式

## 🔧 WebSocket 连接管理

### 1. 客户端结构定义

创建 `internal/services/hub.go`：

```go
package services

import (
	"encoding/json"
	"gin-chat-room/internal/database"
	"gin-chat-room/internal/models"
	"log"
	"sync"
	"time"
)

// WebSocketConnection WebSocket 连接接口
type WebSocketConnection interface {
	ReadPump(client *Client)
	WritePump(client *Client)
	Close() error
}

// Client WebSocket 客户端
type Client struct {
	ID     string
	UserID uint
	RoomID uint
	Conn   WebSocketConnection
	Send   chan []byte
	Hub    *Hub
}

// Hub WebSocket 连接管理中心
type Hub struct {
	// 注册的客户端
	clients map[*Client]bool

	// 按房间分组的客户端
	rooms map[uint]map[*Client]bool

	// 按用户分组的客户端
	users map[uint]*Client

	// 注册客户端的通道
	register chan *Client

	// 注销客户端的通道
	unregister chan *Client

	// 广播消息的通道
	broadcast chan *BroadcastMessage

	// 互斥锁
	mutex sync.RWMutex
}

// BroadcastMessage 广播消息结构
type BroadcastMessage struct {
	RoomID  uint        `json:"room_id"`
	Message interface{} `json:"message"`
}

// WebSocketMessage WebSocket 消息结构
type WebSocketMessage struct {
	Type    string      `json:"type"`
	RoomID  uint        `json:"room_id,omitempty"`
	Content string      `json:"content,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// NewHub 创建新的 Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[uint]map[*Client]bool),
		users:      make(map[uint]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage),
	}
}
```

### 2. Hub 运行逻辑

```go
// Run 运行 Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastToRoom(message.RoomID, message.Message)
		}
	}
}

// registerClient 注册客户端
func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// 添加到客户端列表
	h.clients[client] = true

	// 添加到房间
	if h.rooms[client.RoomID] == nil {
		h.rooms[client.RoomID] = make(map[*Client]bool)
	}
	h.rooms[client.RoomID][client] = true

	// 添加到用户映射
	h.users[client.UserID] = client

	// 设置用户在线状态
	SetUserOnline(client.UserID, client.RoomID)

	// 更新数据库中的用户在线状态
	database.DB.Model(&models.User{}).Where("id = ?", client.UserID).Update("is_online", true)

	log.Printf("Client registered: UserID=%d, RoomID=%d", client.UserID, client.RoomID)

	// 通知房间内其他用户有新用户加入
	h.notifyUserJoined(client)

	// 发送在线用户列表给新加入的用户
	h.sendOnlineUsers(client)
}

// unregisterClient 注销客户端
func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, ok := h.clients[client]; ok {
		// 从客户端列表中移除
		delete(h.clients, client)

		// 从房间中移除
		if room, exists := h.rooms[client.RoomID]; exists {
			delete(room, client)
			if len(room) == 0 {
				delete(h.rooms, client.RoomID)
			}
		}

		// 从用户映射中移除
		delete(h.users, client.UserID)

		// 关闭发送通道
		close(client.Send)

		// 设置用户离线状态
		SetUserOffline(client.UserID)

		// 更新数据库中的用户离线状态
		now := time.Now()
		database.DB.Model(&models.User{}).Where("id = ?", client.UserID).Updates(map[string]interface{}{
			"is_online": false,
			"last_seen": &now,
		})

		log.Printf("Client unregistered: UserID=%d, RoomID=%d", client.UserID, client.RoomID)

		// 通知房间内其他用户有用户离开
		h.notifyUserLeft(client)
	}
}

// broadcastToRoom 向房间广播消息
func (h *Hub) broadcastToRoom(roomID uint, message interface{}) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if room, exists := h.rooms[roomID]; exists {
		jsonData, err := json.Marshal(message)
		if err != nil {
			log.Printf("Error marshaling message: %v", err)
			return
		}

		for client := range room {
			select {
			case client.Send <- jsonData:
			default:
				// 如果发送失败，关闭客户端
				close(client.Send)
				delete(h.clients, client)
				delete(room, client)
			}
		}
	}
}
```

### 3. 用户状态管理

```go
// notifyUserJoined 通知用户加入
func (h *Hub) notifyUserJoined(client *Client) {
	// 获取用户信息
	var user models.User
	if err := database.DB.First(&user, client.UserID).Error; err != nil {
		return
	}

	message := WebSocketMessage{
		Type:   "user_joined",
		RoomID: client.RoomID,
		Data: map[string]interface{}{
			"user": user.ToJSON(),
		},
	}

	h.broadcastToRoom(client.RoomID, message)
}

// notifyUserLeft 通知用户离开
func (h *Hub) notifyUserLeft(client *Client) {
	message := WebSocketMessage{
		Type:   "user_left",
		RoomID: client.RoomID,
		Data: map[string]interface{}{
			"user_id": client.UserID,
		},
	}

	h.broadcastToRoom(client.RoomID, message)
}

// sendOnlineUsers 发送在线用户列表
func (h *Hub) sendOnlineUsers(client *Client) {
	onlineUsers, err := GetOnlineUsers(client.RoomID)
	if err != nil {
		log.Printf("Error getting online users: %v", err)
		return
	}

	// 获取用户详细信息
	var users []models.User
	if len(onlineUsers) > 0 {
		database.DB.Where("id IN ?", onlineUsers).Find(&users)
	}

	var userList []map[string]interface{}
	for _, user := range users {
		userList = append(userList, user.ToJSON())
	}

	message := WebSocketMessage{
		Type:   "online_users",
		RoomID: client.RoomID,
		Data: map[string]interface{}{
			"users": userList,
		},
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return
	}

	select {
	case client.Send <- jsonData:
	default:
		close(client.Send)
	}
}

// BroadcastMessage 广播消息到房间
func (h *Hub) BroadcastMessage(roomID uint, message interface{}) {
	h.broadcast <- &BroadcastMessage{
		RoomID:  roomID,
		Message: message,
	}
}

// RegisterClient 注册客户端
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient 注销客户端
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}
```

## 🔌 WebSocket 连接处理

创建 `internal/websocket/connection.go`：

```go
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
```

### 消息处理逻辑

```go
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
```

## 📡 WebSocket 处理器

创建 `internal/handlers/websocket.go`：

```go
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
```

## 🔄 Redis 集成

创建 `internal/services/redis.go`：

```go
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"gin-chat-room/config"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var ctx = context.Background()

// InitRedis 初始化 Redis 连接
func InitRedis() error {
	cfg := config.AppConfig.Redis

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	_, err := RedisClient.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("Redis connected successfully")
	return nil
}

// SetUserOnline 设置用户在线状态
func SetUserOnline(userID uint, roomID uint) error {
	if RedisClient == nil {
		return nil // Redis 未连接，跳过
	}
	
	key := fmt.Sprintf("user:online:%d", userID)
	data := map[string]interface{}{
		"user_id":   userID,
		"room_id":   roomID,
		"timestamp": time.Now().Unix(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return RedisClient.Set(ctx, key, jsonData, 30*time.Minute).Err()
}

// SetUserOffline 设置用户离线状态
func SetUserOffline(userID uint) error {
	if RedisClient == nil {
		return nil // Redis 未连接，跳过
	}
	
	key := fmt.Sprintf("user:online:%d", userID)
	return RedisClient.Del(ctx, key).Err()
}

// GetOnlineUsers 获取房间内在线用户
func GetOnlineUsers(roomID uint) ([]uint, error) {
	if RedisClient == nil {
		return []uint{}, nil // Redis 未连接，返回空列表
	}
	
	pattern := "user:online:*"
	keys, err := RedisClient.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}

	var onlineUsers []uint
	for _, key := range keys {
		data, err := RedisClient.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var userInfo map[string]interface{}
		if err := json.Unmarshal([]byte(data), &userInfo); err != nil {
			continue
		}

		if userRoomID, ok := userInfo["room_id"].(float64); ok && uint(userRoomID) == roomID {
			if userID, ok := userInfo["user_id"].(float64); ok {
				onlineUsers = append(onlineUsers, uint(userID))
			}
		}
	}

	return onlineUsers, nil
}

// CacheMessage 缓存消息
func CacheMessage(roomID uint, message interface{}) error {
	if RedisClient == nil {
		return nil // Redis 未连接，跳过
	}
	
	key := fmt.Sprintf("room:messages:%d", roomID)
	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// 使用列表存储最近的消息，保留最近100条
	pipe := RedisClient.Pipeline()
	pipe.LPush(ctx, key, jsonData)
	pipe.LTrim(ctx, key, 0, 99) // 保留最近100条消息
	pipe.Expire(ctx, key, 24*time.Hour) // 24小时过期

	_, err = pipe.Exec(ctx)
	return err
}
```

## 🔍 WebSocket 消息类型

### 消息类型定义

```go
// 客户端发送的消息类型
const (
    MessageTypeChat     = "message"     // 聊天消息
    MessageTypeJoinRoom = "join_room"   // 加入房间
    MessageTypeLeaveRoom = "leave_room" // 离开房间
)

// 服务器发送的消息类型
const (
    MessageTypeMessage     = "message"      // 聊天消息
    MessageTypeUserJoined  = "user_joined"  // 用户加入
    MessageTypeUserLeft    = "user_left"    // 用户离开
    MessageTypeOnlineUsers = "online_users" // 在线用户列表
)
```

### 消息格式示例

```json
// 发送聊天消息
{
  "type": "message",
  "room_id": 1,
  "content": "Hello, World!"
}

// 接收聊天消息
{
  "type": "message",
  "room_id": 1,
  "data": {
    "id": 1,
    "user_id": 1,
    "content": "Hello, World!",
    "created_at": "2023-01-01T00:00:00Z",
    "user": {
      "id": 1,
      "username": "testuser",
      "nickname": "测试用户"
    }
  }
}

// 用户加入通知
{
  "type": "user_joined",
  "room_id": 1,
  "data": {
    "user": {
      "id": 2,
      "username": "newuser",
      "nickname": "新用户"
    }
  }
}

// 在线用户列表
{
  "type": "online_users",
  "room_id": 1,
  "data": {
    "users": [
      {
        "id": 1,
        "username": "user1",
        "nickname": "用户1"
      },
      {
        "id": 2,
        "username": "user2",
        "nickname": "用户2"
      }
    ]
  }
}
```

## 📊 性能优化

### 1. 连接池管理

```go
// 设置合理的缓冲区大小
const (
    readBufferSize  = 1024
    writeBufferSize = 1024
    sendChannelSize = 256
)

// 限制最大连接数
const maxConnections = 10000

// 连接超时设置
const (
    writeWait  = 10 * time.Second
    pongWait   = 60 * time.Second
    pingPeriod = (pongWait * 9) / 10
)
```

### 2. 消息批处理

```go
// 批量发送消息
n := len(client.Send)
for i := 0; i < n; i++ {
    w.Write([]byte{'\n'})
    w.Write(<-client.Send)
}
```

### 3. 内存优化

```go
// 使用对象池减少 GC 压力
var messagePool = sync.Pool{
    New: func() interface{} {
        return &WebSocketMessage{}
    },
}

func getMessage() *WebSocketMessage {
    return messagePool.Get().(*WebSocketMessage)
}

func putMessage(msg *WebSocketMessage) {
    msg.Reset()
    messagePool.Put(msg)
}
```

## 📚 最佳实践

1. **连接管理**: 及时清理断开的连接，避免内存泄漏
2. **消息验证**: 验证所有接收到的消息格式和内容
3. **错误处理**: 优雅处理连接错误和消息解析错误
4. **性能监控**: 监控连接数、消息吞吐量等指标
5. **安全考虑**: 验证用户权限，防止跨房间消息泄露

## 🎯 下一步

在下一章节中，我们将详细介绍聊天室管理 API 的实现，包括：
- 房间创建和管理
- 用户加入和离开
- 权限控制
- 消息历史查询

通过本章节的学习，您应该已经掌握了：
- WebSocket 连接的管理和维护
- 实时消息的广播机制
- 在线用户状态管理
- Redis 在实时系统中的应用
