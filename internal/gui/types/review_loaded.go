package types

// ReviewLoaded is a review opened, or why it could not be.
type ReviewLoaded struct {
	ID   string
	Data ReviewData
	Err  error
}
