package models

import (
	"time"

	"gorm.io/gorm"
)

// MessageType 消息类型
type MessageType string

const (
	MessageTypeText   MessageType = "text"   // 文本消息
	MessageTypeImage  MessageType = "image"  // 图片消息
	MessageTypeFile   MessageType = "file"   // 文件消息
	MessageTypeSystem MessageType = "system" // 系统消息
)

// Message 消息模型
type Message struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	RoomID    uint           `json:"room_id" gorm:"not null;index"`
	UserID    uint           `json:"user_id" gorm:"not null;index"`
	Type      MessageType    `json:"type" gorm:"default:'text';size:20"`
	Content   string         `json:"content" gorm:"not null;type:text"`
	FileURL   string         `json:"file_url,omitempty" gorm:"size:500"`
	FileName  string         `json:"file_name,omitempty" gorm:"size:255"`
	FileSize  int64          `json:"file_size,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Room Room `json:"room" gorm:"foreignKey:RoomID"`
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// ToJSON 转换为 JSON 格式
func (m *Message) ToJSON() map[string]interface{} {
	result := map[string]interface{}{
		"id":         m.ID,
		"room_id":    m.RoomID,
		"user_id":    m.UserID,
		"type":       m.Type,
		"content":    m.Content,
		"created_at": m.CreatedAt,
		"user": map[string]interface{}{
			"id":       m.User.ID,
			"username": m.User.Username,
			"nickname": m.User.Nickname,
			"avatar":   m.User.Avatar,
		},
	}

	// 如果是文件消息，添加文件信息
	if m.Type == MessageTypeFile || m.Type == MessageTypeImage {
		result["file_url"] = m.FileURL
		result["file_name"] = m.FileName
		result["file_size"] = m.FileSize
	}

	return result
}

// CreateSystemMessage 创建系统消息
func CreateSystemMessage(roomID uint, content string) *Message {
	return &Message{
		RoomID:  roomID,
		UserID:  0, // 系统消息用户ID为0
		Type:    MessageTypeSystem,
		Content: content,
	}
}
