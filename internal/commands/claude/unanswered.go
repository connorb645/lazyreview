package claude

// Unanswered is why Claude gave no answer. NoSession says the session a
// follow-up was to go on from was not found.
type Unanswered struct {
	Why       string
	NoSession bool
}

// Error is why, in words.
func (u Unanswered) Error() string {
	return u.Why
}
