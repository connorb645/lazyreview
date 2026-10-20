// Package components reads the config a change is drawn with: its
// components, packages, areas, named links, words on the arrows between
// components, groups of look-alike types and how much unchanged code to
// draw around it.
package components

// Config is how a change is drawn, every optional part filled in. With no
// config at all, files are grouped by folder.
type Config struct {
	Title        string
	Summary      string
	Areas        []Area
	Components   []Component
	Links        []Link
	Summaries    []Summary
	Groups       []Group
	ContextTypes ContextTypes
}
