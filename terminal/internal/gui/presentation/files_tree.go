package presentation

import (
	"path"
	"slices"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
)

// TreeFiles are the files the Files panel lists, in its order: by
// component, then by folder, those the filter shows.
func TreeFiles(d types.ReviewData, filter types.PullFilter) []server.FileEntry {
	var out []server.FileEntry
	for _, c := range d.Page.Model.Components {
		for _, folder := range byFolder(DiffFiles(d, c.ID, filter)) {
			out = append(out, folder...)
		}
	}
	return out
}

// ChosenFiles are the files that rows of the Files panel hold, by the rows'
// keys, in the tree's order: a file's row holds its file, and a folder's or
// a component's every file under it that the filter shows.
func ChosenFiles(d types.ReviewData, keys []string, filter types.PullFilter) []server.FileEntry {
	var out []server.FileEntry
	for _, f := range TreeFiles(d, filter) {
		if slices.ContainsFunc(keys, func(key string) bool { return Holds(key, f) }) {
			out = append(out, f)
		}
	}
	return out
}

// Holds says whether the row of the Files panel with a key holds a file:
// the file's own row, its folder's or its component's.
func Holds(key string, f server.FileEntry) bool {
	return key == FileKey(f.Path) || key == DirKey(f.Component, f.Path) || key == ComponentKey(f.Component)
}

// LitRows are rows of the Files panel with those of the keys lit, as a
// selection's rows are, and the rows of the files a key holds whose own row
// does not show, such as a folder the filter leaves with no row.
func LitRows(rows []SideRow, keys []string) []SideRow {
	var unshown []string
	for _, key := range keys {
		if !slices.ContainsFunc(rows, func(r SideRow) bool { return r.Key == key }) {
			unshown = append(unshown, key)
		}
	}
	out := make([]SideRow, len(rows))
	for i, r := range rows {
		r.Lit = slices.Contains(keys, r.Key) || r.Item.Kind == "file" && slices.ContainsFunc(unshown, func(key string) bool {
			return Holds(key, server.FileEntry{Path: r.Item.Place.Path, Component: r.Item.Component})
		})
		out[i] = r
	}
	return out
}

func byFolder(files []server.FileEntry) [][]server.FileEntry {
	var folders [][]server.FileEntry
	at := map[string]int{}
	for _, f := range files {
		dir := path.Dir(f.Path)
		i, seen := at[dir]
		if !seen {
			i = len(folders)
			at[dir] = i
			folders = append(folders, nil)
		}
		folders[i] = append(folders[i], f)
	}
	return folders
}
