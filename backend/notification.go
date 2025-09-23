package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Notification represents a notification in the system
type Notification struct {
	ID             int    `json:"notification_id"`
	UserID         int    `json:"user_id"`
	Type           string `json:"type"`
	Message        string `json:"message"`
	ReadStatus     bool   `json:"read_status"`
	CreatedAt      string `json:"created_at"`
	RelatedUserID  *int   `json:"related_user_id,omitempty"`
	RelatedGroupID *int   `json:"related_group_id,omitempty"`
	SenderName     string `json:"sender_name,omitempty"`
	GroupName      string `json:"group_name,omitempty"`
}

// NotificationData for WebSocket transmission
type NotificationData struct {
	Type         string       `json:"type"`
	Notification Notification `json:"notification"`
}

// Create a new notification and send it via WebSocket
func createNotification(userID int, notificationType, message string, relatedUserID, relatedGroupID *int) error {
	// Insert notification into database
	result, err := db.Exec(`
		INSERT INTO notifications (user_id, type, message, read_status, created_at) 
		VALUES (?, ?, ?, 0, ?)
	`, userID, notificationType, message, time.Now().Format("2006-01-02 15:04:05"))

	if err != nil {
		log.Printf("Error creating notification: %v", err)
		return err
	}

	notificationID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error getting notification ID: %v", err)
		return err
	}

	// Create notification object
	notification := Notification{
		ID:         int(notificationID),
		UserID:     userID,
		Type:       notificationType,
		Message:    message,
		ReadStatus: false,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}

	// Add related IDs if provided
	if relatedUserID != nil {
		notification.RelatedUserID = relatedUserID
		// Get sender name
		var senderName string
		if err := db.QueryRow("SELECT nickname FROM users WHERE id = ?", *relatedUserID).Scan(&senderName); err == nil {
			notification.SenderName = senderName
		}
	}

	if relatedGroupID != nil {
		notification.RelatedGroupID = relatedGroupID
		// Get group name
		var groupName string
		if err := db.QueryRow("SELECT title FROM groups WHERE group_id = ?", *relatedGroupID).Scan(&groupName); err == nil {
			notification.GroupName = groupName
		}
	}

	// Send notification via WebSocket
	sendNotificationViaWebSocket(userID, notification)

	return nil
}

// Send notification to user via WebSocket
func sendNotificationViaWebSocket(userID int, notification Notification) {
	clientsMux.Lock()
	client, exists := clients[userID]
	clientsMux.Unlock()

	if exists {
		notificationData := NotificationData{
			Type:         "notification",
			Notification: notification,
		}

		if err := client.conn.WriteJSON(notificationData); err != nil {
			log.Printf("Error sending notification to user %d: %v", userID, err)
		} else {
			log.Printf("Notification sent to user %d via WebSocket", userID)
		}
	} else {
		log.Printf("User %d not connected, notification stored in database", userID)
	}
}

// Get all notifications for the current user
func getNotificationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	if err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		log.Printf("User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	unreadOnly := r.URL.Query().Get("unread_only") == "true"

	limit := 50 // default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0 // default offset
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Build query
	query := `
		SELECT n.notification_id, n.user_id, n.type, n.message, n.read_status, n.created_at
		FROM notifications n
		WHERE n.user_id = ?
	`
	args := []interface{}{userID}

	if unreadOnly {
		query += " AND n.read_status = 0"
	}

	query += " ORDER BY n.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying notifications: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var notifications []Notification
	for rows.Next() {
		var notification Notification
		var createdAt time.Time

		if err := rows.Scan(&notification.ID, &notification.UserID, &notification.Type,
			&notification.Message, &notification.ReadStatus, &createdAt); err != nil {
			log.Printf("Error scanning notification: %v", err)
			continue
		}

		notification.CreatedAt = createdAt.Format(time.RFC3339)
		notifications = append(notifications, notification)
	}

	// Get total count
	var totalCount int
	countQuery := "SELECT COUNT(*) FROM notifications WHERE user_id = ?"
	countArgs := []interface{}{userID}

	if unreadOnly {
		countQuery += " AND read_status = 0"
	}

	db.QueryRow(countQuery, countArgs...).Scan(&totalCount)

	// Mark all retrieved notifications as read
	if len(notifications) > 0 {
		notificationIDs := make([]interface{}, len(notifications))
		for i, n := range notifications {
			notificationIDs[i] = n.ID
		}

		// Create placeholders for the IN clause
		placeholders := ""
		for i := 0; i < len(notificationIDs); i++ {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
		}

		updateQuery := "UPDATE notifications SET read_status = 1 WHERE notification_id IN (" + placeholders + ")"
		if _, err := db.Exec(updateQuery, notificationIDs...); err != nil {
			log.Printf("Error marking notifications as read: %v", err)
		} else {
			log.Printf("Marked %d notifications as read for user %d", len(notifications), userID)
		}
	}

	response := map[string]interface{}{
		"notifications": notifications,
		"total_count":   totalCount,
		"limit":         limit,
		"offset":        offset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Mark specific notification as read
func markNotificationReadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	if err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		log.Printf("User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	notificationIDStr := r.URL.Query().Get("notification_id")
	if notificationIDStr == "" {
		http.Error(w, "notification_id parameter is required", http.StatusBadRequest)
		return
	}

	notificationID, err := strconv.Atoi(notificationIDStr)
	if err != nil {
		http.Error(w, "Invalid notification_id", http.StatusBadRequest)
		return
	}

	// Update notification to mark as read, ensuring it belongs to the current user
	result, err := db.Exec(`
		UPDATE notifications 
		SET read_status = 1 
		WHERE notification_id = ? AND user_id = ?
	`, notificationID, userID)

	if err != nil {
		log.Printf("Error marking notification as read: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Notification not found or already read", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Notification marked as read"})
}

// Mark all notifications as read for current user
func markAllNotificationsReadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	if err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		log.Printf("User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	result, err := db.Exec(`
		UPDATE notifications 
		SET read_status = 1 
		WHERE user_id = ? AND read_status = 0
	`, userID)

	if err != nil {
		log.Printf("Error marking all notifications as read: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "All notifications marked as read",
		"count":   rowsAffected,
	})
}

// Get unread notification count
func getUnreadNotificationCountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	if err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		log.Printf("User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var unreadCount int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM notifications 
		WHERE user_id = ? AND read_status = 0
	`, userID).Scan(&unreadCount)

	if err != nil {
		log.Printf("Error getting unread count: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"unread_count": unreadCount,
	})
}

// Helper functions to create specific notification types

// Create follow request notification
func createFollowRequestNotification(targetUserID, requesterID int) error {
	var requesterName string
	err := db.QueryRow("SELECT nickname FROM users WHERE id = ?", requesterID).Scan(&requesterName)
	if err != nil {
		log.Printf("Error getting requester name: %v", err)
		requesterName = "Someone"
	}

	message := requesterName + " sent you a follow request"
	return createNotification(targetUserID, "follow_request", message, &requesterID, nil)
}

// Create group invitation notification
func createGroupInviteNotification(targetUserID, inviterID, groupID int) error {
	var inviterName, groupName string

	err := db.QueryRow("SELECT nickname FROM users WHERE id = ?", inviterID).Scan(&inviterName)
	if err != nil {
		log.Printf("Error getting inviter name: %v", err)
		inviterName = "Someone"
	}

	err = db.QueryRow("SELECT title FROM groups WHERE group_id = ?", groupID).Scan(&groupName)
	if err != nil {
		log.Printf("Error getting group name: %v", err)
		groupName = "a group"
	}

	message := inviterName + " invited you to join '" + groupName + "'"
	return createNotification(targetUserID, "group_invite", message, &inviterID, &groupID)
}

// Create group join request notification
func createGroupJoinRequestNotification(creatorID, requesterID, groupID int) error {
	var requesterName, groupName string

	err := db.QueryRow("SELECT nickname FROM users WHERE id = ?", requesterID).Scan(&requesterName)
	if err != nil {
		log.Printf("Error getting requester name: %v", err)
		requesterName = "Someone"
	}

	err = db.QueryRow("SELECT title FROM groups WHERE group_id = ?", groupID).Scan(&groupName)
	if err != nil {
		log.Printf("Error getting group name: %v", err)
		groupName = "your group"
	}

	message := requesterName + " requested to join '" + groupName + "'"
	return createNotification(creatorID, "group_request", message, &requesterID, &groupID)
}

// Delete notification (optional - for cleaning up old notifications)
func deleteNotificationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	if err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		log.Printf("User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	notificationIDStr := r.URL.Query().Get("notification_id")
	if notificationIDStr == "" {
		http.Error(w, "notification_id parameter is required", http.StatusBadRequest)
		return
	}

	notificationID, err := strconv.Atoi(notificationIDStr)
	if err != nil {
		http.Error(w, "Invalid notification_id", http.StatusBadRequest)
		return
	}

	result, err := db.Exec(`
		DELETE FROM notifications 
		WHERE notification_id = ? AND user_id = ?
	`, notificationID, userID)

	if err != nil {
		log.Printf("Error deleting notification: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error checking rows affected: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Notification not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Notification deleted"})
}
