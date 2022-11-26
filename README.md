# Websocket Quizzipy
![user(3)](https://user-images.githubusercontent.com/56477571/204069227-319a7c07-742e-4318-aa61-8ae05ee9e19c.jpg)

this is a exsample of creating websocket server to handle duplex communication.
there are two type of node that server define:
- Admin this role can create a room and comunication with all node in the room and there is only one Admin in a room.
to be come an Admin you must connect into server throghout endpoin `/ws/play?code=<room-code>`.
- Participant this role is only can join a room that is created by Admin dan only can communication with Admin room.
tobe come a Participants you must connect into server throghout endpoin `/ws/join?code=<room-code>&nickname<username>`.

## How to test
1. run the server by runing `main.go` script or build docker image and run it.
2. connect to the server by using script `client.go -url <ws://[host]:[port]/[role]>` in client folder
