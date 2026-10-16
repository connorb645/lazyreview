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
	if a.screen == screenReview {
		left, right = " "+p.Bold.Render("Review"), ""
		if d := a.review; d != nil {
			title := d.state.Title
			if d.state.Written != nil && d.state.Written.Title != "" {
				title = d.state.Written.Title
			}
			left = " " + p.Bold.Render(title) + p.Muted.Render(" › Review")
			if d.page.Stack != nil {
				var pills []string
				for _, m := range d.state.Members {
					pills = append(pills, p.pull(m.Color).Render(fmt.Sprintf("#%d", m.Pull)))
				}
				right = strings.Join(pills, "  ") + " "
			}
		}
	}
	return spread(left, right, a.width)
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
	case a.screen == screenReview && a.review == nil:
		out = append(out, "", " "+a.p.Muted.Render("No review is open. gh lists every review."))
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
	if a.screen == screenReview {
		place = "Review"
		if a.review != nil && a.rv.view.cursor < len(a.rv.view.items) {
			if words := itemWords(a.rv.view.items[a.rv.view.cursor]); words != "" {
				place += " · " + words
			}
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
	case a.panel != nil:
		hints = [][2]string{{"j k", "move"}, {"q", "close"}}
	case a.screen == screenReview:
		hints = [][2]string{{"j k", "move"}, {"]] [[", "sections"}, {"za", "fold"}, {"gh", "every review"}, {"Space ?", "keys"}, {":q", "quit"}}
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
