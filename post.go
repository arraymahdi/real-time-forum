package main

import (
	"encoding/json"
	"database/sql"
	"fmt"
	"log"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Structs
type Post struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Category  string `json:"category"`
	ImagePath string `json:"image_path,omitempty"`
	VideoPath string `json:"video_path,omitempty"`
	UserID    int    `json:"user_id"`
	CreatedAt string `json:"created_at"`
}

var uploadDir = "uploads"

// Middleware for JWT authentication
func jwtMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if tokenString == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		email, ok := claims["email"].(string)
		if !ok {
			http.Error(w, "Invalid token payload", http.StatusUnauthorized)
			return
		}

		// Store user email in request context
		r.Header.Set("User-Email", email)
		next(w, r)
	}
}

// File Upload Helper
func saveFile(r *http.Request, fieldName, uploadDir string) (string, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		if err == http.ErrMissingFile {
			return "", nil // No file uploaded
		}
		return "", err
	}
	defer file.Close()

	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.MkdirAll(uploadDir, os.ModePerm)
	}

	fileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	filePath := filepath.Join(uploadDir, fileName)

	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}

	return filePath, nil
}

func createPostHandler(w http.ResponseWriter, r *http.Request) {
	println("try")
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	userEmail := r.Header.Get("User-Email")
	println(userEmail)
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")
	category := r.FormValue("category")

	imagePath, err := saveFile(r, "image", uploadDir)
	if err != nil {
		http.Error(w, "Image upload failed", http.StatusInternalServerError)
		return
	}

	videoPath, err := saveFile(r, "video", uploadDir)
	if err != nil {
		http.Error(w, "Video upload failed", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(`INSERT INTO posts (user_id, title, content, category, image_path, video_path) 
                  VALUES (?, ?, ?, ?, ?, ?)`, userID, title, content, category, imagePath, videoPath)
	if err != nil {
		http.Error(w, "Error saving post", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Post created successfully"})
}

func getPostsHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
        SELECT posts.id, posts.user_id, posts.title, posts.content, posts.category, 
               posts.image_path, posts.video_path, posts.created_at, users.nickname 
        FROM posts
        JOIN users ON posts.user_id = users.id
        ORDER BY posts.created_at DESC
    `)
	if err != nil {
		http.Error(w, "Error retrieving posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []struct {
		Post
		Nickname string `json:"nickname"`
	}
	
	for rows.Next() {
		var post Post
		var nickname string
		err := rows.Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.Category, &post.ImagePath, &post.VideoPath, &post.CreatedAt, &nickname)
		if err != nil {
			http.Error(w, "Error scanning posts", http.StatusInternalServerError)
			return
		}

		posts = append(posts, struct {
			Post
			Nickname string `json:"nickname"`
		}{post, nickname})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

func GetPostByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Extract post ID from URL
	postID := strings.TrimPrefix(r.URL.Path, "/post/")
	if postID == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	var post Post
	var nickname string

	// Fetch post and user nickname
	err := db.QueryRow(`
		SELECT p.id, p.user_id, p.title, p.content, p.category, p.image_path, p.video_path, p.created_at, u.nickname
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = ?`, postID).
		Scan(&post.ID, &post.UserID, &post.Title, &post.Content, &post.Category, &post.ImagePath, &post.VideoPath, &post.CreatedAt, &nickname)

	if err == sql.ErrNoRows {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Println("Database error:", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Construct response
	response := map[string]interface{}{
		"id":         post.ID,
		"user_id":    post.UserID,
		"title":      post.Title,
		"content":    post.Content,
		"category":   post.Category,
		"image_path": post.ImagePath,
		"video_path": post.VideoPath,
		"created_at": post.CreatedAt,
		"nickname":   nickname,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}







