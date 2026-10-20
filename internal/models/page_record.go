package models

// PageRecord is a review's page as page.json keeps it: the model and layout
// the diagram draws, each file's part of the diff, the whole text of each
// changed file it kept, the code of each unchanged file of the context, as
// lines, and on a stack where a comment on each line goes.
type PageRecord struct {
	Model   Model               `json:"model"`
	Layout  Layout              `json:"layout"`
	Diff    map[string]string   `json:"diff"`
	Sources map[string][]string `json:"sources"`
	Context map[string][]string `json:"context"`
	Stack   *StackRecord        `json:"stack"`
}
