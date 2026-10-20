package models

// QueueMember is one pull request's part of the queue: its comments, the
// reviews sent from them, and for each action on offer its set and words.
type QueueMember struct {
	Pull     int            `json:"pull"`
	Head     string         `json:"head"`
	Title    string         `json:"title"`
	URL      string         `json:"url"`
	Author   string         `json:"author"`
	State    string         `json:"state"`
	EndedAt  *string        `json:"ended_at"`
	Sets     map[string]Set `json:"sets"`
	Comments []Comment      `json:"comments"`
	Posted   []Posted       `json:"posted"`
}
