package presentation

import (
	"fmt"
	"strings"
	"time"

	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

// BuildDiff makes the diff rows of files: each file's hunks, runs, whole
// text or rendered blocks as the view has them, and the notes on its lines.
func BuildDiff(tr *i18n.TranslationSet, md *Markdown, loc *time.Location, d types.ReviewData, v DiffView, files []models.FileEntry, filter types.PullFilter) []DiffRow {
	var rows []DiffRow
	for _, f := range files {
		rows = append(rows, DiffRow{Kind: "file", Path: f.Path, Text: f.Path})
		if v.Folded[f.Path] {
			continue
		}
		file, loaded := v.Files[f.Path]
		switch {
		case !loaded:
			rows = append(rows, DiffRow{Kind: "note", Path: f.Path, Text: tr.Loading})
		case f.Status == "":
			rows = append(rows, wholeRows(f.Path, file.Whole, nil)...)
		case f.Binary:
			rows = append(rows, DiffRow{Kind: "note", Path: f.Path, Text: tr.BinaryFile})
		case len(file.Hunks) == 0 && f.Status == "R":
			rows = append(rows, DiffRow{Kind: "note", Path: f.Path, Text: tr.RenamedOnly})
		case len(file.Hunks) == 0:
			rows = append(rows, DiffRow{Kind: "note", Path: f.Path, Text: tr.ModeOnly})
		case v.Rendered[f.Path] && file.Rendered != nil:
			rows = append(rows, blockRows(tr, md, f.Path, *file.Rendered, v)...)
		case v.Whole[f.Path] && file.Whole != nil:
			added := map[int]*models.Row{}
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
	return withNotes(tr, loc, d, v, rows)
}

func blockRows(tr *i18n.TranslationSet, md *Markdown, path string, rendered models.Rendered, v DiffView) []DiffRow {
	var rows []DiffRow
	if rendered.Partial {
		rows = append(rows, DiffRow{Kind: "note", Path: path, Text: tr.PartialRendered})
	}
	for i := range rendered.Blocks {
		b := &rendered.Blocks[i]
		if rendered.Partial && i > 0 && b.Part != rendered.Blocks[i-1].Part {
			rows = append(rows, DiffRow{Kind: "note", Path: path, Text: "···"})
		}
		drawn := withoutMargin(md.Lines(b.Markdown, max(20, v.Width-blockGutterOf(v))))
		if len(drawn) == 0 {
			drawn = []string{""}
		}
		for j, text := range drawn {
			rows = append(rows, DiffRow{Kind: "block", Path: path, Text: text, Block: b, Last: j == len(drawn)-1})
		}
	}
	return rows
}

func withoutMargin(lines []string) []string {
	for _, l := range lines {
		if text := utils.Stripped(l); text != "" && !strings.HasPrefix(text, "  ") {
			return lines
		}
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = strings.Replace(l, "  ", "", 1)
	}
	return out
}

func wholeRows(path string, whole []string, added map[int]*models.Row) []DiffRow {
	var rows []DiffRow
	for i, text := range whole {
		n := i + 1
		if r, ok := added[n]; ok {
			rows = append(rows, DiffRow{Kind: "line", Path: path, Line: r, Added: true})
			continue
		}
		rows = append(rows, DiffRow{Kind: "line", Path: path, Line: &models.Row{Kind: "same", New: n, Text: text}})
	}
	return rows
}

func hunkRows(path string, file models.File, v DiffView) []DiffRow {
	var rows []DiffRow
	run := func(before int) {
		for _, g := range file.Gaps {
			if g.Before != before {
				continue
			}
			key := fmt.Sprintf("%s:%d", path, g.Before)
			if !v.Opened[key] {
				rows = append(rows, DiffRow{Kind: "gap", Path: path, Key: key, Count: g.To - g.From + 1})
				continue
			}
			for n := g.From; n <= g.To; n++ {
				text := ""
				if n-1 < len(file.Whole) {
					text = file.Whole[n-1]
				}
				same := &models.Row{Kind: "same", Old: n + g.Delta, New: n, Text: text}
				if v.Split {
					rows = append(rows, DiffRow{Kind: "line", Path: path, Key: key, Left: same, Right: same})
				} else {
					rows = append(rows, DiffRow{Kind: "line", Path: path, Key: key, Line: same})
				}
			}
		}
	}
	for h := range file.Hunks {
		run(h)
		hunk := &file.Hunks[h]
		rows = append(rows, DiffRow{Kind: "hunk", Path: path, Key: hunk.Header, Text: hunk.Header})
		if v.Split {
			rows = append(rows, pairs(path, hunk.Rows)...)
			continue
		}
		for i := range hunk.Rows {
			rows = append(rows, DiffRow{Kind: "line", Path: path, Line: &hunk.Rows[i]})
		}
	}
	run(len(file.Hunks))
	return rows
}

func pairs(path string, hunkRows []models.Row) []DiffRow {
	var rows []DiffRow
	var dels, adds []*models.Row
	flush := func() {
		for i := 0; i < max(len(dels), len(adds)); i++ {
			r := DiffRow{Kind: "line", Path: path}
			if i < len(dels) {
				r.Left = dels[i]
			}
			if i < len(adds) {
				r.Right = adds[i]
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
			rows = append(rows, DiffRow{Kind: "note", Path: path, Text: r.Text})
		default:
			flush()
			rows = append(rows, DiffRow{Kind: "line", Path: path, Left: r, Right: r})
		}
	}
	flush()
	return rows
}

// ExpandTabs is a line with each tab as four spaces.
func ExpandTabs(text string) string {
	return strings.ReplaceAll(text, "\t", "    ")
}
