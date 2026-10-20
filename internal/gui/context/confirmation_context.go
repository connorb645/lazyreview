package context

// ConfirmationContext asks whether to drop a queued comment, of a pull
// request, at the version shown.
type ConfirmationContext struct {
	Comment int
	Pull    int
	Version string
}
