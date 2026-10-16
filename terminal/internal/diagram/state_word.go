package diagram

// StateWord is the word a card shows for a type's state: new, changed,
// removed or unchanged, or nothing.
func StateWord(state string) string {
	switch state {
	case "new", "changed", "removed":
		return state
	case "same", "context":
		return "unchanged"
	}
	return ""
}
