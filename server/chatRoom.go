package server

import (
	"chat-server/models"
	"fmt"
)

func NewChatServer() *ChatServer {
	return &ChatServer{
		rooms:              make(map[string]*models.Room),
		users:              make(map[string]*models.User),
		displayNames:       make(map[string]bool),
		clientsPrivateMsgs: make(map[string][]chan string),
		clientsRoomMsgs:    make(map[string][]chan string),
	}
}

func (s *ChatServer) JoinRoom(user *models.User, roomName string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	chatRoom, exists := s.rooms[roomName]
	if exists {
		for _, roomMember := range chatRoom.Members {
			if roomMember == user.ID {
				return false, fmt.Errorf("user already exists in the room")
			}
		}
		chatRoom.Members = append(chatRoom.Members, user.ID)
		chatRoom.NewUserSignal <- struct{}{}
		user.ChatRooms = append(user.ChatRooms, roomName)
		fmt.Println("users added to an existing channel")
		return true, nil
	} else {
		room := &models.Room{Name: roomName, Members: []string{user.ID}, NewUserSignal: make(chan struct{}, 100), RoomMessages: make(chan string, 1000)}
		s.rooms[roomName] = room
		user.ChatRooms = append(user.ChatRooms, roomName)
		fmt.Println("new channel has been created and user has been added")
		return true, nil
	}
}

func (s *ChatServer) ListRooms() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var roomNames []string
	for name, _ := range s.rooms {
		roomNames = append(roomNames, name)
	}
	return roomNames
}
