package tests

import (
	"gin-chat-room/internal/models"
	"testing"
)

func TestUserModel(t *testing.T) {
	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
	}

	// 测试设置密码
	password := "testpassword123"
	err := user.SetPassword(password)
	if err != nil {
		t.Fatalf("Failed to set password: %v", err)
	}

	if user.Password == "" {
		t.Error("Password should not be empty after setting")
	}

	if user.Password == password {
		t.Error("Password should be hashed, not stored in plain text")
	}

	// 测试验证密码
	if !user.CheckPassword(password) {
		t.Error("Password verification should succeed with correct password")
	}

	if user.CheckPassword("wrongpassword") {
		t.Error("Password verification should fail with incorrect password")
	}

	// 测试 ToJSON 方法
	jsonData := user.ToJSON()
	if jsonData["username"] != user.Username {
		t.Error("ToJSON should include username")
	}

	if jsonData["email"] != user.Email {
		t.Error("ToJSON should include email")
	}

	if _, exists := jsonData["password"]; exists {
		t.Error("ToJSON should not include password")
	}
}

func TestMessageModel(t *testing.T) {
	message := &models.Message{
		RoomID:  1,
		UserID:  1,
		Type:    models.MessageTypeText,
		Content: "Hello, world!",
		User: models.User{
			ID:       1,
			Username: "testuser",
			Nickname: "Test User",
		},
	}

	// 测试 ToJSON 方法
	jsonData := message.ToJSON()
	
	if jsonData["room_id"] != message.RoomID {
		t.Error("ToJSON should include room_id")
	}

	if jsonData["user_id"] != message.UserID {
		t.Error("ToJSON should include user_id")
	}

	if jsonData["type"] != message.Type {
		t.Error("ToJSON should include type")
	}

	if jsonData["content"] != message.Content {
		t.Error("ToJSON should include content")
	}

	// 检查用户信息
	userInfo, ok := jsonData["user"].(map[string]interface{})
	if !ok {
		t.Error("ToJSON should include user information")
	}

	if userInfo["username"] != message.User.Username {
		t.Error("User information should include username")
	}
}

func TestCreateSystemMessage(t *testing.T) {
	roomID := uint(1)
	content := "System message"

	message := models.CreateSystemMessage(roomID, content)

	if message.RoomID != roomID {
		t.Errorf("Expected RoomID %d, got %d", roomID, message.RoomID)
	}

	if message.UserID != 0 {
		t.Errorf("System message should have UserID 0, got %d", message.UserID)
	}

	if message.Type != models.MessageTypeSystem {
		t.Errorf("Expected Type %s, got %s", models.MessageTypeSystem, message.Type)
	}

	if message.Content != content {
		t.Errorf("Expected Content %s, got %s", content, message.Content)
	}
}
