package comment

// Comment model
type Comment struct {
	ID        int    `json:"id"`
	PostID    int    `json:"post_id"`
	UserID    int    `json:"user_id"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	Nickname  string `json:"nickname,omitempty"`
}
