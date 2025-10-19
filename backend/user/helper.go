package user

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// -------------------- Constants --------------------
const AvatarDir = "./uploads/avatars"

var JwtSecret = []byte("love-love-love")

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
func SignHMACSHA256(message, secret []byte) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write(message)
	return h.Sum(nil)
}

// keep verifyHMACSHA256
func VerifyHMACSHA256(message, secret, signature []byte) bool {
	expected := SignHMACSHA256(message, secret)
	return hmac.Equal(expected, signature)
}

func Base64Encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func Base64Decode(data string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(data)
}

func GenerateJWT(userID int, email string) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	payload := map[string]interface{}{
		"id":    userID,
		"email": email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	}

	headerJSON, _ := json.Marshal(header)
	payloadJSON, _ := json.Marshal(payload)

	headerEnc := Base64Encode(headerJSON)
	payloadEnc := Base64Encode(payloadJSON)
	signature := Base64Encode(SignHMACSHA256([]byte(headerEnc+"."+payloadEnc), JwtSecret))

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
	sig, err := Base64Decode(parts[2])
	if err != nil {
		return 0, err
	}

	if !VerifyHMACSHA256([]byte(headerPayload), JwtSecret, sig) {
		return 0, errors.New("invalid token signature")
	}

	payloadJSON, err := Base64Decode(parts[1])
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

// Middleware for JWT auth
func JwtMiddleware(next http.HandlerFunc) http.HandlerFunc {
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

// ExtractEmailFromToken extracts email from custom JWT
func ExtractEmailFromToken(tokenString string) (string, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid token format")
	}

	headerPayload := parts[0] + "." + parts[1]

	sig, err := Base64Decode(parts[2])
	if err != nil {
		return "", errors.New("invalid token signature encoding")
	}
	if !VerifyHMACSHA256([]byte(headerPayload), JwtSecret, sig) {
		return "", errors.New("invalid token signature")
	}

	payloadJSON, err := Base64Decode(parts[1])
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
