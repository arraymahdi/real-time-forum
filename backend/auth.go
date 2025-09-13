package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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

	"golang.org/x/crypto/bcrypt"
)

// -------------------- Constants --------------------
const avatarDir = "./uploads/avatars"

var jwtSecret = []byte("love-love-love")

// -------------------- Models --------------------
type User struct {
	ID          int    `json:"id"`
	Email       string `json:"email"`
	Password    string `json:"password,omitempty"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	DateOfBirth string `json:"date_of_birth"`
	Avatar      string `json:"avatar"`
	Nickname    string `json:"nickname"`
	AboutMe     string `json:"about_me"`
	ProfileType string `json:"profile_type"`
}

// -------------------- JWT Utilities --------------------
func signHMACSHA256(message, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(message)
	return h.Sum(nil)
}

// keep verifyHMACSHA256
func verifyHMACSHA256(message, secret, signature []byte) bool {
	expected := signHMACSHA256(message, secret)
	return hmac.Equal(expected, signature)
}

func base64Encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64Decode(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(data)
}

func generateJWT(userID int, email string) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	payload := map[string]interface{}{
		"id":    userID,
		"email": email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}

	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)

	headerEnc := base64Encode(headerJSON)
	payloadEnc := base64Encode(payloadJSON)
	signature := base64Encode(signHMACSHA256([]byte(headerEnc+"."+payloadEnc), jwtSecret))

	token := fmt.Sprintf("%s.%s.%s", headerEnc, payloadEnc, signature)
	log.Printf("[JWT] Generated token for id=%d", userID)
	return token, nil
}

func ExtractUserIDFromToken(token string) (int, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return 0, errors.New("invalid token format")
	}

	headerPayload := parts[0] + "." + parts[1]
	sig, err := base64Decode(parts[2])
	if err != nil {
		return 0, err
	}

	if !verifyHMACSHA256([]byte(headerPayload), jwtSecret, sig) {
		return 0, errors.New("invalid token signature")
	}

	payloadJSON, err := base64Decode(parts[1])
	if err != nil {
		return 0, err
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return 0, err
	}

	if uid, ok := claims["id"].(float64); ok {
		return int(uid), nil
	}
	return 0, errors.New("id not found")
}

// -------------------- Handlers --------------------
func registerHandler(w http.ResponseWriter, r *http.Request) {
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

	_, err = db.Exec(`
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

func loginHandler(w http.ResponseWriter, r *http.Request) {
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
	err := db.QueryRow(`SELECT id, password FROM users WHERE email=?`, req.Email).Scan(&userID, &storedPassword)
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

	token, _ := generateJWT(userID, req.Email)
	log.Printf("[Login] User %d logged in", userID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, email, first_name, last_name, nickname, avatar, about_me, profile_type, date_of_birth FROM users")
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

func uploadAvatarHandler(w http.ResponseWriter, r *http.Request) {
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
	if _, err := os.Stat(avatarDir); os.IsNotExist(err) {
		log.Printf("[Avatar] Directory %s does not exist, creating...\n", avatarDir)
		if err := os.MkdirAll(avatarDir, os.ModePerm); err != nil {
			log.Printf("[Avatar][ERROR] Failed to create directory: %v\n", err)
			http.Error(w, "Failed to create directory", http.StatusInternalServerError)
			return
		}
	}
	log.Printf("[Avatar] Directory ready: %s\n", avatarDir)

	// Generate a unique suffix using timestamp
	timestamp := time.Now().UnixNano()
	fileName := fmt.Sprintf("%d_%d_%s", userID, timestamp, handler.Filename)
	filePath := filepath.Join(avatarDir, fileName)
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
	_, err = db.Exec(`UPDATE users SET avatar=? WHERE id=?`, filePath, userID)
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
func getCurrentUserProfileHandler(w http.ResponseWriter, r *http.Request) {
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

	err := db.QueryRow("SELECT id, nickname, avatar, email FROM users WHERE email = ?", userEmail).
		Scan(&user.ID, &user.Nickname, &user.Avatar, &user.Email)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// Add this new endpoint to get user by ID
func getUserByIDHandler(w http.ResponseWriter, r *http.Request) {
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
		ID       int    `json:"id"`
		Nickname string `json:"nickname"`
		Avatar   string `json:"avatar"`
	}

	err = db.QueryRow("SELECT id, nickname, avatar FROM users WHERE id = ?", userID).
		Scan(&user.ID, &user.Nickname, &user.Avatar)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// getFullUserProfileHandler - Get full profile of the authenticated user
func getFullUserProfileHandler(w http.ResponseWriter, r *http.Request) {
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
	err = db.QueryRow(`
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
func updateUserProfileHandler(w http.ResponseWriter, r *http.Request) {
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

	result, err := db.Exec(query, args...)
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
