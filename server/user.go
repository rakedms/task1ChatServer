package server

import (
	"chat-server/models"
	"fmt"
	"math/rand"
	"time"
)

func GenerateUserID() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("user-%d", r.Intn(10000))
}

func (s *ChatServer) CreateUser(displayName string) *models.User {
	user := &models.User{
		ID:                   GenerateUserID(),
		DisplayName:          displayName,
		ChatRooms:            make([]string, 0),
		BroadcastMessageChan: make(chan string, 1000),
		PrivateMessageChan:   make(chan string, 100),
	}

	s.mu.Lock()
	s.users[user.ID] = user
	s.mu.Unlock()

	return user
}
