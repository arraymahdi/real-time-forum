package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Message struct {
	SenderID   int    `json:"sender_id"`
	ReceiverID int    `json:"receiver_id"`
	Content    string `json:"content"`
}

type Client struct {
	conn *websocket.Conn
	id   int
}

var (
	clients    = make(map[int]*Client)
	clientsMux sync.Mutex
)

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	// Read token from client
	var authData struct {
		Token string `json:"token"`
	}
	err = conn.ReadJSON(&authData)
	if err != nil {
		log.Println("Failed to read authentication data:", err)
		conn.WriteJSON(map[string]string{"error": "Invalid token data"})
		return
	}

	// Extract user ID from the token
	userID, err := ExtractUserIDFromToken(authData.Token)
	if err != nil {
		log.Println("Invalid token:", err)
		conn.WriteJSON(map[string]string{"error": "Unauthorized"})
		return
	}

	// Register client
	clientsMux.Lock()
	clients[userID] = &Client{conn: conn, id: userID}
	clientsMux.Unlock()

	log.Printf("User %d connected via WebSocket", userID)

	// Ensure client is removed on disconnect
	defer func() {
		clientsMux.Lock()
		delete(clients, userID)
		clientsMux.Unlock()
		log.Printf("User %d disconnected", userID)
	}()

	// Send confirmation to the client
	conn.WriteJSON(map[string]string{"status": "connected", "user_id": fmt.Sprintf("%d", userID)})

	// Keep connection open for messages
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Read error:", err)
			break // Close connection if read fails
		}
		saveMessage(msg)
		forwardMessage(msg)
	}
}


func saveMessage(msg Message) {
	_, err := db.Exec("INSERT INTO messages (sender_id, receiver_id, content) VALUES (?, ?, ?)", msg.SenderID, msg.ReceiverID, msg.Content)
	if err != nil {
		log.Println("Failed to save message:", err)
	}
}

func forwardMessage(msg Message) {
	clientsMux.Lock()
	receiver, exists := clients[msg.ReceiverID]
	clientsMux.Unlock()

	if exists {
		receiver.conn.WriteJSON(msg)
	}
}

func getMessagesHandler(w http.ResponseWriter, r *http.Request) {
	var userID1, userID2 int
	fmt.Sscanf(r.URL.Query().Get("user1"), "%d", &userID1)
	fmt.Sscanf(r.URL.Query().Get("user2"), "%d", &userID2)

	rows, err := db.Query("SELECT sender_id, receiver_id, content FROM messages WHERE (sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?) ORDER BY sent_at", userID1, userID2, userID2, userID1)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		rows.Scan(&msg.SenderID, &msg.ReceiverID, &msg.Content)
		messages = append(messages, msg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func getOnlineUsers(w http.ResponseWriter, r *http.Request) {
	clientsMux.Lock()
	users := make([]int, 0, len(clients))
	for id := range clients {
		users = append(users, id)
	}
	clientsMux.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}