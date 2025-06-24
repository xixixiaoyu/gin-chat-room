# 05 - 聊天室管理 API

## 📋 概述

本章节将详细介绍聊天室管理 API 的实现，包括房间的创建、查询、加入、离开等核心功能，以及权限控制和消息历史查询。

## 🎯 学习目标

- 掌握 RESTful API 的设计原则
- 学会实现房间管理的完整功能
- 理解权限控制和安全验证
- 掌握分页查询和数据过滤

## 🏠 房间管理功能

### 功能概览

```
房间管理功能
├── 房间列表查询
│   ├── 分页查询
│   ├── 搜索过滤
│   └── 权限过滤
├── 房间详情查询
│   ├── 基本信息
│   ├── 成员统计
│   └── 权限验证
├── 房间创建
│   ├── 基本信息设置
│   ├── 私有房间支持
│   └── 创建者权限
├── 房间加入
│   ├── 权限验证
│   ├── 密码验证
│   └── 成员限制
└── 房间离开
    ├── 成员移除
    ├── 权限检查
    └── 系统通知
```

## 🔧 房间处理器实现

创建 `internal/handlers/room.go`：

```go
package handlers

import (
	"gin-chat-room/internal/database"
	"gin-chat-room/internal/middleware"
	"gin-chat-room/internal/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateRoomRequest 创建房间请求结构
type CreateRoomRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=100"`
	Description string `json:"description,omitempty"`
	IsPrivate   bool   `json:"is_private"`
	Password    string `json:"password,omitempty"`
	MaxMembers  int    `json:"max_members,omitempty"`
}

// JoinRoomRequest 加入房间请求结构
type JoinRoomRequest struct {
	Password string `json:"password,omitempty"`
}

// GetRooms 获取房间列表
func GetRooms(c *gin.Context) {
	var rooms []models.Room
	
	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 搜索参数
	search := c.Query("search")
	
	query := database.DB.Model(&models.Room{}).Preload("Creator")
	
	// 只显示公开房间，除非用户是房间成员
	userID, _ := middleware.GetCurrentUserID(c)
	query = query.Where("is_private = ? OR creator_id = ? OR id IN (SELECT room_id FROM room_members WHERE user_id = ?)", 
		false, userID, userID)

	if search != "" {
		query = query.Where("name ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	var total int64
	query.Count(&total)

	// 获取房间列表
	if err := query.Offset(offset).Limit(pageSize).Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch rooms",
		})
		return
	}

	// 转换为 JSON 格式
	var roomList []map[string]interface{}
	for _, room := range rooms {
		roomList = append(roomList, room.ToJSON(database.DB))
	}

	c.JSON(http.StatusOK, gin.H{
		"rooms": roomList,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// CreateRoom 创建房间
func CreateRoom(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data: " + err.Error(),
		})
		return
	}

	// 创建房间
	room := models.Room{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		IsPrivate:   req.IsPrivate,
		MaxMembers:  req.MaxMembers,
		CreatorID:   userID,
	}

	// 设置默认最大成员数
	if room.MaxMembers <= 0 {
		room.MaxMembers = 100
	}

	// 如果是私有房间，设置密码
	if req.IsPrivate && req.Password != "" {
		room.Password = req.Password // 在生产环境中应该加密密码
	}

	// 保存房间
	if err := database.DB.Create(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create room",
		})
		return
	}

	// 创建者自动加入房间
	roomMember := models.RoomMember{
		RoomID:   room.ID,
		UserID:   userID,
		Role:     "admin",
		JoinedAt: time.Now(),
	}
	database.DB.Create(&roomMember)

	// 预加载创建者信息
	database.DB.Preload("Creator").First(&room, room.ID)

	c.JSON(http.StatusCreated, gin.H{
		"room": room.ToJSON(database.DB),
	})
}

// GetRoom 获取房间详情
func GetRoom(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid room ID",
		})
		return
	}

	var room models.Room
	if err := database.DB.Preload("Creator").First(&room, roomID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Room not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database error",
			})
		}
		return
	}

	// 检查用户是否有权限查看房间
	userID, _ := middleware.GetCurrentUserID(c)
	if room.IsPrivate && room.CreatorID != userID && !room.IsMember(database.DB, userID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"room": room.ToJSON(database.DB),
	})
}
```

### 房间加入和离开

```go
// JoinRoom 加入房间
func JoinRoom(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid room ID",
		})
		return
	}

	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	var req JoinRoomRequest
	c.ShouldBindJSON(&req)

	// 获取房间信息
	var room models.Room
	if err := database.DB.First(&room, roomID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Room not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database error",
			})
		}
		return
	}

	// 检查用户是否已经是房间成员
	if room.IsMember(database.DB, userID) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Already a member of this room",
		})
		return
	}

	// 检查房间是否已满
	if room.GetMemberCount(database.DB) >= int64(room.MaxMembers) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Room is full",
		})
		return
	}

	// 如果是私有房间，验证密码
	if room.IsPrivate && room.Password != "" && room.Password != req.Password {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Invalid password",
		})
		return
	}

	// 添加用户到房间
	roomMember := models.RoomMember{
		RoomID:   uint(roomID),
		UserID:   userID,
		Role:     "member",
		JoinedAt: time.Now(),
	}

	if err := database.DB.Create(&roomMember).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to join room",
		})
		return
	}

	// 创建系统消息
	var user models.User
	database.DB.First(&user, userID)
	
	systemMessage := models.CreateSystemMessage(uint(roomID), user.Nickname+" 加入了房间")
	database.DB.Create(systemMessage)

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully joined room",
	})
}

// LeaveRoom 离开房间
func LeaveRoom(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid room ID",
		})
		return
	}

	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// 检查用户是否是房间成员
	var roomMember models.RoomMember
	if err := database.DB.Where("room_id = ? AND user_id = ?", roomID, userID).First(&roomMember).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Not a member of this room",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database error",
			})
		}
		return
	}

	// 检查是否是房间创建者
	var room models.Room
	database.DB.First(&room, roomID)
	if room.CreatorID == userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Room creator cannot leave the room",
		})
		return
	}

	// 删除房间成员记录
	if err := database.DB.Delete(&roomMember).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to leave room",
		})
		return
	}

	// 创建系统消息
	var user models.User
	database.DB.First(&user, userID)
	
	systemMessage := models.CreateSystemMessage(uint(roomID), user.Nickname+" 离开了房间")
	database.DB.Create(systemMessage)

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully left room",
	})
}
```

## 💬 消息管理 API

创建 `internal/handlers/message.go`：

```go
package handlers

import (
	"gin-chat-room/internal/database"
	"gin-chat-room/internal/middleware"
	"gin-chat-room/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetMessages 获取房间消息历史
func GetMessages(c *gin.Context) {
	roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid room ID",
		})
		return
	}

	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// 检查用户是否是房间成员
	var room models.Room
	if err := database.DB.First(&room, roomID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Room not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Database error",
			})
		}
		return
	}

	// 检查权限
	if room.IsPrivate && room.CreatorID != userID && !room.IsMember(database.DB, userID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Access denied",
		})
		return
	}

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize

	// 获取消息
	var messages []models.Message
	var total int64

	query := database.DB.Model(&models.Message{}).Where("room_id = ?", roomID)
	query.Count(&total)

	if err := query.Preload("User").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch messages",
		})
		return
	}

	// 转换为 JSON 格式
	var messageList []map[string]interface{}
	for i := len(messages) - 1; i >= 0; i-- { // 反转顺序，最新的在后面
		messageList = append(messageList, messages[i].ToJSON())
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": messageList,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}
```

## 🔐 权限控制系统

### 权限级别定义

```go
// 房间权限级别
const (
    RoleCreator = "creator" // 创建者
    RoleAdmin   = "admin"   // 管理员
    RoleMember  = "member"  // 普通成员
)

// 权限检查函数
func (r *Room) HasPermission(db *gorm.DB, userID uint, action string) bool {
    // 创建者拥有所有权限
    if r.CreatorID == userID {
        return true
    }

    // 检查用户角色
    var member RoomMember
    if err := db.Where("room_id = ? AND user_id = ?", r.ID, userID).First(&member).Error; err != nil {
        return false
    }

    switch action {
    case "view":
        return true // 所有成员都可以查看
    case "send_message":
        return true // 所有成员都可以发送消息
    case "manage_members":
        return member.Role == RoleAdmin || member.Role == RoleCreator
    case "delete_room":
        return member.Role == RoleCreator
    default:
        return false
    }
}
```

### 权限中间件

```go
// RoomPermissionMiddleware 房间权限中间件
func RoomPermissionMiddleware(action string) gin.HandlerFunc {
    return func(c *gin.Context) {
        roomID, err := strconv.ParseUint(c.Param("id"), 10, 32)
        if err != nil {
            c.JSON(http.StatusBadRequest, gin.H{
                "error": "Invalid room ID",
            })
            c.Abort()
            return
        }

        userID, exists := middleware.GetCurrentUserID(c)
        if !exists {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": "User not authenticated",
            })
            c.Abort()
            return
        }

        var room models.Room
        if err := database.DB.First(&room, roomID).Error; err != nil {
            c.JSON(http.StatusNotFound, gin.H{
                "error": "Room not found",
            })
            c.Abort()
            return
        }

        if !room.HasPermission(database.DB, userID, action) {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "Permission denied",
            })
            c.Abort()
            return
        }

        c.Set("room", &room)
        c.Next()
    }
}
```

## 📊 数据验证和过滤

### 输入验证

```go
// 房间名称验证
func validateRoomName(name string) error {
    name = strings.TrimSpace(name)
    if len(name) < 1 {
        return errors.New("room name cannot be empty")
    }
    if len(name) > 100 {
        return errors.New("room name too long")
    }
    // 检查特殊字符
    if strings.ContainsAny(name, "<>\"'&") {
        return errors.New("room name contains invalid characters")
    }
    return nil
}

// 密码强度验证
func validatePassword(password string) error {
    if len(password) < 6 {
        return errors.New("password must be at least 6 characters")
    }
    if len(password) > 50 {
        return errors.New("password too long")
    }
    return nil
}
```

### 数据过滤

```go
// 房间列表过滤
func filterRooms(query *gorm.DB, userID uint, filters map[string]interface{}) *gorm.DB {
    // 只显示用户有权限查看的房间
    query = query.Where("is_private = ? OR creator_id = ? OR id IN (SELECT room_id FROM room_members WHERE user_id = ?)", 
        false, userID, userID)

    // 按类型过滤
    if roomType, exists := filters["type"]; exists {
        switch roomType {
        case "public":
            query = query.Where("is_private = ?", false)
        case "private":
            query = query.Where("is_private = ?", true)
        case "joined":
            query = query.Where("id IN (SELECT room_id FROM room_members WHERE user_id = ?)", userID)
        }
    }

    // 按成员数过滤
    if minMembers, exists := filters["min_members"]; exists {
        // 这里需要子查询来计算成员数
        query = query.Having("(SELECT COUNT(*) FROM room_members WHERE room_id = rooms.id) >= ?", minMembers)
    }

    return query
}
```

## 🔍 搜索功能

### 全文搜索

```go
// SearchRooms 搜索房间
func SearchRooms(c *gin.Context) {
    keyword := c.Query("q")
    if keyword == "" {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Search keyword is required",
        })
        return
    }

    userID, _ := middleware.GetCurrentUserID(c)
    
    var rooms []models.Room
    query := database.DB.Model(&models.Room{}).Preload("Creator")
    
    // 权限过滤
    query = query.Where("is_private = ? OR creator_id = ? OR id IN (SELECT room_id FROM room_members WHERE user_id = ?)", 
        false, userID, userID)
    
    // 搜索条件
    searchPattern := "%" + keyword + "%"
    query = query.Where("name ILIKE ? OR description ILIKE ?", searchPattern, searchPattern)
    
    // 按相关性排序
    query = query.Order("CASE WHEN name ILIKE ? THEN 1 ELSE 2 END, created_at DESC", searchPattern)
    
    if err := query.Limit(20).Find(&rooms).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Search failed",
        })
        return
    }

    var roomList []map[string]interface{}
    for _, room := range rooms {
        roomList = append(roomList, room.ToJSON(database.DB))
    }

    c.JSON(http.StatusOK, gin.H{
        "rooms": roomList,
        "keyword": keyword,
        "count": len(roomList),
    })
}
```

## 📈 性能优化

### 1. 数据库查询优化

```go
// 使用索引优化查询
type Room struct {
    Name      string `gorm:"index"`           // 为搜索添加索引
    IsPrivate bool   `gorm:"index"`           // 为过滤添加索引
    CreatorID uint   `gorm:"index"`           // 为权限检查添加索引
}

// 批量预加载
query.Preload("Creator").Preload("RoomMembers.User")

// 使用原生 SQL 优化复杂查询
db.Raw(`
    SELECT r.*, COUNT(rm.id) as member_count 
    FROM rooms r 
    LEFT JOIN room_members rm ON r.id = rm.room_id 
    WHERE r.is_private = false 
    GROUP BY r.id 
    ORDER BY member_count DESC
`).Scan(&rooms)
```

### 2. 缓存策略

```go
// 缓存热门房间列表
func GetPopularRooms() ([]models.Room, error) {
    cacheKey := "popular_rooms"
    
    // 尝试从 Redis 获取
    if RedisClient != nil {
        cached, err := RedisClient.Get(ctx, cacheKey).Result()
        if err == nil {
            var rooms []models.Room
            json.Unmarshal([]byte(cached), &rooms)
            return rooms, nil
        }
    }
    
    // 从数据库查询
    var rooms []models.Room
    err := database.DB.Raw(`
        SELECT r.* FROM rooms r 
        LEFT JOIN room_members rm ON r.id = rm.room_id 
        WHERE r.is_private = false 
        GROUP BY r.id 
        ORDER BY COUNT(rm.id) DESC 
        LIMIT 10
    `).Find(&rooms).Error
    
    if err != nil {
        return nil, err
    }
    
    // 缓存结果
    if RedisClient != nil {
        data, _ := json.Marshal(rooms)
        RedisClient.Set(ctx, cacheKey, data, 5*time.Minute)
    }
    
    return rooms, nil
}
```

## 📚 API 设计最佳实践

### 1. RESTful 设计

```
GET    /api/v1/rooms           # 获取房间列表
POST   /api/v1/rooms           # 创建房间
GET    /api/v1/rooms/:id       # 获取房间详情
PUT    /api/v1/rooms/:id       # 更新房间信息
DELETE /api/v1/rooms/:id       # 删除房间

POST   /api/v1/rooms/:id/join  # 加入房间
POST   /api/v1/rooms/:id/leave # 离开房间

GET    /api/v1/rooms/:id/messages # 获取房间消息
GET    /api/v1/rooms/:id/members  # 获取房间成员
```

### 2. 统一响应格式

```go
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Meta    interface{} `json:"meta,omitempty"`
}

// 成功响应
func SuccessResponse(c *gin.Context, data interface{}) {
    c.JSON(http.StatusOK, APIResponse{
        Success: true,
        Data:    data,
    })
}

// 错误响应
func ErrorResponse(c *gin.Context, code int, message string) {
    c.JSON(code, APIResponse{
        Success: false,
        Error:   message,
    })
}
```

### 3. 分页和排序

```go
type PaginationParams struct {
    Page     int    `form:"page,default=1"`
    PageSize int    `form:"page_size,default=20"`
    Sort     string `form:"sort,default=created_at"`
    Order    string `form:"order,default=desc"`
}

func (p *PaginationParams) Validate() error {
    if p.Page < 1 {
        p.Page = 1
    }
    if p.PageSize < 1 || p.PageSize > 100 {
        p.PageSize = 20
    }
    if p.Order != "asc" && p.Order != "desc" {
        p.Order = "desc"
    }
    return nil
}
```

## 🎯 下一步

在下一章节中，我们将详细介绍前端界面的开发，包括：
- HTML 页面结构设计
- CSS 样式和响应式布局
- JavaScript 交互逻辑
- WebSocket 客户端实现

通过本章节的学习，您应该已经掌握了：
- RESTful API 的设计和实现
- 房间管理的完整功能
- 权限控制和安全验证
- 数据查询和性能优化
