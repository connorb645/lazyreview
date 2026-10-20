package models

// Notice is a change notice for a review: the state's new version, the
// page's new build, or both at once when a client first listens.
type Notice struct {
	State string `json:"state"`
	Build string `json:"build"`
}
