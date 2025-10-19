package notification

import (
	"backend/chat"
	"backend/db"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

// Create a new notification and send it via WebSocket
func CreateNotification(userID int, notificationType, message string, relatedUserID, relatedGroupID *int) error {
	// Insert notification into database
	result, err := db.Instance.Exec(`
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
		if err := db.Instance.QueryRow("SELECT nickname FROM users WHERE id = ?", *relatedUserID).Scan(&senderName); err == nil {
			notification.SenderName = senderName
		}
	}

	if relatedGroupID != nil {
		notification.RelatedGroupID = relatedGroupID
		// Get group name
		var groupName string
		if err := db.Instance.QueryRow("SELECT title FROM groups WHERE group_id = ?", *relatedGroupID).Scan(&groupName); err == nil {
			notification.GroupName = groupName
		}
	}

	// Send notification via WebSocket
	SendNotificationViaWebSocket(userID, notification)

	return nil
}

// Send notification to user via WebSocket
func SendNotificationViaWebSocket(userID int, notification Notification) {
	chat.ClientsMux.Lock()
	client, exists := chat.Clients[userID]
	chat.ClientsMux.Unlock()

	if exists {
		notificationData := NotificationData{
			Type:         "notification",
			Notification: notification,
		}

		if err := client.Conn.WriteJSON(notificationData); err != nil {
			log.Printf("Error sending notification to user %d: %v", userID, err)
		} else {
			log.Printf("Notification sent to user %d via WebSocket", userID)
		}
	} else {
		log.Printf("User %d not connected, notification stored in database", userID)
	}
}

// Get all notifications for the current user
func GetNotificationsHandler(w http.ResponseWriter, r *http.Request) {
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
	if err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
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

	rows, err := db.Instance.Query(query, args...)
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

	db.Instance.QueryRow(countQuery, countArgs...).Scan(&totalCount)

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

		// FIXED: Added user_id check for performance and security
		updateQuery := "UPDATE notifications SET read_status = 1 WHERE notification_id IN (" + placeholders + ") AND user_id = ?"
		notificationIDs = append(notificationIDs, userID)

		if _, err := db.Instance.Exec(updateQuery, notificationIDs...); err != nil {
			log.Printf("Error marking notifications as read: %v", err)
		} else {
			log.Printf("Marked %d notifications as read for user %d", len(notifications)-1, userID)
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
func MarkNotificationReadHandler(w http.ResponseWriter, r *http.Request) {
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
	if err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
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
	result, err := db.Instance.Exec(`
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
func MarkAllNotificationsReadHandler(w http.ResponseWriter, r *http.Request) {
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
	if err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		log.Printf("User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	result, err := db.Instance.Exec(`
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
func GetUnreadNotificationCountHandler(w http.ResponseWriter, r *http.Request) {
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
	if err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		log.Printf("User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var unreadCount int
	err := db.Instance.QueryRow(`
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

// Delete notification (optional - for cleaning up old notifications)
func DeleteNotificationHandler(w http.ResponseWriter, r *http.Request) {
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
	if err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
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

	result, err := db.Instance.Exec(`
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
