package controllers

// Count is the count typed before an action's keys, such as the 5 of 5j:
// N is 1 when none was typed, and Typed says whether one was.
type Count struct {
	N     int
	Typed bool
}
