package models

// GitHubThread is a review thread already on GitHub, as the last load read
// it. At is the line the app shows it on, or nil when it shows on no line.
type GitHubThread struct {
	ID           string          `json:"id"`
	Pull         int             `json:"pull"`
	Path         string          `json:"path"`
	Side         string          `json:"side"`
	Line         int             `json:"line"`
	OriginalLine int             `json:"original_line"`
	Resolved     bool            `json:"resolved"`
	ResolvedBy   string          `json:"resolved_by"`
	Outdated     bool            `json:"outdated"`
	File         bool            `json:"file"`
	Comments     []GitHubComment `json:"comments"`
	More         int             `json:"more"`
	Hunk         string          `json:"hunk"`
	At           *Lines          `json:"at"`
}
