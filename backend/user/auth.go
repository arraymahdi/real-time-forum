package user

import (
	"backend/db"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// -------------------- Handlers --------------------
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		log.Printf("[Register] JSON decode error: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[Register] Password hash error: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	_, err = db.Instance.Exec(`
        INSERT INTO users(email, password, first_name, last_name, date_of_birth, avatar, nickname, about_me, profile_type)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.Email, string(hashed), user.FirstName, user.LastName, user.DateOfBirth,
		user.Avatar, user.Nickname, user.AboutMe, user.ProfileType)
	if err != nil {
		log.Printf("[Register] DB insert error: %v", err)
		http.Error(w, "Failed to register user", http.StatusInternalServerError)
		return
	}

	log.Printf("[Register] User %s registered successfully", user.Email)
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "User registered successfully")
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	var req User
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	var storedPassword string
	var userID int
	err := db.Instance.QueryRow(`SELECT id, password FROM users WHERE email=?`, req.Email).Scan(&userID, &storedPassword)
	if err != nil {
		log.Printf("[Login] User not found: %v", err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(req.Password)); err != nil {
		log.Printf("[Login] Invalid password for id=%d", userID)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, _ := GenerateJWT(userID, req.Email)
	log.Printf("[Login] User %d logged in", userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func GetAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Instance.Query("SELECT id, email, first_name, last_name, nickname, avatar, about_me, profile_type, date_of_birth FROM users")
	if err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.Nickname, &u.Avatar, &u.AboutMe, &u.ProfileType, &u.DateOfBirth); err != nil {
			http.Error(w, "Error scanning user", http.StatusInternalServerError)
			return
		}
		users = append(users, u)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func UploadAvatarHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[Avatar] Upload request received")

	token := r.Header.Get("Authorization")
	if token == "" {
		log.Println("[Avatar][ERROR] Missing token")
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}
	log.Printf("[Avatar] Token received: %s\n", token)

	userID, err := ExtractUserIDFromToken(token)
	if err != nil {
		log.Printf("[Avatar][ERROR] Invalid token: %v\n", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	log.Printf("[Avatar] Extracted userID: %d\n", userID)

	// Parse up to 10MB file
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		log.Printf("[Avatar][ERROR] Failed to parse multipart form: %v\n", err)
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	log.Println("[Avatar] Multipart form parsed")

	file, handler, err := r.FormFile("avatar")
	if err != nil {
		log.Printf("[Avatar][ERROR] Failed to get form file: %v\n", err)
		http.Error(w, "Failed to upload file", http.StatusBadRequest)
		return
	}
	defer file.Close()
	log.Printf("[Avatar] File received: %s (%d bytes)\n", handler.Filename, handler.Size)

	// Ensure avatar directory exists
	if _, err := os.Stat(AvatarDir); os.IsNotExist(err) {
		log.Printf("[Avatar] Directory %s does not exist, creating...\n", AvatarDir)
		if err := os.MkdirAll(AvatarDir, os.ModePerm); err != nil {
			log.Printf("[Avatar][ERROR] Failed to create directory: %v\n", err)
			http.Error(w, "Failed to create directory", http.StatusInternalServerError)
			return
		}
	}
	log.Printf("[Avatar] Directory ready: %s\n", AvatarDir)

	// Generate a unique suffix using timestamp
	timestamp := time.Now().UnixNano()
	fileName := fmt.Sprintf("%d_%d_%s", userID, timestamp, handler.Filename)
	filePath := filepath.Join(AvatarDir, fileName)
	log.Printf("[Avatar] Saving file as: %s\n", filePath)

	// Save file
	dst, err := os.Create(filePath)
	if err != nil {
		log.Printf("[Avatar][ERROR] Failed to create destination file: %v\n", err)
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		log.Printf("[Avatar][ERROR] Failed to copy file contents: %v\n", err)
		http.Error(w, "Failed to write file", http.StatusInternalServerError)
		return
	}
	log.Printf("[Avatar] File saved successfully: %s\n", filePath)

	// Update database
	_, err = db.Instance.Exec(`UPDATE users SET avatar=? WHERE id=?`, filePath, userID)
	if err != nil {
		log.Printf("[Avatar][ERROR] Failed to update DB: %v\n", err)
		http.Error(w, "Failed to update avatar", http.StatusInternalServerError)
		return
	}
	log.Printf("[Avatar] DB updated for user %d with avatar %s\n", userID, filePath)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Avatar uploaded successfully")
	log.Println("[Avatar] Upload completed successfully")
}

// Add this new endpoint to get current user profile
func GetCurrentUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var user struct {
		ID       int    `json:"id"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
		Email    string `json:"email"`
	}

	err := db.Instance.QueryRow("SELECT id, nickname, avatar, email FROM users WHERE email = ?", userEmail).
		Scan(&user.ID, &user.Nickname, &user.Avatar, &user.Email)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// Add this new endpoint to get user by ID
func GetUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	userEmail := r.Header.Get("User-Email")
	if userEmail == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract user ID from URL path
	userIDStr := r.URL.Path[len("/user/"):]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var user struct {
		ID          int    `json:"id"`
		Nickname    string `json:"nickname"`
		ProfileType string `json:"profile_type"`
		Avatar      string `json:"avatar"`
	}

	err = db.Instance.QueryRow("SELECT id, nickname, profile_type, avatar FROM users WHERE id = ?", userID).
		Scan(&user.ID, &user.Nickname, &user.ProfileType, &user.Avatar)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// getFullUserProfileHandler - Get full profile of the authenticated user
func GetFullUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from token
	token := r.Header.Get("Authorization")
	if token == "" {
		log.Println("[Profile][ERROR] Missing token")
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	userID, err := ExtractUserIDFromToken(token)
	if err != nil {
		log.Printf("[Profile][ERROR] Invalid token: %v", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	var user User
	err = db.Instance.QueryRow(`
		SELECT id, email, first_name, last_name, date_of_birth, avatar, nickname, about_me, profile_type 
		FROM users WHERE id = ?`, userID).
		Scan(&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.DateOfBirth,
			&user.Avatar, &user.Nickname, &user.AboutMe, &user.ProfileType)

	if err != nil {
		log.Printf("[Profile][ERROR] User not found: %v", err)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Don't send password in response
	user.Password = ""

	log.Printf("[Profile] Retrieved full profile for user %d", userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// updateUserProfileHandler - Update profile of the authenticated user
func UpdateUserProfileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	// Extract user ID from token
	token := r.Header.Get("Authorization")
	if token == "" {
		log.Println("[Update][ERROR] Missing token")
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	userID, err := ExtractUserIDFromToken(token)
	if err != nil {
		log.Printf("[Update][ERROR] Invalid token: %v", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	var updateData struct {
		FirstName   *string `json:"first_name,omitempty"`
		LastName    *string `json:"last_name,omitempty"`
		DateOfBirth *string `json:"date_of_birth,omitempty"`
		Nickname    *string `json:"nickname,omitempty"`
		AboutMe     *string `json:"about_me,omitempty"`
		ProfileType *string `json:"profile_type,omitempty"`
		Password    *string `json:"password,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		log.Printf("[Update] JSON decode error: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Build dynamic query based on provided fields
	var setParts []string
	var args []interface{}

	if updateData.FirstName != nil {
		setParts = append(setParts, "first_name = ?")
		args = append(args, *updateData.FirstName)
	}
	if updateData.LastName != nil {
		setParts = append(setParts, "last_name = ?")
		args = append(args, *updateData.LastName)
	}
	if updateData.DateOfBirth != nil {
		setParts = append(setParts, "date_of_birth = ?")
		args = append(args, *updateData.DateOfBirth)
	}
	if updateData.Nickname != nil {
		setParts = append(setParts, "nickname = ?")
		args = append(args, *updateData.Nickname)
	}
	if updateData.AboutMe != nil {
		setParts = append(setParts, "about_me = ?")
		args = append(args, *updateData.AboutMe)
	}
	if updateData.ProfileType != nil {
		setParts = append(setParts, "profile_type = ?")
		args = append(args, *updateData.ProfileType)
	}
	if updateData.Password != nil {
		// Hash the new password
		hashed, err := bcrypt.GenerateFromPassword([]byte(*updateData.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("[Update] Password hash error: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}
		setParts = append(setParts, "password = ?")
		args = append(args, string(hashed))
	}

	if len(setParts) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	// Add userID to args for WHERE clause
	args = append(args, userID)

	query := fmt.Sprintf("UPDATE users SET %s WHERE id = ?", strings.Join(setParts, ", "))

	result, err := db.Instance.Exec(query, args...)
	if err != nil {
		log.Printf("[Update] DB update error: %v", err)
		http.Error(w, "Failed to update profile", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("[Update] Error checking rows affected: %v", err)
		http.Error(w, "Update verification failed", http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	log.Printf("[Update] Profile updated successfully for user %d", userID)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Profile updated successfully")
}
