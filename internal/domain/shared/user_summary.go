package shared

type AccessInfo struct {
	IsAllowed bool
	Role      Role
}

type UserSummary struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	PostCount int    `json:"post_count"`
	IsAllowed bool   `json:"is_allowed"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type ParentPostInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Content  string `json:"content"`
}
