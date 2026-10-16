package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// View draws the screen: a title line, the view, a status line and a line
// of key hints.
func (a App) View() string {
	if a.width == 0 || a.height == 0 {
		return ""
	}
	out := []string{a.titleLine()}
	out = append(out, a.bodyLines()...)
	out = append(out, a.statusLine(), a.hintLine())
	return strings.Join(out, "\n")
}

func (a App) titleLine() string {
	p := a.p
	left, right := " "+p.Bold.Render("Every review"), p.Muted.Render(plural(len(a.reviews.reviews), "review")+" ")
	d := a.review
	switch {
	case a.screen == screenReviews:
	case d == nil:
		left, right = " "+p.Bold.Render("Review"), ""
	case a.screen == screenDiagram:
		left, right = a.diagramTitle(), a.filterPills()
	case a.screen == screenDiff:
		left, right = " "+p.Bold.Render(componentTitle(*d, a.diff.component)), a.filterPills()
		if path := a.diff.file(); path != "" {
			left += p.Muted.Render(" › ") + p.Ink.Render(path)
		}
	default:
		title := d.state.Title
		if d.state.Written != nil && d.state.Written.Title != "" {
			title = d.state.Written.Title
		}
		left, right = " "+p.Bold.Render(title)+p.Muted.Render(" › Review"), a.filterPills()
	}
	return spread(left, right, a.width)
}

// filterPills name the pull request filter on a stack: All, then each pull request, the one in use marked.
func (a App) filterPills() string {
	p := a.p
	if a.review == nil || a.review.page.Stack == nil {
		return ""
	}
	pills := []string{p.Muted.Render("All")}
	if a.filter == 0 {
		pills[0] = p.Bold.Render("[All]")
	}
	for _, m := range a.review.state.Members {
		pill := fmt.Sprintf("#%d", m.Pull)
		if a.filter == m.Pull {
			pill = "[" + pill + "]"
		}
		pills = append(pills, p.pull(m.Color).Render(pill))
	}
	return strings.Join(pills, "  ") + " "
}

func componentTitle(d reviewData, id string) string {
	for _, c := range d.page.Model.Components {
		if c.ID == id {
			return c.Title
		}
	}
	return id
}

func (a App) bodyLines() []string {
	h := a.bodyHeight()
	view := a.reviews.view
	if a.screen == screenReview {
		view = a.rv.view
	}
	var out []string
	switch {
	case a.loading:
		out = append(out, "", " "+a.p.Muted.Render("Loading…"))
	case a.screen != screenReviews && a.review == nil:
		out = append(out, "", " "+a.p.Muted.Render("No review is open. gh lists every review."))
	case a.screen == screenDiff:
		out = diffLines(*a.review, a.diff, h, a.filter, a.p)
	case a.screen == screenDiagram && len(a.dg.level.Cards) == 0:
		out = append(out, "", " "+a.p.Muted.Render("Nothing to draw: this change has no parts the diagram knows."))
	case a.screen == screenDiagram:
		out = a.dg.lines(*a.review, a.width, h, a.filter, a.styles)
	default:
		for i := view.top; i < view.top+h && i < len(view.lines); i++ {
			l := view.lines[i]
			text := l.text
			if l.item >= 0 && l.item == view.cursor {
				text = a.p.Accent.Render("▌") + strings.TrimPrefix(text, " ")
			}
			out = append(out, text)
		}
	}
	for len(out) < h {
		out = append(out, "")
	}
	for i := range out {
		out[i] = fit(out[i], a.width)
	}
	if a.panel != nil {
		drawn := a.panel.lines(a.width, min(h, a.panelRows()+3), a.p)
		copy(out[h-len(drawn):], drawn)
	}
	return out
}

func (a App) statusLine() string {
	p := a.p
	if a.prompting != "" {
		return fit(a.prompt.View(), a.width)
	}
	place := "every review"
	switch a.screen {
	case screenReview:
		place = "Review"
		if a.review != nil && a.rv.view.cursor < len(a.rv.view.items) {
			if words := itemWords(a.rv.view.items[a.rv.view.cursor]); words != "" {
				place += " · " + words
			}
		}
	case screenDiff:
		place = a.diffPlace()
	case screenDiagram:
		if a.review != nil {
			place = a.boxWords()
		}
	}
	left := " " + p.Key.Render("NORMAL") + "  " + p.Muted.Render(place)
	right := p.Muted.Render(a.keys.Typed()) + " "
	if a.notice != "" {
		style := p.Ink
		if a.bad {
			style = p.Bad
		}
		right = style.Render(a.notice) + " "
	}
	return p.Bar.Render(spread(left, right, a.width))
}

func (a App) hintLine() string {
	p := a.p
	var hints [][2]string
	switch {
	case a.keys.Pending() == "space":
		for _, k := range a.bindings().After("space") {
			hints = append(hints, [2]string{"Space " + k[0], leaderWords[k[1]]})
		}
	case a.panel != nil && a.panel.finding:
		hints = [][2]string{{"type", "narrow"}, {"Ctrl-n Ctrl-p", "move"}, {"Enter", "go"}, {"Esc", "close"}}
	case a.panel != nil && len(a.panel.rows) > 0 && a.panel.rows[0].open != nil:
		hints = [][2]string{{"j k", "move"}, {"Enter", "go"}, {"q", "close"}}
	case a.panel != nil:
		hints = [][2]string{{"j k", "move"}, {"q", "close"}}
	case a.screen == screenDiagram && a.dg.labels != nil:
		hints = [][2]string{{"a label", "jumps to its box"}, {"Esc", "stop"}}
	case a.screen == screenDiagram:
		hints = a.diagramHints()
	case a.screen == screenReview:
		hints = [][2]string{{"j k", "move"}, {"Enter", "open"}, {"]] [[", "sections"}, {"za", "fold"}, {"gh", "every review"}, {"Space ?", "keys"}, {":q", "quit"}}
	case a.screen == screenDiff:
		hints = [][2]string{{"j k", "move"}, {"]c [c", "hunks"}, {"]f [f", "files"}, {"za", "fold"}, {"/", "search"}, {"-", "back"}, {"Space ?", "keys"}}
	default:
		hints = [][2]string{{"j k", "move"}, {"Enter", "open"}, {"/", "find"}, {"Space ?", "keys"}, {":q", "quit"}}
	}
	var parts []string
	for _, h := range hints {
		parts = append(parts, p.Key.Render(h[0])+" "+p.Muted.Render(h[1]))
	}
	return fit(" "+strings.Join(parts, "   "), a.width)
}

// spread puts left at the start of a line width cells wide and right at its end.
func spread(left, right string, width int) string {
	gap := width - ansi.StringWidth(left) - ansi.StringWidth(right)
	if gap < 1 {
		return fit(left, width)
	}
	return left + strings.Repeat(" ", gap) + right
}

// diffPlace says where the cursor is in the diff: the line, and on a stack
// who changed it and where a comment on it goes.
func (a App) diffPlace() string {
	v := a.diff
	if v.cursor >= len(v.rows) {
		return ""
	}
	r := v.rows[v.cursor]
	side, line, ok := r.at()
	if !ok {
		return map[string]string{"file": "file header", "hunk": "hunk header", "gap": "unchanged lines", "note": ""}[r.kind]
	}
	words := linesWords(side, line, 0, "")
	if w := r.words(); w != "" {
		words += " · " + w
	}
	return words
}

func itemWords(it item) string {
	switch it.kind {
	case "section":
		return it.title
	case "comment":
		return fmt.Sprintf("queued comment %d", it.comment)
	case "link", "thread", "conversation":
		if it.place != nil {
			return fmt.Sprintf("%s %s", it.place.Path, linesWords(it.place.Side, it.place.Line, it.place.StartLine, it.place.StartSide))
		}
		return it.kind
	case "component":
		return it.title
	}
	return ""
}
