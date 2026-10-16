package app

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"review-desk/terminal/internal/diagram"
)

// drawDiagram draws a level on a canvas: links first, their words over
// them, cards over both, then where links meet the cards' borders, and the
// focused card's stubs. It returns the canvas, each row's first line, and
// what the stubs take.
func drawDiagram(d reviewData, v diagramView, filter int) (canvas, []int, stubs) {
	lvl, g := v.level, v.grid
	stubLines := stubWords(lvl, v)
	routes, lanes := routesOf(lvl, g, v.focus, v.all, len(stubLines) > 0)
	focusRow := -1
	if len(stubLines) > 0 {
		focusRow = g.Spots[v.focus].Row
	}
	below := stubs{lines: len(stubLines)}
	if focusRow >= 0 {
		s := g.Spots[v.focus]
		for _, line := range stubLines {
			below.width = max(below.width, s.Width-4+ansi.StringWidth(line))
		}
	}
	tops := make([]int, len(g.Rows))
	y := 1
	for r := range g.Rows {
		tops[r] = y
		gap := 2 + lanes[r]
		if r == focusRow {
			gap += below.lines
		}
		y += diagram.CardHeight + gap
	}
	width := g.Width + 2
	if focusRow >= 0 {
		width = max(width, g.Spots[v.focus].Col+below.width+2)
	}
	c := newCanvas(width, y)
	laneY := func(gap, lane int) int {
		ly := tops[gap] + diagram.CardHeight + 1 + lane
		if gap == focusRow {
			ly += below.lines
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
	for l, at := range v.labels {
		if strings.HasPrefix(l, v.typed) {
			labels[at] = l
		}
	}
	for i, card := range lvl.Cards {
		drawCard(c, d, lvl, card, g.Spots[i], tops[g.Spots[i].Row], i == v.focus, dimmed(card, filter), labels[i])
	}
	for _, pass := range []func(route) bool{isFaint, isPlain, isLit} {
		for _, r := range routes {
			if pass(r) {
				drawEnds(c, g, tops, r, laneY)
			}
		}
	}
	if focusRow >= 0 {
		s := g.Spots[v.focus]
		x, bottom := s.Col+s.Width-4, tops[focusRow]+diagram.CardHeight-1
		c.put(x, bottom, joined(c.cells[bottom][x].r, down), "focus")
		for i, line := range stubLines {
			tree := "├"
			if i == len(stubLines)-1 {
				tree = "└"
			}
			c.text(x, bottom+1+i, tree+line, "lit")
		}
	}
	return c, tops, below
}

// stubWords are the focused card's links to parts outside its level, out
// of it and then into it, each as the line its stub shows.
func stubWords(lvl diagram.Level, v diagramView) []string {
	card, ok := v.card()
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
			name, place, _ := strings.Cut(l.Outside, " · in ")
			in = append(in, "◀─ "+name+" "+l.Words+" · in "+place)
		}
	}
	return append(out, in...)
}

func isFaint(r route) bool { return r.faint }
func isPlain(r route) bool { return !r.faint && !r.lit }
func isLit(r route) bool   { return r.lit }

// routesOf finds how each link of a level that shows runs, links between
// the same two cards drawn as one, and how many lanes each gap below a row
// needs. The focused card keeps the right of its bottom border for stubs
// when it has them.
func routesOf(lvl diagram.Level, g diagram.Grid, focus int, all, stubbed bool) ([]route, []int) {
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

// placePorts spreads the ends of the routes on each border along it, in
// the order of where they go, and runs a route straight down when it is
// alone at both ends and the cards overlap.
func placePorts(routes []route, g diagram.Grid, focus int, stubbed bool) {
	ends := map[[2]int][]port{}
	centre := func(i int) int { return g.Spots[i].Col + g.Spots[i].Width/2 }
	for i, r := range routes {
		if r.side {
			continue
		}
		sf, st := g.Spots[r.from], g.Spots[r.to]
		leaves, arrives := down, up
		switch {
		case st.Row == sf.Row:
			arrives = down
		case st.Row < sf.Row:
			leaves, arrives = up, down
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
		if stubbed && key[0] == focus && key[1] == down {
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
		leaves, arrives := down, up
		if st.Row < sf.Row {
			leaves, arrives = up, down
		}
		if len(ends[[2]int{r.from, leaves}]) != 1 || len(ends[[2]int{r.to, arrives}]) != 1 {
			continue
		}
		lo, hi := max(sf.Col, st.Col)+2, min(sf.Col+sf.Width, st.Col+st.Width)-3
		if stubbed && (r.from == focus && leaves == down || r.to == focus && arrives == down) {
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

// laneRoutes gives each route a lane in each gap it crosses, in an order
// that crosses fewer lines: links that turn back into their own row first,
// the narrowest highest; then those running right, the one that starts
// furthest right highest; then those running left, mirrored. A straight
// run takes a lane only for its words. It returns each gap's lanes.
func laneRoutes(routes []route, g diagram.Grid) []int {
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
				r.lanes[s.at] = -1
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

// freeColumn is the column nearest x that no card of rows first to last
// covers, so a link can pass the rows between its ends.
func freeColumn(g diagram.Grid, first, last, x int) int {
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

func sideBySide(g diagram.Grid, a, b int) bool {
	row := g.Rows[g.Spots[a].Row]
	for i, at := range row {
		if at == a {
			return (i > 0 && row[i-1] == b) || (i+1 < len(row) && row[i+1] == b)
		}
	}
	return false
}

func dimmed(card diagram.Card, filter int) bool {
	if filter == 0 || card.Kind == "component" || card.Kind == "package" {
		return false
	}
	return !containsInt(card.Pulls, filter)
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

// path is the points a route between rows passes, from the border it
// leaves to the border its arrow sits on, and the direction the arrow
// points.
func (r route) path(g diagram.Grid, tops []int, laneY func(gap, lane int) int) ([][2]int, int) {
	sf, st := g.Spots[r.from], g.Spots[r.to]
	top := func(s diagram.Spot) int { return tops[s.Row] }
	bottom := func(s diagram.Spot) int { return tops[s.Row] + diagram.CardHeight - 1 }
	ya, yb, arrow := bottom(sf), top(st), down
	switch {
	case st.Row == sf.Row:
		yb, arrow = bottom(st), up
	case st.Row < sf.Row:
		ya, yb, arrow = top(sf), bottom(st), up
	}
	points := [][2]int{{r.xa, ya}}
	xs := []int{r.xa, r.xb}
	if len(r.gaps) == 2 {
		xs = []int{r.xa, r.via, r.via, r.xb}
	}
	for k, gap := range r.gaps {
		if r.lanes[k] < 0 {
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

func drawRoute(c canvas, g diagram.Grid, tops []int, r route, laneY func(gap, lane int) int) {
	style := routeStyle(r)
	if r.side {
		x0, x1, y := sideEnds(g, tops, r)
		c.segment(x0, y, x1, y, style)
		return
	}
	points, _ := r.path(g, tops, laneY)
	for i := 1; i < len(points); i++ {
		c.segment(points[i-1][0], points[i-1][1], points[i][0], points[i][1], style)
	}
}

// sideEnds are where a link between cards side by side runs: from the
// column of the border it leaves to the column its arrow sits on, on line y.
func sideEnds(g diagram.Grid, tops []int, r route) (int, int, int) {
	sf, st := g.Spots[r.from], g.Spots[r.to]
	y := tops[sf.Row] + 1 + r.across
	if st.Col > sf.Col {
		return sf.Col + sf.Width - 1, st.Col - 1, y
	}
	return sf.Col, st.Col + st.Width, y
}

func drawWords(c canvas, g diagram.Grid, tops []int, r route, laneY func(gap, lane int) int) {
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
			c.text(x, line, words, style)
		}
		return
	}
	points, _ := r.path(g, tops, laneY)
	if len(points) < 3 {
		return
	}
	last, before := points[len(points)-2], points[len(points)-3]
	y := last[1]
	rightX, rightWords, rightRoom := wordsIn(c, y, last[0]+2, c.w-1, r.words, -1)
	leftX, leftWords, leftRoom := wordsIn(c, y, 0, last[0]-2, r.words, 1)
	if before[1] == y && before[0] > last[0] {
		rightX, rightWords, rightRoom = wordsIn(c, y, before[0]+2, c.w-1, r.words, -1)
	} else if before[1] == y {
		leftRoom = 0
	}
	whole := ansi.StringWidth(r.words)
	leftFirst := before[1] == y && before[0] > last[0]
	switch {
	case leftFirst && leftRoom >= whole, !leftFirst && rightRoom < whole && leftRoom >= whole:
		c.text(leftX, y, leftWords, style)
	case rightRoom >= whole, rightRoom >= leftRoom && rightRoom > 0:
		c.text(rightX, y, rightWords, style)
	case leftRoom > 0:
		c.text(leftX, y, leftWords, style)
	}
}

// wordsIn fits a link's words on line y between columns lo and hi, in the
// run of free cells nearest lo (lean -1), hi (lean 1) or the widest, cut to
// fit it. It returns where they start, the words as cut, and the run's room,
// which is 0 where too little is free.
func wordsIn(c canvas, y, lo, hi int, words string, lean int) (int, string, int) {
	lo, hi = max(lo, 0), min(hi, c.w-1)
	free := func(x int) bool { return c.cells[y][x].r == ' ' }
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

// drawEnds puts a route's arrow on the border it points at, and joins its
// line to the border it leaves.
func drawEnds(c canvas, g diagram.Grid, tops []int, r route, laneY func(gap, lane int) int) {
	style := routeStyle(r)
	if r.side {
		x0, x1, y := sideEnds(g, tops, r)
		if x1 > x0 {
			c.put(x0, y, joined(c.cells[y][x0].r, right), c.cells[y][x0].style)
			c.put(x1, y, '▶', style)
		} else {
			c.put(x0, y, joined(c.cells[y][x0].r, left), c.cells[y][x0].style)
			c.put(x1, y, '◀', style)
		}
		return
	}
	points, arrow := r.path(g, tops, laneY)
	start, end := points[0], points[len(points)-1]
	leaves := down
	if points[1][1] < start[1] {
		leaves = up
	}
	if c.in(start[0], start[1]) {
		c.put(start[0], start[1], joined(c.cells[start[1]][start[0]].r, leaves), c.cells[start[1]][start[0]].style)
	}
	head := '▼'
	if arrow == up {
		head = '▲'
	}
	c.put(end[0], end[1], head, style)
}

// joined is a card's border rune with a line leaving it in a direction.
func joined(border rune, dir int) rune {
	joints := map[int]map[rune]rune{
		down:  {'─': '┬', '━': '┯', '┄': '┬'},
		up:    {'─': '┴', '━': '┷', '┄': '┴'},
		right: {'│': '├', '┃': '┠', '┆': '├'},
		left:  {'│': '┤', '┃': '┨', '┆': '┤'},
	}
	if r, ok := joints[dir][border]; ok {
		return r
	}
	return border
}

func drawAreas(c canvas, d reviewData, lvl diagram.Level, g diagram.Grid, tops []int) {
	for _, area := range d.page.Model.Areas {
		x0, y0, x1, y1, found := 0, 0, 0, 0, false
		for i, card := range lvl.Cards {
			if card.Area != area.ID {
				continue
			}
			s := g.Spots[i]
			cx0, cy0, cx1, cy1 := s.Col-2, tops[s.Row]-1, s.Col+s.Width+1, tops[s.Row]+diagram.CardHeight
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
		c.text(x0+2, max(0, y0), " "+area.Title+" ", "area")
	}
}

func drawCard(c canvas, d reviewData, lvl diagram.Level, card diagram.Card, s diagram.Spot, y int, focused, dim bool, label string) {
	x, w := s.Col, s.Width
	border := "border"
	if lvl.Kind == "components" {
		border = "comp:" + componentColour(d, card.ID)
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
	for yy := y; yy < y+diagram.CardHeight; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.put(xx, yy, ' ', "")
		}
	}
	box(c, x, y, x+w-1, y+diagram.CardHeight-1, border, tl, tr, bl, br, h, v)
	if len(card.Pulls) > 0 && d.page.Stack != nil && !focused && !dim {
		inner := w - 2
		for i, pull := range card.Pulls {
			from, to := x+1+inner*i/len(card.Pulls), x+1+inner*(i+1)/len(card.Pulls)
			for xx := from; xx < to; xx++ {
				c.put(xx, y, '▀', "pull:"+itoa(memberColor(d.state, pull)))
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
	c.text(x+2, y+1, ansi.Truncate(card.Title, room, "…"), titleStyle)
	state := diagram.StateWord(card.State)
	c.text(x+2, y+2, ansi.Truncate(card.Detail, room-ansi.StringWidth(state)-1, "…"), detailStyle)
	if state != "" {
		stateStyle := "state:" + card.State
		if dim {
			stateStyle = "dim"
		}
		c.text(x+w-2-ansi.StringWidth(state), y+2, state, stateStyle)
	}
	if label != "" {
		c.text(x+1, y, label, "tag")
	}
}

func box(c canvas, x0, y0, x1, y1 int, style string, tl, tr, bl, br, h, v rune) {
	for x := x0 + 1; x < x1; x++ {
		c.put(x, y0, h, style)
		c.put(x, y1, h, style)
	}
	for y := y0 + 1; y < y1; y++ {
		c.put(x0, y, v, style)
		c.put(x1, y, v, style)
	}
	c.put(x0, y0, tl, style)
	c.put(x1, y0, tr, style)
	c.put(x0, y1, bl, style)
	c.put(x1, y1, br, style)
}

func componentColour(d reviewData, id string) string {
	for _, c := range d.page.Model.Components {
		if c.ID == id {
			return c.Color
		}
	}
	return "grey"
}

// diagramStyles are the styles the diagram's canvas names.
func diagramStyles(p palette) map[string]lipgloss.Style {
	styles := map[string]lipgloss.Style{
		"line": p.Muted, "lit": p.Accent, "faint": p.Muted.Faint(true), "border": p.Muted, "focus": p.Accent, "dim": p.Muted.Faint(true),
		"title": p.Ink, "bold": p.Bold, "detail": p.Muted, "area": p.Muted, "tag": p.Key.Reverse(true),
		"state:new": p.Add, "state:changed": p.Warn, "state:removed": p.Del, "state:same": p.Muted, "state:context": p.Muted,
	}
	for i := 1; i <= 8; i++ {
		styles["pull:"+itoa(i)] = p.pull(i)
	}
	for name := range p.components {
		styles["comp:"+name] = p.componentColor(name)
	}
	return styles
}
