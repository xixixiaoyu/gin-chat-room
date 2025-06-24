package tests

import (
	"gin-chat-room/internal/auth"
	"gin-chat-room/config"
	"testing"
	"time"
)

func TestJWTToken(t *testing.T) {
	// 初始化配置
	config.AppConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret:     "test-secret",
			ExpireTime: 24,
		},
	}

	// 测试生成 token
	userID := uint(1)
	username := "testuser"
	email := "test@example.com"

	token, err := auth.GenerateToken(userID, username, email)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Fatal("Generated token is empty")
	}

	// 测试解析 token
	claims, err := auth.ParseToken(token)
	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected UserID %d, got %d", userID, claims.UserID)
	}

	if claims.Username != username {
		t.Errorf("Expected Username %s, got %s", username, claims.Username)
	}

	if claims.Email != email {
		t.Errorf("Expected Email %s, got %s", email, claims.Email)
	}

	// 测试过期时间
	if claims.ExpiresAt.Time.Before(time.Now()) {
		t.Error("Token should not be expired")
	}

	expectedExpiry := time.Now().Add(24 * time.Hour)
	if claims.ExpiresAt.Time.After(expectedExpiry.Add(time.Minute)) {
		t.Error("Token expiry time is too far in the future")
	}
}

func TestInvalidToken(t *testing.T) {
	// 初始化配置
	config.AppConfig = &config.Config{
		JWT: config.JWTConfig{
			Secret:     "test-secret",
			ExpireTime: 24,
		},
	}

	// 测试无效 token
	_, err := auth.ParseToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}

	// 测试空 token
	_, err = auth.ParseToken("")
	if err == nil {
		t.Error("Expected error for empty token")
	}
}
