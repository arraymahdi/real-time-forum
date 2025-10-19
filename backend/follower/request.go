package follower

import (
	"backend/db"
	"backend/notification"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Updated handleFollowRequestHandler with better error handling
func HandleFollowRequestHandler(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var req struct {
		RequestID int    `json:"request_id"`
		Action    string `json:"action"` // "accept" or "decline"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Action != "accept" && req.Action != "decline" {
		http.Error(w, "Action must be 'accept' or 'decline'", http.StatusBadRequest)
		return
	}

	// Start transaction
	tx, err := db.Instance.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Verify the request belongs to the current user and get requester info
	var requesterID int
	err = tx.QueryRow("SELECT requester_id FROM follow_requests WHERE id = ? AND target_id = ? AND status = 'pending'",
		req.RequestID, userID).Scan(&requesterID)
	if err != nil {
		http.Error(w, "Follow request not found", http.StatusNotFound)
		return
	}

	if req.Action == "accept" {
		// Update request status
		_, err = tx.Exec("UPDATE follow_requests SET status = 'accepted', responded_at = ? WHERE id = ?",
			time.Now().Format("2006-01-02 15:04:05"), req.RequestID)
		if err != nil {
			log.Printf("Error accepting follow request: %v", err)
			http.Error(w, "Error accepting follow request", http.StatusInternalServerError)
			return
		}

		// Add to followers table (avoid duplicate key errors)
		_, err = tx.Exec("INSERT OR IGNORE INTO followers (follower_id, following_id, status, requested_at) VALUES (?, ?, 'accepted', ?)",
			requesterID, userID, time.Now().Format("2006-01-02 15:04:05"))
		if err != nil {
			log.Printf("Error adding to followers: %v", err)
			http.Error(w, "Error adding follower relationship", http.StatusInternalServerError)
			return
		}

		// Create acceptance notification
		var accepterName string
		tx.QueryRow("SELECT nickname FROM users WHERE id = ?", userID).Scan(&accepterName)
		if accepterName == "" {
			accepterName = "Someone"
		}

		// Commit transaction first
		if err = tx.Commit(); err != nil {
			log.Printf("Error committing transaction: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		// Create notification after successful commit
		message := accepterName + " accepted your follow request"
		notification.CreateNotification(requesterID, "follow_request", message, &userID, nil)

	} else {
		// Decline request
		_, err = tx.Exec("UPDATE follow_requests SET status = 'declined', responded_at = ? WHERE id = ?",
			time.Now().Format("2006-01-02 15:04:05"), req.RequestID)
		if err != nil {
			log.Printf("Error declining follow request: %v", err)
			http.Error(w, "Error declining follow request", http.StatusInternalServerError)
			return
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			log.Printf("Error committing transaction: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Follow request " + req.Action + "ed successfully",
	})
}

// getFollowingHandler gets all users the current user is following
func GetFollowingHandler(w http.ResponseWriter, r *http.Request) {
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
	err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID)
	if err == sql.ErrNoRows {
		log.Printf("User not found with email: %s", userEmail)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("Database error getting user by email '%s': %v", userEmail, err)
		http.Error(w, "Database error while finding user", http.StatusInternalServerError)
		return
	}

	rows, err := db.Instance.Query(`
		SELECT u.id, u.nickname, u.first_name, u.last_name, u.email, u.avatar, f.requested_at
		FROM users u
		JOIN followers f ON u.id = f.following_id
		WHERE f.follower_id = ?
		ORDER BY f.requested_at DESC
	`, userID)
	if err != nil {
		log.Printf("Database error querying following for id %d: %v", userID, err)
		http.Error(w, "Database error retrieving following", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var following []map[string]interface{}
	for rows.Next() {
		var id int
		var nickname, firstName, lastName, email, avatar, createdAt string
		err := rows.Scan(&id, &nickname, &firstName, &lastName, &email, &avatar, &createdAt)
		if err != nil {
			log.Printf("Database error scanning following row: %v", err)
			http.Error(w, "Error processing following data", http.StatusInternalServerError)
			return
		}

		following = append(following, map[string]interface{}{
			"id":          id,
			"nickname":    nickname,
			"first_name":  firstName,
			"last_name":   lastName,
			"email":       email,
			"avatar":      avatar,
			"followed_at": createdAt,
		})
	}

	if err = rows.Err(); err != nil {
		log.Printf("Database error during following rows iteration: %v", err)
		http.Error(w, "Error processing following data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(following)
}

// getUserFollowStatusHandler gets a specific user's follow status relative to current user
func GetUserFollowStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	nickname := r.URL.Query().Get("nickname")
	if nickname == "" {
		http.Error(w, "Nickname parameter is required", http.StatusBadRequest)
		return
	}

	var currentUserID int
	err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&currentUserID)
	if err == sql.ErrNoRows {
		log.Printf("User not found with email: %s", userEmail)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("Database error getting user by email '%s': %v", userEmail, err)
		http.Error(w, "Database error while finding user", http.StatusInternalServerError)
		return
	}

	var targetUser struct {
		ID        int    `json:"id"`
		Nickname  string `json:"nickname"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
	}

	err = db.Instance.QueryRow("SELECT id, nickname, first_name, last_name, email FROM users WHERE nickname = ?",
		nickname).Scan(&targetUser.ID, &targetUser.Nickname, &targetUser.FirstName, &targetUser.LastName, &targetUser.Email)
	if err == sql.ErrNoRows {
		log.Printf("Target user not found with nickname: %s", nickname)
		http.Error(w, "Target user not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Database error getting target user by nickname '%s': %v", nickname, err)
		http.Error(w, "Database error while finding target user", http.StatusInternalServerError)
		return
	}

	// Check if current user follows target user
	var isFollowing bool
	err = db.Instance.QueryRow("SELECT EXISTS(SELECT 1 FROM followers WHERE follower_id = ? AND following_id = ?)",
		currentUserID, targetUser.ID).Scan(&isFollowing)
	if err != nil {
		log.Printf("Database error checking if user %d follows user %d: %v", currentUserID, targetUser.ID, err)
		http.Error(w, "Database error checking follow status", http.StatusInternalServerError)
		return
	}

	// Check if target user follows current user
	var followsYou bool
	err = db.Instance.QueryRow("SELECT EXISTS(SELECT 1 FROM followers WHERE follower_id = ? AND following_id = ?)",
		targetUser.ID, currentUserID).Scan(&followsYou)
	if err != nil {
		log.Printf("Database error checking if user %d follows user %d: %v", targetUser.ID, currentUserID, err)
		http.Error(w, "Database error checking follow back status", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"id":           targetUser.ID,
		"nickname":     targetUser.Nickname,
		"first_name":   targetUser.FirstName,
		"last_name":    targetUser.LastName,
		"email":        targetUser.Email,
		"is_following": isFollowing,
		"follows_you":  followsYou,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 3. GET pending follow requests
func GetFollowRequestsHandler(w http.ResponseWriter, r *http.Request) {
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

	// Get pending follow requests received by current user
	rows, err := db.Instance.Query(`
		SELECT fr.id, fr.requester_id, fr.created_at, u.nickname, u.first_name, u.last_name
		FROM follow_requests fr
		JOIN users u ON fr.requester_id = u.id
		WHERE fr.target_id = ? AND fr.status = 'pending'
		ORDER BY fr.created_at DESC
	`, userID)
	if err != nil {
		log.Printf("Error querying follow requests: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var requests []map[string]interface{}
	for rows.Next() {
		var id, requesterID int
		var createdAt time.Time
		var nickname, firstName, lastName string

		if err := rows.Scan(&id, &requesterID, &createdAt, &nickname, &firstName, &lastName); err != nil {
			log.Printf("Error scanning follow request: %v", err)
			continue
		}

		requests = append(requests, map[string]interface{}{
			"request_id":     id,
			"requester_id":   requesterID,
			"requester_name": nickname,
			"first_name":     firstName,
			"last_name":      lastName,
			"created_at":     createdAt.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(requests)
}

// Add this new endpoint to check if user has pending follow request to specific user
func CheckFollowRequestStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	nickname := r.URL.Query().Get("nickname")
	if nickname == "" {
		http.Error(w, "Nickname parameter is required", http.StatusBadRequest)
		return
	}

	var currentUserID int
	err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&currentUserID)
	if err != nil {
		log.Printf("User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var targetUserID int
	err = db.Instance.QueryRow("SELECT id FROM users WHERE nickname = ?", nickname).Scan(&targetUserID)
	if err != nil {
		log.Printf("Target user lookup failed: %v", err)
		http.Error(w, "Target user not found", http.StatusNotFound)
		return
	}

	// Check if there's a pending follow request
	var requestStatus string
	err = db.Instance.QueryRow("SELECT status FROM follow_requests WHERE requester_id = ? AND target_id = ?",
		currentUserID, targetUserID).Scan(&requestStatus)

	hasPendingRequest := (err == nil && requestStatus == "pending")

	response := map[string]interface{}{
		"has_pending_request": hasPendingRequest,
		"request_status":      requestStatus,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
