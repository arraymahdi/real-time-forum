package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Gender string

const (
	Male   Gender = "Male"
	Female Gender = "Female"
	Other  Gender = "Other"
)

type User struct {
	ID        int    `json:"id"`
	Nickname  string `json:"nickname"`
	Age       int    `json:"age"`
	Gender    Gender `json:"gender"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Avatar    string `json:"avatar"`
}

const (
	avatarDir = "../uploads/avatars"
)

var jwtSecret = []byte("love-love-love")

// base64Encode encodes data to base64 URL encoding without padding.
func base64Encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// signHMACSHA256 creates an HMAC-SHA256 signature.
func signHMACSHA256(message, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(message)
	return h.Sum(nil)
}

// generateJWT manually generates a JWT token.
func generateJWT(userID int, email string) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}
	headerJSON, _ := json.Marshal(header)

	payload := map[string]interface{}{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	payloadJSON, _ := json.Marshal(payload)

	headerEncoded := base64Encode(headerJSON)
	payloadEncoded := base64Encode(payloadJSON)

	// Create the signature
	signature := signHMACSHA256([]byte(headerEncoded+"."+payloadEncoded), jwtSecret)
	signatureEncoded := base64Encode(signature)

	// Concatenate all parts to form the final JWT
	return fmt.Sprintf("%s.%s.%s", headerEncoded, payloadEncoded, signatureEncoded), nil
}

func isValidGender(g Gender) bool {
	return g == Male || g == Female || g == Other
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if !isValidGender(user.Gender) {
		http.Error(w, "Invalid gender", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error hashing password", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO users (nickname, age, gender, first_name, last_name, email, password) VALUES (?, ?, ?, ?, ?, ?, ?)",
		user.Nickname, user.Age, user.Gender, user.FirstName, user.LastName, user.Email, string(hashedPassword))
	if err != nil {
		http.Error(w, "Error registering user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintln(w, "User registered successfully")
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	var storedPassword string
	err := db.QueryRow("SELECT id, password FROM users WHERE email = ? OR nickname = ?", user.Email, user.Nickname).Scan(&user.ID, &storedPassword)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(user.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := generateJWT(user.ID, user.Email)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// base64Decode decodes a base64 URL-encoded string.
func base64Decode(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(data)
}

// verifyHMACSHA256 verifies the HMAC-SHA256 signature.
func verifyHMACSHA256(message, secret, signature []byte) bool {
	expectedSignature := signHMACSHA256(message, secret)
	return hmac.Equal(expectedSignature, signature)
}

// ExtractUserIDFromToken extracts the user ID from a manually created JWT.
func ExtractUserIDFromToken(tokenString string) (int, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return 0, errors.New("invalid token format")
	}

	headerPayload := parts[0] + "." + parts[1]

	// Decode and verify the signature
	signature, err := base64Decode(parts[2])
	if err != nil {
		return 0, errors.New("invalid token signature encoding")
	}

	if !verifyHMACSHA256([]byte(headerPayload), jwtSecret, signature) {
		return 0, errors.New("invalid token signature")
	}

	// Decode the payload
	payloadJSON, err := base64Decode(parts[1])
	if err != nil {
		return 0, errors.New("invalid token payload encoding")
	}

	// Parse JSON and extract user ID
	var claims map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return 0, errors.New("invalid token payload")
	}

	// Extract user ID
	if userID, ok := claims["user_id"].(float64); ok {
		return int(userID), nil
	}

	return 0, errors.New("user_id not found in token")
}

func getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, nickname, email FROM users")
	if err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id int
		var nickname, email string
		if err := rows.Scan(&id, &nickname, &email); err != nil {
			http.Error(w, "Error scanning user", http.StatusInternalServerError)
			return
		}
		users = append(users, map[string]interface{}{
			"id":       id,
			"nickname": nickname,
			"email":    email,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func uploadAvatarHandler(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("Authorization")
	if token == "" {
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}
	userID, err := ExtractUserIDFromToken(token)
	if err != nil {
		http.Error(w, "Invalid or missing token", http.StatusUnauthorized)
		return
	}

	r.ParseMultipartForm(100 << 20) // Limit upload size to ~10MB
	file, handler, err := r.FormFile("avatar")
	if err != nil {
		http.Error(w, "Failed to upload file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create directory if not exists
	if _, err := os.Stat(avatarDir); os.IsNotExist(err) {
		if err := os.MkdirAll(avatarDir, os.ModePerm); err != nil {
			http.Error(w, "Failed to create directory", http.StatusInternalServerError)
			return
		}
	}

	// Create file path
	filePath := filepath.Join(avatarDir, fmt.Sprintf("%d_%s", userID, handler.Filename))

	// Create a new file on disk
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Copy file content
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Failed to save file", http.StatusInternalServerError)
		return
	}

	// Update the user's avatar in the database
	_, err = db.Exec("UPDATE users SET avatar = ? WHERE id = ?", filePath, userID)
	if err != nil {
		http.Error(w, "Failed to update user avatar", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Avatar uploaded successfully")
}
