package group

import (
	"backend/db"
	"backend/notification"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Request to join a group
func RequestJoinGroupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
		log.Printf("[Groups] User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var req struct {
		GroupID int `json:"group_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if user already has a membership record
	var existingStatus string
	err := db.Instance.QueryRow("SELECT status FROM group_memberships WHERE group_id = ? AND user_id = ?",
		req.GroupID, userID).Scan(&existingStatus)
	if err == nil {
		if existingStatus == "accepted" {
			http.Error(w, "You are already a member", http.StatusBadRequest)
			return
		} else if existingStatus == "pending" {
			http.Error(w, "You have already requested to join", http.StatusBadRequest)
			return
		} else if existingStatus == "invited" {
			http.Error(w, "You have already been invited, cannot request to join", http.StatusBadRequest)
			return
		}
	}

	// Create join request
	_, err = db.Instance.Exec(`INSERT INTO group_memberships (user_id, group_id, role, status) VALUES (?, ?, 'member', 'pending')`,
		userID, req.GroupID)
	if err != nil {
		log.Printf("[Groups] Join request failed: %v", err)
		http.Error(w, "Error sending join request", http.StatusInternalServerError)
		return
	}

	// Notify group creator with notification
	var creatorID int
	if err := db.Instance.QueryRow("SELECT creator_id FROM groups WHERE group_id = ?", req.GroupID).Scan(&creatorID); err == nil {
		if err := notification.CreateGroupJoinRequestNotification(creatorID, userID, req.GroupID); err != nil {
			log.Printf("[Groups] Notification failed: %v", err)
		}
	}

	log.Printf("[Groups] User %d requested to join group %d", userID, req.GroupID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Join request sent successfully"})
}

// Accept/Reject group invitation or join request
func RespondToGroupRequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
		log.Printf("[Groups] User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var req struct {
		GroupID      int    `json:"group_id"`
		TargetUserID int    `json:"target_user_id,omitempty"`
		Action       string `json:"action"`
		RequestType  string `json:"request_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Action != "accept" && req.Action != "reject" {
		http.Error(w, "Action must be 'accept' or 'reject'", http.StatusBadRequest)
		return
	}

	// Start transaction
	tx, err := db.Instance.Begin()
	if err != nil {
		log.Printf("[Groups] Error starting transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var targetUserID int
	var notifyUserID int
	var currentStatus string

	if req.RequestType == "invitation" {
		// User responding to their own invitation
		targetUserID = userID

		// Verify invitation exists
		err := tx.QueryRow("SELECT status FROM group_memberships WHERE group_id = ? AND user_id = ?",
			req.GroupID, userID).Scan(&currentStatus)
		if err != nil || currentStatus != "invited" {
			http.Error(w, "No pending invitation found", http.StatusBadRequest)
			return
		}

		// Get the group creator who should be notified
		err = tx.QueryRow("SELECT creator_id FROM groups WHERE group_id = ?", req.GroupID).Scan(&notifyUserID)
		if err != nil {
			log.Printf("[Groups] Error getting group creator: %v", err)
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

	} else if req.RequestType == "join_request" {
		// Creator responding to a join request
		targetUserID = req.TargetUserID
		notifyUserID = targetUserID // Notify the requester

		// Verify current user is the creator
		var creatorID int
		err := tx.QueryRow("SELECT creator_id FROM groups WHERE group_id = ?", req.GroupID).Scan(&creatorID)
		if err != nil || creatorID != userID {
			http.Error(w, "Only group creator can accept/reject join requests", http.StatusForbidden)
			return
		}

		// Verify request exists
		err = tx.QueryRow("SELECT status FROM group_memberships WHERE group_id = ? AND user_id = ?",
			req.GroupID, targetUserID).Scan(&currentStatus)
		if err != nil || currentStatus != "pending" {
			http.Error(w, "No pending join request found", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Invalid request type", http.StatusBadRequest)
		return
	}

	// Get group name for notification
	var groupName string
	if err := tx.QueryRow("SELECT title FROM groups WHERE group_id = ?", req.GroupID).Scan(&groupName); err != nil {
		log.Printf("[Groups] Error getting group name: %v", err)
		groupName = "the group"
	}

	log.Printf("[Groups] Processing %s for group %d, target user %d, notify user %d",
		req.Action, req.GroupID, targetUserID, notifyUserID)

	if req.Action == "accept" {
		// Accept - update membership
		_, err := tx.Exec(
			`UPDATE group_memberships SET status = 'accepted', joined_at = ? WHERE group_id = ? AND user_id = ?`,
			time.Now().Format("2006-01-02 15:04:05"),
			req.GroupID, targetUserID,
		)
		if err != nil {
			log.Printf("[Groups] Accept failed: %v", err)
			http.Error(w, "Error accepting request", http.StatusInternalServerError)
			return
		}

		// Commit transaction first
		if err = tx.Commit(); err != nil {
			log.Printf("[Groups] Error committing transaction: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Create notification AFTER successful commit
		var message string
		if req.RequestType == "invitation" {
			// Notify the inviter (creator) that invitation was accepted
			var accepterName string
			if err := db.Instance.QueryRow("SELECT nickname FROM users WHERE id = ?", userID).Scan(&accepterName); err == nil {
				message = accepterName + " accepted your invitation to join '" + groupName + "'"
			} else {
				message = "Someone accepted your invitation to join '" + groupName + "'"
			}
		} else {
			// Notify the requester that their request was accepted
			message = "Your request to join '" + groupName + "' was accepted"
		}

		log.Printf("[Groups] Sending notification to user %d: %s", notifyUserID, message)
		if err := notification.CreateNotification(notifyUserID, "other", message, &userID, &req.GroupID); err != nil {
			log.Printf("[Groups] ERROR creating notification: %v", err)
		} else {
			log.Printf("[Groups] SUCCESS: Notification sent to user %d", notifyUserID)
		}

	} else {
		// Reject - delete membership
		_, err := tx.Exec(`DELETE FROM group_memberships WHERE group_id = ? AND user_id = ?`,
			req.GroupID, targetUserID)
		if err != nil {
			log.Printf("[Groups] Reject failed: %v", err)
			http.Error(w, "Error rejecting request", http.StatusInternalServerError)
			return
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			log.Printf("[Groups] Error committing transaction: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Create notification AFTER successful commit
		var message string
		if req.RequestType == "invitation" {
			// Notify the inviter that invitation was declined
			var declinerName string
			if err := db.Instance.QueryRow("SELECT nickname FROM users WHERE id = ?", userID).Scan(&declinerName); err == nil {
				message = declinerName + " declined your invitation to join '" + groupName + "'"
			} else {
				message = "Someone declined your invitation to join '" + groupName + "'"
			}
		} else {
			// Notify the requester that their request was rejected
			message = "Your request to join '" + groupName + "' was rejected"
		}

		log.Printf("[Groups] Sending notification to user %d: %s", notifyUserID, message)
		if err := notification.CreateNotification(notifyUserID, "other", message, &userID, &req.GroupID); err != nil {
			log.Printf("[Groups] ERROR creating notification: %v", err)
		}
	}

	log.Printf("[Groups] User %d %sed request for user %d in group %d", userID, req.Action, targetUserID, req.GroupID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Request " + req.Action + "ed successfully",
	})
}

// Get pending invitations and join requests for the current user
func GetPendingRequestsHandler(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	if err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		log.Printf("[Groups] User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	requestType := r.URL.Query().Get("type") // "invitations" or "requests"

	var query string
	var args []interface{}

	if requestType == "invitations" {
		// Get invitations for the current user
		query = `
			SELECT gm.user_id, gm.group_id, gm.role, gm.status, g.title, g.description, u.nickname
			FROM group_memberships gm
			JOIN groups g ON gm.group_id = g.group_id
			JOIN users u ON g.creator_id = u.id
			WHERE gm.user_id = ? AND gm.status = 'invited'
			ORDER BY gm.joined_at DESC
		`
		args = []interface{}{userID}
	} else if requestType == "requests" {
		// Get join requests for groups created by the current user
		query = `
			SELECT gm.user_id, gm.group_id, gm.role, gm.status, g.title, g.description, u.nickname
			FROM group_memberships gm
			JOIN groups g ON gm.group_id = g.group_id
			JOIN users u ON gm.user_id = u.id
			WHERE g.creator_id = ? AND gm.status = 'pending'
			ORDER BY gm.joined_at DESC
		`
		args = []interface{}{userID}
	} else {
		http.Error(w, "Invalid request type. Use 'invitations' or 'requests'", http.StatusBadRequest)
		return
	}

	rows, err := db.Instance.Query(query, args...)
	if err != nil {
		log.Printf("[Groups] Query pending requests failed: %v", err)
		http.Error(w, "Error retrieving pending requests", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var requests []map[string]interface{}
	for rows.Next() {
		var userIDVal, groupID int
		var role, status, title, description, userName string
		if err := rows.Scan(&userIDVal, &groupID, &role, &status, &title, &description, &userName); err != nil {
			log.Printf("[Groups] Scan pending request failed: %v", err)
			continue
		}

		request := map[string]interface{}{
			"user_id":           userIDVal,
			"group_id":          groupID,
			"role":              role,
			"status":            status,
			"group_title":       title,
			"group_description": description,
			"user_name":         userName,
		}
		requests = append(requests, request)
	}

	log.Printf("[Groups] Returning %d pending %s for user %d", len(requests), requestType, userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requests)
}
