package model

import "github.com/connorb645/lazyreview/internal/build/languages"

// FileImports is the imports of one file.
type FileImports struct {
	Path    string
	Imports []languages.Import
}
