package model

// ContextOptions says which unchanged types ChooseContext keeps: with
// Enabled, up to Max of the best-scoring, the names in Include always and
// those in Exclude never.
type ContextOptions struct {
	Enabled bool
	Max     int
	Include []string
	Exclude []string
}
