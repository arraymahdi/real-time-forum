package chat

import (
	"backend/db"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func BroadcastToGroupMembers(msg Message) {
	// Get all group members
	rows, err := db.Instance.Query(`
		SELECT user_id FROM group_memberships 
		WHERE group_id = ? AND status = 'accepted'
	`, msg.GroupID)
	if err != nil {
		log.Printf("Error getting group members: %v", err)
		return
	}
	defer rows.Close()

	ClientsMux.Lock()
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err == nil {
			if client, exists := Clients[userID]; exists && userID != msg.SenderID {
				client.Conn.WriteJSON(msg)
			}
		}
	}
	ClientsMux.Unlock()
}

func BroadcastTypingToUser(msg Message) {
	ClientsMux.Lock()
	receiver, exists := Clients[msg.ReceiverID]
	ClientsMux.Unlock()

	if exists {
		receiver.Conn.WriteJSON(msg)
	}
}

func BroadcastTypingToGroup(msg Message) {
	ClientsMux.Lock()
	for _, client := range Clients {
		if client.ID != msg.SenderID {
			// Check if user is in the group
			for _, groupID := range client.Groups {
				if groupID == msg.GroupID {
					client.Conn.WriteJSON(msg)
					break
				}
			}
		}
	}
	ClientsMux.Unlock()
}

func BroadcastOnlineUsers() {
	// Collect IDs of online users
	ClientsMux.Lock()
	onlineUsers := make([]int, 0, len(Clients))
	for id := range Clients {
		onlineUsers = append(onlineUsers, id)
	}
	// make a copy of Clients so we don’t hold lock during writes
	copies := make([]*Client, 0, len(Clients))
	for _, client := range Clients {
		copies = append(copies, client)
	}
	ClientsMux.Unlock()

	// Build the message
	notification := map[string]interface{}{
		"type":         "online_users",
		"online_users": onlineUsers,
	}

	// Send to each client (outside the lock!)
	for _, client := range copies {
		if err := client.Conn.WriteJSON(notification); err != nil {
			log.Printf("Error sending online users: %v", err)
		}
	}
}
