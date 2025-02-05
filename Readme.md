Run the chat server
>> go run main.go

--------------------------------------------------------------------------------------
connect user to server
POST http://localhost:8080/connect
body:
    {
        "display_name": "Ram"
    }

resp:

    {
        "user_id": "user-1875"
    }
--------------------------------------------------------------------------------------
create or join room

POST http://localhost:8080/join
body:
    {
      "user_id": "user-1875",
      "room_name": "dummy2"
    }

--------------------------------------------------------------------------------------
Get all rooms in server


GET http://localhost:8080/all-rooms-in-server/user-1875

resp:
    {
        "rooms": null
    }

--------------------------------------------------------------------------------------
Get all rooms of a user

GET http://localhost:8080/all-rooms-of-user/user-1875

resp:

    {
        "rooms": [
            "dummy1",
            "dummy2"
        ]
    }

--------------------------------------------------------------------------------------
Initiate SSE for

All room message of a user
GET http://localhost:8080/all-room-messages/user-7828

Get Private messages
GET http://localhost:8080/private-messages/user-6178

Get Details of a Chat room
GET http://localhost:8080/get-room-contents/user-6178/dummy1

--------------------------------------------------------------------------------------

SEND POST REQ FOR CHAT ROOM/ PVT MSG

Send Broadcast message to a chat room

POST http://localhost:8080/broadcast-message
Body:

    {
      "user_id": "user-2923",
      "room_name": "dummy1",
      "message": "broadcastmsp1"
    }

Resp:

    {
        "message": "Message sent"
    }

--------------------------------------------------------------------------------------
send private message
POST http://localhost:8080/send-private-messages

body:
    {
      "user_id": "user-2923",
      "to_user": "user-7828",
      "message": "pvtMsp1"
    }
    
resp:
    {
        "message": "Message sent"
    }