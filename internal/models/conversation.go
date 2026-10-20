package models

// Conversation is a conversation with Claude on lines of the review, as its
// file keeps it: the lines, the messages, the Claude Code session that a
// follow-up goes on in once Claude was first asked, and what Claude has done
// so far while it answers. Code is the quote conversations kept before they
// stopped keeping one, written back as it was read.
type Conversation struct {
	ID        int       `json:"id"`
	Path      string    `json:"path"`
	Side      string    `json:"side"`
	Line      int       `json:"line"`
	StartLine int       `json:"start_line,omitempty"`
	StartSide string    `json:"start_side,omitempty"`
	Code      *string   `json:"code,omitempty"`
	CreatedAt string    `json:"created_at"`
	Messages  []Message `json:"messages"`
	Session   string    `json:"session,omitempty"`
	Working   *Working  `json:"working,omitempty"`
}
