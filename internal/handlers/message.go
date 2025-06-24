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
