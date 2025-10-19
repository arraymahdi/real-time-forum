package event

import (
	"backend/db"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Respond to event (going/not_going)
func RespondToEventHandler(w http.ResponseWriter, r *http.Request) {
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
		EventID  int    `json:"event_id"`
		Response string `json:"response"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Response != "going" && req.Response != "not_going" {
		http.Error(w, "Response must be 'going' or 'not_going'", http.StatusBadRequest)
		return
	}

	// Check if user is member of the group that the event belongs to
	var groupID int
	err := db.Instance.QueryRow("SELECT group_id FROM events WHERE event_id = ?", req.EventID).Scan(&groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Event not found", http.StatusNotFound)
		} else {
			log.Printf("[Groups] Error finding event: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	var role string
	err = db.Instance.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		groupID, userID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "You are not a member of this group", http.StatusForbidden)
		} else {
			log.Printf("[Groups] Error checking group membership: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	// Insert or update event response
	_, err = db.Instance.Exec(`INSERT OR REPLACE INTO event_responses (event_id, user_id, response, responded_at) VALUES (?, ?, ?, ?)`,
		req.EventID, userID, req.Response, time.Now().Format("2006-01-02 15:04:05"))
	if err != nil {
		log.Printf("[Groups] Event response failed: %v", err)
		http.Error(w, "Error saving event response", http.StatusInternalServerError)
		return
	}

	log.Printf("[Groups] User %d responded '%s' to event %d", userID, req.Response, req.EventID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message":  "Event response saved successfully",
		"event_id": strconv.Itoa(req.EventID),
		"response": req.Response,
	})
}

// Delete event (creators and event creators only)
func DeleteEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
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

	eventIDStr := strings.TrimPrefix(r.URL.Path, "/event/")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Get event and group info
	var eventCreatorID, groupID int
	err = db.Instance.QueryRow("SELECT creator_id, group_id FROM events WHERE event_id = ?", eventID).Scan(&eventCreatorID, &groupID)
	if err == sql.ErrNoRows {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("[Groups] Query event failed: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	// Check if user can delete the event
	canDelete := false

	// Event creator can delete
	if eventCreatorID == userID {
		canDelete = true
	} else {
		// Group creator can delete any event in their group
		var groupCreatorID int
		err = db.Instance.QueryRow("SELECT creator_id FROM groups WHERE group_id = ?", groupID).Scan(&groupCreatorID)
		if err == nil && groupCreatorID == userID {
			canDelete = true
		}
	}

	if !canDelete {
		http.Error(w, "You don't have permission to delete this event", http.StatusForbidden)
		return
	}

	// Delete event responses first
	_, err = db.Instance.Exec("DELETE FROM event_responses WHERE event_id = ?", eventID)
	if err != nil {
		log.Printf("[Groups] Delete event responses failed: %v", err)
		http.Error(w, "Error deleting event", http.StatusInternalServerError)
		return
	}

	// Delete event
	_, err = db.Instance.Exec("DELETE FROM events WHERE event_id = ?", eventID)
	if err != nil {
		log.Printf("[Groups] Delete event failed: %v", err)
		http.Error(w, "Error deleting event", http.StatusInternalServerError)
		return
	}

	log.Printf("[Groups] User %d deleted event %d", userID, eventID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Event deleted successfully"})
}
