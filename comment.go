package main

import (
	"encoding/json"
	"net/http"
)

type Comment struct {
	ID        int    `json:"id"`
	PostID    int    `json:"post_id"`
	UserID    int    `json:"user_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

func createCommentHandler(w http.ResponseWriter, r *http.Request) {
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
	println(userEmail)
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}
	println(userID)

	var comment Comment
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&comment)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if comment.Content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(`INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)`, comment.PostID, userID, comment.Content)
	if err != nil {
		http.Error(w, "Error saving comment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Comment added successfully"})
}

func getCommentsHandler(w http.ResponseWriter, r *http.Request) {
	postID := r.URL.Query().Get("post_id")
	if postID == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	rows, err := db.Query("SELECT id, post_id, user_id, content, created_at FROM comments WHERE post_id = ? ORDER BY created_at DESC", postID)
	if err != nil {
		http.Error(w, "Error retrieving comments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Content, &comment.CreatedAt)
		if err != nil {
			http.Error(w, "Error scanning comments", http.StatusInternalServerError)
			return
		}
		comments = append(comments, comment)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}
