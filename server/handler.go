package server

import (
	"chat-server/models"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type ChatServer struct {
	mu    sync.Mutex
	rooms map[string]*models.Room
	users map[string]*models.User
}

func (s *ChatServer) NewConnection(c *gin.Context) {
	var req struct {
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user := s.CreateUser(req.DisplayName)
	c.JSON(http.StatusOK, gin.H{"user_id": user.ID})
}

func (s *ChatServer) HandleListRooms(c *gin.Context) {
	rooms := s.ListRooms()
	c.JSON(http.StatusOK, gin.H{"rooms": rooms})
}

func (s *ChatServer) HandleJoinRoom(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id"`
		RoomName string `json:"room_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	s.mu.Lock()
	user, exists := s.users[req.UserID]
	s.mu.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	s.JoinRoom(user, req.RoomName)

	c.JSON(http.StatusOK, gin.H{"message": "Joined room successfully"})
}

func (s *ChatServer) SendBroadcastMessage(c *gin.Context) {
	var req struct {
		UserID   string `json:"user_id"`
		RoomName string `json:"room_name"`
		Message  string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	s.mu.Lock()
	_, exists := s.users[req.UserID]
	room, roomExists := s.rooms[req.RoomName]
	s.mu.Unlock()

	if !exists || !roomExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User or Room not found"})
		return
	}

	randNum := rand.New(rand.NewSource(time.Now().UnixNano()))
	msgId := fmt.Sprintf("message-%d", randNum.Intn(10000))

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		room.RoomMessages <- fmt.Sprintf("From User: %s, Message: %s", req.UserID, req.Message)
	}()

	for _, member := range room.Members {
		wg.Add(1)
		go func(member *models.User) {
			defer wg.Done()
			member.BroadcastMessageChan <- fmt.Sprintf("From Room: %s, To User: %s, MsgId: %s, Message: %s", room.Name, member.DisplayName, msgId, req.Message)
			room.Messages[msgId] = fmt.Sprintf("From Room: %s, To User: %s, MsgId: %s, Message: %s", room.Name, member.DisplayName, msgId, req.Message)
		}(member)
	}

	wg.Wait()

	c.JSON(http.StatusOK, gin.H{"message": "Message sent"})
}

func (s *ChatServer) SendPrivateMessage(c *gin.Context) {
	var req struct {
		FromUserID string `json:"user_id"`
		ToUserID   string `json:"to_user"`
		Message    string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	_, FromUserExists := s.users[req.FromUserID]
	if !FromUserExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "user who sends message doesn't exists"})
		return
	}
	ToUser, ToUserExists := s.users[req.FromUserID]

	if !ToUserExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "user who receives message doesn't exists"})
		return
	}

	ToUser.PrivateMessageChan <- fmt.Sprintf("from: %s, message: %s", req.FromUserID, req.Message)
	c.JSON(http.StatusOK, gin.H{"message": "Message sent"})
}

func (s *ChatServer) GetMessagesFromAllRooms(c *gin.Context) {
	userID := c.Param("userID")

	s.mu.Lock()
	user, exists := s.users[userID]
	s.mu.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
		return
	}

	notify := c.Writer.CloseNotify()

	go func() {
		for {
			select {
			case msg := <-user.BroadcastMessageChan:
				fmt.Fprintf(c.Writer, "data: %s\n\n", msg)
				flusher.Flush()
			case <-notify:
				log.Println("Client closed connection")
				return
			}
		}
	}()
}

func (s *ChatServer) GetPrivateMessage(c *gin.Context) {
	userID := c.Param("userID")

	s.mu.Lock()
	user, exists := s.users[userID]
	s.mu.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Set SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
		return
	}

	notify := c.Writer.CloseNotify()

	go func() {
		for {
			select {
			case msg := <-user.PrivateMessageChan:
				fmt.Fprintf(c.Writer, "data: %s\n\n", msg)
				flusher.Flush()
			case <-notify:
				log.Println("Client closed connection")
				return
			}
		}
	}()
}

func (s *ChatServer) GetChatRoomContents(c *gin.Context) {
	userID := c.Param("userID")
	roomID := c.Param("roomID")

	s.mu.Lock()
	_, exists := s.users[userID]
	room, roomExists := s.rooms[roomID]
	s.mu.Unlock()

	if !exists || !roomExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User or Room not found"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
		return
	}

	notify := c.Writer.CloseNotify()

	func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		initialMessages, err := json.Marshal(room.Messages)
		if err == nil {
			fmt.Fprintf(c.Writer, "data: %s\n\n", initialMessages)
			flusher.Flush()
		} else {
			log.Println("Error marshaling messages:", err)
		}
		initialMembers, err := json.Marshal(room.Members)
		if err == nil {
			fmt.Fprintf(c.Writer, "users: %s\n\n", initialMembers)
			flusher.Flush()
		} else {
			log.Println("Error marshaling members:", err)
		}
	}()

	go func() {
		for {
			select {
			case _ = <-room.RoomMessages:
				messagesJSON, err := json.Marshal(room.Messages)
				if err != nil {
					log.Println("Error converting members to JSON:", err)
					return
				}
				fmt.Fprintf(c.Writer, "data: %s\n\n", messagesJSON)
				flusher.Flush()
			case _ = <-room.NewUserSignal:
				membersJSON, err := json.Marshal(room.Members)
				if err != nil {
					log.Println("Error converting members to JSON:", err)
					return
				}
				fmt.Fprintf(c.Writer, "users: %s\n\n", membersJSON)
				flusher.Flush()
			case <-notify:
				log.Println("Client closed connection")
				return
			}
		}
	}()

}
