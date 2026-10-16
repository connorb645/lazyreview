package app

// reviewLoaded is a review the server gave, or why it gave none.
type reviewLoaded struct {
	data reviewData
	err  error
}
