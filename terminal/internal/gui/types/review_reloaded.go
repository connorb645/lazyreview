package types

// ReviewReloaded is the review open loaded again, with the load's notes,
// or why not.
type ReviewReloaded struct {
	Data  ReviewData
	Notes []string
	Err   error
}
