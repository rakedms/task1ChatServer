package routes

import (
	"chat-server/server"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, s *server.ChatServer) {
	r.POST("/connect", s.NewConnection)
	r.GET("/rooms", s.HandleListRooms)
	r.POST("/join", s.HandleJoinRoom)
	r.POST("/broadcast-message", s.SendBroadcastMessage)
	r.GET("/all-room-messages/:userID", s.GetMessagesFromAllRooms)
	r.GET("/private-messages/:userID", s.GetPrivateMessage)
	r.GET("/getRoomContents/:userID/:roomID", s.GetChatRoomContents)
}
