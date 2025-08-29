package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Post model aligned with schema
type Post struct {
	ID               int    `json:"post_id"`
	UserID           int    `json:"user_id"`
	GroupID          *int   `json:"group_id,omitempty"`
	AllowedFollowers []int  `json:"allowed_followers,omitempty"`
	Content          string `json:"content"`
	Media            string `json:"media,omitempty"`
	Privacy          string `json:"privacy"`
	CreatedAt        string `json:"created_at"`
	Nickname         string `json:"nickname,omitempty"`
}

var uploadDir = "uploads"

// ExtractEmailFromToken extracts email from custom JWT
func ExtractEmailFromToken(tokenString string) (string, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid token format")
	}

	headerPayload := parts[0] + "." + parts[1]

	sig, err := base64Decode(parts[2])
	if err != nil {
		return "", errors.New("invalid token signature encoding")
	}
	if !verifyHMACSHA256([]byte(headerPayload), jwtSecret, sig) {
		return "", errors.New("invalid token signature")
	}

	payloadJSON, err := base64Decode(parts[1])
	if err != nil {
		return "", errors.New("invalid token payload encoding")
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return "", errors.New("invalid token payload")
	}

	email, ok := claims["email"].(string)
	if !ok {
		return "", errors.New("email not found in token")
	}
	return email, nil
}

// Middleware for JWT auth
func jwtMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tokenString == "" {
			http.Error(w, "Authorization header is required", http.StatusUnauthorized)
			return
		}

		email, err := ExtractEmailFromToken(tokenString)
		if err != nil {
			log.Printf("[JWT] Invalid token: %v", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		r.Header.Set("User-Email", email)
		next(w, r)
	}
}

// File uploader
func saveFile(r *http.Request, fieldName string) (string, error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		if err == http.ErrMissingFile {
			return "", nil
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

// Create post
func createPostHandler(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("[Posts] User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	content := r.FormValue("content")
	privacy := r.FormValue("privacy")
	groupIDStr := r.FormValue("group_id")
	allowedFollowersStr := r.FormValue("allowed_followers") // comma-separated user IDs

	if privacy == "" {
		privacy = "public"
	}
	if content == "" {
		http.Error(w, "Content is required", http.StatusBadRequest)
		return
	}

	// handle optional group_id
	var groupID *int
	if groupIDStr != "" {
		var gid int
		if _, err := fmt.Sscanf(groupIDStr, "%d", &gid); err == nil {
			groupID = &gid
		}
	}

	mediaPath, err := saveFile(r, "media")
	if err != nil {
		log.Printf("[Posts] Media upload failed: %v", err)
		http.Error(w, "Media upload failed", http.StatusInternalServerError)
		return
	}

	// insert post and get last inserted ID
	res, err := db.Exec(`INSERT INTO posts (user_id, group_id, content, media, privacy) VALUES (?, ?, ?, ?, ?)`,
		userID, groupID, content, mediaPath, privacy)
	if err != nil {
		log.Printf("[Posts] Insert failed: %v", err)
		http.Error(w, "Error saving post", http.StatusInternalServerError)
		return
	}

	postID, err := res.LastInsertId()
	if err != nil {
		log.Printf("[Posts] Getting post ID failed: %v", err)
		http.Error(w, "Error saving post", http.StatusInternalServerError)
		return
	}

	// handle allowed followers for private posts only if it's NOT a group post
	var allowedFollowers []int
	if privacy == "private" && groupID == nil && allowedFollowersStr != "" {
		followerIDs := strings.Split(allowedFollowersStr, ",")
		stmt, err := db.Prepare("INSERT INTO post_allowed_followers (post_id, follower_id) VALUES (?, ?)")
		if err != nil {
			log.Printf("[Posts] Prepare allowed followers failed: %v", err)
			http.Error(w, "Error saving allowed followers", http.StatusInternalServerError)
			return
		}
		defer stmt.Close()

		for _, fidStr := range followerIDs {
			fidStr = strings.TrimSpace(fidStr)
			fid, err := strconv.Atoi(fidStr)
			if err != nil {
				continue // skip invalid IDs
			}
			_, err = stmt.Exec(postID, fid)
			if err != nil {
				log.Printf("[Posts] Inserting allowed follower failed for user %d: %v", fid, err)
				continue
			}
			allowedFollowers = append(allowedFollowers, fid)
		}
	}

	// build response post object
	post := Post{
		ID:               int(postID),
		UserID:           userID,
		GroupID:          groupID,
		Content:          content,
		Media:            mediaPath,
		Privacy:          privacy,
		AllowedFollowers: allowedFollowers,
	}

	log.Printf("[Posts] User %d created new post (ID: %d)", userID, postID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(post)
}

func getPostsHandler(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	if err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		log.Printf("[Posts] User lookup failed for email '%s': %v", userEmail, err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}
	log.Printf("[Posts] Logged in user ID: %d", userID)

	rows, err := db.Query(`
		SELECT p.post_id, p.user_id, p.group_id, p.content, p.media, p.privacy, p.created_at, u.nickname
		FROM posts p
		JOIN users u ON p.user_id = u.id
		ORDER BY p.created_at DESC
	`)
	if err != nil {
		log.Printf("[Posts] Query failed: %v", err)
		http.Error(w, "Error retrieving posts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var posts []Post

	for rows.Next() {
		var post Post
		var groupID sql.NullInt64
		if err := rows.Scan(&post.ID, &post.UserID, &groupID, &post.Content, &post.Media, &post.Privacy, &post.CreatedAt, &post.Nickname); err != nil {
			log.Printf("[Posts] Scan failed: %v", err)
			http.Error(w, "Error scanning posts", http.StatusInternalServerError)
			return
		}
		if groupID.Valid {
			val := int(groupID.Int64)
			post.GroupID = &val
		}

		show := false
		log.Printf("[Posts] Checking post ID %d by user %d (privacy: %s)", post.ID, post.UserID, post.Privacy)

		// Always allow creator
		if post.UserID == userID {
			show = true
			log.Printf("[Posts] User is creator, showing post ID %d", post.ID)
		} else {
			switch post.Privacy {
			case "public":
				show = true
				log.Printf("[Posts] Post ID %d is public, showing", post.ID)
			case "almost_private":
				var exists int
				err := db.QueryRow("SELECT 1 FROM followers WHERE follower_id = ? AND following_id = ? AND status = 'accepted'", userID, post.UserID).Scan(&exists)
				if err == nil {
					show = true
					log.Printf("[Posts] User %d follows post owner %d, showing post ID %d", userID, post.UserID, post.ID)
				} else {
					log.Printf("[Posts] User %d does NOT follow post owner %d, hiding post ID %d", userID, post.UserID, post.ID)
				}
			case "private":
				if post.GroupID != nil {
					var exists int
					err := db.QueryRow("SELECT 1 FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'", *post.GroupID, userID).Scan(&exists)
					if err == nil {
						show = true
						log.Printf("[Posts] User %d is member of group %d, showing private post ID %d", userID, *post.GroupID, post.ID)
					} else {
						log.Printf("[Posts] User %d NOT in group %d, hiding private post ID %d", userID, *post.GroupID, post.ID)
					}
				} else {
					var exists int
					err := db.QueryRow("SELECT 1 FROM post_allowed_followers WHERE post_id = ? AND follower_id = ?", post.ID, userID).Scan(&exists)
					if err == nil {
						show = true
						log.Printf("[Posts] User %d is allowed follower, showing private post ID %d", userID, post.ID)
					} else {
						log.Printf("[Posts] User %d NOT allowed follower, hiding private post ID %d", userID, post.ID)
					}
				}
			}
		}

		if show {
			if post.Privacy == "private" && post.GroupID == nil {
				rowsAF, _ := db.Query("SELECT follower_id FROM post_allowed_followers WHERE post_id = ?", post.ID)
				defer rowsAF.Close()
				var allowedFollowers []int
				for rowsAF.Next() {
					var fid int
					rowsAF.Scan(&fid)
					allowedFollowers = append(allowedFollowers, fid)
				}
				post.AllowedFollowers = allowedFollowers
				log.Printf("[Posts] Post ID %d allowed followers: %v", post.ID, allowedFollowers)
			}
			posts = append(posts, post)
		}
	}

	log.Printf("[Posts] Returning %d posts for user %d", len(posts), userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}

// Get single post by ID with privacy check, always allow creator
func GetPostByIDHandler(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	if err := db.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		log.Printf("[Posts] User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	postIDStr := strings.TrimPrefix(r.URL.Path, "/post/")
	if postIDStr == "" {
		http.Error(w, "Post ID is required", http.StatusBadRequest)
		return
	}

	var post Post
	var groupID sql.NullInt64

	err := db.QueryRow(`
		SELECT p.post_id, p.user_id, p.group_id, p.content, p.media, p.privacy, p.created_at, u.nickname
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.post_id = ?`, postIDStr).
		Scan(&post.ID, &post.UserID, &groupID, &post.Content, &post.Media, &post.Privacy, &post.CreatedAt, &post.Nickname)

	if err == sql.ErrNoRows {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("[Posts] Query by ID failed: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if groupID.Valid {
		val := int(groupID.Int64)
		post.GroupID = &val
	}

	// Privacy check: **always allow creator**
	show := false
	if post.UserID == userID {
		show = true
	} else {
		switch post.Privacy {
		case "public":
			show = true
		case "almost_private":
			var exists int
			err := db.QueryRow("SELECT 1 FROM followers WHERE follower_id = ? AND following_id = ? AND status = 'accepted'", userID, post.UserID).Scan(&exists)
			if err == nil {
				show = true
			}
		case "private":
			if post.GroupID != nil {
				var exists int
				err := db.QueryRow("SELECT 1 FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'", *post.GroupID, userID).Scan(&exists)
				if err == nil {
					show = true
				}
			} else {
				var exists int
				err := db.QueryRow("SELECT 1 FROM post_allowed_followers WHERE post_id = ? AND follower_id = ?", post.ID, userID).Scan(&exists)
				if err == nil {
					show = true
				}
			}
		}
	}

	if !show {
		http.Error(w, "You are not allowed to view this post", http.StatusForbidden)
		return
	}

	// Populate allowed followers if private and not group
	if post.Privacy == "private" && post.GroupID == nil {
		rowsAF, _ := db.Query("SELECT follower_id FROM post_allowed_followers WHERE post_id = ?", post.ID)
		defer rowsAF.Close()
		var allowedFollowers []int
		for rowsAF.Next() {
			var fid int
			rowsAF.Scan(&fid)
			allowedFollowers = append(allowedFollowers, fid)
		}
		post.AllowedFollowers = allowedFollowers
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(post)
}
