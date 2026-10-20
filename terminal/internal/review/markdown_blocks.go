package review

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"

	"github.com/connorb645/lazyreview/terminal/internal/models"
)

var (
	markdown       = goldmark.New(goldmark.WithExtensions(extension.GFM))
	closingFence   = regexp.MustCompile("^ {0,3}(`{3,}|~{3,})\\s*$")
	setextLine     = regexp.MustCompile(`^ {0,3}(=+|-+)\s*$`)
	orderedNumber  = regexp.MustCompile(`^(\s*)\d+([.)])`)
	trailingSpaces = regexp.MustCompile(`\s+$`)
)

// MarkdownBlocks is the Rendered view of a Markdown file in a diff, as blocks
// the app draws one at a time, each with the lines it was read from: an
// added file's blocks are added, a deleted file's removed, and a changed
// file's compared block by block, before and after. With the whole text the
// blocks cover the whole file; without it, Partial, they cover the changed
// parts only, each block naming the hunk it comes from. A list is read item
// by item, so a list that changes is marked item by item; when a run of
// removed and added blocks holds exactly one list of a kind, ordered or not,
// on each side, it is one list edited, and its items are compared.
func MarkdownBlocks(file models.FileEntry, hunks []models.Hunk, whole []string) models.Rendered {
	side := func(h models.Hunk, keep string) []string {
		var lines []string
		for _, r := range h.Rows {
			if r.Kind == "same" || r.Kind == keep {
				lines = append(lines, r.Text)
			}
		}
		return lines
	}
	switch {
	case file.Status == "A":
		return models.Rendered{Blocks: drawn("add", markdownTokens(side(hunks[0], "add"), 1, "RIGHT"), nil)}
	case file.Status == "D":
		return models.Rendered{Blocks: drawn("del", markdownTokens(side(hunks[0], "del"), 1, "LEFT"), nil)}
	case whole != nil:
		var before []string
		next := 1
		for _, h := range hunks {
			first := h.NewStart
			if h.NewCount == 0 {
				first = h.NewStart + 1
			}
			before = append(before, slice(whole, next-1, first-1)...)
			before = append(before, side(h, "del")...)
			next = first + h.NewCount
		}
		before = append(before, slice(whole, next-1, len(whole))...)
		return models.Rendered{Blocks: changes(markdownTokens(before, 1, "LEFT"), markdownTokens(whole, 1, "RIGHT"))}
	}
	blocks := []models.Block{}
	for part, h := range hunks {
		for _, b := range changes(markdownTokens(side(h, "del"), max(h.OldStart, 1), "LEFT"), markdownTokens(side(h, "add"), max(h.NewStart, 1), "RIGHT")) {
			b.Part = part
			blocks = append(blocks, b)
		}
	}
	return models.Rendered{Blocks: blocks, Partial: true}
}

func markdownTokens(lines []string, first int, side string) []mdToken {
	source := []byte(strings.Join(lines, "\n"))
	doc := markdown.Parser().Parse(text.NewReader(source))
	starts := lineStarts(source)
	var out []mdToken
	children := topLevel(doc)
	end := -1
	for _, node := range children {
		from, to := lineRange(node, source, starts, lines, end)
		end = to
		t := mdToken{raw: rawLines(lines, from, to), span: mdSpan{side: side, startLine: first + from, line: first + to}}
		if list, ok := node.(*ast.List); ok {
			t.list, t.ordered, t.start = true, list.IsOrdered(), 1
			if list.IsOrdered() {
				t.start = list.Start
			}
			itemEnd := from - 1
			for item := list.FirstChild(); item != nil; item = item.NextSibling() {
				iFrom, iTo := lineRange(item, source, starts, lines, itemEnd)
				itemEnd = iTo
				t.items = append(t.items, mdToken{raw: rawLines(lines, iFrom, iTo), span: mdSpan{side: side, startLine: first + iFrom, line: first + iTo}})
			}
		}
		out = append(out, t)
	}
	return out
}

func topLevel(doc ast.Node) []ast.Node {
	var out []ast.Node
	for n := doc.FirstChild(); n != nil; n = n.NextSibling() {
		out = append(out, n)
	}
	return out
}

// lineRange is the first and last line, from 0, a block spans: from the
// first line its text is on, or a fenced block's opening fence, to the last
// line its text reaches, with a fenced block's closing fence or a heading's
// underline. A block with no text, such as a rule, is the first line after
// the block before it that holds anything.
func lineRange(node ast.Node, source []byte, starts []int, lines []string, before int) (int, int) {
	lo, hi, found := segments(node)
	from := before + 1
	for from < len(lines) && strings.TrimSpace(lines[from]) == "" {
		from++
	}
	to := from
	if found {
		from, to = lineOf(starts, lo), lineOf(starts, max(hi-1, lo))
	}
	switch n := node.(type) {
	case *ast.FencedCodeBlock:
		if found && n.Info == nil {
			from--
		}
		to = closingAfter(lines, from, to, n.Lines().Len() > 0)
	case *ast.Heading:
		if to+1 < len(lines) && setextLine.MatchString(lines[to+1]) && !strings.HasPrefix(strings.TrimSpace(lines[from]), "#") {
			to++
		}
	}
	return max(from, 0), min(max(to, from), len(lines)-1)
}

func closingAfter(lines []string, from, to int, hasContent bool) int {
	at := to + 1
	if !hasContent {
		at = from + 1
	}
	for i := at; i < len(lines); i++ {
		if closingFence.MatchString(lines[i]) {
			return i
		}
		if hasContent {
			break
		}
	}
	return to
}

func segments(node ast.Node) (int, int, bool) {
	lo, hi, found := 0, 0, false
	add := func(start, stop int) {
		if !found || start < lo {
			lo = start
		}
		if !found || stop > hi {
			hi = stop
		}
		found = true
	}
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if n.Type() == ast.TypeBlock {
			lines := n.Lines()
			for i := 0; i < lines.Len(); i++ {
				s := lines.At(i)
				add(s.Start, s.Stop)
			}
			if f, ok := n.(*ast.FencedCodeBlock); ok && f.Info != nil {
				add(f.Info.Segment.Start, f.Info.Segment.Stop)
			}
		}
		if t, ok := n.(*ast.Text); ok {
			add(t.Segment.Start, t.Segment.Stop)
		}
		return ast.WalkContinue, nil
	})
	return lo, hi, found
}

func lineStarts(source []byte) []int {
	starts := []int{0}
	for i, b := range source {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func lineOf(starts []int, offset int) int {
	return sort.Search(len(starts), func(i int) bool { return starts[i] > offset }) - 1
}

func rawLines(lines []string, from, to int) string {
	return trailingSpaces.ReplaceAllString(strings.Join(slice(lines, from, to+1), "\n"), "")
}

func slice(lines []string, from, to int) []string {
	from, to = max(from, 0), min(to, len(lines))
	if from >= to {
		return nil
	}
	return lines[from:to]
}

func block(state string, markdown string, span mdSpan, left *mdSpan) models.Block {
	b := models.Block{State: state, Markdown: markdown, Side: span.side, StartLine: span.startLine, Line: span.line}
	if left != nil {
		b.Left = &models.Span{StartLine: left.startLine, Line: left.line}
	}
	return b
}

func itemMarkdown(list mdToken, item mdToken, number int) string {
	if !list.ordered {
		return item.raw
	}
	return orderedNumber.ReplaceAllString(item.raw, "${1}"+strconv.Itoa(number)+"${2}")
}

func drawn(state string, tokens []mdToken, olds []mdToken) []models.Block {
	blocks := []models.Block{}
	for i, t := range tokens {
		var old *mdToken
		if i < len(olds) {
			old = &olds[i]
		}
		if !t.list {
			var left *mdSpan
			if old != nil {
				left = &old.span
			}
			blocks = append(blocks, block(state, t.raw, t.span, left))
			continue
		}
		for k, item := range t.items {
			var left *mdSpan
			if old != nil && k < len(old.items) {
				left = &old.items[k].span
			}
			blocks = append(blocks, block(state, itemMarkdown(t, item, t.start+k), item.span, left))
		}
	}
	return blocks
}

func listChange(oldList, newList mdToken) []models.Block {
	var blocks []models.Block
	for _, step := range lcsSteps(keys(oldList.items), keys(newList.items)) {
		switch step.kind {
		case "add":
			blocks = append(blocks, block("add", itemMarkdown(newList, newList.items[step.b], newList.start+step.b), newList.items[step.b].span, nil))
		case "del":
			blocks = append(blocks, block("del", itemMarkdown(oldList, oldList.items[step.a], oldList.start+step.a), oldList.items[step.a].span, nil))
		default:
			left := oldList.items[step.a].span
			blocks = append(blocks, block("same", itemMarkdown(newList, newList.items[step.b], newList.start+step.b), newList.items[step.b].span, &left))
		}
	}
	return blocks
}

func run(dels, adds []mdToken) []models.Block {
	edited := map[int]int{}
	paired := map[int]bool{}
	for _, ordered := range []bool{true, false} {
		var olds, news []int
		for i, t := range dels {
			if t.list && t.ordered == ordered {
				olds = append(olds, i)
			}
		}
		for i, t := range adds {
			if t.list && t.ordered == ordered {
				news = append(news, i)
			}
		}
		if len(olds) == 1 && len(news) == 1 {
			edited[news[0]] = olds[0]
			paired[olds[0]] = true
		}
	}
	var kept []mdToken
	for i, t := range dels {
		if !paired[i] {
			kept = append(kept, t)
		}
	}
	blocks := drawn("del", kept, nil)
	for i, t := range adds {
		if old, ok := edited[i]; ok {
			blocks = append(blocks, listChange(dels[old], t)...)
			continue
		}
		blocks = append(blocks, drawn("add", []mdToken{t}, nil)...)
	}
	return blocks
}

func changes(olds, news []mdToken) []models.Block {
	path := lcsSteps(keys(olds), keys(news))
	blocks := []models.Block{}
	for k := 0; k < len(path); {
		if path[k].kind == "same" {
			blocks = append(blocks, drawn("same", []mdToken{news[path[k].b]}, []mdToken{olds[path[k].a]})...)
			k++
			continue
		}
		var dels, adds []mdToken
		for ; k < len(path) && path[k].kind != "same"; k++ {
			if path[k].kind == "del" {
				dels = append(dels, olds[path[k].a])
			} else {
				adds = append(adds, news[path[k].b])
			}
		}
		blocks = append(blocks, run(dels, adds)...)
	}
	return blocks
}

func keys(tokens []mdToken) []string {
	out := make([]string, len(tokens))
	for i, t := range tokens {
		out[i] = t.raw
	}
	return out
}

// lcsSteps are the steps, by the longest common subsequence, that turn
// before into after: same, del or add, in reading order.
func lcsSteps(before, after []string) []lcsStep {
	n, m := len(before), len(after)
	width := m + 1
	table := make([]int, (n+1)*width)
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if before[i] == after[j] {
				table[i*width+j] = table[(i+1)*width+j+1] + 1
			} else {
				table[i*width+j] = max(table[(i+1)*width+j], table[i*width+j+1])
			}
		}
	}
	var out []lcsStep
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case before[i] == after[j]:
			out = append(out, lcsStep{kind: "same", a: i, b: j})
			i++
			j++
		case table[(i+1)*width+j] >= table[i*width+j+1]:
			out = append(out, lcsStep{kind: "del", a: i})
			i++
		default:
			out = append(out, lcsStep{kind: "add", b: j})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, lcsStep{kind: "del", a: i})
	}
	for ; j < m; j++ {
		out = append(out, lcsStep{kind: "add", b: j})
	}
	return out
}
