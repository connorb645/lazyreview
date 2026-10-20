package config

import "github.com/connorb645/lazyreview/terminal/internal/i18n"

type reader struct {
	tr      *i18n.TranslationSet
	anchors map[string]any
}
