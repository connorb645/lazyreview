package server

// Posted is a review sent to GitHub from the queue.
type Posted struct {
	Set    string `json:"set"`
	Action string `json:"action"`
	Review string `json:"review"`
	At     string `json:"at"`
	Count  int    `json:"count"`
}
