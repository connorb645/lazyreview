package diagram

import model "github.com/connorb645/lazyreview/internal/diagram"

// Scene is a level of the diagram as it is drawn: its cards and links,
// where each card sits, the card in focus, whether every link shows, and the
// labels a key jumps to, with what has been typed of one.
type Scene struct {
	Level  model.Level
	Grid   model.Grid
	Focus  int
	All    bool
	Labels map[string]int
	Typed  string
}

// Card is the card in focus, if the level has one.
func (v Scene) Card() (model.Card, bool) {
	if v.Focus < 0 || v.Focus >= len(v.Level.Cards) {
		return model.Card{}, false
	}
	return v.Level.Cards[v.Focus], true
}
