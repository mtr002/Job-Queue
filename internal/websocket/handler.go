package websocket

import (
	"encoding/json"
	"log"
	"net/http"
)

func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	client.hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}

func BroadcastJobUpdate(hub *Hub, job interface{}) {
	message, err := json.Marshal(map[string]interface{}{
		"type": "job_update",
		"data": job,
	})
	if err != nil {
		log.Printf("Failed to marshal job update: %v", err)
		return
	}

	hub.Broadcast(message)
}

