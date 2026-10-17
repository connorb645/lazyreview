package server

// Group is many look-alike types the diagram draws as one box.
type Group struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Members []string `json:"members"`
}
