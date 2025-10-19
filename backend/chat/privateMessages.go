package chat

import (
	"backend/db"
	"backend/user"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func HandleConnections(w http.ResponseWriter, r *http.Request) {
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

	userID, err := user.ExtractUserIDFromToken(authData.Token)
	if err != nil {
		log.Println("Invalid token:", err)
		conn.WriteJSON(map[string]string{"error": "Unauthorized"})
		return
	}

	// Get user's groups
	userGroups, err := GetUserGroups(userID)
	if err != nil {
		log.Printf("Error getting user groups: %v", err)
		userGroups = []int{} // Continue with empty groups
	}

	ClientsMux.Lock()
	Clients[userID] = &Client{Conn: conn, ID: userID, Groups: userGroups}
	ClientsMux.Unlock()

	log.Printf("User %d connected with groups: %v", userID, userGroups)

	// Notify all clients about the new online user
	BroadcastOnlineUsers()

	defer func() {
		ClientsMux.Lock()
		delete(Clients, userID)
		ClientsMux.Unlock()
		log.Printf("User %d disconnected", userID)
		BroadcastOnlineUsers()
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

		// Get sender name
		senderName, _ := GetUserName(userID)
		msg.SenderName = senderName

		switch msg.Type {
		case "typing":
			HandleTypingNotification(msg)
		case "group":
			HandleGroupMessage(msg)
		default: // private message
			msg.Type = "private"
			HandlePrivateMessage(msg)
		}
	}
}

func HandlePrivateMessage(msg Message) {
	// Check if users can message each other
	canMessage, err := CanUsersMessage(msg.SenderID, msg.ReceiverID)
	if err != nil {
		log.Printf("Error checking message permissions: %v", err)
		return
	}

	if !canMessage {
		log.Printf("User %d cannot message user %d - not following or public", msg.SenderID, msg.ReceiverID)
		// Send error back to sender
		ClientsMux.Lock()
		sender, exists := Clients[msg.SenderID]
		ClientsMux.Unlock()
		if exists {
			sender.Conn.WriteJSON(map[string]string{
				"error": "Cannot send message: You must follow this user or they must have a public profile",
			})
		}
		return
	}
	// Save private message
	SavePrivateMessage(msg)

	// Forward to recipient if online
	ForwardPrivateMessage(msg)
}

// Update the handleGroupMessage function to include sender_name in the broadcast
func HandleGroupMessage(msg Message) {
	// Check if user is member of the group
	if !IsUserInGroup(msg.SenderID, msg.GroupID) {
		log.Printf("User %d is not a member of group %d", msg.SenderID, msg.GroupID)
		return
	}

	// Get sender name if not already set
	if msg.SenderName == "" {
		senderName, _ := GetUserName(msg.SenderID)
		msg.SenderName = senderName
	}

	// Save group message
	SaveGroupMessage(msg)

	// Broadcast to all group members
	BroadcastToGroupMembers(msg)
}

func HandleTypingNotification(msg Message) {
	if msg.GroupID > 0 {
		// Group typing notification
		BroadcastTypingToGroup(msg)
	} else {
		// Private typing notification
		BroadcastTypingToUser(msg)
	}
}

// Updated canUsersMessage function
func CanUsersMessage(senderID, receiverID int) (bool, error) {
	// Get both users' profile types
	var senderProfileType, receiverProfileType string
	err := db.Instance.QueryRow("SELECT profile_type FROM users WHERE id = ?", senderID).Scan(&senderProfileType)
	if err != nil {
		return false, err
	}

	err = db.Instance.QueryRow("SELECT profile_type FROM users WHERE id = ?", receiverID).Scan(&receiverProfileType)
	if err != nil {
		return false, err
	}

	// If both users are public, they can message each other
	if senderProfileType == "public" && receiverProfileType == "public" {
		return true, nil
	}

	// Check if there's at least one follow relationship (either direction)
	var followExists int
	err = db.Instance.QueryRow(`
		SELECT 1 FROM followers 
		WHERE ((follower_id = ? AND following_id = ?) OR (follower_id = ? AND following_id = ?)) 
		AND status = 'accepted'
		LIMIT 1
	`, senderID, receiverID, receiverID, senderID).Scan(&followExists)

	if err != nil && err.Error() != "sql: no rows in result set" {
		return false, err
	}

	// If there's a follow relationship, allow messaging
	if followExists == 1 {
		return true, nil
	}

	// If receiver is public, anyone can message them
	if receiverProfileType == "public" {
		return true, nil
	}

	return false, nil
}

// Updated getMessageableUsersAndGroupsHandler - returns only followed users and user's groups
func GetMessageableUsersAndGroupsHandler(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	if err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Get online users
	ClientsMux.Lock()
	onlineUserIDs := make(map[int]bool)
	for id := range Clients {
		onlineUserIDs[id] = true
	}
	ClientsMux.Unlock()

	// Combined result structure
	type ChatItem struct {
		ID              int    `json:"id"`
		Type            string `json:"type"` // "user" or "group"
		Name            string `json:"name"`
		ProfileType     string `json:"profile_type,omitempty"` // only for users
		LastMessageTime string `json:"last_message_time"`
		LastMessage     string `json:"last_message,omitempty"`
		IsOnline        bool   `json:"is_online,omitempty"`    // only for users
		MemberCount     int    `json:"member_count,omitempty"` // only for groups
	}

	var chatItems []ChatItem

	// Get only users that the current user follows (with accepted status)
	userRows, err := db.Instance.Query(`
		SELECT DISTINCT u.id, u.nickname, u.profile_type,
		       COALESCE(latest.created_at, '1970-01-01T00:00:00Z') as last_message_time,
		       COALESCE(latest.content, '') as last_message
		FROM users u
		INNER JOIN followers f ON f.following_id = u.id
		LEFT JOIN (
			SELECT 
				CASE 
					WHEN sender_id = ? THEN receiver_id 
					ELSE sender_id 
				END as other_user_id,
				content,
				created_at,
				ROW_NUMBER() OVER (
					PARTITION BY CASE 
						WHEN sender_id = ? THEN receiver_id 
						ELSE sender_id 
					END 
					ORDER BY created_at DESC
				) as rn
			FROM messages 
			WHERE sender_id = ? OR receiver_id = ?
		) latest ON u.id = latest.other_user_id AND latest.rn = 1
		WHERE f.follower_id = ? AND f.status = 'accepted'
		ORDER BY last_message_time DESC, u.nickname ASC
	`, userID, userID, userID, userID, userID)

	if err != nil {
		log.Printf("Database query error for users: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer userRows.Close()

	for userRows.Next() {
		var item ChatItem
		if err := userRows.Scan(&item.ID, &item.Name, &item.ProfileType, &item.LastMessageTime, &item.LastMessage); err == nil {
			item.Type = "user"
			item.IsOnline = onlineUserIDs[item.ID]
			chatItems = append(chatItems, item)
		}
	}

	// Get user's groups with their latest messages
	groupRows, err := db.Instance.Query(`
		SELECT g.group_id, g.title, 
		       COALESCE(latest.created_at, '1970-01-01T00:00:00Z') as last_message_time,
		       COALESCE(latest.content, '') as last_message,
		       COUNT(gm2.user_id) as member_count
		FROM groups g
		JOIN group_memberships gm ON g.group_id = gm.group_id
		LEFT JOIN (
			SELECT group_id, content, created_at,
			       ROW_NUMBER() OVER (PARTITION BY group_id ORDER BY created_at DESC) as rn
			FROM group_messages
		) latest ON g.group_id = latest.group_id AND latest.rn = 1
		LEFT JOIN group_memberships gm2 ON g.group_id = gm2.group_id AND gm2.status = 'accepted'
		WHERE gm.user_id = ? AND gm.status = 'accepted'
		GROUP BY g.group_id
		ORDER BY last_message_time DESC, g.title ASC
	`, userID)

	if err != nil {
		log.Printf("Database query error for groups: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer groupRows.Close()

	for groupRows.Next() {
		var item ChatItem
		if err := groupRows.Scan(&item.ID, &item.Name, &item.LastMessageTime, &item.LastMessage, &item.MemberCount); err == nil {
			item.Type = "group"
			chatItems = append(chatItems, item)
		}
	}

	// Sort all items by last message time (most recent first)
	for i := 0; i < len(chatItems); i++ {
		for j := i + 1; j < len(chatItems); j++ {
			if chatItems[i].LastMessageTime < chatItems[j].LastMessageTime {
				chatItems[i], chatItems[j] = chatItems[j], chatItems[i]
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"chat_items":  chatItems,
		"total_count": len(chatItems),
	})
}

func IsUserInGroup(userID, groupID int) bool {
	var exists int
	err := db.Instance.QueryRow(`
		SELECT 1 FROM group_memberships 
		WHERE user_id = ? AND group_id = ? AND status = 'accepted'
	`, userID, groupID).Scan(&exists)

	return err == nil && exists == 1
}

func GetUserGroups(userID int) ([]int, error) {
	rows, err := db.Instance.Query(`
		SELECT group_id FROM group_memberships 
		WHERE user_id = ? AND status = 'accepted'
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []int
	for rows.Next() {
		var groupID int
		if err := rows.Scan(&groupID); err == nil {
			groups = append(groups, groupID)
		}
	}
	return groups, nil
}

func GetUserName(userID int) (string, error) {
	var name string
	err := db.Instance.QueryRow("SELECT nickname FROM users WHERE id = ?", userID).Scan(&name)
	return name, err
}

func SavePrivateMessage(msg Message) {
	_, err := db.Instance.Exec("INSERT INTO messages (sender_id, receiver_id, content, created_at) VALUES (?, ?, ?, ?)",
		msg.SenderID, msg.ReceiverID, msg.Content, msg.SentAt)
	if err != nil {
		log.Printf("Failed to save private message: %v", err)
	}
}

func SaveGroupMessage(msg Message) {
	_, err := db.Instance.Exec("INSERT INTO group_messages (group_id, sender_id, content, created_at) VALUES (?, ?, ?, ?)",
		msg.GroupID, msg.SenderID, msg.Content, msg.SentAt)
	if err != nil {
		log.Printf("Failed to save group message: %v", err)
	}
}

func ForwardPrivateMessage(msg Message) {
	ClientsMux.Lock()
	receiver, exists := Clients[msg.ReceiverID]
	ClientsMux.Unlock()

	if exists {
		receiver.Conn.WriteJSON(msg)
	}
}

// Get private messages between two users
func GetPrivateMessagesHandler(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var currentUserID int
	if err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&currentUserID); err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var otherUserID, offset int
	fmt.Sscanf(r.URL.Query().Get("other_user"), "%d", &otherUserID)
	fmt.Sscanf(r.URL.Query().Get("offset"), "%d", &offset)

	// Check if users can message each other
	canMessage, err := CanUsersMessage(currentUserID, otherUserID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	if !canMessage {
		http.Error(w, "Cannot view messages with this user", http.StatusForbidden)
		return
	}

	rows, err := db.Instance.Query(`
		SELECT m.sender_id, m.receiver_id, m.content, m.created_at, u.nickname
		FROM messages m
		JOIN users u ON m.sender_id = u.id
		WHERE (m.sender_id = ? AND m.receiver_id = ?) OR (m.sender_id = ? AND m.receiver_id = ?) 
		ORDER BY m.created_at DESC 
		LIMIT 20 OFFSET ?`, currentUserID, otherUserID, otherUserID, currentUserID, offset)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var createdAt time.Time
		if err := rows.Scan(&msg.SenderID, &msg.ReceiverID, &msg.Content, &createdAt, &msg.SenderName); err == nil {
			msg.SentAt = createdAt.Format(time.RFC3339)
			msg.Type = "private"
			messages = append(messages, msg)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// Update the existing getGroupMessagesHandler to ensure it returns data in the correct format
func GetGroupMessagesHandler(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	if err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var groupID, offset int
	fmt.Sscanf(r.URL.Query().Get("group_id"), "%d", &groupID)
	fmt.Sscanf(r.URL.Query().Get("offset"), "%d", &offset)

	// Check if user is member of the group
	if !IsUserInGroup(userID, groupID) {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	rows, err := db.Instance.Query(`
		SELECT gm.message_id, gm.group_id, gm.sender_id, gm.content, 
		       COALESCE(gm.media, '') as media, gm.created_at, u.nickname
		FROM group_messages gm
		JOIN users u ON gm.sender_id = u.id
		WHERE gm.group_id = ?
		ORDER BY gm.created_at DESC 
		LIMIT 20 OFFSET ?`, groupID, offset)

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []GroupMessage
	for rows.Next() {
		var msg GroupMessage
		var createdAt time.Time
		if err := rows.Scan(&msg.MessageID, &msg.GroupID, &msg.SenderID, &msg.Content,
			&msg.Media, &createdAt, &msg.SenderName); err == nil {
			msg.CreatedAt = createdAt.Format(time.RFC3339)
			messages = append(messages, msg)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}
