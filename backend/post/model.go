package post

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
