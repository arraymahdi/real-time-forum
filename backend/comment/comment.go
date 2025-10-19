package comment

import (
	"backend/db"
	"backend/user"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
)

// Create a new comment
func CreateCommentHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[createCommentHandler] Incoming request...")

	if r.Method != http.MethodPost {
		log.Println("[createCommentHandler] Invalid method:", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	// Extract token and user ID
	token := r.Header.Get("Authorization")
	if token == "" {
		log.Println("[createCommentHandler] Missing token")
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	userID, err := user.ExtractUserIDFromToken(token)
	if err != nil {
		log.Println("[createCommentHandler] Token extraction failed:", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Println("[createCommentHandler] User ID from token:", userID)

	// Get post_id from query
	postIDStr := r.URL.Query().Get("post_id")
	if postIDStr == "" {
		log.Println("[createCommentHandler] Missing post_id in query")
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		log.Println("[createCommentHandler] Invalid Post ID:", postIDStr, "error:", err)
		http.Error(w, "Invalid Post ID", http.StatusBadRequest)
		return
	}
	log.Println("[createCommentHandler] Post ID:", postID)

	// Decode comment payload
	var comment Comment
	if err := json.NewDecoder(r.Body).Decode(&comment); err != nil {
		log.Println("[createCommentHandler] JSON decode failed:", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	log.Printf("[createCommentHandler] Decoded Comment: %+v\n", comment)

	// Fetch post owner and privacy
	var postOwnerID int
	var privacy string
	err = db.Instance.QueryRow("SELECT user_id, privacy FROM posts WHERE post_id = ?", postID).Scan(&postOwnerID, &privacy)
	if err != nil {
		log.Println("[createCommentHandler] Failed to fetch post:", err)
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	// Check privacy rules
	allowed := false
	if userID == postOwnerID {
		allowed = true
	} else {
		switch privacy {
		case "public":
			allowed = true
		case "almost_private":
			err = db.Instance.QueryRow("SELECT 1 FROM followers WHERE follower_id=? AND following_id=?", userID, postOwnerID).Scan(new(int))
			if err == nil {
				allowed = true
			}
		case "private":
			err = db.Instance.QueryRow("SELECT 1 FROM post_allowed_followers WHERE post_id=? AND follower_id=?", postID, userID).Scan(new(int))
			if err == nil {
				allowed = true
			}
		}
	}

	if !allowed {
		log.Println("[createCommentHandler] User not allowed to comment due to privacy settings")
		http.Error(w, "You do not have permission to comment on this post", http.StatusForbidden)
		return
	}

	// Insert comment into DB
	stmt, err := db.Instance.Prepare("INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)")
	if err != nil {
		log.Println("[createCommentHandler] DB prepare failed:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(postID, userID, comment.Content)
	if err != nil {
		log.Println("[createCommentHandler] INSERT failed:", err)
		http.Error(w, "Failed to create comment", http.StatusInternalServerError)
		return
	}

	log.Println("[createCommentHandler] Comment created successfully for Post ID:", postID, "by User ID:", userID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Comment created successfully"})
}

// Get all comments for a post
func GetCommentsByPostHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[getCommentsByPostHandler] Incoming request...")
	if r.Method != http.MethodGet {
		log.Println("[getCommentsByPostHandler] Invalid method:", r.Method)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	postIDStr := r.URL.Query().Get("post_id")
	if postIDStr == "" {
		log.Println("[getCommentsByPostHandler] Missing post_id in query")
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		log.Println("[getCommentsByPostHandler] Invalid Post ID:", postIDStr, "error:", err)
		http.Error(w, "Invalid Post ID", http.StatusBadRequest)
		return
	}
	log.Println("[getCommentsByPostHandler] Post ID:", postID)

	query := `
		SELECT c.comment_id, c.post_id, c.user_id, c.content, c.created_at, u.nickname
		FROM comments c
		LEFT JOIN users u ON c.user_id = u.id
		WHERE c.post_id = ?
		ORDER BY c.created_at ASC
	`
	rows, err := db.Instance.Query(query, postID)
	if err != nil {
		log.Println("[getCommentsByPostHandler] Query failed:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Content, &comment.CreatedAt, &comment.Nickname); err != nil {
			log.Println("[getCommentsByPostHandler] Row scan failed:", err)
			http.Error(w, "Error scanning row", http.StatusInternalServerError)
			return
		}
		log.Printf("[getCommentsByPostHandler] Scanned comment: %+v\n", comment)
		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		log.Println("[getCommentsByPostHandler] Rows iteration error:", err)
		http.Error(w, "Error processing rows", http.StatusInternalServerError)
		return
	}

	log.Println("[getCommentsByPostHandler] Retrieved", len(comments), "comments for Post ID:", postID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}
