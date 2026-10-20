package presentation

import "github.com/connorb645/lazyreview/internal/i18n"

// PanelNames are the panels' names by number: the main panel, then the
// four on the left.
func PanelNames(tr *i18n.TranslationSet) [5]string {
	return [5]string{tr.MainTitle, tr.ReviewTitle, tr.FilesTitle, tr.NotesTitle, tr.DiagramTitle}
}
