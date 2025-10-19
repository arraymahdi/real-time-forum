package notification

import (
	"backend/db"
	"log"
)

// Create follow request notification
func CreateFollowRequestNotification(targetUserID, requesterID int) error {
	var requesterName string
	err := db.Instance.QueryRow("SELECT nickname FROM users WHERE id = ?", requesterID).Scan(&requesterName)
	if err != nil {
		log.Printf("Error getting requester name: %v", err)
		requesterName = "Someone"
	}

	message := requesterName + " sent you a follow request"
	return CreateNotification(targetUserID, "follow_request", message, &requesterID, nil)
}
