package review

import "github.com/connorb645/lazyreview/terminal/internal/models"

// PageView is what the app reads of a review's page to draw the diagram and
// the file lists: its build id, its model, each unchanged file of the
// context with its count of lines, its layout, and for a stack its members
// and its warnings. It leaves out the diff, the whole texts, the code of
// the context and the stack's line map, which FileView gives a file at a
// time.
func PageView(build string, page *models.PageRecord) models.Page {
	model := page.Model
	model.ContextFiles = make([]models.FileEntry, len(page.Model.ContextFiles))
	for i, f := range page.Model.ContextFiles {
		f.Lines = len(page.Context[f.Path])
		model.ContextFiles[i] = f
	}
	view := models.Page{Build: build, Model: model, Layout: page.Layout}
	if page.Stack != nil {
		view.Stack = &models.PageStack{Members: page.Stack.Members, Warnings: page.Stack.Warnings}
	}
	return view
}
