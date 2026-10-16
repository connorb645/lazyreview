package diagram

import (
	"sort"

	"github.com/charmbracelet/x/ansi"
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
func Arrange(cards []Card, gap int) Grid {
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
			w := CardWidth(cards[i])
			g.Spots[i] = Spot{Row: r, Col: col, Width: w}
			col += w + gap
		}
		g.Width = max(g.Width, col-gap+2)
	}
	return g
}

// CardWidth is a card's width: its words, a space either side and its border.
func CardWidth(c Card) int {
	w := max(ansi.StringWidth(c.Title), ansi.StringWidth(c.Detail)+1+len(StateWord(c.State))) + 4
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

// Nearest is the card nearest card `from` in a direction, one of h, j, k
// and l, or -1 when there is none that way. Cards within a 90 degree cone
// come first, then any card on that side; each scores its distance along the
// direction plus twice its distance across. A row counts as rowLines lines,
// and a line as two columns, so the score measures what the eye sees.
func (g Grid) Nearest(from int, dir string, rowLines int) int {
	x0, y0 := g.centre(from, rowLines)
	best, bestScore, bestInCone := -1, 0.0, false
	for i := range g.Spots {
		if i == from {
			continue
		}
		x, y := g.centre(i, rowLines)
		along, across := 0.0, 0.0
		switch dir {
		case "l":
			along, across = x-x0, y-y0
		case "h":
			along, across = x0-x, y-y0
		case "j":
			along, across = y-y0, x-x0
		case "k":
			along, across = y0-y, x-x0
		}
		if along <= 0 {
			continue
		}
		if across < 0 {
			across = -across
		}
		inCone := across <= along
		score := along + 2*across
		if best < 0 || (inCone && !bestInCone) || (inCone == bestInCone && score < bestScore) {
			best, bestScore, bestInCone = i, score, inCone
		}
	}
	return best
}

func (g Grid) centre(i, rowLines int) (float64, float64) {
	s := g.Spots[i]
	return float64(s.Col) + float64(s.Width)/2, float64(s.Row*rowLines*2 + CardHeight)
}
