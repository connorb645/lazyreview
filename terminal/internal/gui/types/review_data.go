// Package types holds the values every layer of the gui shares.
package types

import "github.com/connorb645/lazyreview/terminal/internal/models"

// ReviewData is a review as the server gives it: its id, its state and the
// page it draws.
type ReviewData struct {
	ID    string
	State models.State
	Page  models.Page
}
