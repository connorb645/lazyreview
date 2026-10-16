package server

// Conversation is a conversation with Claude on lines of the review.
type Conversation struct {
	ID        int       `json:"id"`
	Path      string    `json:"path"`
	Side      string    `json:"side"`
	Line      int       `json:"line"`
	StartLine int       `json:"start_line"`
	StartSide string    `json:"start_side"`
	CreatedAt string    `json:"created_at"`
	Messages  []Message `json:"messages"`
}
