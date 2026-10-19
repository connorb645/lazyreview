package presentation

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const gutter = 14

func badges(tr *i18n.TranslationSet) map[string]string {
	return map[string]string{"A": tr.New, "D": tr.Deleted, "R": tr.Renamed, "M": tr.Changed, "": tr.Unchanged}
}

// DiffLines draws the rows of a diff from its first one shown, as many as
// fit in a height, the cursor's and the selection's on their backgrounds.
// active is whether the diff has the keys, and labels label the keys a
// note's first row names.
func DiffLines(tr *i18n.TranslationSet, th theme.Theme, labels KeyLabels, d types.ReviewData, v DiffView, height int, filter types.PullFilter, active bool) []string {
	var out []string
	for i := v.Top; i < len(v.Rows) && len(out) < height; i++ {
		lines := drawRow(tr, labels, d, v, i, filter, th, active)
		if !th.Plain && (i == v.Cursor || v.Selecting && i >= min(v.Anchor, v.Cursor) && i <= max(v.Anchor, v.Cursor)) {
			for j := range lines {
				lines[j] = th.Selected(lines[j], v.Width, active)
			}
		}
		out = append(out, lines...)
	}
	if len(out) > height {
		out = out[:height]
	}
	return out
}

func drawRow(tr *i18n.TranslationSet, labels KeyLabels, d types.ReviewData, v DiffView, i int, filter types.PullFilter, th theme.Theme, active bool) []string {
	r := v.Rows[i]
	mark := " "
	switch {
	case !th.Plain:
	case i == v.Cursor && active:
		mark = "▌"
	case v.Selecting && i >= min(v.Anchor, v.Cursor) && i <= max(v.Anchor, v.Cursor):
		mark = "┃"
	}
	switch r.Kind {
	case "notehead", "notetext":
		return []string{mark + noteLine(tr, labels, d, v, r, th)}
	case "block":
		return []string{mark + blockLine(v, i, th)}
	case "file":
		return []string{mark + fileHeader(tr, d, v, r, filter, th)}
	case "hunk":
		return []string{mark + "  " + th.Muted.Render(r.Text)}
	case "gap":
		return []string{mark + "  " + th.Muted.Render(fmt.Sprintf("··· %s ···", utils.Plural(r.Count, tr.UnchangedLineNoun))) + "  " + th.Key.Render("zo")}
	case "note":
		return []string{mark + "  " + th.Muted.Render(r.Text)}
	}
	if r.Line == nil {
		return splitRow(d, v, r, mark, filter, th)
	}
	return unifiedRow(d, v, r, mark, filter, th)
}

const blockGutter = 12

func blockLine(v DiffView, i int, th theme.Theme) string {
	r := v.Rows[i]
	b := r.Block
	label := ""
	if i == 0 || v.Rows[i-1].Block != b {
		label = blockLabel(b)
	}
	bar := " "
	switch b.State {
	case "add":
		bar = th.Added.Render("▌")
	case "del":
		bar = th.Removed.Render("▌")
	}
	return th.Muted.Render(fmt.Sprintf("%8s ", label)) + bar + th.Muted.Render("│") + " " + r.Text
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

func noteLine(tr *i18n.TranslationSet, labels KeyLabels, d types.ReviewData, v DiffView, r DiffRow, th theme.Theme) string {
	indent := strings.Repeat(" ", gutter-1)
	n, ok := NoteOf(d, r.Path, r.Key)
	if !ok {
		return ""
	}
	if r.Kind == "notehead" {
		head, keys := NoteHead(tr, labels, d, n, v.NoteOpen(n))
		style := map[string]lipgloss.Style{"queued": th.Accent, "conversation": th.Bold, "thread": th.Bold}[n.Kind]
		if n.Kind == "queued" {
			style = th.PullRequest(MemberColor(d.State, n.Pull))
		}
		return utils.Spread(indent+style.Render(head), th.Key.Render(keys)+" ", v.Width-1)
	}
	style := map[string]lipgloss.Style{"body": th.Text, "you": th.Bold, "claude": th.Text, "error": th.Bad, "stale": th.Warning, "author": th.Muted, "waiting": th.Muted, "step": th.Muted, "refused": th.Bad}[r.Part]
	return indent + "  " + style.Render(r.Text)
}

func fileHeader(tr *i18n.TranslationSet, d types.ReviewData, v DiffView, r DiffRow, filter types.PullFilter, th theme.Theme) string {
	fold := "▾"
	if v.Folded[r.Path] {
		fold = "▸"
	}
	entry := FileEntry(d, r.Path)
	head, tail := " "+th.Muted.Render(fold)+" ", "  "+th.Muted.Render(badges(tr)[entry.Status])
	for _, pull := range entry.Pulls {
		tail += "  " + th.PullRequest(MemberColor(d.State, pull)).Render(fmt.Sprintf("#%d", pull))
	}
	if v.Whole[r.Path] {
		tail += "  " + th.Accent.Render(tr.WholeBadge)
	}
	if v.Rendered[r.Path] {
		tail += "  " + th.Accent.Render(tr.RenderedBadge)
	}
	if entry.Status == "" {
		tail += "  " + th.Muted.Render(utils.Plural(entry.Lines, tr.LineNoun))
	} else {
		adds, dels := filter.Lines(entry)
		tail += "  " + th.Added.Render(fmt.Sprintf("+%d", adds)) + " " + th.Removed.Render(fmt.Sprintf("−%d", dels))
	}
	path := r.Path
	if room := v.Width - 1 - ansi.StringWidth(head+tail); room > 0 {
		path = utils.FitTitle(path, room)
	}
	return head + th.Bold.Render(path) + tail
}

func unifiedRow(d types.ReviewData, v DiffView, r DiffRow, mark string, filter types.PullFilter, th theme.Theme) []string {
	l := r.Line
	sign := map[string]string{"add": "+", "del": "-", "same": " ", "note": " "}[l.Kind]
	if r.Added {
		sign = "+"
	}
	head := mark + stripe(d, l, filter, th) + th.Muted.Render(fmt.Sprintf("%4s %4s ", number(l.Old), number(l.New)))
	style := rowStyle(l, r.Added, filter, th)
	body := v.Width - gutter
	texts := cut(ExpandTabs(l.Text), body, v.Wrap)
	out := []string{head + style.Render(sign+" "+texts[0])}
	for _, more := range texts[1:] {
		out = append(out, strings.Repeat(" ", gutter)+style.Render(more))
	}
	return out
}

func splitRow(d types.ReviewData, v DiffView, r DiffRow, mark string, filter types.PullFilter, th theme.Theme) []string {
	half := (v.Width - 3) / 2
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
		style := rowStyle(l, false, filter, th)
		texts := cut(ExpandTabs(l.Text), half-7, v.Wrap)
		out := []string{th.Muted.Render(fmt.Sprintf("%4s ", number(n))) + style.Render(sign+" "+texts[0])}
		for _, more := range texts[1:] {
			out = append(out, "       "+style.Render(more))
		}
		for i := range out {
			out[i] = utils.Fit(out[i], half)
		}
		return out
	}
	left, right := side(r.Left, true), side(r.Right, false)
	owner := r.Right
	if owner == nil || owner.Kind == "same" {
		owner = r.Left
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
			lead = mark + stripe(d, owner, filter, th)
		}
		out = append(out, lead+a+th.Muted.Render("│")+b)
	}
	return out
}

func stripe(d types.ReviewData, l *server.Row, filter types.PullFilter, th theme.Theme) string {
	if l == nil || l.Owner == 0 || l.Kind == "same" {
		return " "
	}
	bar := "▎"
	if !filter.Shows(l.Owner) {
		bar = "·"
	}
	return th.PullRequest(MemberColor(d.State, l.Owner)).Render(bar)
}

func rowStyle(l *server.Row, added bool, filter types.PullFilter, th theme.Theme) lipgloss.Style {
	if l.Kind != "same" && !filter.Shows(l.Owner) {
		return th.Muted
	}
	switch {
	case l.Kind == "add" || added:
		return th.Added
	case l.Kind == "del":
		return th.Removed
	case l.Kind == "note":
		return th.Muted
	}
	return th.Text
}

// RowHeight is how many lines a row of the diff takes: more than one only
// for a line wrapped to the view's width.
func RowHeight(r DiffRow, v DiffView) int {
	if !v.Wrap || r.Kind != "line" {
		return 1
	}
	if r.Line != nil {
		return len(cut(ExpandTabs(r.Line.Text), v.Width-gutter, true))
	}
	half := (v.Width - 3) / 2
	h := 1
	for _, l := range []*server.Row{r.Left, r.Right} {
		if l != nil {
			h = max(h, len(cut(ExpandTabs(l.Text), half-7, true)))
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
