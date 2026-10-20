package layout

import "github.com/connorb645/lazyreview/terminal/internal/models"

type block struct {
	id         string
	area       *models.Area
	components []string
	w, h       float64
	at         map[string]box
	inner      []summaryPair
}
