package reply

type Reply struct {
	PostID       int64 `json:"post_id"`
	ParentPostID int64 `json:"parent_post_id"`
}
