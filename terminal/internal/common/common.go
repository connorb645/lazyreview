// Package common holds what every layer of the terminal app shares, built
// once when the app starts.
package common

import "github.com/connorb645/lazyreview/terminal/internal/i18n"

// Common is what every layer shares: the strings the app shows.
type Common struct {
	Tr *i18n.TranslationSet
}
