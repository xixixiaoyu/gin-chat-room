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

// PublishMessage 发布消息到 Redis
func PublishMessage(roomID uint, message interface{}) error {
	if RedisClient == nil {
		return nil // Redis 未连接，跳过
	}

	channel := fmt.Sprintf("room:%d", roomID)
	jsonData, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return RedisClient.Publish(ctx, channel, jsonData).Err()
}

// SubscribeRoom 订阅房间消息
func SubscribeRoom(roomID uint) *redis.PubSub {
	if RedisClient == nil {
		return nil // Redis 未连接，返回 nil
	}

	channel := fmt.Sprintf("room:%d", roomID)
	return RedisClient.Subscribe(ctx, channel)
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

// GetCachedMessages 获取缓存的消息
func GetCachedMessages(roomID uint, limit int) ([]string, error) {
	if RedisClient == nil {
		return []string{}, nil // Redis 未连接，返回空列表
	}

	key := fmt.Sprintf("room:messages:%d", roomID)
	return RedisClient.LRange(ctx, key, 0, int64(limit-1)).Result()
}
