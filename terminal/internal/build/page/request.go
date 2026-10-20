package page

import (
	"github.com/connorb645/lazyreview/terminal/internal/build/components"
	"github.com/connorb645/lazyreview/terminal/internal/models"
)

// Request is what a page is built from: the base, the heads to read,
// bottom first, the base branch the heading names, the title, the subject
// the heading opens with, the config, whether to draw the unchanged code
// the change touches, and the title of a component holding the whole
// repository.
type Request struct {
	Base       string
	Members    []Member
	BaseBranch string
	Title      string
	Subject    []models.HeadingPart
	Config     components.Config
	Context    bool
	RootTitle  string
}
