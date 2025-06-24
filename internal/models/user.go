package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Username  string         `json:"username" gorm:"uniqueIndex;not null;size:50"`
	Email     string         `json:"email" gorm:"uniqueIndex;not null;size:100"`
	Password  string         `json:"-" gorm:"not null"`
	Nickname  string         `json:"nickname" gorm:"size:50"`
	Avatar    string         `json:"avatar" gorm:"size:255"`
	IsOnline  bool           `json:"is_online" gorm:"default:false"`
	LastSeen  *time.Time     `json:"last_seen"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	Messages    []Message    `json:"-" gorm:"foreignKey:UserID"`
	RoomMembers []RoomMember `json:"-" gorm:"foreignKey:UserID"`
}

// BeforeCreate 创建前的钩子函数
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// 如果没有设置昵称，使用用户名作为昵称
	if u.Nickname == "" {
		u.Nickname = u.Username
	}
	return nil
}

// SetPassword 设置密码（加密）
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword 验证密码
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// ToJSON 转换为 JSON 格式（不包含敏感信息）
func (u *User) ToJSON() map[string]interface{} {
	return map[string]interface{}{
		"id":        u.ID,
		"username":  u.Username,
		"email":     u.Email,
		"nickname":  u.Nickname,
		"avatar":    u.Avatar,
		"is_online": u.IsOnline,
		"last_seen": u.LastSeen,
	}
}
