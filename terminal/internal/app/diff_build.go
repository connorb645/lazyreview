package app

import (
	"fmt"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/server"
)

func diffFiles(d reviewData, component string, filter int) []server.FileEntry {
	var out []server.FileEntry
	for _, f := range d.page.Model.Files {
		if f.Component == component && (filter == 0 || containsInt(f.Pulls, filter)) {
			out = append(out, f)
		}
	}
	for _, f := range d.page.Model.ContextFiles {
		if f.Component == component {
			out = append(out, f)
		}
	}
	return out
}

func buildDiff(d reviewData, v diffView, filter int) []diffRow {
	var rows []diffRow
	for _, f := range diffFiles(d, v.component, filter) {
		rows = append(rows, diffRow{kind: "file", path: f.Path, text: f.Path})
		if v.folded[f.Path] {
			continue
		}
		file, loaded := v.files[f.Path]
		switch {
		case !loaded:
			rows = append(rows, diffRow{kind: "note", path: f.Path, text: "Loading…"})
		case f.Status == "":
			rows = append(rows, wholeRows(f.Path, file.Whole, nil)...)
		case f.Binary:
			rows = append(rows, diffRow{kind: "note", path: f.Path, text: "A binary file. Its contents are not shown."})
		case len(file.Hunks) == 0 && f.Status == "R":
			rows = append(rows, diffRow{kind: "note", path: f.Path, text: "Renamed, with no change to its contents."})
		case len(file.Hunks) == 0:
			rows = append(rows, diffRow{kind: "note", path: f.Path, text: "No change to its contents, only to its mode."})
		case v.rendered[f.Path] && file.Rendered != nil:
			rows = append(rows, blockRows(f.Path, *file.Rendered, v)...)
		case v.whole[f.Path] && file.Whole != nil:
			added := map[int]*server.Row{}
			for h := range file.Hunks {
				for i := range file.Hunks[h].Rows {
					if r := &file.Hunks[h].Rows[i]; r.Kind == "add" && f.Status != "A" {
						added[r.New] = r
					}
				}
			}
			rows = append(rows, wholeRows(f.Path, file.Whole, added)...)
		default:
			rows = append(rows, hunkRows(f.Path, file, v)...)
		}
	}
	return withNotes(d, v, rows)
}

func blockRows(path string, rendered server.Rendered, v diffView) []diffRow {
	var rows []diffRow
	if rendered.Partial {
		rows = append(rows, diffRow{kind: "note", path: path, text: "Only the changed parts show: the page kept no whole text of this file."})
	}
	for i := range rendered.Blocks {
		b := &rendered.Blocks[i]
		if rendered.Partial && i > 0 && b.Part != rendered.Blocks[i-1].Part {
			rows = append(rows, diffRow{kind: "note", path: path, text: "···"})
		}
		drawn := withoutMargin(v.md.lines(b.Markdown, max(20, v.width-blockGutter)))
		if len(drawn) == 0 {
			drawn = []string{""}
		}
		for j, text := range drawn {
			rows = append(rows, diffRow{kind: "block", path: path, text: text, block: b, last: j == len(drawn)-1})
		}
	}
	return rows
}

func withoutMargin(lines []string) []string {
	for _, l := range lines {
		if text := stripped(l); text != "" && !strings.HasPrefix(text, "  ") {
			return lines
		}
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = strings.Replace(l, "  ", "", 1)
	}
	return out
}

func wholeRows(path string, whole []string, added map[int]*server.Row) []diffRow {
	var rows []diffRow
	for i, text := range whole {
		n := i + 1
		if r, ok := added[n]; ok {
			rows = append(rows, diffRow{kind: "line", path: path, line: r, added: true})
			continue
		}
		rows = append(rows, diffRow{kind: "line", path: path, line: &server.Row{Kind: "same", New: n, Text: text}})
	}
	return rows
}

func hunkRows(path string, file server.File, v diffView) []diffRow {
	var rows []diffRow
	run := func(before int) {
		for _, g := range file.Gaps {
			if g.Before != before {
				continue
			}
			key := fmt.Sprintf("%s:%d", path, g.Before)
			if !v.opened[key] {
				rows = append(rows, diffRow{kind: "gap", path: path, key: key, count: g.To - g.From + 1})
				continue
			}
			for n := g.From; n <= g.To; n++ {
				text := ""
				if n-1 < len(file.Whole) {
					text = file.Whole[n-1]
				}
				same := &server.Row{Kind: "same", Old: n + g.Delta, New: n, Text: text}
				if v.split {
					rows = append(rows, diffRow{kind: "line", path: path, key: key, left: same, right: same})
				} else {
					rows = append(rows, diffRow{kind: "line", path: path, key: key, line: same})
				}
			}
		}
	}
	for h := range file.Hunks {
		run(h)
		hunk := &file.Hunks[h]
		rows = append(rows, diffRow{kind: "hunk", path: path, key: hunk.Header, text: hunk.Header})
		if v.split {
			rows = append(rows, pairs(path, hunk.Rows)...)
			continue
		}
		for i := range hunk.Rows {
			rows = append(rows, diffRow{kind: "line", path: path, line: &hunk.Rows[i]})
		}
	}
	run(len(file.Hunks))
	return rows
}

func pairs(path string, hunkRows []server.Row) []diffRow {
	var rows []diffRow
	var dels, adds []*server.Row
	flush := func() {
		for i := 0; i < max(len(dels), len(adds)); i++ {
			r := diffRow{kind: "line", path: path}
			if i < len(dels) {
				r.left = dels[i]
			}
			if i < len(adds) {
				r.right = adds[i]
			}
			rows = append(rows, r)
		}
		dels, adds = nil, nil
	}
	for i := range hunkRows {
		r := &hunkRows[i]
		switch r.Kind {
		case "del":
			dels = append(dels, r)
		case "add":
			adds = append(adds, r)
		case "note":
			flush()
			rows = append(rows, diffRow{kind: "note", path: path, text: r.Text})
		default:
			flush()
			rows = append(rows, diffRow{kind: "line", path: path, left: r, right: r})
		}
	}
	flush()
	return rows
}

func containsInt(list []int, n int) bool {
	for _, x := range list {
		if x == n {
			return true
		}
	}
	return false
}

func expandTabs(text string) string {
	return strings.ReplaceAll(text, "\t", "    ")
}
