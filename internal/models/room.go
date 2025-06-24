package models

import (
	"time"

	"gorm.io/gorm"
)

// Room 聊天室模型
type Room struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"not null;size:100"`
	Description string         `json:"description" gorm:"size:500"`
	IsPrivate   bool           `json:"is_private" gorm:"default:false"`
	Password    string         `json:"-" gorm:"size:255"` // 私有房间密码
	MaxMembers  int            `json:"max_members" gorm:"default:100"`
	CreatorID   uint           `json:"creator_id" gorm:"not null"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Creator     User         `json:"creator" gorm:"foreignKey:CreatorID"`
	Messages    []Message    `json:"-" gorm:"foreignKey:RoomID"`
	RoomMembers []RoomMember `json:"-" gorm:"foreignKey:RoomID"`
}

// RoomMember 聊天室成员模型
type RoomMember struct {
	ID       uint      `json:"id" gorm:"primaryKey"`
	RoomID   uint      `json:"room_id" gorm:"not null"`
	UserID   uint      `json:"user_id" gorm:"not null"`
	Role     string    `json:"role" gorm:"default:'member';size:20"` // admin, member
	JoinedAt time.Time `json:"joined_at"`

	// 关联关系
	Room Room `json:"room" gorm:"foreignKey:RoomID"`
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// GetMemberCount 获取房间成员数量
func (r *Room) GetMemberCount(db *gorm.DB) int64 {
	var count int64
	db.Model(&RoomMember{}).Where("room_id = ?", r.ID).Count(&count)
	return count
}

// IsMember 检查用户是否是房间成员
func (r *Room) IsMember(db *gorm.DB, userID uint) bool {
	var count int64
	db.Model(&RoomMember{}).Where("room_id = ? AND user_id = ?", r.ID, userID).Count(&count)
	return count > 0
}

// IsAdmin 检查用户是否是房间管理员
func (r *Room) IsAdmin(db *gorm.DB, userID uint) bool {
	var count int64
	db.Model(&RoomMember{}).Where("room_id = ? AND user_id = ? AND role = ?", r.ID, userID, "admin").Count(&count)
	return count > 0 || r.CreatorID == userID
}

// ToJSON 转换为 JSON 格式
func (r *Room) ToJSON(db *gorm.DB) map[string]interface{} {
	return map[string]interface{}{
		"id":           r.ID,
		"name":         r.Name,
		"description":  r.Description,
		"is_private":   r.IsPrivate,
		"max_members":  r.MaxMembers,
		"creator_id":   r.CreatorID,
		"member_count": r.GetMemberCount(db),
		"created_at":   r.CreatedAt,
	}
}
