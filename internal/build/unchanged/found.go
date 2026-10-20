// Package unchanged finds the unchanged code a change touches, which the
// diagram may draw around it as context.
package unchanged

import "github.com/connorb645/lazyreview/internal/build/model"

// Found is the unchanged types found, as nodes, and the text of every file
// read for them, by path.
type Found struct {
	Nodes []*model.Node
	Texts map[string]*string
}
