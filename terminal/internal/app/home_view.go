package app

import "github.com/connorb645/lazyreview/terminal/internal/server"

type homeView struct {
	list        *server.Pulls
	failed      string
	reading     bool
	loading     string
	loadingName string
	panels      []homePanel
	focus       int
	from        screen
	fromFocus   int
	fromDriving int
}

func (h homeView) rebuilt(p palette) homeView {
	was, _ := h.selected()
	kept := map[string]sidePanel{}
	for _, panel := range h.panels {
		kept[panel.title] = panel.rows
	}
	panels := homePanels(h, p)
	for i, panel := range panels {
		if rows, ok := kept[panel.title]; ok {
			panels[i].rows = rows.reset(panel.rows.rows)
		}
	}
	h.panels, h.focus = panels, clamp(h.focus, 0, len(panels)-1)
	for i, panel := range panels {
		if was.key != "" && panel.rows.index(was.key) >= 0 {
			h.focus = i
		}
	}
	return h
}

func (h homeView) focusedRows() sidePanel {
	if h.focus < 0 || h.focus >= len(h.panels) {
		return sidePanel{cursor: -1}
	}
	return h.panels[h.focus].rows
}

func (h homeView) selected() (sideRow, bool) {
	return h.focusedRows().selected()
}

func (h homeView) pull() (server.PullRepo, server.Unit, server.OpenPull, bool) {
	row, ok := h.selected()
	if !ok || row.it.kind != "pull" || h.list == nil {
		return server.PullRepo{}, server.Unit{}, server.OpenPull{}, false
	}
	repo := h.list.Repos[h.panels[h.focus].repo]
	for _, u := range repo.Units {
		for _, m := range u.Members {
			if m.Number == row.it.pull {
				return repo, u, m, true
			}
		}
	}
	return server.PullRepo{}, server.Unit{}, server.OpenPull{}, false
}

func (h homeView) moved(by int) homeView {
	for n := by * sign(by); n > 0; n-- {
		next, ok := h.stepped(sign(by))
		if !ok {
			break
		}
		h = next
	}
	return h
}

func (h homeView) stepped(dir int) (homeView, bool) {
	rows := h.focusedRows()
	if rows.cursor >= 0 {
		if next := rows.landing(rows.cursor+dir, dir); next >= 0 && (next-rows.cursor)*dir > 0 {
			return h.at(h.focus, next), true
		}
	}
	for i := h.focus + dir; i >= 0 && i < len(h.panels); i += dir {
		from := 0
		if dir < 0 {
			from = len(h.panels[i].rows.rows) - 1
		}
		if at := h.panels[i].rows.landing(from, dir); at >= 0 {
			return h.at(i, at), true
		}
	}
	return h, false
}

func (h homeView) ends(last bool) homeView {
	for n := range h.panels {
		i, from, dir := n, 0, 1
		if last {
			i = len(h.panels) - 1 - n
			from, dir = len(h.panels[i].rows.rows)-1, -1
		}
		if at := h.panels[i].rows.landing(from, dir); at >= 0 {
			return h.at(i, at)
		}
	}
	return h
}

func (h homeView) toRepo(number int) homeView {
	for i, p := range h.panels {
		if p.number == number {
			if p.rows.cursor >= 0 {
				return h.at(i, p.rows.cursor)
			}
			h.focus = i
			return h
		}
	}
	return h
}

func (h homeView) numbers(most int) []string {
	var out []string
	for _, p := range h.panels {
		if p.number > 0 && len(out) < most {
			out = append(out, itoa(p.number))
		}
	}
	return out
}

func (h homeView) at(panel, row int) homeView {
	panels := append([]homePanel{}, h.panels...)
	panels[panel].rows.cursor = row
	h.panels, h.focus = panels, panel
	return h
}

func (h homeView) shown(heights []int) homeView {
	panels := append([]homePanel{}, h.panels...)
	for i := range panels {
		panels[i].rows = panels[i].rows.shown(max(1, heights[i]))
	}
	h.panels = panels
	return h
}

func (h homeView) wants() []int {
	out := make([]int, len(h.panels))
	for i, p := range h.panels {
		out[i] = max(1, len(p.rows.rows)) + 2
	}
	return out
}

func (h homeView) withReview(unit, id string) homeView {
	if h.list == nil {
		return h
	}
	list := *h.list
	list.Repos = append([]server.PullRepo{}, list.Repos...)
	for r, repo := range list.Repos {
		repo.Units = append([]server.Unit{}, repo.Units...)
		for u, un := range repo.Units {
			if unitKey(repo, un) != unit {
				continue
			}
			un.Review = id
			un.Members = append([]server.OpenPull{}, un.Members...)
			for m := range un.Members {
				un.Members[m].Review = id
			}
			repo.Units[u] = un
		}
		list.Repos[r] = repo
	}
	h.list = &list
	return h
}
