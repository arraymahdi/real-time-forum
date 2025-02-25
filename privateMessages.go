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
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	// Read user ID as JSON
	var user struct {
		UserID int `json:"user_id"`
	}
	err = conn.ReadJSON(&user)
	if err != nil {
		log.Println("Failed to read user ID:", err)
		return
	}

	userID := user.UserID

	// Register client
	clientsMux.Lock()
	clients[userID] = &Client{conn: conn, id: userID}
	clientsMux.Unlock()

	log.Printf("User %d connected", userID)

	// Ensure connection stays open
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