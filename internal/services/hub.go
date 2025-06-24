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
