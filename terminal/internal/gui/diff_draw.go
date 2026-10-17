package gui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const gutter = 14

func badges(tr *i18n.TranslationSet) map[string]string {
	return map[string]string{"A": tr.New, "D": tr.Deleted, "R": tr.Renamed, "M": tr.Changed, "": tr.Unchanged}
}

func diffLines(tr *i18n.TranslationSet, d reviewData, v diffView, height int, filter pullFilter, p palette, active bool) []string {
	var out []string
	for i := v.top; i < len(v.rows) && len(out) < height; i++ {
		lines := drawRow(tr, d, v, i, filter, p, active)
		if !p.plain && (i == v.cursor || v.selecting && i >= min(v.anchor, v.cursor) && i <= max(v.anchor, v.cursor)) {
			for j := range lines {
				lines[j] = p.highlight(lines[j], v.width, active)
			}
		}
		out = append(out, lines...)
	}
	if len(out) > height {
		out = out[:height]
	}
	return out
}

func drawRow(tr *i18n.TranslationSet, d reviewData, v diffView, i int, filter pullFilter, p palette, active bool) []string {
	r := v.rows[i]
	mark := " "
	switch {
	case !p.plain:
	case i == v.cursor && active:
		mark = "▌"
	case v.selecting && i >= min(v.anchor, v.cursor) && i <= max(v.anchor, v.cursor):
		mark = "┃"
	}
	switch r.kind {
	case "notehead", "notetext":
		return []string{mark + noteLine(tr, d, v, r, p)}
	case "block":
		return []string{mark + blockLine(v, i, p)}
	case "file":
		return []string{mark + fileHeader(tr, d, v, r, p)}
	case "hunk":
		return []string{mark + "  " + p.Muted.Render(r.text)}
	case "gap":
		return []string{mark + "  " + p.Muted.Render(fmt.Sprintf("··· %s ···", utils.Plural(r.count, tr.UnchangedLineNoun))) + "  " + p.Key.Render("zo")}
	case "note":
		return []string{mark + "  " + p.Muted.Render(r.text)}
	}
	if r.line == nil {
		return splitRow(d, v, r, mark, filter, p)
	}
	return unifiedRow(d, v, r, mark, filter, p)
}

const blockGutter = 12

func blockLine(v diffView, i int, p palette) string {
	r := v.rows[i]
	b := r.block
	label := ""
	if i == 0 || v.rows[i-1].block != b {
		label = blockLabel(b)
	}
	bar := " "
	switch b.State {
	case "add":
		bar = p.Add.Render("▌")
	case "del":
		bar = p.Del.Render("▌")
	}
	return p.Muted.Render(fmt.Sprintf("%8s ", label)) + bar + p.Muted.Render("│") + " " + r.text
}

func blockLabel(b *server.Block) string {
	label := fmt.Sprint(b.StartLine)
	if b.Line != b.StartLine {
		label += fmt.Sprintf("-%d", b.Line)
	}
	if b.Side == "LEFT" {
		label = "-" + label
	}
	return label
}

func noteLine(tr *i18n.TranslationSet, d reviewData, v diffView, r diffRow, p palette) string {
	indent := strings.Repeat(" ", gutter-1)
	n, ok := noteOf(d, r.path, r.key)
	if !ok {
		return ""
	}
	if r.kind == "notehead" {
		head, keys := noteHead(tr, d, n, v.noteOpen(n))
		style := map[string]lipgloss.Style{"queued": p.Accent, "conversation": p.Bold, "thread": p.Bold}[n.kind]
		if n.kind == "queued" {
			style = p.pull(memberColor(d.state, n.pull))
		}
		return spread(indent+style.Render(head), p.Key.Render(keys)+" ", v.width-1)
	}
	style := map[string]lipgloss.Style{"body": p.Ink, "you": p.Bold, "claude": p.Ink, "error": p.Bad, "stale": p.Warn, "author": p.Muted, "waiting": p.Muted}[r.part]
	return indent + "  " + style.Render(r.text)
}

func fileHeader(tr *i18n.TranslationSet, d reviewData, v diffView, r diffRow, p palette) string {
	fold := "▾"
	if v.folded[r.path] {
		fold = "▸"
	}
	entry := fileEntry(d, r.path)
	head, tail := " "+p.Muted.Render(fold)+" ", "  "+p.Muted.Render(badges(tr)[entry.Status])
	for _, pull := range entry.Pulls {
		tail += "  " + p.pull(memberColor(d.state, pull)).Render(fmt.Sprintf("#%d", pull))
	}
	if v.whole[r.path] {
		tail += "  " + p.Accent.Render(tr.WholeBadge)
	}
	if v.rendered[r.path] {
		tail += "  " + p.Accent.Render(tr.RenderedBadge)
	}
	if entry.Status == "" {
		tail += "  " + p.Muted.Render(utils.Plural(entry.Lines, tr.LineNoun))
	} else {
		tail += "  " + p.Add.Render(fmt.Sprintf("+%d", entry.Adds)) + " " + p.Del.Render(fmt.Sprintf("−%d", entry.Dels))
	}
	path := r.path
	if room := v.width - 1 - ansi.StringWidth(head+tail); room > 0 {
		path = utils.FitTitle(path, room)
	}
	return head + p.Bold.Render(path) + tail
}

func unifiedRow(d reviewData, v diffView, r diffRow, mark string, filter pullFilter, p palette) []string {
	l := r.line
	sign := map[string]string{"add": "+", "del": "-", "same": " ", "note": " "}[l.Kind]
	if r.added {
		sign = "+"
	}
	head := mark + stripe(d, l, p) + p.Muted.Render(fmt.Sprintf("%4s %4s ", number(l.Old), number(l.New)))
	style := rowStyle(l, r.added, filter, p)
	body := v.width - gutter
	texts := cut(expandTabs(l.Text), body, v.wrap)
	out := []string{head + style.Render(sign+" "+texts[0])}
	for _, more := range texts[1:] {
		out = append(out, strings.Repeat(" ", gutter)+style.Render(more))
	}
	return out
}

func splitRow(d reviewData, v diffView, r diffRow, mark string, filter pullFilter, p palette) []string {
	half := (v.width - 3) / 2
	side := func(l *server.Row, old bool) []string {
		if l == nil {
			return []string{strings.Repeat(" ", half)}
		}
		n := l.New
		sign := "+"
		if old {
			n = l.Old
			sign = "-"
		}
		if l.Kind == "same" {
			sign = " "
		}
		style := rowStyle(l, false, filter, p)
		texts := cut(expandTabs(l.Text), half-7, v.wrap)
		out := []string{p.Muted.Render(fmt.Sprintf("%4s ", number(n))) + style.Render(sign+" "+texts[0])}
		for _, more := range texts[1:] {
			out = append(out, "       "+style.Render(more))
		}
		for i := range out {
			out[i] = utils.Fit(out[i], half)
		}
		return out
	}
	left, right := side(r.left, true), side(r.right, false)
	owner := r.right
	if owner == nil || owner.Kind == "same" {
		owner = r.left
	}
	var out []string
	for i := 0; i < max(len(left), len(right)); i++ {
		a, b := strings.Repeat(" ", half), strings.Repeat(" ", half)
		if i < len(left) {
			a = left[i]
		}
		if i < len(right) {
			b = right[i]
		}
		lead := "  "
		if i == 0 {
			lead = mark + stripe(d, owner, p)
		}
		out = append(out, lead+a+p.Muted.Render("│")+b)
	}
	return out
}

func stripe(d reviewData, l *server.Row, p palette) string {
	if l == nil || l.Owner == 0 || l.Kind == "same" {
		return " "
	}
	return p.pull(memberColor(d.state, l.Owner)).Render("▎")
}

func rowStyle(l *server.Row, added bool, filter pullFilter, p palette) lipgloss.Style {
	if l.Kind != "same" && !filter.shows(l.Owner) {
		return p.Muted
	}
	switch {
	case l.Kind == "add" || added:
		return p.Add
	case l.Kind == "del":
		return p.Del
	case l.Kind == "note":
		return p.Muted
	}
	return p.Ink
}

func rowHeight(r diffRow, v diffView) int {
	if !v.wrap || r.kind != "line" {
		return 1
	}
	if r.line != nil {
		return len(cut(expandTabs(r.line.Text), v.width-gutter, true))
	}
	half := (v.width - 3) / 2
	h := 1
	for _, l := range []*server.Row{r.left, r.right} {
		if l != nil {
			h = max(h, len(cut(expandTabs(l.Text), half-7, true)))
		}
	}
	return h
}

func cut(text string, width int, wrapping bool) []string {
	if width < 4 {
		width = 4
	}
	if !wrapping || ansi.StringWidth(text) <= width {
		return []string{ansi.Truncate(text, width, "…")}
	}
	return strings.Split(ansi.Hardwrap(text, width, true), "\n")
}

func number(n int) string {
	if n == 0 {
		return ""
	}
	return fmt.Sprint(n)
}

func fileEntry(d reviewData, path string) server.FileEntry {
	for _, f := range d.page.Model.Files {
		if f.Path == path {
			return f
		}
	}
	for _, f := range d.page.Model.ContextFiles {
		if f.Path == path {
			return f
		}
	}
	return server.FileEntry{Path: path}
}
