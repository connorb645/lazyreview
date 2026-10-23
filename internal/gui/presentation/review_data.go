package presentation

import (
	"fmt"
	"strings"

	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

// FileEntry is a file of the review's change or of its unchanged context,
// or a bare entry of the path when it is neither.
func FileEntry(d types.ReviewData, path string) models.FileEntry {
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
	return models.FileEntry{Path: path}
}

// DiffFiles are a component's files in its diff: the changed ones the
// filter shows, then its unchanged context.
func DiffFiles(d types.ReviewData, component string, filter types.PullFilter) []models.FileEntry {
	var out []models.FileEntry
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

// FilterWords names the filter's members, such as "#21 and #22".
func FilterWords(tr *i18n.TranslationSet, d types.ReviewData, f types.PullFilter) string {
	names := make([]string, len(f))
	for i, pull := range f {
		names[i] = MemberName(d, pull)
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

// MemberName names a member of the review's stack, or its one pull
// request: a pull request by its number, such as #21, and a branch by its
// name.
func MemberName(d types.ReviewData, pull int) string {
	if d.Page.Stack != nil {
		for _, m := range d.Page.Stack.Members {
			if m.Pull == pull && m.Branch != "" {
				return m.Branch
			}
		}
	}
	return fmt.Sprintf("#%d", pull)
}

// MemberNamed is the member of the review's stack that name names: a pull
// request by its number, with or without its #, or a branch by its name;
// and false when none does.
func MemberNamed(d types.ReviewData, name string) (int, bool) {
	if d.Page.Stack == nil {
		return 0, false
	}
	for _, m := range d.Page.Stack.Members {
		if own := MemberName(d, m.Pull); own == name || own == "#"+name {
			return m.Pull, true
		}
	}
	return 0, false
}

// StackMember is a pull request's own part of a stack.
func StackMember(stack *models.PageStack, pull int) models.StackMember {
	for _, m := range stack.Members {
		if m.Pull == pull {
			return m
		}
	}
	return models.StackMember{Pull: pull}
}

// MemberColor is a pull request's colour number in its stack, or 0.
func MemberColor(s models.State, pull int) int {
	for _, m := range append(append([]models.Member{}, s.Members...), s.Earlier...) {
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
func PackageTitle(page models.Page, component, id string) string {
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
func NodeOf(page models.Page, id string) (models.Node, bool) {
	for _, n := range page.Model.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return models.Node{}, false
}
