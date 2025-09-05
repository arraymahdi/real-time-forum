package main

import (
	// "crypto/hmac"
	// "crypto/sha256"
	"database/sql"
	// "encoding/base64"
	"encoding/json"
	// "errors"
	// "fmt"
	// "io"
	"log"
	"net/http"

	// "os"
	// "path/filepath"
	"strconv"
	"strings"
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

// Browse all groups
func browseGroupsHandler(w http.ResponseWriter, r *http.Request) {
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

	rows, err := db.Query(`
		SELECT g.group_id, g.title, g.description, g.creator_id, g.created_at, u.nickname,
		       COUNT(gm.user_id) as member_count,
		       COALESCE(um.role, '') as user_role
		FROM groups g
		JOIN users u ON g.creator_id = u.id
		LEFT JOIN group_memberships gm ON g.group_id = gm.group_id AND gm.status = 'accepted'
		LEFT JOIN group_memberships um ON g.group_id = um.group_id AND um.user_id = ? AND um.status = 'accepted'
		GROUP BY g.group_id
		ORDER BY g.created_at DESC
	`, userID)
	if err != nil {
		log.Printf("[Groups] Query failed: %v", err)
		http.Error(w, "Error retrieving groups", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var group Group
		if err := rows.Scan(&group.GroupID, &group.Title, &group.Description, &group.CreatorID,
			&group.CreatedAt, &group.CreatorName, &group.MemberCount, &group.UserRole); err != nil {
			log.Printf("[Groups] Scan failed: %v", err)
			continue
		}
		groups = append(groups, group)
	}

	log.Printf("[Groups] Returning %d groups for browsing", len(groups))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}

// Get user's groups (where user is a member)
func getUserGroupsHandler(w http.ResponseWriter, r *http.Request) {
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

	rows, err := db.Query(`
		SELECT g.group_id, g.title, g.description, g.creator_id, g.created_at, u.nickname,
		       gm.role, gm.joined_at
		FROM groups g
		JOIN users u ON g.creator_id = u.id
		JOIN group_memberships gm ON g.group_id = gm.group_id
		WHERE gm.user_id = ? AND gm.status = 'accepted'
		ORDER BY gm.joined_at DESC
	`, userID)
	if err != nil {
		log.Printf("[Groups] Query user groups failed: %v", err)
		http.Error(w, "Error retrieving user groups", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var group Group
		var joinedAt sql.NullString
		if err := rows.Scan(&group.GroupID, &group.Title, &group.Description, &group.CreatorID,
			&group.CreatedAt, &group.CreatorName, &group.UserRole, &joinedAt); err != nil {
			log.Printf("[Groups] Scan failed: %v", err)
			continue
		}
		groups = append(groups, group)
	}

	log.Printf("[Groups] Returning %d groups for user %d", len(groups), userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(groups)
}

// Invite user to group
func inviteToGroupHandler(w http.ResponseWriter, r *http.Request) {
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
		GroupID       int `json:"group_id"`
		InvitedUserID int `json:"invited_user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if user is member of the group and can invite
	var role string
	err := db.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		req.GroupID, userID).Scan(&role)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	// Check if invited user already has a membership record
	var existingStatus string
	err = db.QueryRow("SELECT status FROM group_memberships WHERE group_id = ? AND user_id = ?",
		req.GroupID, req.InvitedUserID).Scan(&existingStatus)
	if err == nil {
		if existingStatus == "accepted" {
			http.Error(w, "User is already a member", http.StatusBadRequest)
			return
		} else if existingStatus == "invited" {
			http.Error(w, "User is already invited", http.StatusBadRequest)
			return
		}
	}

	// Create invitation
	_, err = db.Exec(`INSERT INTO group_memberships (user_id, group_id, role, status) VALUES (?, ?, 'member', 'invited')`,
		req.InvitedUserID, req.GroupID)
	if err != nil {
		log.Printf("[Groups] Invitation failed: %v", err)
		http.Error(w, "Error sending invitation", http.StatusInternalServerError)
		return
	}

	// Create notification
	_, err = db.Exec(`INSERT INTO notifications (user_id, type, message) VALUES (?, 'group_invite', ?)`,
		req.InvitedUserID, "You have been invited to join a group")
	if err != nil {
		log.Printf("[Groups] Notification failed: %v", err)
	}

	log.Printf("[Groups] User %d invited user %d to group %d", userID, req.InvitedUserID, req.GroupID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Invitation sent successfully"})
}

// Accept/Reject group invitation or join request
func respondToGroupRequestHandler(w http.ResponseWriter, r *http.Request) {
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
		GroupID      int    `json:"group_id"`
		TargetUserID int    `json:"target_user_id,omitempty"` // for creators accepting join requests
		Action       string `json:"action"`                   // accept, reject
		RequestType  string `json:"request_type"`             // invitation, join_request
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Action != "accept" && req.Action != "reject" {
		http.Error(w, "Action must be 'accept' or 'reject'", http.StatusBadRequest)
		return
	}

	var targetUserID int
	var currentStatus string

	if req.RequestType == "invitation" {
		// User responding to their own invitation
		targetUserID = userID
		err := db.QueryRow("SELECT status FROM group_memberships WHERE group_id = ? AND user_id = ?",
			req.GroupID, userID).Scan(&currentStatus)
		if err != nil || currentStatus != "invited" {
			http.Error(w, "No pending invitation found", http.StatusBadRequest)
			return
		}
	} else if req.RequestType == "join_request" {
		// Creator responding to a join request
		targetUserID = req.TargetUserID

		// Check if current user is the creator
		var creatorID int
		err := db.QueryRow("SELECT creator_id FROM groups WHERE group_id = ?", req.GroupID).Scan(&creatorID)
		if err != nil || creatorID != userID {
			http.Error(w, "Only group creator can accept join requests", http.StatusForbidden)
			return
		}

		err = db.QueryRow("SELECT status FROM group_memberships WHERE group_id = ? AND user_id = ?",
			req.GroupID, targetUserID).Scan(&currentStatus)
		if err != nil || currentStatus != "pending" {
			http.Error(w, "No pending join request found", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Invalid request type", http.StatusBadRequest)
		return
	}

	// Update membership status
	if req.Action == "accept" {
		_, err := db.Exec(`UPDATE group_memberships SET status = 'accepted', joined_at = ? WHERE group_id = ? AND user_id = ?`,
			time.Now().Format("2006-01-02 15:04:05"), req.GroupID, targetUserID)
		if err != nil {
			log.Printf("[Groups] Accept failed: %v", err)
			http.Error(w, "Error accepting request", http.StatusInternalServerError)
			return
		}
	} else {
		_, err := db.Exec(`UPDATE group_memberships SET status = 'rejected' WHERE group_id = ? AND user_id = ?`,
			req.GroupID, targetUserID)
		if err != nil {
			log.Printf("[Groups] Reject failed: %v", err)
			http.Error(w, "Error rejecting request", http.StatusInternalServerError)
			return
		}
	}

	log.Printf("[Groups] User %d %sed request for user %d in group %d", userID, req.Action, targetUserID, req.GroupID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Request processed successfully"})
}

// Request to join group
func requestJoinGroupHandler(w http.ResponseWriter, r *http.Request) {
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
		GroupID int `json:"group_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if user already has a membership record
	var existingStatus string
	err := db.QueryRow("SELECT status FROM group_memberships WHERE group_id = ? AND user_id = ?",
		req.GroupID, userID).Scan(&existingStatus)
	if err == nil {
		if existingStatus == "accepted" {
			http.Error(w, "You are already a member", http.StatusBadRequest)
			return
		} else if existingStatus == "pending" {
			http.Error(w, "You have already requested to join", http.StatusBadRequest)
			return
		}
	}

	// Create join request
	_, err = db.Exec(`INSERT INTO group_memberships (user_id, group_id, role, status) VALUES (?, ?, 'member', 'pending')`,
		userID, req.GroupID)
	if err != nil {
		log.Printf("[Groups] Join request failed: %v", err)
		http.Error(w, "Error sending join request", http.StatusInternalServerError)
		return
	}

	// Notify group creator
	var creatorID int
	if err := db.QueryRow("SELECT creator_id FROM groups WHERE group_id = ?", req.GroupID).Scan(&creatorID); err == nil {
		_, err = db.Exec(`INSERT INTO notifications (user_id, type, message) VALUES (?, 'group_request', ?)`,
			creatorID, "A user has requested to join your group")
		if err != nil {
			log.Printf("[Groups] Notification failed: %v", err)
		}
	}

	log.Printf("[Groups] User %d requested to join group %d", userID, req.GroupID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Join request sent successfully"})
}

// Get pending invitations and join requests for the current user
func getPendingRequestsHandler(w http.ResponseWriter, r *http.Request) {
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

	rows, err := db.Query(query, args...)
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

// Get single group by ID
func getGroupByIDHandler(w http.ResponseWriter, r *http.Request) {
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

	groupIDStr := strings.TrimPrefix(r.URL.Path, "/group/")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	var group Group
	err = db.QueryRow(`
		SELECT g.group_id, g.title, g.description, g.creator_id, g.created_at, u.nickname,
		       COUNT(gm.user_id) as member_count,
		       COALESCE(um.role, '') as user_role
		FROM groups g
		JOIN users u ON g.creator_id = u.id
		LEFT JOIN group_memberships gm ON g.group_id = gm.group_id AND gm.status = 'accepted'
		LEFT JOIN group_memberships um ON g.group_id = um.group_id AND um.user_id = ? AND um.status = 'accepted'
		WHERE g.group_id = ?
		GROUP BY g.group_id
	`, userID, groupID).Scan(&group.GroupID, &group.Title, &group.Description, &group.CreatorID,
		&group.CreatedAt, &group.CreatorName, &group.MemberCount, &group.UserRole)

	if err == sql.ErrNoRows {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("[Groups] Query group by ID failed: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(group)
}

// Get group members
func getGroupMembersHandler(w http.ResponseWriter, r *http.Request) {
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

	groupIDStr := strings.TrimPrefix(r.URL.Path, "/group/")
	groupIDStr = strings.TrimSuffix(groupIDStr, "/members")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	// Check if user is member of the group
	var userRole string
	err = db.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		groupID, userID).Scan(&userRole)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	rows, err := db.Query(`
		SELECT gm.user_id, gm.group_id, gm.role, gm.status, gm.joined_at, u.nickname
		FROM group_memberships gm
		JOIN users u ON gm.user_id = u.id
		WHERE gm.group_id = ? AND gm.status = 'accepted'
		ORDER BY gm.role DESC, gm.joined_at ASC
	`, groupID)
	if err != nil {
		log.Printf("[Groups] Query members failed: %v", err)
		http.Error(w, "Error retrieving group members", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var members []GroupMembership
	for rows.Next() {
		var member GroupMembership
		var joinedAt sql.NullString
		if err := rows.Scan(&member.UserID, &member.GroupID, &member.Role, &member.Status,
			&joinedAt, &member.UserName); err != nil {
			log.Printf("[Groups] Scan member failed: %v", err)
			continue
		}
		if joinedAt.Valid {
			member.JoinedAt = joinedAt.String
		}
		members = append(members, member)
	}

	log.Printf("[Groups] Returning %d members for group %d", len(members), groupID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(members)
}

// Get posts for a specific group
func getGroupPostsHandler(w http.ResponseWriter, r *http.Request) {
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

	groupIDStr := strings.TrimPrefix(r.URL.Path, "/group/")
	groupIDStr = strings.TrimSuffix(groupIDStr, "/posts")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	// Check if user is member of the group
	var role string
	err = db.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		groupID, userID).Scan(&role)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	rows, err := db.Query(`
		SELECT p.post_id, p.user_id, p.group_id, p.content, p.media, p.created_at, u.nickname
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.group_id = ?
		ORDER BY p.created_at DESC
	`, groupID)
	if err != nil {
		log.Printf("[Groups] Query group posts failed: %v", err)
		http.Error(w, "Error retrieving group posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []GroupPost
	for rows.Next() {
		var post GroupPost
		if err := rows.Scan(&post.ID, &post.UserID, &post.GroupID, &post.Content,
			&post.Media, &post.CreatedAt, &post.Nickname); err != nil {
			log.Printf("[Groups] Scan group post failed: %v", err)
			continue
		}
		posts = append(posts, post)
	}

	log.Printf("[Groups] Returning %d posts for group %d", len(posts), groupID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

// Leave group
func leaveGroupHandler(w http.ResponseWriter, r *http.Request) {
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
		GroupID int `json:"group_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if user is a member and get their role
	var role string
	err := db.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		req.GroupID, userID).Scan(&role)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusBadRequest)
		return
	}

	// Creators cannot leave their own group
	if role == "creator" {
		http.Error(w, "Group creators cannot leave their own group", http.StatusBadRequest)
		return
	}

	// Remove user from group
	_, err = db.Exec("DELETE FROM group_memberships WHERE group_id = ? AND user_id = ?", req.GroupID, userID)
	if err != nil {
		log.Printf("[Groups] Leave group failed: %v", err)
		http.Error(w, "Error leaving group", http.StatusInternalServerError)
		return
	}

	log.Printf("[Groups] User %d left group %d", userID, req.GroupID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Successfully left group"})
}

// Remove member from group (creators and admins only)
func removeMemberHandler(w http.ResponseWriter, r *http.Request) {
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
		GroupID        int `json:"group_id"`
		MemberToRemove int `json:"member_to_remove"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if current user has permission to remove members
	var currentUserRole string
	err := db.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		req.GroupID, userID).Scan(&currentUserRole)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	if currentUserRole != "creator" && currentUserRole != "admin" {
		http.Error(w, "Only creators and admins can remove members", http.StatusForbidden)
		return
	}

	// Check the role of the member to be removed
	var memberRole string
	err = db.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		req.GroupID, req.MemberToRemove).Scan(&memberRole)
	if err != nil {
		http.Error(w, "Member not found in this group", http.StatusBadRequest)
		return
	}

	// Creators cannot be removed
	if memberRole == "creator" {
		http.Error(w, "Cannot remove group creator", http.StatusBadRequest)
		return
	}

	// Admins can only be removed by creators
	if memberRole == "admin" && currentUserRole != "creator" {
		http.Error(w, "Only creators can remove admins", http.StatusForbidden)
		return
	}

	// Remove member from group
	_, err = db.Exec("DELETE FROM group_memberships WHERE group_id = ? AND user_id = ?", req.GroupID, req.MemberToRemove)
	if err != nil {
		log.Printf("[Groups] Remove member failed: %v", err)
		http.Error(w, "Error removing member", http.StatusInternalServerError)
		return
	}

	log.Printf("[Groups] User %d removed user %d from group %d", userID, req.MemberToRemove, req.GroupID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Member removed successfully"})
}

// Promote member to admin (creators only)
func promoteMemberHandler(w http.ResponseWriter, r *http.Request) {
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
		GroupID         int    `json:"group_id"`
		MemberToPromote int    `json:"member_to_promote"`
		NewRole         string `json:"new_role"` // admin or member
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.NewRole != "admin" && req.NewRole != "member" {
		http.Error(w, "New role must be 'admin' or 'member'", http.StatusBadRequest)
		return
	}

	// Check if current user is the creator
	var currentUserRole string
	err := db.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		req.GroupID, userID).Scan(&currentUserRole)
	if err != nil || currentUserRole != "creator" {
		http.Error(w, "Only group creators can change member roles", http.StatusForbidden)
		return
	}

	// Check if target member exists in group
	var memberRole string
	err = db.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		req.GroupID, req.MemberToPromote).Scan(&memberRole)
	if err != nil {
		http.Error(w, "Member not found in this group", http.StatusBadRequest)
		return
	}

	if memberRole == "creator" {
		http.Error(w, "Cannot change creator role", http.StatusBadRequest)
		return
	}

	// Update member role
	_, err = db.Exec("UPDATE group_memberships SET role = ? WHERE group_id = ? AND user_id = ?",
		req.NewRole, req.GroupID, req.MemberToPromote)
	if err != nil {
		log.Printf("[Groups] Promote member failed: %v", err)
		http.Error(w, "Error updating member role", http.StatusInternalServerError)
		return
	}

	log.Printf("[Groups] User %d changed role of user %d to %s in group %d", userID, req.MemberToPromote, req.NewRole, req.GroupID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Member role updated successfully"})
}

// Update group info (creators and admins only)
func updateGroupHandler(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("[Groups] User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	var req struct {
		GroupID     int    `json:"group_id"`
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

	// Check if user has permission to update group
	var role string
	err := db.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		req.GroupID, userID).Scan(&role)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	if role != "creator" && role != "admin" {
		http.Error(w, "Only creators and admins can update group info", http.StatusForbidden)
		return
	}

	// Update group
	_, err = db.Exec("UPDATE groups SET title = ?, description = ? WHERE group_id = ?",
		req.Title, req.Description, req.GroupID)
	if err != nil {
		log.Printf("[Groups] Update group failed: %v", err)
		http.Error(w, "Error updating group", http.StatusInternalServerError)
		return
	}

	log.Printf("[Groups] User %d updated group %d", userID, req.GroupID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Group updated successfully"})
}

// Get events for a group
func getGroupEventsHandler(w http.ResponseWriter, r *http.Request) {
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

	groupIDStr := strings.TrimPrefix(r.URL.Path, "/group/")
	groupIDStr = strings.TrimSuffix(groupIDStr, "/events")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	// Check if user is member of the group
	var role string
	err = db.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		groupID, userID).Scan(&role)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	rows, err := db.Query(`
		SELECT e.event_id, e.group_id, e.creator_id, e.title, e.description, e.event_time, e.created_at, u.nickname,
		       COALESCE(er.response, '') as user_response
		FROM events e
		JOIN users u ON e.creator_id = u.id
		LEFT JOIN event_responses er ON e.event_id = er.event_id AND er.user_id = ?
		WHERE e.group_id = ?
		ORDER BY e.event_time ASC
	`, userID, groupID)
	if err != nil {
		log.Printf("[Groups] Query events failed: %v", err)
		http.Error(w, "Error retrieving events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.EventID, &event.GroupID, &event.CreatorID, &event.Title,
			&event.Description, &event.EventTime, &event.CreatedAt, &event.CreatorName, &event.UserResponse); err != nil {
			log.Printf("[Groups] Scan event failed: %v", err)
			continue
		}
		events = append(events, event)
	}

	log.Printf("[Groups] Returning %d events for group %d", len(events), groupID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// Helper function to handle dynamic group routes
func handleGroupDynamicRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Handle /group/{id}/members
	if strings.HasSuffix(path, "/members") && strings.HasPrefix(path, "/group/") {
		jwtMiddleware(getGroupMembersHandler)(w, r)
		return
	}

	// Handle /group/{id}/posts
	if strings.HasSuffix(path, "/posts") && strings.HasPrefix(path, "/group/") {
		jwtMiddleware(getGroupPostsHandler)(w, r)
		return
	}

	// Handle /group/{id}/events
	if strings.HasSuffix(path, "/events") && strings.HasPrefix(path, "/group/") {
		jwtMiddleware(getGroupEventsHandler)(w, r)
		return
	}

	// Handle single group by ID /group/{id}
	if !strings.Contains(strings.TrimPrefix(path, "/group/"), "/") {
		jwtMiddleware(getGroupByIDHandler)(w, r)
		return
	}

	http.Error(w, "Group route not found", http.StatusNotFound)
}
