package review

import "github.com/connorb645/lazyreview/terminal/internal/models"

// WholeText is the whole text of a file of a review's page, as lines: a
// changed file's text at the head when the page kept it, an added file's
// from its diff, an unchanged file of the context its own, and nil for any
// other, a deleted file or one too large to keep.
func WholeText(page *models.PageRecord, path string) []string {
	if lines, ok := page.Sources[path]; ok && lines != nil {
		return lines
	}
	if lines, ok := page.Context[path]; ok && lines != nil {
		return lines
	}
	file, ok := fileOf(page, path)
	if !ok || file.Status != "A" {
		return nil
	}
	lines := []string{}
	for _, h := range ReadHunks(page.Diff[path]) {
		for _, r := range h.Rows {
			if r.Kind == "add" {
				lines = append(lines, r.Text)
			}
		}
	}
	return lines
}

func fileOf(page *models.PageRecord, path string) (models.FileEntry, bool) {
	for _, f := range page.Model.Files {
		if f.Path == path {
			return f, true
		}
	}
	return models.FileEntry{}, false
}
