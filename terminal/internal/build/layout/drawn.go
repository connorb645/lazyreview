package layout

import "github.com/connorb645/lazyreview/terminal/internal/models"

type drawn struct {
	components map[string]box
	boxes      map[string]box
	arrows     []models.PackageArrow
	summaries  []models.ComponentArrow
}
