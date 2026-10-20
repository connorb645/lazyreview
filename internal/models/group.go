package models

// Group is many look-alike types the diagram draws as one box.
type Group struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Members     []string `json:"members"`
}
