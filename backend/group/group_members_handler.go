package group

import (
	"backend/db"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// Remove member from group (creators and admins only)
func RemoveMemberHandler(w http.ResponseWriter, r *http.Request) {
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
		GroupID        int `json:"group_id"`
		MemberToRemove int `json:"member_to_remove"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check if current user has permission to remove members
	var currentUserRole string
	err := db.Instance.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
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
	err = db.Instance.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
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
	_, err = db.Instance.Exec("DELETE FROM group_memberships WHERE group_id = ? AND user_id = ?", req.GroupID, req.MemberToRemove)
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
func PromoteMemberHandler(w http.ResponseWriter, r *http.Request) {
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
	err := db.Instance.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		req.GroupID, userID).Scan(&currentUserRole)
	if err != nil || currentUserRole != "creator" {
		http.Error(w, "Only group creators can change member roles", http.StatusForbidden)
		return
	}

	// Check if target member exists in group
	var memberRole string
	err = db.Instance.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
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
	_, err = db.Instance.Exec("UPDATE group_memberships SET role = ? WHERE group_id = ? AND user_id = ?",
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

// Get group members
func GetGroupMembersHandler(w http.ResponseWriter, r *http.Request) {
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
	groupIDStr = strings.TrimSuffix(groupIDStr, "/members")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	// Check if user is member of the group
	var userRole string
	err = db.Instance.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		groupID, userID).Scan(&userRole)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	rows, err := db.Instance.Query(`
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
func GetGroupPostsHandler(w http.ResponseWriter, r *http.Request) {
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
	groupIDStr = strings.TrimSuffix(groupIDStr, "/posts")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	// Check if user is member of the group
	var role string
	err = db.Instance.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		groupID, userID).Scan(&role)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	rows, err := db.Instance.Query(`
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

// Get membership status for all groups (for current user)
func GetMembershipStatusesHandler(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	if err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	rows, err := db.Instance.Query(`
        SELECT group_id, status 
        FROM group_memberships
        WHERE user_id = ?
    `, userID)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	statuses := make(map[int]string)
	for rows.Next() {
		var groupID int
		var status string
		if err := rows.Scan(&groupID, &status); err == nil {
			statuses[groupID] = status
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statuses)
}
