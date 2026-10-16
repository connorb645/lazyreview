package server

// Summary is one line of the list of every review: a pull request's or a
// stack's, or a branch's with no pull request yet.
type Summary struct {
	ID       string   `json:"id"`
	Kind     string   `json:"kind"`
	Repo     string   `json:"repo"`
	Title    string   `json:"title"`
	LoadedAt string   `json:"loaded_at"`
	Moved    *Moved   `json:"moved"`
	Members  []Member `json:"members"`
	Branch   string   `json:"branch"`
	Base     string   `json:"base"`
	Queued   int      `json:"queued"`
}
