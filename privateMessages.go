package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
var connections = make(map[int]*websocket.Conn)
var onlineUsers = make(map[int]bool)
var mutex sync.Mutex

// WebSocket handler
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not open WebSocket connection", http.StatusBadRequest)
		return
	}
	defer conn.Close()

	userID, err := strconv.Atoi(r.URL.Query().Get("userId"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	mutex.Lock()
	connections[userID] = conn
	onlineUsers[userID] = true
	mutex.Unlock()

	notifyUserStatusChange(userID, true)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			mutex.Lock()
			delete(connections, userID)
			delete(onlineUsers, userID)
			mutex.Unlock()

			notifyUserStatusChange(userID, false)
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err == nil {
			saveMessage(msg)
			broadcastMessage(msg)
		}
	}
}

func notifyUserStatusChange(userID int, isOnline bool) {
	message, _ := json.Marshal(map[string]interface{}{
		"user_id": userID,
		"online":  isOnline,
		"type":    "user_status",
	})

	mutex.Lock()
	for _, conn := range connections {
		conn.WriteMessage(websocket.TextMessage, message)
	}
	mutex.Unlock()
}

// Save message
func saveMessage(msg Message) {
	_, err := db.Exec(`INSERT INTO messages (sender_id, receiver_id, content) VALUES (?, ?, ?)`, msg.SenderID, msg.ReceiverID, msg.Content)
	if err != nil {
		log.Println("Error saving message:", err)
	}
}

// Broadcast message
func broadcastMessage(msg Message) {
	mutex.Lock()
	conn, ok := connections[msg.ReceiverID]
	mutex.Unlock()

	if ok {
		message, _ := json.Marshal(msg)
		conn.WriteMessage(websocket.TextMessage, message)
	}
}

// Load messages with pagination
func getMessagesHandler(w http.ResponseWriter, r *http.Request) {
	senderID := r.URL.Query().Get("sender_id")
	receiverID := r.URL.Query().Get("receiver_id")
	limit := 10
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	rows, err := db.Query(`SELECT id, sender_id, receiver_id, content, sent_at FROM messages 
		WHERE (sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?) 
		ORDER BY sent_at DESC LIMIT ? OFFSET ?`, senderID, receiverID, receiverID, senderID, limit, offset)

	if err != nil {
		http.Error(w, "Error loading messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		rows.Scan(&msg.ID, &msg.SenderID, &msg.ReceiverID, &msg.Content, &msg.SentAt)
		messages = append(messages, msg)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// Get users with online status
func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, nickname FROM users ORDER BY nickname ASC`)
	if err != nil {
		http.Error(w, "Error loading users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id int
		var nickname string
		rows.Scan(&id, &nickname)

		mutex.Lock()
		isOnline := onlineUsers[id]
		mutex.Unlock()

		users = append(users, map[string]interface{}{
			"id":       id,
			"nickname": nickname,
			"online":   isOnline,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// Message struct
type Message struct {
	ID         int    `json:"id"`
	SenderID   int    `json:"sender_id"`
	ReceiverID int    `json:"receiver_id"`
	Content    string `json:"content"`
	SentAt     string `json:"sent_at"`
}
