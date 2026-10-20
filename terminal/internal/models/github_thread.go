package models

// GitHubThread is a review thread already on GitHub, as the last load read
// it. Line, StartLine and OriginalLine are 0 and ResolvedBy and StartSide
// empty where GitHub gave none. At is the line the app shows it on, or nil
// when it shows on no line.
type GitHubThread struct {
	ID           string                `json:"id"`
	Pull         int                   `json:"pull"`
	Path         string                `json:"path"`
	Side         string                `json:"side"`
	Line         int                   `json:"line"`
	StartLine    int                   `json:"start_line"`
	StartSide    string                `json:"start_side"`
	OriginalLine int                   `json:"original_line"`
	Resolved     bool                  `json:"resolved"`
	ResolvedBy   string                `json:"resolved_by"`
	Outdated     bool                  `json:"outdated"`
	File         bool                  `json:"file"`
	Comments     []GitHubThreadComment `json:"comments"`
	More         int                   `json:"more"`
	Hunk         string                `json:"hunk"`
	At           *Lines                `json:"at"`
}

// MarshalJSON writes the thread as github.json has always kept it: null
// for a line, a side or a login GitHub gave none of.
func (t GitHubThread) MarshalJSON() ([]byte, error) {
	return plainJSON(struct {
		ID           string                `json:"id"`
		Pull         int                   `json:"pull"`
		Path         string                `json:"path"`
		Side         string                `json:"side"`
		Line         *int                  `json:"line"`
		StartLine    *int                  `json:"start_line"`
		StartSide    *string               `json:"start_side"`
		OriginalLine *int                  `json:"original_line"`
		Resolved     bool                  `json:"resolved"`
		ResolvedBy   *string               `json:"resolved_by"`
		Outdated     bool                  `json:"outdated"`
		File         bool                  `json:"file"`
		Comments     []GitHubThreadComment `json:"comments"`
		More         int                   `json:"more"`
		Hunk         string                `json:"hunk"`
		At           *Lines                `json:"at"`
	}{t.ID, t.Pull, t.Path, t.Side, nonZero(t.Line), nonZero(t.StartLine), nonEmpty(t.StartSide), nonZero(t.OriginalLine),
		t.Resolved, nonEmpty(t.ResolvedBy), t.Outdated, t.File, orNone(t.Comments), t.More, t.Hunk, t.At})
}

func nonZero(n int) *int {
	if n == 0 {
		return nil
	}
	return &n
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
