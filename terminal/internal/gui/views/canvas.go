// Package views draws the framed rectangles the gui shows its rows in, and
// the canvas the diagram is drawn on. It knows nothing of what the rows say.
package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// The directions a line leaves a cell.
const (
	Up = 1 << iota
	Down
	Left
	Right
)

var lineRunes = map[int]rune{
	Up: '│', Down: '│', Up | Down: '│', Left: '─', Right: '─', Left | Right: '─',
	Down | Right: '╭', Down | Left: '╮', Up | Right: '╰', Up | Left: '╯',
	Up | Down | Right: '├', Up | Down | Left: '┤', Left | Right | Down: '┬', Left | Right | Up: '┴', Up | Down | Left | Right: '┼',
}

// Canvas is a grid of cells that lines and words are drawn on, each cell
// keeping the name of its style until Lines turns them into text.
type Canvas struct {
	W, H  int
	Cells [][]Cell
}

// NewCanvas is a canvas of spaces, w cells wide and h high.
func NewCanvas(w, h int) Canvas {
	c := Canvas{W: w, H: h, Cells: make([][]Cell, h)}
	for y := range c.Cells {
		c.Cells[y] = make([]Cell, w)
		for x := range c.Cells[y] {
			c.Cells[y][x] = Cell{R: ' '}
		}
	}
	return c
}

// In says whether a cell is on the canvas.
func (c Canvas) In(x, y int) bool {
	return x >= 0 && y >= 0 && x < c.W && y < c.H
}

// Put sets one cell, if it is on the canvas.
func (c Canvas) Put(x, y int, r rune, style string) {
	if c.In(x, y) {
		c.Cells[y][x] = Cell{R: r, Style: style}
	}
}

// Text writes words from a cell rightward, a wide rune taking two cells.
func (c Canvas) Text(x, y int, s, style string) {
	for _, r := range s {
		w := runewidth.RuneWidth(r)
		c.Put(x, y, r, style)
		if w == 2 {
			c.Put(x+1, y, 0, style)
		}
		x += max(w, 1)
	}
}

// Joint adds directions to a line's cell, which then draws the rune that
// joins them. It leaves alone a cell that holds words.
func (c Canvas) Joint(x, y, dirs int, style string) {
	if !c.In(x, y) {
		return
	}
	at := &c.Cells[y][x]
	if at.R != ' ' && at.Dirs == 0 && at.Style != "area" {
		return
	}
	at.Dirs |= dirs
	at.R = lineRunes[at.Dirs]
	at.Style = style
}

// Segment draws a straight line between two cells in a row or a column.
func (c Canvas) Segment(x0, y0, x1, y1 int, style string) {
	if y0 == y1 {
		if x0 > x1 {
			x0, x1 = x1, x0
		}
		for x := x0; x <= x1; x++ {
			dirs := Left | Right
			if x == x0 {
				dirs = Right
			}
			if x == x1 {
				dirs = Left
			}
			if x0 == x1 {
				dirs = Left | Right
			}
			c.Joint(x, y0, dirs, style)
		}
		return
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		dirs := Up | Down
		if y == y0 {
			dirs = Down
		}
		if y == y1 {
			dirs = Up
		}
		c.Joint(x0, y, dirs, style)
	}
}

// Lines is a part of the canvas as text, each run of cells in its style.
func (c Canvas) Lines(x0, y0, w, h int, styles map[string]lipgloss.Style) []string {
	out := make([]string, 0, h)
	for y := y0; y < y0+h; y++ {
		var b strings.Builder
		run, style := []rune{}, ""
		flush := func() {
			if len(run) == 0 {
				return
			}
			if s, ok := styles[style]; ok && style != "" {
				b.WriteString(s.Render(string(run)))
			} else {
				b.WriteString(string(run))
			}
			run = run[:0]
		}
		for x := x0; x < x0+w; x++ {
			ch, st := ' ', ""
			if c.In(x, y) {
				ch, st = c.Cells[y][x].R, c.Cells[y][x].Style
			}
			if ch == 0 {
				continue
			}
			if st != style {
				flush()
				style = st
			}
			run = append(run, ch)
		}
		flush()
		out = append(out, b.String())
	}
	return out
}
