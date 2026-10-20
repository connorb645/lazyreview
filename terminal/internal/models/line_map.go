package models

// LineMap is where a comment on each line of one side of a file in a
// stack's combined diff goes, by the line's number as text.
type LineMap map[string]LineEntry
