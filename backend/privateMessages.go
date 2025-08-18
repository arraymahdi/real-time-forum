package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

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
	SentAt     string `json:"sent_at"`
	Type       string `json:"type,omitempty"` // Added for typing notification
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

	var authData struct {
		Token string `json:"token"`
	}
	err = conn.ReadJSON(&authData)
	if err != nil {
		log.Println("Failed to read authentication data:", err)
		conn.WriteJSON(map[string]string{"error": "Invalid token data"})
		return
	}

	userID, err := ExtractUserIDFromToken(authData.Token)
	if err != nil {
		log.Println("Invalid token:", err)
		conn.WriteJSON(map[string]string{"error": "Unauthorized"})
		return
	}

	clientsMux.Lock()
	clients[userID] = &Client{conn: conn, id: userID}
	clientsMux.Unlock()

	log.Printf("User %d connected", userID)

	// Notify all clients about the new online user
	broadcastOnlineUsers()

	defer func() {
		clientsMux.Lock()
		delete(clients, userID)
		clientsMux.Unlock()
		log.Printf("User %d disconnected", userID)
		broadcastOnlineUsers()
	}()

	// Send confirmation to the connected client
	conn.WriteJSON(map[string]string{"status": "connected", "user_id": fmt.Sprintf("%d", userID)})

	// Listen for messages from the user
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		msg.SenderID = userID
		msg.SentAt = time.Now().Format(time.RFC3339)

		if msg.Type == "typing" {
			broadcastTyping(msg)
		} else {
			saveMessage(msg)
			forwardMessage(msg)
		}
	}
}

func broadcastOnlineUsers() {
	clientsMux.Lock()
	onlineUsers := make([]int, 0, len(clients))
	for id := range clients {
		onlineUsers = append(onlineUsers, id)
	}
	clientsMux.Unlock()

	notification := map[string]interface{}{
		"type":         "online_users",
		"online_users": onlineUsers,
	}

	clientsMux.Lock()
	for _, client := range clients {
		client.conn.WriteJSON(notification)
	}
	clientsMux.Unlock()
}

func broadcastTyping(msg Message) {
	clientsMux.Lock()
	receiver, exists := clients[msg.ReceiverID]
	clientsMux.Unlock()

	if exists {
		receiver.conn.WriteJSON(msg)
	}
}

func saveMessage(msg Message) {
	_, err := db.Exec("INSERT INTO messages (sender_id, receiver_id, content, sent_at) VALUES (?, ?, ?, ?)",
		msg.SenderID, msg.ReceiverID, msg.Content, msg.SentAt)
	if err != nil {
		log.Println("Failed to save message:", err)
	}
}

func forwardMessage(msg Message) {
	clientsMux.Lock()
	receiver, exists := clients[msg.ReceiverID]
	clientsMux.Unlock()

	if exists {
		msg.SentAt = time.Now().Format(time.RFC3339)
		receiver.conn.WriteJSON(msg)
	}
}

func getMessagesHandler(w http.ResponseWriter, r *http.Request) {
	var userID1, userID2, offset int
	fmt.Sscanf(r.URL.Query().Get("user1"), "%d", &userID1)
	fmt.Sscanf(r.URL.Query().Get("user2"), "%d", &userID2)
	fmt.Sscanf(r.URL.Query().Get("offset"), "%d", &offset)

	rows, err := db.Query(`
		SELECT sender_id, receiver_id, content, sent_at 
		FROM messages 
		WHERE (sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?) 
		ORDER BY sent_at DESC 
		LIMIT 10 OFFSET ?`, userID1, userID2, userID2, userID1, offset)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var sentAt time.Time
		if err := rows.Scan(&msg.SenderID, &msg.ReceiverID, &msg.Content, &sentAt); err == nil {
			msg.SentAt = sentAt.Format(time.RFC3339)
			messages = append(messages, msg)
		}
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

func getSortedUsersHandler(w http.ResponseWriter, r *http.Request) {
	var userID int
	fmt.Sscanf(r.URL.Query().Get("user_id"), "%d", &userID)

	rows, err := db.Query(`
        SELECT DISTINCT u.id, u.nickname, MAX(m.sent_at) as last_message_time
        FROM users u
        LEFT JOIN messages m ON (m.sender_id = u.id OR m.receiver_id = u.id)
        WHERE u.id != ? AND (m.sender_id = ? OR m.receiver_id = ?)
        GROUP BY u.id
        ORDER BY last_message_time DESC
    `, userID, userID, userID)

	if err != nil {
		log.Printf("Database query error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var usersWithMessages []struct {
		ID              int    `json:"id"`
		Nickname        string `json:"nickname"`
		LastMessageTime string `json:"last_message_time"`
	}

	for rows.Next() {
		var user struct {
			ID              int    `json:"id"`
			Nickname        string `json:"nickname"`
			LastMessageTime string `json:"last_message_time"`
		}
		if err := rows.Scan(&user.ID, &user.Nickname, &user.LastMessageTime); err == nil {
			usersWithMessages = append(usersWithMessages, user)
		} else {
			log.Printf("Error scanning row: %v", err)
		}
	}

	rows2, err := db.Query(`
        SELECT id, nickname
        FROM users
        WHERE id != ? AND id NOT IN (
            SELECT DISTINCT receiver_id FROM messages WHERE sender_id = ?
            UNION
            SELECT DISTINCT sender_id FROM messages WHERE receiver_id = ?
        )
        ORDER BY nickname ASC
    `, userID, userID, userID)

	if err != nil {
		log.Printf("Database query error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows2.Close()

	var usersNoMessages []struct {
		ID       int    `json:"id"`
		Nickname string `json:"nickname"`
	}

	for rows2.Next() {
		var user struct {
			ID       int    `json:"id"`
			Nickname string `json:"nickname"`
		}
		if err := rows2.Scan(&user.ID, &user.Nickname); err == nil {
			usersNoMessages = append(usersNoMessages, user)
		} else {
			log.Printf("Error scanning row: %v", err)
		}
	}

	var sortedUsers []struct {
		ID              int    `json:"id"`
		LastMessageTime string `json:"last_message_time"`
		Nickname        string `json:"nickname"`
	}

	for _, user := range usersWithMessages {
		sortedUsers = append(sortedUsers, struct {
			ID              int    `json:"id"`
			LastMessageTime string `json:"last_message_time"`
			Nickname        string `json:"nickname"`
		}{
			ID:              user.ID,
			Nickname:        user.Nickname,
			LastMessageTime: user.LastMessageTime,
		})
	}

	for _, user := range usersNoMessages {
		sortedUsers = append(sortedUsers, struct {
			ID              int    `json:"id"`
			LastMessageTime string `json:"last_message_time"`
			Nickname        string `json:"nickname"`
		}{
			ID:       user.ID,
			Nickname: user.Nickname,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sortedUsers)
}
