package models

// Choice is one pull request of a stack that can take a comment on an
// unchanged line, at its own path and line.
type Choice struct {
	Pull int    `json:"pull"`
	Path string `json:"path"`
	Line int    `json:"line"`
}
