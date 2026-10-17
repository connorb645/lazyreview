package gui

import "github.com/connorb645/lazyreview/terminal/internal/gui/types"

type reviewLoaded struct {
	id   string
	data types.ReviewData
	err  error
}
