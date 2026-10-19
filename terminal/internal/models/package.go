package models

// Package is a folder of a component the diagram frames. A bare package is
// drawn with no header of its own.
type Package struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Path        string `json:"path"`
	Bare        bool   `json:"bare"`
}
