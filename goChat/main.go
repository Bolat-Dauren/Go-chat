package main

import (
	"golang.org/x/net/websocket"
	"log"
	"net/http"
)

func main() {
	http.Handle("/ws", websocket.Handler(HandleNewConnection))
	http.Handle("/admin", websocket.Handler(HandleAdminConnection))
	http.Handle("/", http.FileServer(http.Dir("./static")))
	log.Fatal(http.ListenAndServe(":3000", nil))

}
