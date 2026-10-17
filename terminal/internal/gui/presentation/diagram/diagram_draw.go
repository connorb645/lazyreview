// Package diagram draws a level of the diagram on a views.Canvas: its cards, the
// links between them routed round the cards, and the links that leave it.
package diagram

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	model "github.com/connorb645/lazyreview/terminal/internal/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const noLane = -1

// Draw draws a scene of the diagram on a canvas as large as it needs, the
// cards the filter leaves out dimmed, and gives the canvas, the top line of
// each row of cards, and the lines of links leaving the level that hang
// below the focused card.
func Draw(tr *i18n.TranslationSet, d types.ReviewData, v Scene, filter types.PullFilter) (views.Canvas, []int, Stubs) {
	lvl, g := v.Level, v.Grid
	stubLines := stubWords(tr, lvl, v)
	routes, lanes := routesOf(lvl, g, v.Focus, v.All, len(stubLines) > 0)
	focusRow := -1
	if len(stubLines) > 0 {
		focusRow = g.Spots[v.Focus].Row
	}
	below := Stubs{Lines: len(stubLines)}
	if focusRow >= 0 {
		s := g.Spots[v.Focus]
		for _, line := range stubLines {
			below.Width = max(below.Width, s.Width-4+ansi.StringWidth(line))
		}
	}
	tops := make([]int, len(g.Rows))
	y := 1
	for r := range g.Rows {
		tops[r] = y
		gap := 2 + lanes[r]
		if r == focusRow {
			gap += below.Lines
		}
		y += model.CardHeight + gap
	}
	width := g.Width + 2
	if focusRow >= 0 {
		width = max(width, g.Spots[v.Focus].Col+below.Width+2)
	}
	c := views.NewCanvas(width, y-1)
	laneY := func(gap, lane int) int {
		ly := tops[gap] + model.CardHeight + 1 + lane
		if gap == focusRow {
			ly += below.Lines
		}
		return ly
	}
	if lvl.Kind == "components" {
		drawAreas(c, d, lvl, g, tops)
	}
	for _, pass := range []func(route) bool{isFaint, isPlain, isLit} {
		for _, r := range routes {
			if pass(r) {
				drawRoute(c, g, tops, r, laneY)
			}
		}
	}
	for _, pass := range []func(route) bool{isFaint, isPlain, isLit} {
		for _, r := range routes {
			if pass(r) {
				drawWords(c, g, tops, r, laneY)
			}
		}
	}
	labels := map[int]string{}
	for l, at := range v.Labels {
		if strings.HasPrefix(l, v.Typed) {
			labels[at] = l
		}
	}
	for i, card := range lvl.Cards {
		drawCard(c, d, lvl, card, model.StateWord(tr, card.State), g.Spots[i], tops[g.Spots[i].Row], i == v.Focus, presentation.Dimmed(card, filter), labels[i])
	}
	for _, pass := range []func(route) bool{isFaint, isPlain, isLit} {
		for _, r := range routes {
			if pass(r) {
				drawEnds(c, g, tops, r, laneY)
			}
		}
	}
	if focusRow >= 0 {
		s := g.Spots[v.Focus]
		x, bottom := s.Col+s.Width-4, tops[focusRow]+model.CardHeight-1
		c.Put(x, bottom, joined(c.Cells[bottom][x].R, views.Down), "focus")
		for i, line := range stubLines {
			tree := "├"
			if i == len(stubLines)-1 {
				tree = "└"
			}
			c.Text(x, bottom+1+i, tree+line, "lit")
		}
	}
	return c, tops, below
}

func stubWords(tr *i18n.TranslationSet, lvl model.Level, v Scene) []string {
	card, ok := v.Card()
	if !ok || lvl.Kind != "types" {
		return nil
	}
	var out, in []string
	for _, l := range lvl.Links {
		switch {
		case l.Outside == "":
		case l.From == card.ID:
			out = append(out, "─▶ "+l.Words+" "+l.Outside)
		case l.To == card.ID:
			place := utils.ResolvePlaceholderString(tr.InPlace, map[string]string{"place": l.OutsidePlace})
			in = append(in, "◀─ "+l.OutsideName+" "+l.Words+" · "+place)
		}
	}
	return append(out, in...)
}

func isFaint(r route) bool { return r.faint }
func isPlain(r route) bool { return !r.faint && !r.lit }
func isLit(r route) bool   { return r.lit }

func routesOf(lvl model.Level, g model.Grid, focus int, all, stubbed bool) ([]route, []int) {
	var routes []route
	at := map[[2]int]int{}
	for _, l := range lvl.Links {
		from, to := lvl.Index(l.From), lvl.Index(l.To)
		if l.Outside != "" || from < 0 || to < 0 || from == to {
			continue
		}
		lit := from == focus || to == focus
		if !lit && !l.AtRest && !all {
			continue
		}
		words := ""
		if lit || lvl.Kind != "types" || l.Kind == "use" && l.AtRest {
			words = l.Words
		}
		if i, ok := at[[2]int{from, to}]; ok {
			r := &routes[i]
			r.lit = r.lit || lit
			r.faint = r.faint && !lit && !l.AtRest
			if words != "" && r.words != "" {
				r.words += " · " + words
			} else if words != "" {
				r.words = words
			}
			continue
		}
		at[[2]int{from, to}] = len(routes)
		routes = append(routes, route{from: from, to: to, words: words, lit: lit, faint: !lit && !l.AtRest})
	}
	centre := func(i int) int { return g.Spots[i].Col + g.Spots[i].Width/2 }
	for i := range routes {
		r := &routes[i]
		if j, back := at[[2]int{r.to, r.from}]; back && j < i {
			r.across = 1
		}
		sf, st := g.Spots[r.from], g.Spots[r.to]
		mid := (centre(r.from) + centre(r.to)) / 2
		switch {
		case sf.Row == st.Row && sideBySide(g, r.from, r.to):
			r.side = true
		case st.Row == sf.Row || st.Row == sf.Row+1:
			r.gaps = []int{sf.Row}
		case st.Row == sf.Row-1:
			r.gaps = []int{st.Row}
		case st.Row > sf.Row:
			r.gaps, r.via = []int{sf.Row, st.Row - 1}, freeColumn(g, sf.Row+1, st.Row-1, mid)
		default:
			r.gaps, r.via = []int{sf.Row - 1, st.Row}, freeColumn(g, st.Row+1, sf.Row-1, mid)
		}
	}
	placePorts(routes, g, focus, stubbed)
	return routes, laneRoutes(routes, g)
}

func placePorts(routes []route, g model.Grid, focus int, stubbed bool) {
	ends := map[[2]int][]port{}
	centre := func(i int) int { return g.Spots[i].Col + g.Spots[i].Width/2 }
	for i, r := range routes {
		if r.side {
			continue
		}
		sf, st := g.Spots[r.from], g.Spots[r.to]
		leaves, arrives := views.Down, views.Up
		switch {
		case st.Row == sf.Row:
			arrives = views.Down
		case st.Row < sf.Row:
			leaves, arrives = views.Up, views.Down
		}
		towardTo, towardFrom := centre(r.to), centre(r.from)
		if len(r.gaps) == 2 {
			towardTo, towardFrom = r.via, r.via
		}
		ends[[2]int{r.from, leaves}] = append(ends[[2]int{r.from, leaves}], port{route: i, source: true, toward: towardTo})
		ends[[2]int{r.to, arrives}] = append(ends[[2]int{r.to, arrives}], port{route: i, toward: towardFrom})
	}
	for key, list := range ends {
		sort.SliceStable(list, func(a, b int) bool { return list[a].toward < list[b].toward })
		s := g.Spots[key[0]]
		lo, hi := s.Col+2, s.Col+s.Width-3
		if stubbed && key[0] == focus && key[1] == views.Down {
			hi = s.Col + s.Width - 6
		}
		for k, p := range list {
			x := lo + (k+1)*(hi-lo)/(len(list)+1)
			if len(list) == 1 {
				x = min(max(s.Col+s.Width/2, lo), hi)
			}
			if p.source {
				routes[p.route].xa = x
			} else {
				routes[p.route].xb = x
			}
		}
	}
	for i := range routes {
		r := &routes[i]
		sf, st := g.Spots[r.from], g.Spots[r.to]
		if r.side || len(r.gaps) != 1 || sf.Row == st.Row {
			continue
		}
		leaves, arrives := views.Down, views.Up
		if st.Row < sf.Row {
			leaves, arrives = views.Up, views.Down
		}
		if len(ends[[2]int{r.from, leaves}]) != 1 || len(ends[[2]int{r.to, arrives}]) != 1 {
			continue
		}
		lo, hi := max(sf.Col, st.Col)+2, min(sf.Col+sf.Width, st.Col+st.Width)-3
		if stubbed && (r.from == focus && leaves == views.Down || r.to == focus && arrives == views.Down) {
			hi = min(hi, g.Spots[focus].Col+g.Spots[focus].Width-6)
		}
		switch {
		case lo > hi:
		case r.xa >= lo && r.xa <= hi:
			r.xb = r.xa
		case r.xb >= lo && r.xb <= hi:
			r.xa = r.xb
		default:
			r.xa, r.xb = (lo+hi)/2, (lo+hi)/2
		}
	}
}

func laneRoutes(routes []route, g model.Grid) []int {
	byGap := map[int][]stretch{}
	for i := range routes {
		r := &routes[i]
		if r.side {
			continue
		}
		r.lanes = make([]int, len(r.gaps))
		sf, st := g.Spots[r.from], g.Spots[r.to]
		xs := []int{r.xa, r.xb}
		if len(r.gaps) == 2 {
			xs = []int{r.xa, r.via, r.via, r.xb}
		}
		for k, gap := range r.gaps {
			upper, lower := xs[2*k], xs[2*k+1]
			if st.Row < sf.Row {
				upper, lower = lower, upper
			}
			if len(r.gaps) == 1 && st.Row < sf.Row {
				upper, lower = r.xb, r.xa
			}
			byGap[gap] = append(byGap[gap], stretch{route: i, at: k, upper: upper, lower: lower, turn: sf.Row == st.Row})
		}
	}
	lanes := make([]int, len(g.Rows)+1)
	for gap, list := range byGap {
		rank := func(s stretch) (int, int) {
			switch {
			case s.turn:
				return 0, abs(s.lower - s.upper)
			case s.lower > s.upper:
				return 1, -s.upper
			case s.lower < s.upper:
				return 2, s.upper
			}
			return 3, 0
		}
		sort.SliceStable(list, func(a, b int) bool {
			ga, ka := rank(list[a])
			gb, kb := rank(list[b])
			return ga < gb || ga == gb && ka < kb
		})
		for _, s := range list {
			r := &routes[s.route]
			if s.upper == s.lower && !s.turn && (r.words == "" || s.at != len(r.gaps)-1) {
				r.lanes[s.at] = noLane
				continue
			}
			r.lanes[s.at] = lanes[gap]
			lanes[gap]++
		}
	}
	return lanes
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func freeColumn(g model.Grid, first, last, x int) int {
	taken := func(col int) bool {
		for r := first; r <= last; r++ {
			for _, i := range g.Rows[r] {
				if s := g.Spots[i]; col >= s.Col-1 && col <= s.Col+s.Width {
					return true
				}
			}
		}
		return false
	}
	for d := 0; ; d++ {
		if !taken(x + d) {
			return x + d
		}
		if x-d > 0 && !taken(x-d) {
			return x - d
		}
	}
}

func sideBySide(g model.Grid, a, b int) bool {
	row := g.Rows[g.Spots[a].Row]
	for i, at := range row {
		if at == a {
			return (i > 0 && row[i-1] == b) || (i+1 < len(row) && row[i+1] == b)
		}
	}
	return false
}

func routeStyle(r route) string {
	switch {
	case r.lit:
		return "lit"
	case r.faint:
		return "faint"
	}
	return "line"
}

func (r route) path(g model.Grid, tops []int, laneY func(gap, lane int) int) ([][2]int, int) {
	sf, st := g.Spots[r.from], g.Spots[r.to]
	top := func(s model.Spot) int { return tops[s.Row] }
	bottom := func(s model.Spot) int { return tops[s.Row] + model.CardHeight - 1 }
	ya, yb, arrow := bottom(sf), top(st), views.Down
	switch {
	case st.Row == sf.Row:
		yb, arrow = bottom(st), views.Up
	case st.Row < sf.Row:
		ya, yb, arrow = top(sf), bottom(st), views.Up
	}
	points := [][2]int{{r.xa, ya}}
	xs := []int{r.xa, r.xb}
	if len(r.gaps) == 2 {
		xs = []int{r.xa, r.via, r.via, r.xb}
	}
	for k, gap := range r.gaps {
		if r.lanes[k] == noLane {
			continue
		}
		ly := laneY(gap, r.lanes[k])
		points = append(points, [2]int{xs[2*k], ly})
		if xs[2*k+1] != xs[2*k] {
			points = append(points, [2]int{xs[2*k+1], ly})
		}
	}
	return append(points, [2]int{r.xb, yb}), arrow
}

func drawRoute(c views.Canvas, g model.Grid, tops []int, r route, laneY func(gap, lane int) int) {
	style := routeStyle(r)
	if r.side {
		x0, x1, y := sideEnds(g, tops, r)
		c.Segment(x0, y, x1, y, style)
		return
	}
	points, _ := r.path(g, tops, laneY)
	for i := 1; i < len(points); i++ {
		c.Segment(points[i-1][0], points[i-1][1], points[i][0], points[i][1], style)
	}
}

func sideEnds(g model.Grid, tops []int, r route) (int, int, int) {
	sf, st := g.Spots[r.from], g.Spots[r.to]
	y := tops[sf.Row] + 1 + r.across
	if st.Col > sf.Col {
		return sf.Col + sf.Width - 1, st.Col - 1, y
	}
	return sf.Col, st.Col + st.Width, y
}

func drawWords(c views.Canvas, g model.Grid, tops []int, r route, laneY func(gap, lane int) int) {
	if r.words == "" {
		return
	}
	style := routeStyle(r)
	if r.side {
		x0, x1, y := sideEnds(g, tops, r)
		line := y - 1 - r.across
		if r.across > 0 {
			line = y + 1
		}
		if x, words, room := wordsIn(c, line, min(x0, x1)+1, max(x0, x1)-1, r.words, 0); room > 0 {
			c.Text(x, line, words, style)
		}
		return
	}
	points, _ := r.path(g, tops, laneY)
	if len(points) < 3 {
		return
	}
	last, before := points[len(points)-2], points[len(points)-3]
	y := last[1]
	rightX, rightWords, rightRoom := wordsIn(c, y, last[0]+2, c.W-1, r.words, -1)
	leftX, leftWords, leftRoom := wordsIn(c, y, 0, last[0]-2, r.words, 1)
	if before[1] == y && before[0] > last[0] {
		rightX, rightWords, rightRoom = wordsIn(c, y, before[0]+2, c.W-1, r.words, -1)
	} else if before[1] == y {
		leftRoom = 0
	}
	whole := ansi.StringWidth(r.words)
	leftFirst := before[1] == y && before[0] > last[0]
	switch {
	case leftFirst && leftRoom >= whole, !leftFirst && rightRoom < whole && leftRoom >= whole:
		c.Text(leftX, y, leftWords, style)
	case rightRoom >= whole, rightRoom >= leftRoom && rightRoom > 0:
		c.Text(rightX, y, rightWords, style)
	case leftRoom > 0:
		c.Text(leftX, y, leftWords, style)
	}
}

func wordsIn(c views.Canvas, y, lo, hi int, words string, lean int) (int, string, int) {
	lo, hi = max(lo, 0), min(hi, c.W-1)
	free := func(x int) bool { return c.Cells[y][x].R == ' ' }
	start, end := -1, -2
	switch lean {
	case -1:
		for start = lo; start <= hi && !free(start); start++ {
		}
		for end = start; end+1 <= hi && free(end+1); end++ {
		}
	case 1:
		for end = hi; end >= lo && !free(end); end-- {
		}
		for start = end; start-1 >= lo && free(start-1); start-- {
		}
	default:
		for x := lo; x <= hi; x++ {
			if !free(x) {
				continue
			}
			from := x
			for x+1 <= hi && free(x+1) {
				x++
			}
			if x-from > end-start {
				start, end = from, x
			}
		}
		start, end = start+1, end-1
	}
	room := end - start + 1
	if start < lo || end > hi || room < min(4, ansi.StringWidth(words)) {
		return 0, "", 0
	}
	words = ansi.Truncate(words, room, "…")
	switch lean {
	case 1:
		return end + 1 - ansi.StringWidth(words), words, room
	case 0:
		return start + (room-ansi.StringWidth(words))/2, words, room
	}
	return start, words, room
}

func drawEnds(c views.Canvas, g model.Grid, tops []int, r route, laneY func(gap, lane int) int) {
	style := routeStyle(r)
	if r.side {
		x0, x1, y := sideEnds(g, tops, r)
		if x1 > x0 {
			c.Put(x0, y, joined(c.Cells[y][x0].R, views.Right), c.Cells[y][x0].Style)
			c.Put(x1, y, '▶', style)
		} else {
			c.Put(x0, y, joined(c.Cells[y][x0].R, views.Left), c.Cells[y][x0].Style)
			c.Put(x1, y, '◀', style)
		}
		return
	}
	points, arrow := r.path(g, tops, laneY)
	start, end := points[0], points[len(points)-1]
	leaves := views.Down
	if points[1][1] < start[1] {
		leaves = views.Up
	}
	if c.In(start[0], start[1]) {
		c.Put(start[0], start[1], joined(c.Cells[start[1]][start[0]].R, leaves), c.Cells[start[1]][start[0]].Style)
	}
	head := '▼'
	if arrow == views.Up {
		head = '▲'
	}
	c.Put(end[0], end[1], head, style)
}

func joined(border rune, dir int) rune {
	joints := map[int]map[rune]rune{
		views.Down:  {'─': '┬', '━': '┯', '┄': '┬'},
		views.Up:    {'─': '┴', '━': '┷', '┄': '┴'},
		views.Right: {'│': '├', '┃': '┠', '┆': '├'},
		views.Left:  {'│': '┤', '┃': '┨', '┆': '┤'},
	}
	if r, ok := joints[dir][border]; ok {
		return r
	}
	return border
}

func drawAreas(c views.Canvas, d types.ReviewData, lvl model.Level, g model.Grid, tops []int) {
	for _, area := range d.Page.Model.Areas {
		x0, y0, x1, y1, found := 0, 0, 0, 0, false
		for i, card := range lvl.Cards {
			if card.Area != area.ID {
				continue
			}
			s := g.Spots[i]
			cx0, cy0, cx1, cy1 := s.Col-2, tops[s.Row]-1, s.Col+s.Width+1, tops[s.Row]+model.CardHeight
			if !found {
				x0, y0, x1, y1, found = cx0, cy0, cx1, cy1, true
				continue
			}
			x0, y0, x1, y1 = min(x0, cx0), min(y0, cy0), max(x1, cx1), max(y1, cy1)
		}
		if !found {
			continue
		}
		box(c, x0, max(0, y0), x1, y1, "area", '╭', '╮', '╰', '╯', '─', '│')
		c.Text(x0+2, max(0, y0), " "+area.Title+" ", "area")
	}
}

func drawCard(c views.Canvas, d types.ReviewData, lvl model.Level, card model.Card, state string, s model.Spot, y int, focused, dim bool, label string) {
	x, w := s.Col, s.Width
	border := "border"
	if lvl.Kind == "components" {
		border = "comp:" + presentation.ComponentColour(d, card.ID)
	}
	if dim {
		border = "dim"
	}
	h, v, tl, tr, bl, br := '─', '│', '╭', '╮', '╰', '╯'
	switch {
	case focused:
		border, h, v, tl, tr, bl, br = "focus", '━', '┃', '┏', '┓', '┗', '┛'
	case card.Context:
		h, v = '┄', '┆'
	}
	for yy := y; yy < y+model.CardHeight; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.Put(xx, yy, ' ', "")
		}
	}
	box(c, x, y, x+w-1, y+model.CardHeight-1, border, tl, tr, bl, br, h, v)
	if len(card.Pulls) > 0 && d.Page.Stack != nil && !focused && !dim {
		inner := w - 2
		for i, pull := range card.Pulls {
			from, to := x+1+inner*i/len(card.Pulls), x+1+inner*(i+1)/len(card.Pulls)
			for xx := from; xx < to; xx++ {
				c.Put(xx, y, '▀', "pull:"+utils.Itoa(presentation.MemberColor(d.State, pull)))
			}
		}
	}
	titleStyle, detailStyle := "title", "detail"
	if focused {
		titleStyle = "bold"
	}
	if dim {
		titleStyle, detailStyle = "dim", "dim"
	}
	room := w - 4
	c.Text(x+2, y+1, utils.FitTitle(card.Title, room), titleStyle)
	c.Text(x+2, y+2, ansi.Truncate(card.Detail, room-ansi.StringWidth(state)-1, "…"), detailStyle)
	if state != "" {
		stateStyle := "state:" + card.State
		if dim {
			stateStyle = "dim"
		}
		c.Text(x+w-2-ansi.StringWidth(state), y+2, state, stateStyle)
	}
	if label != "" {
		c.Text(x+1, y, label, "tag")
	}
}

func box(c views.Canvas, x0, y0, x1, y1 int, style string, tl, tr, bl, br, h, v rune) {
	for x := x0 + 1; x < x1; x++ {
		c.Put(x, y0, h, style)
		c.Put(x, y1, h, style)
	}
	for y := y0 + 1; y < y1; y++ {
		c.Put(x0, y, v, style)
		c.Put(x1, y, v, style)
	}
	c.Put(x0, y0, tl, style)
	c.Put(x1, y0, tr, style)
	c.Put(x0, y1, bl, style)
	c.Put(x1, y1, br, style)
}

// Styles are the styles a drawn canvas names, in a theme.
func Styles(th theme.Theme) map[string]lipgloss.Style {
	styles := map[string]lipgloss.Style{
		"line": th.Muted, "lit": th.Accent, "faint": th.Muted.Faint(true), "border": th.Muted, "focus": th.Accent, "dim": th.Muted.Faint(true),
		"title": th.Text, "bold": th.Bold, "detail": th.Muted, "area": th.Muted, "tag": th.Key.Reverse(true),
		"state:new": th.Added, "state:changed": th.Warning, "state:removed": th.Removed, "state:same": th.Muted, "state:context": th.Muted,
	}
	for i := 1; i <= 8; i++ {
		styles["pull:"+utils.Itoa(i)] = th.PullRequest(i)
	}
	for name := range th.Colours.ComponentColors {
		styles["comp:"+name] = th.Component(name)
	}
	return styles
}
