package notification

// Notification represents a notification in the system
type Notification struct {
	ID             int    `json:"notification_id"`
	UserID         int    `json:"user_id"`
	Type           string `json:"type"`
	Message        string `json:"message"`
	ReadStatus     bool   `json:"read_status"`
	CreatedAt      string `json:"created_at"`
	RelatedUserID  *int   `json:"related_user_id,omitempty"`
	RelatedGroupID *int   `json:"related_group_id,omitempty"`
	SenderName     string `json:"sender_name,omitempty"`
	GroupName      string `json:"group_name,omitempty"`
}

// NotificationData for WebSocket transmission
type NotificationData struct {
	Type         string       `json:"type"`
	Notification Notification `json:"notification"`
}
