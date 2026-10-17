package presentation

import (
	"fmt"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// FileEntry is a file of the review's change or of its unchanged context,
// or a bare entry of the path when it is neither.
func FileEntry(d types.ReviewData, path string) server.FileEntry {
	for _, f := range d.Page.Model.Files {
		if f.Path == path {
			return f
		}
	}
	for _, f := range d.Page.Model.ContextFiles {
		if f.Path == path {
			return f
		}
	}
	return server.FileEntry{Path: path}
}

// DiffFiles are a component's files in its diff: the changed ones the
// filter shows, then its unchanged context.
func DiffFiles(d types.ReviewData, component string, filter types.PullFilter) []server.FileEntry {
	var out []server.FileEntry
	for _, f := range d.Page.Model.Files {
		if f.Component == component && filter.ShowsAny(f.Pulls) {
			out = append(out, f)
		}
	}
	for _, f := range d.Page.Model.ContextFiles {
		if f.Component == component {
			out = append(out, f)
		}
	}
	return out
}

// ShowsFile says whether the filter shows a file: one of the context, or
// one a pull request it shows changed.
func ShowsFile(filter types.PullFilter, d types.ReviewData, path string) bool {
	for _, c := range d.Page.Model.ContextFiles {
		if c.Path == path {
			return true
		}
	}
	return filter.ShowsAny(FileEntry(d, path).Pulls)
}

// FilterWords names the filter's pull requests, such as "#21 and #22".
func FilterWords(tr *i18n.TranslationSet, f types.PullFilter) string {
	names := make([]string, len(f))
	for i, pull := range f {
		names[i] = fmt.Sprintf("#%d", pull)
	}
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return utils.ResolvePlaceholderString(tr.ListAnd, map[string]string{"list": strings.Join(names[:len(names)-1], ", "), "last": names[len(names)-1]})
}

// StackPulls are a stack's pull requests bottom first, or none for a review
// of one.
func StackPulls(d types.ReviewData) []int {
	if d.Page.Stack == nil {
		return nil
	}
	var pulls []int
	for _, m := range d.Page.Stack.Members {
		pulls = append(pulls, m.Pull)
	}
	return pulls
}

// StackMember is a pull request's own part of a stack.
func StackMember(stack *server.PageStack, pull int) server.StackMember {
	for _, m := range stack.Members {
		if m.Pull == pull {
			return m
		}
	}
	return server.StackMember{Pull: pull}
}

// MemberColor is a pull request's colour number in its stack, or 0.
func MemberColor(s server.State, pull int) int {
	for _, m := range append(append([]server.Member{}, s.Members...), s.Earlier...) {
		if m.Pull == pull {
			return m.Color
		}
	}
	return 0
}

// ComponentTitle is a component's title, or its id when it has none.
func ComponentTitle(d types.ReviewData, id string) string {
	for _, c := range d.Page.Model.Components {
		if c.ID == id {
			return c.Title
		}
	}
	return id
}

// ComponentColour is the name of a component's colour, or grey.
func ComponentColour(d types.ReviewData, id string) string {
	for _, c := range d.Page.Model.Components {
		if c.ID == id {
			return c.Color
		}
	}
	return "grey"
}

// PackageTitle is a component's package's title, its path when it has no
// title, or its id when there is no such package.
func PackageTitle(page server.Page, component, id string) string {
	for _, c := range page.Model.Components {
		for _, pkg := range c.Packages {
			if c.ID == component && pkg.ID == id {
				if pkg.Title != "" {
					return pkg.Title
				}
				return pkg.Path
			}
		}
	}
	return id
}

// NodeOf is the type or file of the model with an id.
func NodeOf(page server.Page, id string) (server.Node, bool) {
	for _, n := range page.Model.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return server.Node{}, false
}
