// Package helpers holds the steps several controllers share, each a pure
// function of the state that gives the next state and what to run. Only
// helpers reach the server and the programs of the machine.
package helpers

// Helpers are every helper, built once and shared by every controller.
type Helpers struct {
	Window    *WindowArrangementHelper
	AppStatus *AppStatusHelper
	Refresh   *RefreshHelper
	Panels    *PanelsHelper
	Diff      *DiffHelper
	Diagram   *DiagramHelper
	Jump      *JumpHelper
	Filter    *FilterHelper
	Home      *HomeHelper
	Review    *ReviewHelper
	Live      *LiveHelper
	Editing   *EditingHelper
	Posting   *PostingHelper
	Menu      *MenuHelper
	Browser   *BrowserHelper
	Prompt    *PromptHelper
}

// NewHelpers builds every helper on what they share, and gives the shared
// part the helpers, so each can reach the others.
func NewHelpers(c *HelperCommon) *Helpers {
	h := &Helpers{
		Window:    &WindowArrangementHelper{c: c},
		AppStatus: &AppStatusHelper{c: c},
		Refresh:   &RefreshHelper{c: c},
		Panels:    &PanelsHelper{c: c},
		Diff:      &DiffHelper{c: c},
		Diagram:   &DiagramHelper{c: c},
		Jump:      &JumpHelper{c: c},
		Filter:    &FilterHelper{c: c},
		Home:      &HomeHelper{c: c},
		Review:    &ReviewHelper{c: c},
		Live:      &LiveHelper{c: c},
		Editing:   &EditingHelper{c: c},
		Posting:   &PostingHelper{c: c},
		Menu:      &MenuHelper{c: c},
		Browser:   &BrowserHelper{c: c},
		Prompt:    &PromptHelper{c: c},
	}
	c.Helpers = h
	return h
}
