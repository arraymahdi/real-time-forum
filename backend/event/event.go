package event

import (
	"backend/db"
	"backend/user"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Create event in group
func CreateEventHandler(w http.ResponseWriter, r *http.Request) {
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
		GroupID     int    `json:"group_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		EventTime   string `json:"event_time"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Title == "" || req.EventTime == "" {
		http.Error(w, "Title and event time are required", http.StatusBadRequest)
		return
	}

	// Check if user is member of the group
	var role string
	err := db.Instance.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		req.GroupID, userID).Scan(&role)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	// Parse and validate event time
	eventTime, err := time.Parse("2006-01-02 15:04:05", req.EventTime)
	if err != nil {
		// Try alternative format
		eventTime, err = time.Parse("2006-01-02T15:04:05", req.EventTime)
		if err != nil {
			http.Error(w, "Invalid event time format. Use YYYY-MM-DD HH:MM:SS", http.StatusBadRequest)
			return
		}
	}

	// Insert event
	res, err := db.Instance.Exec(`INSERT INTO events (group_id, creator_id, title, description, event_time) VALUES (?, ?, ?, ?, ?)`,
		req.GroupID, userID, req.Title, req.Description, eventTime.Format("2006-01-02 15:04:05"))
	if err != nil {
		log.Printf("[Groups] Event insert failed: %v", err)
		http.Error(w, "Error creating event", http.StatusInternalServerError)
		return
	}

	eventID, err := res.LastInsertId()
	if err != nil {
		log.Printf("[Groups] Getting event ID failed: %v", err)
		http.Error(w, "Error creating event", http.StatusInternalServerError)
		return
	}

	event := Event{
		EventID:     int(eventID),
		GroupID:     req.GroupID,
		CreatorID:   userID,
		Title:       req.Title,
		Description: req.Description,
		EventTime:   eventTime.Format("2006-01-02 15:04:05"),
	}

	log.Printf("[Groups] User %d created event %d in group %d", userID, eventID, req.GroupID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

// Get events for a group
func GetGroupEventsHandler(w http.ResponseWriter, r *http.Request) {
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

	groupIDStr := strings.TrimPrefix(r.URL.Path, "/group/")
	groupIDStr = strings.TrimSuffix(groupIDStr, "/events")
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	// Check if user is member of the group
	var role string
	err = db.Instance.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		groupID, userID).Scan(&role)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	rows, err := db.Instance.Query(`
		SELECT e.event_id, e.group_id, e.creator_id, e.title, e.description, e.event_time, e.created_at, u.nickname,
		       COALESCE(er.response, '') as user_response
		FROM events e
		JOIN users u ON e.creator_id = u.id
		LEFT JOIN event_responses er ON e.event_id = er.event_id AND er.user_id = ?
		WHERE e.group_id = ?
		ORDER BY e.event_time ASC
	`, userID, groupID)
	if err != nil {
		log.Printf("[Groups] Query events failed: %v", err)
		http.Error(w, "Error retrieving events", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.EventID, &event.GroupID, &event.CreatorID, &event.Title,
			&event.Description, &event.EventTime, &event.CreatedAt, &event.CreatorName, &event.UserResponse); err != nil {
			log.Printf("[Groups] Scan event failed: %v", err)
			continue
		}
		events = append(events, event)
	}

	log.Printf("[Groups] Returning %d events for group %d", len(events), groupID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// Get single event by ID
func GetEventByIDHandler(w http.ResponseWriter, r *http.Request) {
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

	var event Event
	var groupID int
	err = db.Instance.QueryRow(`
		SELECT e.event_id, e.group_id, e.creator_id, e.title, e.description, e.event_time, e.created_at, u.nickname,
		       COALESCE(er.response, '') as user_response
		FROM events e
		JOIN users u ON e.creator_id = u.id
		LEFT JOIN event_responses er ON e.event_id = er.event_id AND er.user_id = ?
		WHERE e.event_id = ?
	`, userID, eventID).Scan(&event.EventID, &event.GroupID, &event.CreatorID, &event.Title,
		&event.Description, &event.EventTime, &event.CreatedAt, &event.CreatorName, &event.UserResponse)

	if err == sql.ErrNoRows {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("[Groups] Query event by ID failed: %v", err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	groupID = event.GroupID

	// Check if user is member of the group
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

	// Get all event responses
	rows, err := db.Instance.Query(`
		SELECT er.event_id, er.user_id, er.response, er.responded_at, u.nickname
		FROM event_responses er
		JOIN users u ON er.user_id = u.id
		WHERE er.event_id = ?
		ORDER BY er.responded_at DESC
	`, eventID)
	if err != nil {
		log.Printf("[Groups] Query event responses failed: %v", err)
		http.Error(w, "Error retrieving event responses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var responses []EventResponse
	for rows.Next() {
		var response EventResponse
		if err := rows.Scan(&response.EventID, &response.UserID, &response.Response,
			&response.RespondedAt, &response.UserName); err != nil {
			log.Printf("[Groups] Scan event response failed: %v", err)
			continue
		}
		responses = append(responses, response)
	}

	// Create response object with event details and all responses
	eventWithResponses := map[string]interface{}{
		"event_id":      event.EventID,
		"group_id":      event.GroupID,
		"creator_id":    event.CreatorID,
		"title":         event.Title,
		"description":   event.Description,
		"event_time":    event.EventTime,
		"created_at":    event.CreatedAt,
		"creator_name":  event.CreatorName,
		"user_response": event.UserResponse,
		"responses":     responses,
		"response_counts": map[string]int{
			"going":     0,
			"not_going": 0,
		},
	}

	// Count responses
	goingCount := 0
	notGoingCount := 0
	for _, resp := range responses {
		if resp.Response == "going" {
			goingCount++
		} else if resp.Response == "not_going" {
			notGoingCount++
		}
	}
	eventWithResponses["response_counts"] = map[string]int{
		"going":     goingCount,
		"not_going": notGoingCount,
		"total":     len(responses),
	}

	log.Printf("[Groups] Returning event %d with %d responses", eventID, len(responses))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(eventWithResponses)
}

// Get event responses
func GetEventResponsesHandler(w http.ResponseWriter, r *http.Request) {
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
	eventIDStr = strings.TrimSuffix(eventIDStr, "/responses")
	eventID, err := strconv.Atoi(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	// Check if user is member of the group that the event belongs to
	var groupID int
	err = db.Instance.QueryRow("SELECT group_id FROM events WHERE event_id = ?", eventID).Scan(&groupID)
	if err != nil {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	var role string
	err = db.Instance.QueryRow("SELECT role FROM group_memberships WHERE group_id = ? AND user_id = ? AND status = 'accepted'",
		groupID, userID).Scan(&role)
	if err != nil {
		http.Error(w, "You are not a member of this group", http.StatusForbidden)
		return
	}

	rows, err := db.Instance.Query(`
		SELECT er.event_id, er.user_id, er.response, er.responded_at, u.nickname
		FROM event_responses er
		JOIN users u ON er.user_id = u.id
		WHERE er.event_id = ?
		ORDER BY er.responded_at DESC
	`, eventID)
	if err != nil {
		log.Printf("[Groups] Query event responses failed: %v", err)
		http.Error(w, "Error retrieving event responses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var responses []EventResponse
	for rows.Next() {
		var response EventResponse
		if err := rows.Scan(&response.EventID, &response.UserID, &response.Response,
			&response.RespondedAt, &response.UserName); err != nil {
			log.Printf("[Groups] Scan event response failed: %v", err)
			continue
		}
		responses = append(responses, response)
	}

	log.Printf("[Groups] Returning %d responses for event %d", len(responses), eventID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responses)
}

// Helper function to handle dynamic event routes
func HandleEventDynamicRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Handle /event/{id}/responses
	if strings.HasSuffix(path, "/responses") && strings.HasPrefix(path, "/event/") {
		user.JwtMiddleware(GetEventResponsesHandler)(w, r)
		return
	}

	// Handle single event by ID /event/{id}
	if !strings.Contains(strings.TrimPrefix(path, "/event/"), "/") {
		user.JwtMiddleware(GetEventByIDHandler)(w, r)
		return
	}

	http.Error(w, "Event route not found", http.StatusNotFound)
}
