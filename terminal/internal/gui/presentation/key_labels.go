package presentation

import (
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/keys"
)

// KeyLabels label the keys bound to actions, as the keys popup and the bar
// show them: the keys of the place the user is in, or, for an action the
// place does not bind, the keys bound to it anywhere.
type KeyLabels struct {
	place map[string][]string
	all   map[string][]string
}

// NewKeyLabels labels the keys of a place, and of every place, each a map
// from an action to its key sequences, as keys.Reader names them.
func NewKeyLabels(place, all map[string][]string) KeyLabels {
	return KeyLabels{place: place, all: all}
}

// Of labels the first key of each action, one after another.
func (l KeyLabels) Of(actions ...string) string {
	labels := make([]string, len(actions))
	for i, action := range actions {
		labels[i] = l.Nth(action, 0)
	}
	return strings.Join(labels, " ")
}

// Nth labels an action's key at a place in its list, or its first key when
// it has fewer.
func (l KeyLabels) Nth(action string, n int) string {
	sequences, ok := l.place[action]
	if !ok {
		sequences = l.all[action]
	}
	if len(sequences) == 0 {
		return ""
	}
	if n >= len(sequences) {
		n = 0
	}
	return keys.Label(sequences[n])
}
