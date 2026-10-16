package app

// posting is the panel that posts a pull request's queue: the actions on
// offer, the one chosen, and whether it is showing what that one sends.
type posting struct {
	pull       int
	actions    []string
	choice     int
	confirming bool
}
