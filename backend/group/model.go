package group

type Group struct {
	GroupID     int    `json:"group_id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	CreatorID   int    `json:"creator_id"`
	CreatedAt   string `json:"created_at"`
	CreatorName string `json:"creator_name,omitempty"`
	MemberCount int    `json:"member_count,omitempty"`
	UserRole    string `json:"user_role,omitempty"` // creator, admin, member, or null if not a member
}

type GroupMembership struct {
	UserID   int    `json:"user_id"`
	GroupID  int    `json:"group_id"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	JoinedAt string `json:"joined_at,omitempty"`
	UserName string `json:"user_name,omitempty"`
}

type GroupPost struct {
	ID        int    `json:"post_id"`
	UserID    int    `json:"user_id"`
	GroupID   int    `json:"group_id"`
	Content   string `json:"content"`
	Media     string `json:"media,omitempty"`
	CreatedAt string `json:"created_at"`
	Nickname  string `json:"nickname,omitempty"`
}
