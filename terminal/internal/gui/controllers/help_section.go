package controllers

import "github.com/connorb645/lazyreview/terminal/internal/gui/presentation"

// HelpSection is a heading of the keys popup and the rows under it.
type HelpSection struct {
	Heading string
	Rows    []presentation.HelpRow
}
