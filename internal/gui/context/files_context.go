package context

// FilesContext is the Files panel: its rows, the components and folders
// folded in it, the keys of the rows whose files the main panel's diff
// shows, and while a selection grows, the key of the row it grew from.
type FilesContext struct {
	SideContext
	Folded   map[string]bool
	Chosen   []string
	GrowFrom string
}

// FromCursor is the panel choosing what its cursor is on: while a selection
// grows, the rows from the one it grew from to the cursor's, else the
// cursor's row alone. A selection whose first row is gone stops growing.
func (f FilesContext) FromCursor() FilesContext {
	row, ok := f.Selected()
	if !ok {
		return f
	}
	from := f.Index(f.GrowFrom)
	if f.GrowFrom == "" || from < 0 {
		f.GrowFrom, f.Chosen = "", []string{row.Key}
		return f
	}
	f.Chosen = nil
	for i := min(from, f.Cursor); i <= max(from, f.Cursor); i++ {
		f.Chosen = append(f.Chosen, f.Rows[i].Key)
	}
	return f
}

// SelectionToggled is the panel starting a selection on its cursor's row,
// or stopping the one that grows; either way the cursor's row is chosen
// alone.
func (f FilesContext) SelectionToggled() FilesContext {
	row, ok := f.Selected()
	if !ok {
		return f
	}
	from := row.Key
	if f.GrowFrom != "" {
		from = ""
	}
	f.GrowFrom, f.Chosen = from, []string{row.Key}
	return f
}

// SelectionStopped is the panel with no selection, its cursor's row chosen
// alone.
func (f FilesContext) SelectionStopped() FilesContext {
	f.GrowFrom = ""
	return f.FromCursor()
}

// HasSelection says whether a selection grows, or the panel chooses more
// than its cursor's row, or another row.
func (f FilesContext) HasSelection() bool {
	row, ok := f.Selected()
	return f.GrowFrom != "" || ok && (len(f.Chosen) != 1 || f.Chosen[0] != row.Key)
}
