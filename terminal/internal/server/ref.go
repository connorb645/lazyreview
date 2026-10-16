package server

// Ref is a branch or a commit a review names, with the commit it was at.
type Ref struct {
	Ref string `json:"ref"`
	Sha string `json:"sha"`
}
