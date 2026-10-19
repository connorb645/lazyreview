package models

// State is everything a review holds apart from what it draws: its pull
// requests, or its branch, its queue, its conversations with Claude, the
// review threads on GitHub and its written review. Version is a short hash
// of the rest, which the change notices name.
type State struct {
	ID       string         `json:"id"`
	Kind     string         `json:"kind"`
	Host     string         `json:"host"`
	Repo     string         `json:"repo"`
	Title    string         `json:"title"`
	Me       string         `json:"me"`
	Account  *string        `json:"account"`
	Base     Ref            `json:"base"`
	Head     Ref            `json:"head"`
	LoadedAt string         `json:"loaded_at"`
	Notes    []string       `json:"notes"`
	Moved    *Moved         `json:"moved"`
	Members  []Member       `json:"members"`
	Earlier  []Member       `json:"earlier"`
	Queue    *Queue         `json:"queue"`
	Threads  []Conversation `json:"threads"`
	GitHub   []GitHubThread `json:"github"`
	Written  *Written       `json:"written"`
	Version  string         `json:"version"`
}
