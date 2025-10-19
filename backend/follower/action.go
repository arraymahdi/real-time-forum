package follower

import (
	"backend/db"
	"backend/notification"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
)

// followUserHandler handles following a user by nickname
func FollowUserHandler(w http.ResponseWriter, r *http.Request) {
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
	err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&followerID)
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
	err = db.Instance.QueryRow("SELECT id, profile_type FROM users WHERE nickname = ?", nickname).Scan(&followingID, &profileType)
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

	// 1. Check if already following each other
	var alreadyFollowing bool
	err = db.Instance.QueryRow("SELECT EXISTS(SELECT 1 FROM followers WHERE follower_id = ? AND following_id = ?)",
		followerID, followingID).Scan(&alreadyFollowing)
	if err != nil {
		log.Printf("Database error checking existing follow relationship: %v", err)
		http.Error(w, "Database error checking follow status", http.StatusInternalServerError)
		return
	}

	if alreadyFollowing {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"message": "Already following this user"})
		return
	}

	// 2. Check for existing follow request
	var existingRequestStatus string
	err = db.Instance.QueryRow("SELECT status FROM follow_requests WHERE requester_id = ? AND target_id = ?",
		followerID, followingID).Scan(&existingRequestStatus)

	if err == nil {
		// Request exists, check status
		if existingRequestStatus == "pending" {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"message": "Follow request already sent and is pending"})
			return
		} else if existingRequestStatus == "declined" || existingRequestStatus == "accepted" {
			// Remove the previous request and create a new one
			_, err = db.Instance.Exec("DELETE FROM follow_requests WHERE requester_id = ? AND target_id = ?",
				followerID, followingID)
			if err != nil {
				log.Printf("Error deleting old follow request: %v", err)
				http.Error(w, "Error creating follow request", http.StatusInternalServerError)
				return
			}

			_, err = db.Instance.Exec("INSERT INTO follow_requests (requester_id, target_id, status, created_at) VALUES (?, ?, 'pending', CURRENT_TIMESTAMP)",
				followerID, followingID)
			if err != nil {
				log.Printf("Error creating new follow request: %v", err)
				http.Error(w, "Error creating follow request", http.StatusInternalServerError)
				return
			}
		}
	} else if err != sql.ErrNoRows {
		log.Printf("Database error checking existing follow request: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	} else {
		// No request exists, create a new one
		_, err = db.Instance.Exec("INSERT INTO follow_requests (requester_id, target_id, status, created_at) VALUES (?, ?, 'pending', CURRENT_TIMESTAMP)",
			followerID, followingID)
		if err != nil {
			log.Printf("Error creating follow request: %v", err)
			http.Error(w, "Error creating follow request", http.StatusInternalServerError)
			return
		}
	}

	// 3. Handle based on profile type
	if profileType == "private" {
		// For private profiles, create or update follow request
		if err == sql.ErrNoRows {
			// No existing request, create new one
			_, err = db.Instance.Exec("INSERT INTO follow_requests (requester_id, target_id, status) VALUES (?, ?, 'pending')",
				followerID, followingID)
			if err != nil {
				log.Printf("Database error inserting follow request: %v", err)
				http.Error(w, "Error creating follow request", http.StatusInternalServerError)
				return
			}
		}
		// If we updated a declined request above, it's already handled

		// Create notification for follow request
		if err := notification.CreateFollowRequestNotification(followingID, followerID); err != nil {
			log.Printf("Error creating follow request notification: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Follow request sent"})
		return
	} else {
		// Public profile - directly follow
		_, err = db.Instance.Exec("INSERT INTO followers (follower_id, following_id, status) VALUES (?, ?, 'accepted')",
			followerID, followingID)
		if err != nil {
			log.Printf("Database error inserting follow relationship: %v", err)
			http.Error(w, "Error creating follow relationship", http.StatusInternalServerError)
			return
		}

		// If there was a follow request, mark it as accepted
		if err != sql.ErrNoRows {
			db.Instance.Exec("UPDATE follow_requests SET status = 'accepted', responded_at = CURRENT_TIMESTAMP WHERE requester_id = ? AND target_id = ?",
				followerID, followingID)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Successfully followed user"})
		return
	}
}

// Updated unfollowUserHandler with better cleanup
func UnfollowUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
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
	err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&followerID)
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
	err = db.Instance.QueryRow("SELECT id FROM users WHERE nickname = ?", nickname).Scan(&followingID)
	if err == sql.ErrNoRows {
		log.Printf("Target user not found with nickname: %s", nickname)
		http.Error(w, "Target user not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("Database error getting user by nickname '%s': %v", nickname, err)
		http.Error(w, "Database error while finding target user", http.StatusInternalServerError)
		return
	}

	// Start transaction to ensure consistency
	tx, err := db.Instance.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Delete from followers table
	result, err := tx.Exec("DELETE FROM followers WHERE follower_id = ? AND following_id = ?",
		followerID, followingID)
	if err != nil {
		log.Printf("Database error deleting follow relationship: %v", err)
		http.Error(w, "Error removing follow relationship", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Database error getting rows affected: %v", err)
		http.Error(w, "Database error checking deletion result", http.StatusInternalServerError)
		return
	}

	// Also clean up any pending follow requests (in case they exist)
	_, err = tx.Exec("DELETE FROM follow_requests WHERE requester_id = ? AND target_id = ?",
		followerID, followingID)
	if err != nil {
		log.Printf("Error cleaning up follow requests: %v", err)
		// Don't fail the request, just log the error
	}

	// Commit the transaction
	if err = tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
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
func GetFollowersHandler(w http.ResponseWriter, r *http.Request) {
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
		var nickname, firstName, lastName, email, avatar, createdAt string
		err := rows.Scan(&id, &nickname, &firstName, &lastName, &email, &avatar, &createdAt)
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
			"avatar":      avatar,
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
