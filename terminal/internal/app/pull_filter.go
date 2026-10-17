package app

import (
	"fmt"
	"strings"
)

type pullFilter []int

func stackPulls(d reviewData) []int {
	if d.page.Stack == nil {
		return nil
	}
	var pulls []int
	for _, m := range d.page.Stack.Members {
		pulls = append(pulls, m.Pull)
	}
	return pulls
}

func pullRun(stack []int, from, to int) pullFilter {
	in := false
	var run pullFilter
	for _, pull := range stack {
		edge := pull == from || pull == to
		if edge || in {
			run = append(run, pull)
		}
		if edge && (in || from == to) {
			break
		}
		if edge {
			in = true
		}
	}
	return run
}

func (f pullFilter) shows(pull int) bool {
	return len(f) == 0 || containsInt(f, pull)
}

func (f pullFilter) showsAny(pulls []int) bool {
	for _, pull := range pulls {
		if containsInt(f, pull) {
			return true
		}
	}
	return len(f) == 0
}

func (f pullFilter) showsFile(d reviewData, path string) bool {
	for _, c := range d.page.Model.ContextFiles {
		if c.Path == path {
			return true
		}
	}
	return f.showsAny(fileEntry(d, path).Pulls)
}

func (f pullFilter) within(stack []int) pullFilter {
	var kept pullFilter
	for _, pull := range f {
		if containsInt(stack, pull) {
			kept = append(kept, pull)
		}
	}
	return kept
}

func (f pullFilter) words() string {
	names := make([]string, len(f))
	for i, pull := range f {
		names[i] = fmt.Sprintf("#%d", pull)
	}
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}
