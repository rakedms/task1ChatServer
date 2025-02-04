package models

type User struct {
	ID                   string
	DisplayName          string
	BroadcastMessageChan chan string
	PrivateMessageChan   chan string
}

type Room struct {
	Name          string
	NewUserSignal chan struct{}
	Members       map[string]*User
	Messages      map[string]string
	RoomMessages  chan string
}
