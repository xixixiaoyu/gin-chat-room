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
