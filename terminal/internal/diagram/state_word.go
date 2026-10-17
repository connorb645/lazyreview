package diagram

import "github.com/connorb645/lazyreview/terminal/internal/i18n"

// StateWord is the word a card shows for a type's state: new, changed,
// removed or unchanged, or nothing.
func StateWord(tr *i18n.TranslationSet, state string) string {
	switch state {
	case "new":
		return tr.New
	case "changed":
		return tr.Changed
	case "removed":
		return tr.Removed
	case "same", "context":
		return tr.Unchanged
	}
	return ""
}
