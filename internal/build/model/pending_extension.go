package model

import (
	"github.com/connorb645/lazyreview/internal/build/diff"
	"github.com/connorb645/lazyreview/internal/build/languages"
)

type pendingExtension struct {
	file        diff.File
	language    *languages.Language
	extended    string
	declaration languages.Declaration
	code        string
}
