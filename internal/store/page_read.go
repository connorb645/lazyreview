package store

import "github.com/connorb645/lazyreview/internal/models"

// PageRead is a review's page with its build id, the short hash of
// page.json's text that names exactly the diff and the diagram a window was
// drawn from.
type PageRead struct {
	Build string
	Page  *models.PageRecord
}
