package models

type User struct {
	ID                   string
	DisplayName          string
	ChatRooms            []string
	BroadcastMessageChan chan string
	PrivateMessageChan   chan string
}

type Room struct {
	Name          string
	NewUserSignal chan struct{}
	Members       []string
	Messages      []string
	RoomMessages  chan string
}
