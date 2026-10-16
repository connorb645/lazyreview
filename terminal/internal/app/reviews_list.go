package app

import (
	"fmt"
	"strings"
	"time"

	"review-desk/terminal/internal/server"
)

// reviewsList is the list of every review the server keeps, newest load first.
type reviewsList struct {
	reviews []server.Summary
	view    scroller
	query   string
}

func (r reviewsList) build(width int, p palette, loc *time.Location) ([]line, []item) {
	b := &builder{width: width, p: p}
	if len(r.reviews) == 0 {
		b.text("", " "+p.Muted.Render("No reviews yet. Load one with review-desk load <pull request address>."))
		return b.lines, b.items
	}
	for _, s := range r.reviews {
		what := s.Branch + " against " + s.Base
		if s.Kind != "branch" {
			var pulls []string
			for _, m := range s.Members {
				pulls = append(pulls, fmt.Sprintf("#%d", m.Pull))
			}
			what = strings.Join(pulls, ", ")
		}
		meta := fmt.Sprintf("%s · %s · loaded %s", what, s.Repo, when(s.LoadedAt, loc))
		if s.Queued > 0 {
			meta += " · " + plural(s.Queued, "queued comment")
		}
		if s.Moved != nil {
			meta += " · moved to " + s.Moved.To
		}
		url := ""
		if len(s.Members) > 0 {
			url = s.Members[0].URL
		}
		b.add(item{kind: "review", section: s.ID, url: url}, " "+p.Bold.Render(s.Title), "   "+p.Muted.Render(meta))
		b.text("")
	}
	return b.lines, b.items
}

// match is the review after the cursor, or before it when back, whose title,
// repository, or pull requests and their titles hold the query, or -1.
func (r reviewsList) match(back bool) int {
	q := strings.ToLower(r.query)
	if q == "" {
		return -1
	}
	n := len(r.reviews)
	for step := 1; step <= n; step++ {
		i := (r.view.cursor + step) % n
		if back {
			i = ((r.view.cursor-step)%n + n) % n
		}
		s := r.reviews[i]
		text := strings.ToLower(s.Title + " " + s.Repo + " " + s.ID + " " + s.Branch)
		for _, m := range s.Members {
			text += fmt.Sprintf(" #%d %s", m.Pull, strings.ToLower(m.Title))
		}
		if strings.Contains(text, q) {
			return i
		}
	}
	return -1
}
