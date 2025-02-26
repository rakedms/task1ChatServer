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
	mu                 sync.Mutex
	rooms              map[string]*models.Room
	users              map[string]*models.User
	displayNames       map[string]bool
	clientsPrivateMsgs map[string][]chan string
	clientsRoomMsgs    map[string][]chan string
}

func (s *ChatServer) NewConnection(c *gin.Context) {
	var req struct {
		DisplayName string `json:"display_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if s.displayNames[req.DisplayName] {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": "the display name exists already",
		})
		return
	}

	user := s.CreateUser(req.DisplayName)
	s.displayNames[req.DisplayName] = true

	c.JSON(http.StatusOK, gin.H{"user_id": user.ID})
}

func (s *ChatServer) HandleListRooms(c *gin.Context) {
	userID := c.Param("userID")
	_, exists := s.users[userID]
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{
			"err": "the user accessing server doesn't exists",
		})
		return
	}
	rooms := s.ListRooms()
	if rooms == nil {
		rooms = []string{}
	}
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

	_, err := s.JoinRoom(user, req.RoomName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

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

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if !roomExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "chat room not found"})
		return
	}

	randNum := rand.New(rand.NewSource(time.Now().UnixNano()))
	msgId := fmt.Sprintf("message-%d", randNum.Intn(10000))

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		room.RoomMessages <- fmt.Sprintf("From User: %s, Message: %s", req.UserID, req.Message)
		room.Messages = append(room.Messages, fmt.Sprintf("From: %s, To Room: %s, MsgId: %s, Message: %s", req.UserID, req.RoomName, msgId, req.Message))
	}()

	for _, member := range room.Members {
		wg.Add(1)
		go func(member string) {
			defer wg.Done()
			s.mu.Lock()
			user := s.users[member]
			lenOfBroadcastChan := len(user.BroadcastMessageChan)
			if len(user.BroadcastMessageChan[lenOfBroadcastChan-1]) == cap(user.BroadcastMessageChan[lenOfBroadcastChan-1]) {
				newMsgChan := make(chan string, 100)
				user.BroadcastMessageChan = append(user.BroadcastMessageChan, newMsgChan)
			}
			user.BroadcastMessageChan[lenOfBroadcastChan-1] <- fmt.Sprintf("From: %s, To Room: %s, MsgId: %s, Message: %s", req.UserID, req.RoomName, msgId, req.Message)
			user.AllRoomMessages = append(user.AllRoomMessages, fmt.Sprintf("From: %s, To Room: %s, MsgId: %s, Message: %s", req.UserID, req.RoomName, msgId, req.Message))
			s.mu.Unlock()
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

	_, FromUserExists := s.users[req.FromUserID]
	toUser, ToUserExists := s.users[req.ToUserID]

	fmt.Println(toUser)

	s.mu.Unlock()
	if !FromUserExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "user who sends message doesn't exists"})
		return
	}

	if !ToUserExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "user who receives message doesn't exists"})
		return
	}

	for _, msgChan := range toUser.PrivateMsgClient {
		fmt.Println(175, cap(msgChan))
		log.Printf("Sending message from %s to %s: %s", req.FromUserID, req.ToUserID, req.Message)
		msgChan <- fmt.Sprintf("from %s to %s: %s", req.FromUserID, req.ToUserID, req.Message)
		log.Println("Message successfully sent to channel")
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent"})
}

func (s *ChatServer) GetUserRooms(c *gin.Context) {
	userID := c.Param("userID")

	s.mu.Lock()
	user, exists := s.users[userID]
	s.mu.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	rooms := user.ChatRooms
	c.JSON(http.StatusOK, gin.H{
		"rooms": rooms,
	})

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

	notify := c.Request.Context().Done()

	sendAllRoomMsgs := func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		allRoomMessagesJson, err := json.Marshal(user.AllRoomMessages)
		if err == nil {
			fmt.Fprintf(c.Writer, "event: room-messages-user\n")
			fmt.Fprintf(c.Writer, "data: %s\n\n", allRoomMessagesJson)
			flusher.Flush()
		} else {
			log.Println("Error marshaling room messages:", err)
		}

	}

	sendAllRoomMsgs()

	totalChan := len(user.BroadcastMessageChan)
	mergedChan := make(chan string, totalChan*100)
	var wg sync.WaitGroup
	for _, msgChan := range user.BroadcastMessageChan {
		wg.Add(1)
		go func(ch chan string) {
			defer wg.Done()
			for msg := range msgChan {
				mergedChan <- msg
			}
		}(msgChan)
	}

	go func() {
		wg.Wait()
		close(mergedChan)
	}()

	for {
		select {
		case mergedChanMsg := <-mergedChan:
			log.Printf("new msg: %s", mergedChanMsg)
			fmt.Fprintf(c.Writer, "event: new msg from a room\n")
			fmt.Fprintf(c.Writer, "data: %s\n\n", mergedChanMsg)
			flusher.Flush()
		case <-notify:
			log.Println("Client closed connection")
			return
		}
	}

}

func (s *ChatServer) GetPrivateMessage(c *gin.Context) {
	userID := c.Param("userID")

	s.mu.Lock()
	user, exists := s.users[userID]
	s.mu.Unlock()

	fmt.Println(user)
	fmt.Println("cap: ", cap(user.PrivateMsgClient[0]))

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
	user.PvtMsgConnections++

	notify := c.Request.Context().Done()

	if user.PvtMsgConnections > 1 {
		msgChn := make(chan string, 100)
		s.mu.Lock()
		user.PrivateMsgClient = append(user.PrivateMsgClient, msgChn)
		s.mu.Unlock()
	}
	log.Printf("Client connected for user: %s, number of connection is %d", userID, user.PvtMsgConnections)

	mergedChan := make(chan string, 100)

	var wg sync.WaitGroup
	for _, msgClient := range user.PrivateMsgClient {
		wg.Add(1)
		go func(ch chan string) {
			defer wg.Done()
			for msg := range ch {
				mergedChan <- msg
			}
		}(msgClient)
	}

	go func() {
		wg.Wait()
		close(mergedChan)
	}()

	for {
		select {
		case msg, ok := <-mergedChan:
			if !ok {
				return
			}
			fmt.Fprintf(c.Writer, "event: message\n")
			fmt.Fprintf(c.Writer, "data: %s\n\n", msg)
			flusher.Flush()
		case <-notify:
			log.Println("Client closed connection")
			return
		}
	}

}

// this will give all the msgs and the details of the chat room including user id and all messages lively

func (s *ChatServer) GetChatRoomContents(c *gin.Context) {
	userID := c.Param("userID")
	roomName := c.Param("roomName")

	s.mu.Lock()
	_, exists := s.users[userID]
	room, roomExists := s.rooms[roomName]
	s.mu.Unlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if !roomExists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Room not found"})
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

	notify := c.Request.Context().Done()

	sendRoomState := func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		messagesJSON, err := json.Marshal(room.Messages)
		if err == nil {
			fmt.Fprintf(c.Writer, "event: all-messages\n")
			fmt.Fprintf(c.Writer, "data: %s\n\n", messagesJSON)
			flusher.Flush()
		} else {
			log.Println("Error marshaling messages:", err)
		}

		membersJSON, err := json.Marshal(room.Members)
		if err == nil {
			fmt.Fprintf(c.Writer, "event: all-users\n")
			fmt.Fprintf(c.Writer, "data: %s\n\n", membersJSON)
			flusher.Flush()
		} else {
			log.Println("Error marshaling members:", err)
		}
	}

	sendRoomState()

	for {
		select {
		case newMessage := <-room.RoomMessages:
			fmt.Println("New message received:", newMessage)
			fmt.Fprintf(c.Writer, "event: new-message\n")
			fmt.Fprintf(c.Writer, "data: %s\n\n", newMessage)
			flusher.Flush()

		case <-room.NewUserSignal:
			fmt.Println("New user joined the room")
			s.mu.Lock()
			membersJSON, err := json.Marshal(room.Members)
			s.mu.Unlock()
			if err == nil {
				fmt.Fprintf(c.Writer, "event: new-user\n")
				fmt.Fprintf(c.Writer, "data: %s\n\n", membersJSON)
				flusher.Flush()
			} else {
				log.Println("Error marshaling members:", err)
			}

		case <-notify:
			log.Println("Client closed connection")
			return
		}
	}

}
