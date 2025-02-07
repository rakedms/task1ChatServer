package models

type User struct {
	ID                   string
	DisplayName          string
	ChatRooms            []string
	AllRoomMessages      []string
	BroadcastMessageChan []chan string
	PrivateMsgClient     []chan string
	PvtMsgConnections    int
}

type Room struct {
	Name          string
	NewUserSignal chan struct{}
	Members       []string
	Messages      []string
	RoomMessages  chan string
}
