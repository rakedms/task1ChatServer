package routes

import (
	"chat-server/server"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, s *server.ChatServer) {
	r.POST("/connect", s.NewConnection)
	r.POST("/join", s.HandleJoinRoom)
	r.POST("/broadcast-message", s.SendBroadcastMessage)
	r.POST("/send-private-messages", s.SendPrivateMessage)

	r.GET("/all-rooms-in-server", s.HandleListRooms)
	r.GET("/all-rooms-of-user/:userID", s.GetUserRooms)
	r.GET("/all-room-messages/:userID", s.GetMessagesFromAllRooms)
	r.GET("/private-messages/:userID", s.GetPrivateMessage)
	r.GET("/getRoomContents/:userID/:roomName", s.GetChatRoomContents)
}
