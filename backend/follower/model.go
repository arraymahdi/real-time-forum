package follower

// Follower represents a follow relationship
type Follower struct {
	ID          int    `json:"id"`
	FollowerID  int    `json:"follower_id"`
	FollowingID int    `json:"following_id"`
	CreatedAt   string `json:"created_at"`
}

// UserWithFollowStatus represents a user with follow status
type UserWithFollowStatus struct {
	ID          int    `json:"id"`
	Nickname    string `json:"nickname"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	IsFollowing bool   `json:"is_following"`
	FollowsYou  bool   `json:"follows_you"`
}
