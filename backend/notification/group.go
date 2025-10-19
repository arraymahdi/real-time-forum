package notification

import (
	"backend/db"
	"log"
)

// Create group invitation notification
func CreateGroupInviteNotification(targetUserID, inviterID, groupID int) error {
	var inviterName, groupName string

	err := db.Instance.QueryRow("SELECT nickname FROM users WHERE id = ?", inviterID).Scan(&inviterName)
	if err != nil {
		log.Printf("Error getting inviter name: %v", err)
		inviterName = "Someone"
	}

	err = db.Instance.QueryRow("SELECT title FROM groups WHERE group_id = ?", groupID).Scan(&groupName)
	if err != nil {
		log.Printf("Error getting group name: %v", err)
		groupName = "a group"
	}

	message := inviterName + " invited you to join '" + groupName + "'"
	return CreateNotification(targetUserID, "group_invite", message, &inviterID, &groupID)
}

// Create group join request notification
func CreateGroupJoinRequestNotification(creatorID, requesterID, groupID int) error {
	var requesterName, groupName string

	err := db.Instance.QueryRow("SELECT nickname FROM users WHERE id = ?", requesterID).Scan(&requesterName)
	if err != nil {
		log.Printf("Error getting requester name: %v", err)
		requesterName = "Someone"
	}

	err = db.Instance.QueryRow("SELECT title FROM groups WHERE group_id = ?", groupID).Scan(&groupName)
	if err != nil {
		log.Printf("Error getting group name: %v", err)
		groupName = "your group"
	}

	message := requesterName + " requested to join '" + groupName + "'"
	return CreateNotification(creatorID, "group_request", message, &requesterID, &groupID)
}

// Create group event notification
func CreateGroupEventNotification(groupID int, eventTitle string, excludeUserID int) error {
	// Get all group members except the event creator
	rows, err := db.Instance.Query(`
		SELECT user_id FROM group_memberships 
		WHERE group_id = ? AND status = 'accepted' AND user_id != ?
	`, groupID, excludeUserID)
	if err != nil {
		log.Printf("Error getting group members for event notification: %v", err)
		return err
	}
	defer rows.Close()

	message := "New event created: " + eventTitle
	for rows.Next() {
		var userID int
		if err := rows.Scan(&userID); err == nil {
			CreateNotification(userID, "group_event", message, &excludeUserID, &groupID)
		}
	}

	return nil
}
