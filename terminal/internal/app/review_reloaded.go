package app

// reviewReloaded is the review read again, to show in place, with the
// server's notes on loading it again from GitHub.
type reviewReloaded struct {
	data  reviewData
	notes []string
	err   error
}
