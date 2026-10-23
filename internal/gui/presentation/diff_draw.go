package presentation

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/theme"
	"github.com/connorb645/lazyreview/internal/utils"
)

const (
	gutter      = 14
	plainGutter = 4
	blockGutter = 12
	plainBlock  = 3
)

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
		return []string{mark + blockLine(v, i, filter, th)}
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
		return splitRow(d, v, i, mark, filter, th)
	}
	return unifiedRow(d, v, i, mark, filter, th)
}

func blockLine(v DiffView, i int, filter types.PullFilter, th theme.Theme) string {
	r := v.Rows[i]
	b := r.Block
	label := ""
	switch {
	case v.Numbers == RelativeNumbers && i != v.Cursor:
		label = distance(v, i)
	case v.Numbers == RelativeNumbers || i == 0 || v.Rows[i-1].Block != b:
		label = blockLabel(b)
	}
	out := len(b.Pulls) > 0 && !filter.ShowsAny(b.Pulls)
	glyph, text := "▌", r.Text
	if out {
		text = th.Faint.Render(ansi.Strip(text))
	}
	if out && th.Plain {
		glyph = "·"
	}
	bar := " "
	switch {
	case b.State == "add" && out:
		bar = th.DimAdded.Render(glyph)
	case b.State == "add":
		bar = th.Added.Render(glyph)
	case b.State == "del" && out:
		bar = th.DimRemoved.Render(glyph)
	case b.State == "del":
		bar = th.Removed.Render(glyph)
	}
	numbers := th.Muted.Render(fmt.Sprintf("%8s ", label))
	if v.Numbers == NoNumbers {
		numbers = ""
	}
	return numbers + bar + th.Muted.Render("│") + " " + text
}

func blockLabel(b *models.Block) string {
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
	indent := strings.Repeat(" ", gutterOf(v)-1)
	n, ok := NoteOf(d, r.Path, r.Key)
	if !ok {
		return ""
	}
	if r.Kind == "notehead" {
		heads, keys := NoteHead(tr, labels, d, n, v.NoteOpen(n))
		style := map[string]lipgloss.Style{"queued": th.Accent, "conversation": th.Bold, "thread": th.Bold}[n.Kind]
		if n.Kind == "queued" {
			style = th.PullRequest(MemberColor(d.State, n.Pull))
		}
		width := v.Width - 1
		if full, keyed := indent+style.Render(heads[0]), th.Key.Render(keys)+" "; ansi.StringWidth(full)+1+ansi.StringWidth(keyed) <= width {
			return utils.Spread(full, keyed, width)
		}
		for _, head := range heads {
			if ansi.StringWidth(indent+head) <= width {
				return indent + style.Render(head)
			}
		}
		return utils.Fit(indent+style.Render(heads[len(heads)-1]), width)
	}
	return indent + "  " + partStyle(th, r.Part).Render(r.Text)
}

func fileHeader(tr *i18n.TranslationSet, d types.ReviewData, v DiffView, r DiffRow, filter types.PullFilter, th theme.Theme) string {
	fold := "▾"
	if v.Folded[r.Path] {
		fold = "▸"
	}
	entry := FileEntry(d, r.Path)
	head, status := " "+th.Muted.Render(fold)+" ", "  "+th.Muted.Render(badges(tr)[entry.Status])
	short := shortBranchNames(tr, d, v.Width)
	tail := ""
	for _, pull := range entry.Pulls {
		name := MemberName(d, pull)
		if short {
			name = utils.FitTitle(name, ansi.StringWidth(lastPart(name))+2)
		}
		tail += "  " + th.PullRequest(MemberColor(d.State, pull)).Render(name)
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
	tail = status + tail
	path := r.Path
	if room := v.Width - 1 - ansi.StringWidth(head+tail); room > 0 {
		path = utils.FitTitle(path, room)
	}
	return head + th.Bold.Render(path) + tail
}

func shortBranchNames(tr *i18n.TranslationSet, d types.ReviewData, width int) bool {
	if d.State.Kind != "branch" || d.Page.Stack == nil {
		return false
	}
	for _, f := range d.Page.Model.Files {
		need := 3 + ansi.StringWidth(lastPart(f.Path)) + 2 + ansi.StringWidth(badges(tr)[f.Status]) + 2 + ansi.StringWidth(fmt.Sprintf("+%d −%d", f.Adds, f.Dels))
		for _, pull := range f.Pulls {
			need += 2 + ansi.StringWidth(MemberName(d, pull))
		}
		if need > width-1 {
			return true
		}
	}
	return false
}

func lastPart(name string) string {
	return name[strings.LastIndex(name, "/")+1:]
}

func unifiedRow(d types.ReviewData, v DiffView, i int, mark string, filter types.PullFilter, th theme.Theme) []string {
	r := v.Rows[i]
	l := r.Line
	sign := map[string]string{"add": "+", "del": "-", "same": " ", "note": " "}[l.Kind]
	if r.Added {
		sign = "+"
	}
	numbers := fmt.Sprintf("%4s %4s ", number(l.Old), number(l.New))
	switch {
	case v.Numbers == NoNumbers:
		numbers = ""
	case v.Numbers == RelativeNumbers && i != v.Cursor:
		numbers = fmt.Sprintf("%9s ", distance(v, i))
	}
	head := mark + stripe(d, l, filter, th) + th.Muted.Render(numbers)
	style := rowStyle(l, r.Added, filter, th)
	texts := codeCut(ExpandTabs(l.Text), v, v.Width-gutterOf(v))
	out := []string{head + style.Render(sign+" "+texts[0])}
	for _, more := range texts[1:] {
		out = append(out, strings.Repeat(" ", gutterOf(v))+style.Render(more))
	}
	return out
}

func splitRow(d types.ReviewData, v DiffView, i int, mark string, filter types.PullFilter, th theme.Theme) []string {
	r := v.Rows[i]
	half := (v.Width - 3) / 2
	side := func(l *models.Row, old bool) []string {
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
		numbers := fmt.Sprintf("%4s ", number(n))
		switch {
		case v.Numbers == NoNumbers:
			numbers = ""
		case v.Numbers == RelativeNumbers && i != v.Cursor:
			numbers = fmt.Sprintf("%4s ", distance(v, i))
		}
		style := rowStyle(l, false, filter, th)
		texts := codeCut(ExpandTabs(l.Text), v, half-sideGutter(v))
		out := []string{th.Muted.Render(numbers) + style.Render(sign+" "+texts[0])}
		for _, more := range texts[1:] {
			out = append(out, strings.Repeat(" ", sideGutter(v))+style.Render(more))
		}
		for i := range out {
			out[i] = utils.Fit(out[i], half)
		}
		return out
	}
	left, right := side(r.Left, true), side(r.Right, false)
	var out []string
	for i := 0; i < max(len(left), len(right)); i++ {
		a, b := strings.Repeat(" ", half), strings.Repeat(" ", half)
		if i < len(left) {
			a = left[i]
		}
		if i < len(right) {
			b = right[i]
		}
		lead, between := "  ", th.Muted.Render("│")
		if i == 0 {
			lead = mark + stripe(d, r.Left, filter, th)
			if bar := stripe(d, r.Right, filter, th); bar != " " {
				between = bar
			}
		}
		out = append(out, lead+a+between+b)
	}
	return out
}

func stripe(d types.ReviewData, l *models.Row, filter types.PullFilter, th theme.Theme) string {
	if l == nil || l.Owner == 0 || l.Kind == "same" {
		return " "
	}
	bar := "▎"
	if !filter.Shows(l.Owner) {
		bar = "·"
	}
	return th.PullRequest(MemberColor(d.State, l.Owner)).Render(bar)
}

func rowStyle(l *models.Row, added bool, filter types.PullFilter, th theme.Theme) lipgloss.Style {
	out := l.Kind != "same" && !filter.Shows(l.Owner)
	switch {
	case (l.Kind == "add" || added) && out:
		return th.DimAdded
	case l.Kind == "add" || added:
		return th.Added
	case l.Kind == "del" && out:
		return th.DimRemoved
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
		return len(cut(ExpandTabs(r.Line.Text), v.Width-gutterOf(v), true))
	}
	half := (v.Width - 3) / 2
	h := 1
	for _, l := range []*models.Row{r.Left, r.Right} {
		if l != nil {
			h = max(h, len(cut(ExpandTabs(l.Text), half-sideGutter(v), true)))
		}
	}
	return h
}

func gutterOf(v DiffView) int {
	if v.Numbers == NoNumbers {
		return plainGutter
	}
	return gutter
}

func sideGutter(v DiffView) int {
	if v.Numbers == NoNumbers {
		return 2
	}
	return 7
}

func blockGutterOf(v DiffView) int {
	if v.Numbers == NoNumbers {
		return plainBlock
	}
	return blockGutter
}

func distance(v DiffView, i int) string {
	return fmt.Sprint(max(i-v.Cursor, v.Cursor-i))
}

// ScrollStep is how far h and l scroll a diff's code: half the width its
// code has, and no less than four columns.
func ScrollStep(v DiffView) int {
	return max(4, codeWidth(v)/2)
}

// ScrollEnd is how far a diff's code scrolls to the left: until the widest
// line of code shown in a height, on either side of a split row, ends at
// the right edge.
func ScrollEnd(v DiffView, height int) int {
	widest, used := 0, 0
	for i := v.Top; i < len(v.Rows) && used < height; i++ {
		r := v.Rows[i]
		used += RowHeight(r, v)
		if r.Kind != "line" {
			continue
		}
		for _, l := range []*models.Row{r.Line, r.Left, r.Right} {
			if l != nil {
				widest = max(widest, ansi.StringWidth(ExpandTabs(l.Text)))
			}
		}
	}
	return max(0, widest-codeWidth(v))
}

func codeWidth(v DiffView) int {
	if v.Split {
		return max(4, (v.Width-3)/2-sideGutter(v))
	}
	return max(4, v.Width-gutterOf(v))
}

func codeCut(text string, v DiffView, width int) []string {
	if v.Wrap || v.Left == 0 {
		return cut(text, width, v.Wrap)
	}
	return []string{shifted(text, v.Left, max(4, width))}
}

func shifted(text string, left, width int) string {
	if ansi.StringWidth(text) == 0 {
		return ""
	}
	var rest strings.Builder
	at, hidden := 0, left+1
	for _, r := range text {
		w := ansi.StringWidth(string(r))
		switch {
		case at >= hidden:
			rest.WriteRune(r)
		case at+w > hidden:
			rest.WriteString(strings.Repeat(" ", at+w-hidden))
		}
		at += w
	}
	if ansi.StringWidth(rest.String()) < width {
		return "…" + rest.String()
	}
	return "…" + ansi.Truncate(rest.String(), width-1, "…")
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
