package server

import (
	"chat-server/models"
)

func NewChatServer() *ChatServer {
	return &ChatServer{
		rooms: make(map[string]*models.Room),
		users: make(map[string]*models.User),
	}
}

func (s *ChatServer) JoinRoom(user *models.User, roomName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	chatRoom, exists := s.rooms[roomName]
	if exists {
		chatRoom.Members[user.ID] = user
		chatRoom.NewUserSignal <- struct{}{}
		user.ChatRooms = append(user.ChatRooms, roomName)
	} else {
		room := &models.Room{Name: roomName, Members: make(map[string]*models.User), NewUserSignal: make(chan struct{}), RoomMessages: make(chan string, 10)}
		s.rooms[roomName] = room
		user.ChatRooms = append(user.ChatRooms, roomName)
	}
}

func (s *ChatServer) ListRooms() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var roomNames []string
	for name := range s.rooms {
		roomNames = append(roomNames, name)
	}
	return roomNames
}
