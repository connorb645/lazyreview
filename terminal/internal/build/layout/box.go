package layout

import (
	"math"

	"github.com/connorb645/lazyreview/terminal/internal/models"
)

type box struct {
	x, y, w, h float64
}

func (b box) rounded() models.Box {
	return models.Box{X: tenths(b.x), Y: tenths(b.y), H: tenths(b.h)}
}

func tenths(v float64) float64 {
	return math.Floor(v*10+0.5) / 10
}
