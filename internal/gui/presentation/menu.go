package presentation

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/gui/views"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/theme"
	"github.com/connorb645/lazyreview/internal/utils"
)

// Menu is a popup list as it is drawn: its title, its rows, the cursor and
// the first row shown, and, for the finder, what has been typed.
type Menu struct {
	Title   string
	Rows    []MenuRow
	Cursor  int
	Top     int
	Finding bool
	Query   string
}

// Choosing says whether the menu's rows are there to choose from, and so
// show a cursor, rather than only to read.
func (m Menu) Choosing() bool {
	for _, r := range m.Rows {
		if r.Open != nil || r.Left != "" {
			return true
		}
	}
	return m.Finding
}

// Opens says whether a row of the menu opens a place.
func (m Menu) Opens() bool {
	for _, r := range m.Rows {
		if r.Open != nil {
			return true
		}
	}
	return false
}

func (m Menu) choices() (int, int) {
	at, n := 0, 0
	for i, r := range m.Rows {
		if r.Heading != "" {
			continue
		}
		n++
		if i <= m.Cursor {
			at = n
		}
	}
	return at, n
}

// Box draws the menu in a frame sized to its rows and the screen's width,
// at most height lines high, from the row that keeps the cursor in view.
func (m Menu) Box(tr *i18n.TranslationSet, th theme.Theme, width, height int) []string {
	left, right := 0, 0
	for _, r := range m.Rows {
		left = max(left, ansi.StringWidth(r.Left))
		right = max(right, ansi.StringWidth(r.Right))
	}
	w := min(width-4, max(left+right+7, width/2, 44))
	if !m.Choosing() {
		w = min(width-4, max(width*3/5, 60))
	}
	inner := w - 2
	rows := height - 2
	var content []string
	if m.Finding {
		rows--
		content = append(content, " "+th.Accent.Render("›")+" "+th.Text.Render(m.Query)+th.Accent.Render("▏"))
	}
	if m.Finding && len(m.Rows) == 0 {
		content = append(content, " "+th.Muted.Render(tr.NothingMatches))
	}
	drawn := make([][]string, len(m.Rows))
	for i, r := range m.Rows {
		drawn[i] = m.drawRow(th, r, i == m.Cursor && m.Choosing(), left, inner)
	}
	top := min(m.Top, m.Cursor)
	for top < m.Cursor && linesIn(drawn, top, m.Cursor) > rows {
		top++
	}
	for i := top; i < len(drawn) && len(content) < height-2; i++ {
		for _, l := range drawn[i] {
			if len(content) < height-2 {
				content = append(content, l)
			}
		}
	}
	footer := ""
	if at, n := m.choices(); n > 0 && m.Choosing() {
		footer = OfTotal(tr, at, n)
	}
	h := len(content) + 2
	if m.Finding {
		h = height
	}
	return views.Framed(content, views.Extent{}, w, h, m.Title, "", footer, true, th)
}

func (m Menu) drawRow(th theme.Theme, r MenuRow, selected bool, left, inner int) []string {
	if r.Heading != "" {
		return []string{" " + th.ActiveBorder.Bold(true).Render(r.Heading)}
	}
	gap := 0
	if left > 0 {
		gap = left + 2
	}
	room := max(8, inner-gap-2)
	words := []string{r.Right}
	if !m.Finding {
		words = utils.Wrap(r.Right, room)
	}
	var out []string
	for i, text := range words {
		key := ""
		if i == 0 {
			key = r.Left
		}
		fitted := ansi.Truncate(text, room, "…")
		if m.Finding {
			fitted = utils.FitTitle(text, room)
		}
		line := " " + th.Key.Render(key) + strings.Repeat(" ", max(0, gap-ansi.StringWidth(key))) + th.Text.Render(fitted)
		if selected {
			line = th.Selected(line, inner, true)
		}
		out = append(out, line)
	}
	return out
}

func linesIn(drawn [][]string, from, to int) int {
	n := 0
	for i := from; i <= to && i < len(drawn); i++ {
		n += len(drawn[i])
	}
	return n
}
