package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Follower represents a follow relationship
type Follower struct {
	ID          int    `json:"id"`
	FollowerID  int    `json:"follower_id"`
	FollowingID int    `json:"following_id"`
	CreatedAt   string `json:"created_at"`
}

// UserWithFollowStatus represents a user with follow status
type UserWithFollowStatus struct {
	ID          int    `json:"id"`
	Nickname    string `json:"nickname"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	IsFollowing bool   `json:"is_following"`
	FollowsYou  bool   `json:"follows_you"`
}

// followUserHandler handles following a user by nickname
func followUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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

	var followerID int
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&followerID)
	if err == sql.ErrNoRows {
		log.Printf("User not found with email: %s", userEmail)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("Database error getting user by email '%s': %v", userEmail, err)
		http.Error(w, "Database error while finding user", http.StatusInternalServerError)
		return
	}

	var followingID int
	var profileType string
	err = db.QueryRow("SELECT id, profile_type FROM users WHERE nickname = ?", nickname).Scan(&followingID, &profileType)
	if err == sql.ErrNoRows {
		log.Printf("Target user not found with nickname: %s", nickname)
		http.Error(w, "Target user not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Database error getting user by nickname '%s': %v", nickname, err)
		http.Error(w, "Database error while finding target user", http.StatusInternalServerError)
		return
	}

	if followerID == followingID {
		http.Error(w, "Cannot follow yourself", http.StatusBadRequest)
		return
	}

	// Check if already following
	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM followers WHERE follower_id = ? AND following_id = ?)",
		followerID, followingID).Scan(&exists)
	if err != nil {
		log.Printf("Database error checking existing follow relationship: %v", err)
		http.Error(w, "Database error checking follow status", http.StatusInternalServerError)
		return
	}

	if exists {
		http.Error(w, "Already following this user", http.StatusConflict)
		return
	}

	// Check if profile is private - if so, create follow request instead
	if profileType == "private" {
		// Insert follow request
		_, err = db.Exec("INSERT INTO follow_requests (requester_id, target_id, status) VALUES (?, ?, 'pending')",
			followerID, followingID)
		if err != nil {
			log.Printf("Database error inserting follow request: %v", err)
			http.Error(w, "Error creating follow request", http.StatusInternalServerError)
			return
		}

		// Create notification for follow request
		if err := createFollowRequestNotification(followingID, followerID); err != nil {
			log.Printf("Error creating follow request notification: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Follow request sent"})
		return
	}

	// Public profile - directly follow
	_, err = db.Exec("INSERT INTO followers (follower_id, following_id, status) VALUES (?, ?, 'accepted')",
		followerID, followingID)
	if err != nil {
		log.Printf("Database error inserting follow relationship: %v", err)
		http.Error(w, "Error creating follow relationship", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Successfully followed user"})
}

// unfollowUserHandler handles unfollowing a user by nickname
func unfollowUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Get current user email from JWT
	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get nickname from query parameter
	nickname := r.URL.Query().Get("nickname")
	if nickname == "" {
		http.Error(w, "Nickname parameter is required", http.StatusBadRequest)
		return
	}

	// Get current user ID
	var followerID int
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&followerID)
	if err == sql.ErrNoRows {
		log.Printf("User not found with email: %s", userEmail)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("Database error getting user by email '%s': %v", userEmail, err)
		http.Error(w, "Database error while finding user", http.StatusInternalServerError)
		return
	}

	// Get target user ID by nickname
	var followingID int
	err = db.QueryRow("SELECT id FROM users WHERE nickname = ?", nickname).Scan(&followingID)
	if err == sql.ErrNoRows {
		log.Printf("Target user not found with nickname: %s", nickname)
		http.Error(w, "Target user not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Database error getting user by nickname '%s': %v", nickname, err)
		http.Error(w, "Database error while finding target user", http.StatusInternalServerError)
		return
	}

	// Delete follow relationship
	result, err := db.Exec("DELETE FROM followers WHERE follower_id = ? AND following_id = ?",
		followerID, followingID)
	if err != nil {
		log.Printf("Database error deleting follow relationship (follower_id=%d, following_id=%d): %v",
			followerID, followingID, err)
		http.Error(w, "Error removing follow relationship", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Database error getting rows affected: %v", err)
		http.Error(w, "Database error checking deletion result", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Not following this user", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Successfully unfollowed user"})
}

// getFollowersHandler gets all followers of the current user
func getFollowersHandler(w http.ResponseWriter, r *http.Request) {
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
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID)
	if err == sql.ErrNoRows {
		log.Printf("User not found with email: %s", userEmail)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("Database error getting user by email '%s': %v", userEmail, err)
		http.Error(w, "Database error while finding user", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query(`
		SELECT u.id, u.nickname, u.first_name, u.last_name, u.email, f.requested_at
		FROM users u
		JOIN followers f ON u.id = f.follower_id
		WHERE f.following_id = ?
		ORDER BY f.requested_at DESC
	`, userID)
	if err != nil {
		log.Printf("Database error querying followers for id %d: %v", userID, err)
		http.Error(w, "Database error retrieving followers", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var followers []map[string]interface{}
	for rows.Next() {
		var id int
		var nickname, firstName, lastName, email, createdAt string
		err := rows.Scan(&id, &nickname, &firstName, &lastName, &email, &createdAt)
		if err != nil {
			log.Printf("Database error scanning follower row: %v", err)
			http.Error(w, "Error processing followers data", http.StatusInternalServerError)
			return
		}

		followers = append(followers, map[string]interface{}{
			"id":          id,
			"nickname":    nickname,
			"first_name":  firstName,
			"last_name":   lastName,
			"email":       email,
			"followed_at": createdAt,
		})
	}

	if err = rows.Err(); err != nil {
		log.Printf("Database error during follower rows iteration: %v", err)
		http.Error(w, "Error processing followers data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(followers)
}

// getFollowingHandler gets all users the current user is following
func getFollowingHandler(w http.ResponseWriter, r *http.Request) {
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
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID)
	if err == sql.ErrNoRows {
		log.Printf("User not found with email: %s", userEmail)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("Database error getting user by email '%s': %v", userEmail, err)
		http.Error(w, "Database error while finding user", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query(`
		SELECT u.id, u.nickname, u.first_name, u.last_name, u.email, f.requested_at
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
		var nickname, firstName, lastName, email, createdAt string
		err := rows.Scan(&id, &nickname, &firstName, &lastName, &email, &createdAt)
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
func getUserFollowStatusHandler(w http.ResponseWriter, r *http.Request) {
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
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&currentUserID)
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

	err = db.QueryRow("SELECT id, nickname, first_name, last_name, email FROM users WHERE nickname = ?",
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
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM followers WHERE follower_id = ? AND following_id = ?)",
		currentUserID, targetUser.ID).Scan(&isFollowing)
	if err != nil {
		log.Printf("Database error checking if user %d follows user %d: %v", currentUserID, targetUser.ID, err)
		http.Error(w, "Database error checking follow status", http.StatusInternalServerError)
		return
	}

	// Check if target user follows current user
	var followsYou bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM followers WHERE follower_id = ? AND following_id = ?)",
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

func handleFollowRequestHandler(w http.ResponseWriter, r *http.Request) {
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
	if err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
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

	// Verify the request belongs to the current user
	var requesterID int
	err := db.QueryRow("SELECT requester_id FROM follow_requests WHERE id = ? AND target_id = ? AND status = 'pending'",
		req.RequestID, userID).Scan(&requesterID)
	if err != nil {
		http.Error(w, "Follow request not found", http.StatusNotFound)
		return
	}

	if req.Action == "accept" {
		// Update request status
		_, err = db.Exec("UPDATE follow_requests SET status = 'accepted', responded_at = ? WHERE id = ?",
			time.Now().Format("2006-01-02 15:04:05"), req.RequestID)
		if err != nil {
			log.Printf("Error accepting follow request: %v", err)
			http.Error(w, "Error accepting follow request", http.StatusInternalServerError)
			return
		}

		// Add to followers table
		_, err = db.Exec("INSERT INTO followers (follower_id, following_id, status) VALUES (?, ?, 'accepted')",
			requesterID, userID)
		if err != nil {
			log.Printf("Error adding to followers: %v", err)
		}

		// Create acceptance notification
		var accepterName string
		db.QueryRow("SELECT nickname FROM users WHERE id = ?", userID).Scan(&accepterName)
		message := accepterName + " accepted your follow request"
		createNotification(requesterID, "follow_request", message, &userID, nil)

	} else {
		// Decline request
		_, err = db.Exec("UPDATE follow_requests SET status = 'declined', responded_at = ? WHERE id = ?",
			time.Now().Format("2006-01-02 15:04:05"), req.RequestID)
		if err != nil {
			log.Printf("Error declining follow request: %v", err)
			http.Error(w, "Error declining follow request", http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Follow request " + req.Action + "ed successfully",
	})
}

// 3. GET pending follow requests
func getFollowRequestsHandler(w http.ResponseWriter, r *http.Request) {
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

	// Get pending follow requests received by current user
	rows, err := db.Query(`
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
