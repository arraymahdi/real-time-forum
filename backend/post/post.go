package post

import (
	"backend/db"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// Create post
func CreatePostHandler(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("[Posts] User lookup failed: %v", err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	content := r.FormValue("content")
	privacy := r.FormValue("privacy")
	groupIDStr := r.FormValue("group_id")
	allowedFollowersStr := r.FormValue("allowed_followers") // comma-separated user IDs

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
		} else {
			http.Error(w, "Invalid group ID", http.StatusBadRequest)
			return
		}
	}

	// enforce group/ privacy rules
	if groupID != nil {
		privacy = "private" // any group post must be private
	} else {
		if privacy == "" {
			privacy = "public" // default for non-group posts
		}
		if privacy == "private" {
			http.Error(w, "Private posts must belong to a group", http.StatusBadRequest)
			return
		}
	}

	mediaPath, err := SaveFile(r, "media")
	if err != nil {
		log.Printf("[Posts] Media upload failed: %v", err)
		http.Error(w, "Media upload failed", http.StatusInternalServerError)
		return
	}

	// insert post and get last inserted ID
	res, err := db.Instance.Exec(`INSERT INTO posts (user_id, group_id, content, media, privacy) VALUES (?, ?, ?, ?, ?)`,
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

	// handle allowed followers: disabled for group posts
	var allowedFollowers []int
	if privacy == "private" && groupID == nil && allowedFollowersStr != "" {
		followerIDs := strings.Split(allowedFollowersStr, ",")
		stmt, err := db.Instance.Prepare("INSERT INTO post_allowed_followers (post_id, follower_id) VALUES (?, ?)")
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

func GetPostsHandler(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	if err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		log.Printf("[Posts] User lookup failed for email '%s': %v", userEmail, err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}
	log.Printf("[Posts] Logged in user ID: %d", userID)

	rows, err := db.Instance.Query(`
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
				err := db.Instance.QueryRow("SELECT 1 FROM followers WHERE follower_id = ? AND following_id = ? AND status = 'accepted'", userID, post.UserID).Scan(&exists)
				if err == nil {
					show = true
					log.Printf("[Posts] User %d follows post owner %d, showing post ID %d", userID, post.UserID, post.ID)
				} else {
					log.Printf("[Posts] User %d does NOT follow post owner %d, hiding post ID %d", userID, post.UserID, post.ID)
				}
			case "private":
				if post.GroupID != nil {
					var exists int
					err := db.Instance.QueryRow("SELECT 1 FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'", *post.GroupID, userID).Scan(&exists)
					if err == nil {
						show = true
						log.Printf("[Posts] User %d is member of group %d, showing private post ID %d", userID, *post.GroupID, post.ID)
					} else {
						log.Printf("[Posts] User %d NOT in group %d, hiding private post ID %d", userID, *post.GroupID, post.ID)
					}
				} else {
					var exists int
					err := db.Instance.QueryRow("SELECT 1 FROM post_allowed_followers WHERE post_id = ? AND follower_id = ?", post.ID, userID).Scan(&exists)
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
				rowsAF, _ := db.Instance.Query("SELECT follower_id FROM post_allowed_followers WHERE post_id = ?", post.ID)
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
	if err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
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

	err := db.Instance.QueryRow(`
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
			err := db.Instance.QueryRow("SELECT 1 FROM followers WHERE follower_id = ? AND following_id = ? AND status = 'accepted'", userID, post.UserID).Scan(&exists)
			if err == nil {
				show = true
			}
		case "private":
			if post.GroupID != nil {
				var exists int
				err := db.Instance.QueryRow("SELECT 1 FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'", *post.GroupID, userID).Scan(&exists)
				if err == nil {
					show = true
				}
			} else {
				var exists int
				err := db.Instance.QueryRow("SELECT 1 FROM post_allowed_followers WHERE post_id = ? AND follower_id = ?", post.ID, userID).Scan(&exists)
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
		rowsAF, _ := db.Instance.Query("SELECT follower_id FROM post_allowed_followers WHERE post_id = ?", post.ID)
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

// Get all posts created by the logged-in user
func GetMyPostsHandler(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int
	if err := db.Instance.QueryRow("SELECT id FROM users WHERE email = ?", userEmail).Scan(&userID); err != nil {
		log.Printf("[Posts] User lookup failed for email '%s': %v", userEmail, err)
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	rows, err := db.Instance.Query(`
		SELECT p.post_id, p.user_id, p.group_id, p.content, p.media, p.privacy, p.created_at, u.nickname
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.user_id = ?
		ORDER BY p.created_at DESC
	`, userID)
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

		// Populate allowed followers if private and not a group post
		if post.Privacy == "private" && post.GroupID == nil {
			rowsAF, _ := db.Instance.Query("SELECT follower_id FROM post_allowed_followers WHERE post_id = ?", post.ID)
			defer rowsAF.Close()
			var allowedFollowers []int
			for rowsAF.Next() {
				var fid int
				rowsAF.Scan(&fid)
				allowedFollowers = append(allowedFollowers, fid)
			}
			post.AllowedFollowers = allowedFollowers
		}

		posts = append(posts, post)
	}

	log.Printf("[Posts] Returning %d posts for user %d", len(posts), userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(posts)
}
