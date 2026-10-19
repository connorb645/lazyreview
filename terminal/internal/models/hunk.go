package models

// Hunk is one hunk of a file's diff, row by row.
type Hunk struct {
	Header   string `json:"header"`
	OldStart int    `json:"old_start"`
	OldCount int    `json:"old_count"`
	NewStart int    `json:"new_start"`
	NewCount int    `json:"new_count"`
	Rows     []Row  `json:"rows"`
}
