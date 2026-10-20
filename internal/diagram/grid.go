package diagram

import (
	"sort"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/i18n"
)

const (
	// CardHeight is a card's lines: a border, two lines of text and a border.
	CardHeight = 4
	minWidth   = 14
	maxWidth   = 36
)

// Spot is where a card sits on the screen: its row, its first column and
// its width.
type Spot struct {
	Row   int
	Col   int
	Width int
}

// Grid is a level arranged in rows and columns: the cards of each row, left
// to right, and each card's spot. Rows read top to bottom from the layout's
// vertical bands, and each row's order left to right from it.
type Grid struct {
	Rows  [][]int
	Spots []Spot
	Width int
}

// Arrange sets a level's cards in rows, `gap` columns apart, each card as
// wide as its words.
func Arrange(tr *i18n.TranslationSet, cards []Card, gap int) Grid {
	order := make([]int, len(cards))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return cards[order[a]].Box.Y < cards[order[b]].Box.Y })
	var rows [][]int
	bottom := 0.0
	for _, i := range order {
		b := cards[i].Box
		if len(rows) == 0 || b.Y+b.H/2 > bottom {
			rows = append(rows, []int{i})
			bottom = b.Y + b.H
			continue
		}
		rows[len(rows)-1] = append(rows[len(rows)-1], i)
		bottom = max(bottom, b.Y+b.H)
	}
	g := Grid{Rows: rows, Spots: make([]Spot, len(cards))}
	for r, row := range rows {
		sort.SliceStable(row, func(a, b int) bool { return cards[row[a]].Box.X < cards[row[b]].Box.X })
		col := 2
		for _, i := range row {
			w := CardWidth(tr, cards[i])
			g.Spots[i] = Spot{Row: r, Col: col, Width: w}
			col += w + gap
		}
		g.Width = max(g.Width, col-gap+2)
	}
	return g
}

// CardWidth is a card's width: its words, a space either side and its border.
func CardWidth(tr *i18n.TranslationSet, c Card) int {
	w := max(ansi.StringWidth(c.Title), ansi.StringWidth(c.Detail)+1+len(StateWord(tr, c.State))) + 4
	return min(max(w, minWidth), maxWidth)
}

// Order is every card in reading order: row by row, left to right.
func (g Grid) Order() []int {
	var out []int
	for _, row := range g.Rows {
		out = append(out, row...)
	}
	return out
}

// Item is card i's place in its row, counted from 0 at the left.
func (g Grid) Item(i int) int {
	for item, at := range g.Rows[g.Spots[i].Row] {
		if at == i {
			return item
		}
	}
	return 0
}

// Along is the card `by` cards along card `from`'s row: to the right, or to
// the left when `by` is negative. It stops at the row's ends.
func (g Grid) Along(from, by int) int {
	row := g.Rows[g.Spots[from].Row]
	return row[max(0, min(g.Item(from)+by, len(row)-1))]
}

// Down is the card `by` rows below card `from`, or above when `by` is
// negative: the one at `item` in that row, or the row's last when the row
// is shorter. It stops at the top and bottom rows, and `from` stays when
// there is no row that way.
func (g Grid) Down(from, by, item int) int {
	at := g.Spots[from].Row
	to := max(0, min(at+by, len(g.Rows)-1))
	if to == at {
		return from
	}
	row := g.Rows[to]
	return row[min(item, len(row)-1)]
}
