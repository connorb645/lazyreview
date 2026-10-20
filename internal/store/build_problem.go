package store

import "github.com/connorb645/lazyreview/internal/i18n"

// BuildProblem says why a window drawn from build given may not change a
// review whose page is now build current, or "" when it may: its line
// numbers may name other code now.
func BuildProblem(tr *i18n.TranslationSet, current, given string) string {
	if given == "" {
		return tr.NoBuild
	}
	if given != current {
		return tr.LoadedAgainSince
	}
	return ""
}
