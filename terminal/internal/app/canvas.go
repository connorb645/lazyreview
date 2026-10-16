package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

const (
	up = 1 << iota
	down
	left
	right
)

var lineRunes = map[int]rune{
	up: '│', down: '│', up | down: '│', left: '─', right: '─', left | right: '─',
	down | right: '╭', down | left: '╮', up | right: '╰', up | left: '╯',
	up | down | right: '├', up | down | left: '┤', left | right | down: '┬', left | right | up: '┴', up | down | left | right: '┼',
}

// canvas is a grid of cells the diagram draws on: each cell's rune, the
// name of its style, and for a line the directions it runs in, so lines
// that meet join.
type canvas struct {
	w, h  int
	cells [][]cell
}

func newCanvas(w, h int) canvas {
	c := canvas{w: w, h: h, cells: make([][]cell, h)}
	for y := range c.cells {
		c.cells[y] = make([]cell, w)
		for x := range c.cells[y] {
			c.cells[y][x] = cell{r: ' '}
		}
	}
	return c
}

func (c canvas) in(x, y int) bool {
	return x >= 0 && y >= 0 && x < c.w && y < c.h
}

func (c canvas) put(x, y int, r rune, style string) {
	if c.in(x, y) {
		c.cells[y][x] = cell{r: r, style: style}
	}
}

// text writes s from x on line y, a wide character taking two cells.
func (c canvas) text(x, y int, s, style string) {
	for _, r := range s {
		w := runewidth.RuneWidth(r)
		c.put(x, y, r, style)
		if w == 2 {
			c.put(x+1, y, 0, style)
		}
		x += max(w, 1)
	}
}

// joint adds a line's directions to a cell, unless something not a line,
// other than an area's frame, is there.
func (c canvas) joint(x, y, dirs int, style string) {
	if !c.in(x, y) {
		return
	}
	at := &c.cells[y][x]
	if at.r != ' ' && at.dirs == 0 && at.style != "area" {
		return
	}
	at.dirs |= dirs
	at.r = lineRunes[at.dirs]
	at.style = style
}

// segment draws a straight line from one point to another, across or down.
func (c canvas) segment(x0, y0, x1, y1 int, style string) {
	if y0 == y1 {
		if x0 > x1 {
			x0, x1 = x1, x0
		}
		for x := x0; x <= x1; x++ {
			dirs := left | right
			if x == x0 {
				dirs = right
			}
			if x == x1 {
				dirs = left
			}
			if x0 == x1 {
				dirs = left | right
			}
			c.joint(x, y0, dirs, style)
		}
		return
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		dirs := up | down
		if y == y0 {
			dirs = down
		}
		if y == y1 {
			dirs = up
		}
		c.joint(x0, y, dirs, style)
	}
}

// lines draws the part of the canvas from column x0 and line y0, w by h,
// each run of one style styled once.
func (c canvas) lines(x0, y0, w, h int, styles map[string]lipgloss.Style) []string {
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
			if c.in(x, y) {
				ch, st = c.cells[y][x].r, c.cells[y][x].style
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
