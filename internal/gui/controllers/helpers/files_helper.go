package helpers

import (
	"slices"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/utils"
)

// FilesHelper chooses the files the main panel's diff shows: the Files
// panel's row under its cursor, a run of its rows selected as Vim's visual
// mode selects lines, or a file or component jumped to from elsewhere.
type FilesHelper struct {
	c *HelperCommon
}

// FromCursor is the state with the Files panel choosing what its cursor is
// on, while that panel has the keys.
func (self *FilesHelper) FromCursor(s context.ContextTree) context.ContextTree {
	if s.Focus() == context.FilesPanel {
		s.FilesPanel = s.FilesPanel.FromCursor()
	}
	return s
}

// Select is the state with a selection started on the Files panel's
// cursor, or the one growing stopped, the main panel following.
func (self *FilesHelper) Select(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	s.FilesPanel = s.FilesPanel.SelectionToggled()
	return self.c.Helpers.Panels.Followed(s)
}

// Stop is the state with the Files panel's selection stopped, its cursor's
// row chosen alone, the main panel following.
func (self *FilesHelper) Stop(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	s.FilesPanel = s.FilesPanel.SelectionStopped()
	return self.c.Helpers.Panels.Followed(s)
}

// ChoseFile is the state with the Files panel choosing one file, as a jump
// into it does: its folder and component unfolded, so its row shows.
func (self *FilesHelper) ChoseFile(s context.ContextTree, path string) context.ContextTree {
	component := presentation.FileEntry(*s.Review, path).Component
	s = self.chose(s, presentation.FileKey(path))
	folds := []string{presentation.ComponentKey(component), presentation.DirKey(component, path)}
	if !slices.ContainsFunc(folds, func(key string) bool { return s.FilesPanel.Folded[key] }) {
		return s
	}
	s.FilesPanel.Folded = utils.CopyBools(s.FilesPanel.Folded)
	for _, key := range folds {
		delete(s.FilesPanel.Folded, key)
	}
	return self.c.Helpers.Refresh.SidesBuilt(s)
}

// ChoseComponent is the state with the Files panel choosing a component's
// row, as the diagram's boxes open it.
func (self *FilesHelper) ChoseComponent(s context.ContextTree, component string) context.ContextTree {
	return self.chose(s, presentation.ComponentKey(component))
}

// Kept is the state with the Files panel's choice kept to the rows that
// still hold a file the filter shows, or its cursor's row when none does.
// While a selection grows, it is the rows the selection now runs over; when
// the row it grew from is gone, it stops growing and keeps what it held.
func (self *FilesHelper) Kept(s context.ContextTree) context.ContextTree {
	f := s.FilesPanel
	if s.Focus() == context.FilesPanel && f.GrowFrom != "" && f.Index(f.GrowFrom) >= 0 {
		s.FilesPanel = f.FromCursor()
		return s
	}
	f.GrowFrom = ""
	d, shown := *s.Review, s.ShownPulls()
	f.Chosen = slices.DeleteFunc(slices.Clone(f.Chosen), func(key string) bool {
		return len(presentation.ChosenFiles(d, []string{key}, shown)) == 0
	})
	if len(f.Chosen) == 0 {
		f = f.SelectionStopped()
	}
	s.FilesPanel = f
	return s
}

// GrowingInFiles is the state with the selection no longer growing once
// the keys leave the Files panel.
func (self *FilesHelper) GrowingInFiles(s context.ContextTree) context.ContextTree {
	if s.Focus() != context.FilesPanel || s.Screen() == context.ScreenHome {
		s.FilesPanel.GrowFrom = ""
	}
	return s
}

func (self *FilesHelper) chose(s context.ContextTree, key string) context.ContextTree {
	s.FilesPanel.GrowFrom, s.FilesPanel.Chosen = "", []string{key}
	return s
}
