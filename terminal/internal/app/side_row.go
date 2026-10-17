package app

import "github.com/charmbracelet/lipgloss"

type sideRow struct {
	key        string
	text       string
	title      string
	titleStyle lipgloss.Style
	right      string
	heading    bool
	it         item
}
