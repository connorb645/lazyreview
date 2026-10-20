package layout

import "github.com/connorb645/lazyreview/terminal/internal/models"

type laid struct {
	w, h   float64
	boxes  map[string]box
	arrows []models.PackageArrow
}
