package gui

import "github.com/connorb645/lazyreview/terminal/internal/gui/types"

type reviewReloaded struct {
	data  types.ReviewData
	notes []string
	err   error
}
