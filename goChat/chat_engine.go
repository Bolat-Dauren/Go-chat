package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	"log"
)

type Admin struct {
	connection *Connection
	NotifyChan chan string
}

var admin *Admin
var allConnections []*Connection
var allMessages []string
var privateChats = make(map[string]*Connection)

type Connection struct {
	name string
	ws   *websocket.Conn
}

func HandleAdminConnection(ws *websocket.Conn) {
	log.Println("Admin connected!")
	connection := Connection{"Admin", ws}
	admin = &Admin{&connection, make(chan string)}
	go admin.ListenNotifications()
	connection.StartListening(PostMessage)
}

func (connection *Connection) StartListening(PostMessage func(string, bool)) {
	buf := make([]byte, 512)

	for {
		n, err := connection.ws.Read(buf)
		if err != nil {
			ReleaseConnection(connection)
			PostMessage(fmt.Sprintf("(%s disconnected)", connection.name), false)
			break
		} else {
			HandleInputMessage(connection, buf[:n], PostMessage)
		}
	}
}

func HandleNewConnection(ws *websocket.Conn) {
	log.Println("New connection!")

	isAdmin := ws.Request().URL.Path == "/admin"

	var name string
	if isAdmin {
		name = "Admin"
	} else {
		name = "Unnamed"
	}

	connection := Connection{name, ws}
	allConnections = append(allConnections, &connection)
	connection.StartListening(PostMessage)
}

func (admin *Admin) ListenNotifications() {
	for {
		message := <-admin.NotifyChan
		admin.connection.ws.Write([]byte(message))
	}
}

func HandleInputMessage(from *Connection, data []byte, PostMessage func(string, bool)) {
	var inputJson map[string]string
	err := json.Unmarshal(data, &inputJson)
	if err != nil {
		log.Println("Error decoding JSON:", err)
		return
	}

	log.Println("New input json:", inputJson)

	var newMessageToAll string

	switch inputJson["action"] {
	case "post_message":
		if inputJson["message"] == "/admin" {
			NotifyAdmin(from.name)
		} else {
			newMessageToAll = fmt.Sprintf("%s: %s", from.name, inputJson["message"])
		}
	case "update_name":
		oldName := from.name
		from.name = inputJson["name"]
		newMessageToAll = fmt.Sprintf("(%s -> %s)", oldName, from.name)
	case "get_history":
		byteMessage, err := json.Marshal(allMessages)
		if err != nil {
			log.Println("Error marshaling message history:", err)
			return
		}
		BroadcastTo(from, byteMessage)
	case "private_message":
		targetName := inputJson["target"]
		privateMsg := inputJson["message"]
		if target, ok := privateChats[targetName]; ok && target != nil {
			PrivateMessage(target, []byte(fmt.Sprintf("Private from %s: %s", from.name, privateMsg)))
		} else {
			PrivateMessage(from, []byte("User not found or not connected"))
		}
	}

	if len(newMessageToAll) > 0 {
		isAdminConnection := from.name == "Admin"
		PostMessage(newMessageToAll, isAdminConnection)
	}
}

func PostMessage(message string, isAdminConnection bool) {
	byteMessage, _ := json.Marshal([]string{message})
	BroadcastToAll(byteMessage)
	if !isAdminConnection {
		NotifyAdmin("")
	}
}

// notifyadmin
func NotifyAdmin(userName string) {
	message := fmt.Sprintf("User %s is calling the admin!", userName)
	if admin != nil {
		privateChats[userName] = nil
		admin.NotifyChan <- message
	}
}
