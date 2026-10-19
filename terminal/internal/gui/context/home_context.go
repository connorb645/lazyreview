package context

import (
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/theme"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// HomeContext is the home screen: the open pull requests read from GitHub,
// or why they could not be, what is being read or loaded, whether the
// repositories the config hides show, a panel for each repository, the one
// with the keys, and the place of the review the home screen was opened
// from.
type HomeContext struct {
	List        *models.Pulls
	Failed      string
	Reading     bool
	Loading     string
	LoadingName string
	ShowHidden  bool
	Panels      []HomePanel
	Focus       int
	From        Screen
	FromFocus   int
	FromDriving int
}

// Rebuilt is the home screen with its panels made again from its list,
// each keeping its selection, the keys staying with the panel that had
// them, or at its place when it is gone.
func (h HomeContext) Rebuilt(tr *i18n.TranslationSet, th theme.Theme) HomeContext {
	wasTitle := ""
	if h.Focus >= 0 && h.Focus < len(h.Panels) {
		wasTitle = h.Panels[h.Focus].Title
	}
	kept := map[string]SideContext{}
	for _, panel := range h.Panels {
		kept[panel.Title] = panel.Rows
	}
	var panels []HomePanel
	for _, p := range presentation.HomePanels(tr, th, h.List, h.Failed, h.ShowHidden) {
		panel := HomePanel{Title: p.Title, Number: p.Number, Repo: p.Repo, Waiting: p.Waiting, Hidden: p.Hidden, Rows: SideContext{}.Reset(p.Rows)}
		if rows, ok := kept[panel.Title]; ok {
			panel.Rows = rows.Reset(p.Rows)
		}
		panels = append(panels, panel)
	}
	h.Panels, h.Focus = panels, utils.Clamp(h.Focus, 0, len(panels)-1)
	for i, panel := range panels {
		if panel.Title == wasTitle {
			h.Focus = i
		}
	}
	return h
}

// Repo is the repository of the panel with the keys, if it is one.
func (h HomeContext) Repo() (models.PullRepo, bool) {
	if h.List == nil || h.Focus < 0 || h.Focus >= len(h.Panels) || h.Panels[h.Focus].Repo < 0 {
		return models.PullRepo{}, false
	}
	return h.List.Repos[h.Panels[h.Focus].Repo], true
}

// WithHidden is the home screen with each repository hidden by what the
// config's hidden repositories now say.
func (h HomeContext) WithHidden(hidden models.Hidden) HomeContext {
	if h.List == nil {
		return h
	}
	by := map[string][]string{}
	for _, r := range hidden.Repos {
		by[r.Name] = r.HiddenBy
	}
	list := *h.List
	list.Repos = append([]models.PullRepo{}, list.Repos...)
	for i, repo := range list.Repos {
		if entries, ok := by[repo.Name]; ok {
			list.Repos[i].HiddenBy = entries
		}
	}
	h.List = &list
	return h
}

func (h HomeContext) focusedRows() SideContext {
	if h.Focus < 0 || h.Focus >= len(h.Panels) {
		return SideContext{Cursor: -1}
	}
	return h.Panels[h.Focus].Rows
}

// Selected is the row selected in the panel with the keys.
func (h HomeContext) Selected() (presentation.SideRow, bool) {
	return h.focusedRows().Selected()
}

// Pull is the pull request selected, its repository and the stack it is
// in, if a pull request is selected.
func (h HomeContext) Pull() (models.PullRepo, models.Unit, models.OpenPull, bool) {
	row, ok := h.Selected()
	if !ok || row.Item.Kind != "pull" || h.List == nil {
		return models.PullRepo{}, models.Unit{}, models.OpenPull{}, false
	}
	repo := h.List.Repos[h.Panels[h.Focus].Repo]
	for _, u := range repo.Units {
		for _, m := range u.Members {
			if m.Number == row.Item.Pull {
				return repo, u, m, true
			}
		}
	}
	return models.PullRepo{}, models.Unit{}, models.OpenPull{}, false
}

// Moved is the home screen with the selection moved by rows in the panel
// with the keys, stopping at its ends.
func (h HomeContext) Moved(by int) HomeContext {
	rows := h.focusedRows()
	if rows.Cursor < 0 {
		return h
	}
	return h.At(h.Focus, rows.Move(by, len(rows.Rows)).Cursor)
}

// Nth is the home screen with the nth row of the panel with the keys
// selected, counting from 1, or the nearest one after it that can be.
func (h HomeContext) Nth(n int) HomeContext {
	rows := h.focusedRows()
	if at := rows.Landing(utils.Clamp(n-1, 0, len(rows.Rows)-1), 1); at >= 0 {
		return h.At(h.Focus, at)
	}
	return h
}

// Ends is the home screen with the first row of the panel with the keys
// selected, or its last.
func (h HomeContext) Ends(last bool) HomeContext {
	rows := h.focusedRows()
	from, dir := 0, 1
	if last {
		from, dir = len(rows.Rows)-1, -1
	}
	if at := rows.Landing(from, dir); at >= 0 {
		return h.At(h.Focus, at)
	}
	return h
}

// ToRepo is the home screen with the keys in the panel of a numbered
// repository.
func (h HomeContext) ToRepo(number int) HomeContext {
	for i, p := range h.Panels {
		if p.Number == number {
			if p.Rows.Cursor >= 0 {
				return h.At(i, p.Rows.Cursor)
			}
			h.Focus = i
			return h
		}
	}
	return h
}

// PanelBy is the home screen with the keys in the panel by panels on,
// going round at the ends.
func (h HomeContext) PanelBy(by int) HomeContext {
	if len(h.Panels) > 0 {
		h.Focus = utils.Around(h.Focus+by, len(h.Panels))
	}
	return h
}

// At is the home screen with a row of a panel selected and the keys there.
func (h HomeContext) At(panel, row int) HomeContext {
	panels := append([]HomePanel{}, h.Panels...)
	panels[panel].Rows.Cursor = row
	h.Panels, h.Focus = panels, panel
	return h
}

// Shown is the home screen with each panel showing its selection in its
// height.
func (h HomeContext) Shown(heights []int) HomeContext {
	panels := append([]HomePanel{}, h.Panels...)
	for i := range panels {
		panels[i].Rows = panels[i].Rows.Shown(max(1, heights[i]))
	}
	h.Panels = panels
	return h
}

// Wants are the heights each panel wants, with its frame.
func (h HomeContext) Wants() []int {
	out := make([]int, len(h.Panels))
	for i, p := range h.Panels {
		out[i] = max(1, len(p.Rows.Rows)) + 2
	}
	return out
}

// WithReview is the home screen with a stack, or a lone pull request,
// marked as loaded into a review.
func (h HomeContext) WithReview(unit, id string) HomeContext {
	if h.List == nil {
		return h
	}
	list := *h.List
	list.Repos = append([]models.PullRepo{}, list.Repos...)
	for r, repo := range list.Repos {
		repo.Units = append([]models.Unit{}, repo.Units...)
		for u, un := range repo.Units {
			if presentation.UnitKey(repo, un) != unit {
				continue
			}
			un.Review = id
			un.Members = append([]models.OpenPull{}, un.Members...)
			for m := range un.Members {
				un.Members[m].Review = id
			}
			repo.Units[u] = un
		}
		list.Repos[r] = repo
	}
	h.List = &list
	return h
}
