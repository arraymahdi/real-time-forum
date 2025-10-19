package event

type Event struct {
	EventID      int    `json:"event_id"`
	GroupID      int    `json:"group_id"`
	CreatorID    int    `json:"creator_id"`
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	EventTime    string `json:"event_time"`
	CreatedAt    string `json:"created_at"`
	CreatorName  string `json:"creator_name,omitempty"`
	UserResponse string `json:"user_response,omitempty"` // going, not_going, or null
}

type EventResponse struct {
	EventID     int    `json:"event_id"`
	UserID      int    `json:"user_id"`
	Response    string `json:"response"`
	RespondedAt string `json:"responded_at"`
	UserName    string `json:"user_name,omitempty"`
}
