package main

import (
	// "crypto/hmac"
	// "crypto/sha256"
	// "database/sql"
	// "encoding/base64"
	"encoding/json"
	// "errors"
	// "fmt"
	// "io"
	"log"
	"net/http"

	// "os"
	// "path/filepath"
	// "strconv"
	// "strings"
	"time"
)

// Global variables (should be initialized elsewhere in your main application)
// var (
// 	db        *sql.DB
// 	jwtSecret = []byte("your-secret-key") // Should be loaded from environment
// )

// // Helper functions for JWT (matching your existing implementation)
// func base64Decode(s string) ([]byte, error) {
// 	// Add padding if necessary
// 	switch len(s) % 4 {
// 	case 2:
// 		s += "=="
// 	case 3:
// 		s += "="
// 	}
// 	return base64.URLEncoding.DecodeString(s)
// }

// func verifyHMACSHA256(message, key, signature []byte) bool {
// 	mac := hmac.New(sha256.New, key)
// 	mac.Write(message)
// 	expectedSignature := mac.Sum(nil)
// 	return hmac.Equal(signature, expectedSignature)
// }

// Group models aligned with schema
type Group struct {
	GroupID     int    `json:"group_id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	CreatorID   int    `json:"creator_id"`
	CreatedAt   string `json:"created_at"`
	CreatorName string `json:"creator_name,omitempty"`
	MemberCount int    `json:"member_count,omitempty"`
	UserRole    string `json:"user_role,omitempty"` // creator, admin, member, or null if not a member
}

type GroupMembership struct {
	UserID   int    `json:"user_id"`
	GroupID  int    `json:"group_id"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	JoinedAt string `json:"joined_at,omitempty"`
	UserName string `json:"user_name,omitempty"`
}

type Event struct {
	EventID      int    `json:"event_id"`
	GroupID      int    `json:"group_id"`
	CreatorID    int    `json:"creator_id"`
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	EventTime    string `json:"event_time"`
	CreatedAt    string `json:"created_at"`
	CreatorName  string `json:"creator_name,omitempty"`
	UserResponse string `json:"user_response,omitempty"` // going, not_going, or null
}

type EventResponse struct {
	EventID     int    `json:"event_id"`
	UserID      int    `json:"user_id"`
	Response    string `json:"response"`
	RespondedAt string `json:"responded_at"`
	UserName    string `json:"user_name,omitempty"`
}

type GroupPost struct {
	ID        int    `json:"post_id"`
	UserID    int    `json:"user_id"`
	GroupID   int    `json:"group_id"`
	Content   string `json:"content"`
	Media     string `json:"media,omitempty"`
	CreatedAt string `json:"created_at"`
	Nickname  string `json:"nickname,omitempty"`
}

// Create a new group
func createGroupHandler(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("[Groups] User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	// Insert group
	res, err := db.Exec(`INSERT INTO groups (title, description, creator_id, created_at) VALUES (?, ?, ?, ?)`,
		req.Title, req.Description, userID, time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		log.Printf("[Groups] Insert failed: %v", err)
		http.Error(w, "Error creating group", http.StatusInternalServerError)
		return
	}

	groupID, err := res.LastInsertId()
	if err != nil {
		log.Printf("[Groups] Getting group ID failed: %v", err)
		http.Error(w, "Error creating group", http.StatusInternalServerError)
		return
	}

	// Add creator as member with creator role
	_, err = db.Exec(`INSERT INTO group_memberships (user_id, group_id, role, status, joined_at) VALUES (?, ?, 'creator', 'accepted', ?)`,
		userID, groupID, time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		log.Printf("[Groups] Adding creator membership failed: %v", err)
		http.Error(w, "Error creating group", http.StatusInternalServerError)
		return
	}

	group := Group{
		GroupID:     int(groupID),
		Title:       req.Title,
		Description: req.Description,
		CreatorID:   userID,
		UserRole:    "creator",
	}

	log.Printf("[Groups] User %d created new group (ID: %d)", userID, groupID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(group)
}
