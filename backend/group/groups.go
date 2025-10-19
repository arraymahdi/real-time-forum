package group

import (
	"backend/db"
	"backend/event"
	"backend/notification"
	"backend/user"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Create a new group
func CreateGroupHandler(w http.ResponseWriter, r *http.Request) {
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
	res, err := db.Instance.Exec(`INSERT INTO groups (title, description, creator_id, created_at) VALUES (?, ?, ?, ?)`,
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
	_, err = db.Instance.Exec(`INSERT INTO group_memberships (user_id, group_id, role, status, joined_at) VALUES (?, ?, 'creator', 'accepted', ?)`,
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
func BrowseGroupsHandler(w http.ResponseWriter, r *http.Request) {
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

	rows, err := db.Instance.Query(`
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
func GetUserGroupsHandler(w http.ResponseWriter, r *http.Request) {
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

	rows, err := db.Instance.Query(`
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
func InviteToGroupHandler(w http.ResponseWriter, r *http.Request) {
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
		GroupID       int `json:"group_id"`
		InvitedUserID int `json:"invited_user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if user is member of the group and can invite
	var role string
	err := db.Instance.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		req.GroupID, userID).Scan(&role)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	// Check if invited user already has a membership record
	var existingStatus string
	err = db.Instance.QueryRow("SELECT status FROM group_memberships WHERE group_id = ? AND user_id = ?",
		req.GroupID, req.InvitedUserID).Scan(&existingStatus)
	if err == nil {
		if existingStatus == "accepted" {
			http.Error(w, "User is already a member", http.StatusBadRequest)
			return
		} else if existingStatus == "invited" {
			http.Error(w, "User is already invited", http.StatusBadRequest)
			return
		} else if existingStatus == "pending" {
			http.Error(w, "User already requested to join, cannot invite", http.StatusBadRequest)
			return
		}
	}

	// Create invitation
	_, err = db.Instance.Exec(`INSERT INTO group_memberships (user_id, group_id, role, status) VALUES (?, ?, 'member', 'invited')`,
		req.InvitedUserID, req.GroupID)
	if err != nil {
		log.Printf("[Groups] Invitation failed: %v", err)
		http.Error(w, "Error sending invitation", http.StatusInternalServerError)
		return
	}

	// Create notification for group invitation
	if err := notification.CreateGroupInviteNotification(req.InvitedUserID, userID, req.GroupID); err != nil {
		log.Printf("[Groups] Notification failed: %v", err)
	}

	log.Printf("[Groups] User %d invited user %d to group %d", userID, req.InvitedUserID, req.GroupID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Invitation sent successfully"})
}

// Get single group by ID
func GetGroupByIDHandler(w http.ResponseWriter, r *http.Request) {
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

	groupIDStr := strings.TrimPrefix(r.URL.Path, "/group/")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	var group Group
	err = db.Instance.QueryRow(`
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

// Leave group
func LeaveGroupHandler(w http.ResponseWriter, r *http.Request) {
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

	// Check if user is a member and get their role
	var role string
	err := db.Instance.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
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
	_, err = db.Instance.Exec("DELETE FROM group_memberships WHERE group_id = ? AND user_id = ?", req.GroupID, userID)
	if err != nil {
		log.Printf("[Groups] Leave group failed: %v", err)
		http.Error(w, "Error leaving group", http.StatusInternalServerError)
		return
	}

	log.Printf("[Groups] User %d left group %d", userID, req.GroupID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Successfully left group"})
}

// Update group info (creators and admins only)
func UpdateGroupHandler(w http.ResponseWriter, r *http.Request) {
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
	err := db.Instance.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
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
	_, err = db.Instance.Exec("UPDATE groups SET title = ?, description = ? WHERE group_id = ?",
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

// Helper function to handle dynamic group routes
func HandleGroupDynamicRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Handle /group/{id}/members
	if strings.HasSuffix(path, "/members") && strings.HasPrefix(path, "/group/") {
		user.JwtMiddleware(GetGroupMembersHandler)(w, r)
		return
	}

	// Handle /group/{id}/posts
	if strings.HasSuffix(path, "/posts") && strings.HasPrefix(path, "/group/") {
		user.JwtMiddleware(GetGroupPostsHandler)(w, r)
		return
	}

	// Handle /group/{id}/events
	if strings.HasSuffix(path, "/events") && strings.HasPrefix(path, "/group/") {
		user.JwtMiddleware(event.GetGroupEventsHandler)(w, r)
		return
	}

	// Handle single group by ID /group/{id}
	if !strings.Contains(strings.TrimPrefix(path, "/group/"), "/") {
		user.JwtMiddleware(GetGroupByIDHandler)(w, r)
		return
	}

	http.Error(w, "Group route not found", http.StatusNotFound)
}
