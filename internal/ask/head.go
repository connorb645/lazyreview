package ask

import (
	"strconv"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/review"
)

// Head is the commit whose code Claude reads to answer, on a stack the
// member it heads, numbered as the stack numbers it, else 0, and where the
// lines asked about are in that code: the file's path, and its first and
// last lines, or no lines for removed ones.
type Head struct {
	Commit string
	Pull   int
	Path   string
	Lines  *[2]int
}

// HeadAsked is the head for a conversation: on a stack, of pull requests
// or of branches, the head of the member that changed its lines, or of the
// stack's top when none did; else the pull request's head, or a branch's
// commit as it was loaded.
func HeadAsked(tr *i18n.TranslationSet, rev models.Review, page models.PageRecord, c models.Conversation) Head {
	members := stackOf(rev)
	switch {
	case rev.IsBranch() && len(members) == 0:
		return placeShown(Head{Commit: rev.Head.Sha}, c)
	case len(members) == 1:
		return placeShown(Head{Commit: members[0].Head}, c)
	}
	if owner := ownerOf(tr, rev, page, c); owner != nil {
		return placeTraced(tr, Head{Commit: owner.Head, Pull: owner.Pull}, page.Stack.Lines, c)
	}
	top := members[len(members)-1]
	return placeShown(Head{Commit: top.Head, Pull: top.Pull}, c)
}

// stackOf is a review's members, bottom first: its pull requests, or a
// stack of branches' branches, each numbered by its place in the stack;
// none for one branch.
func stackOf(rev models.Review) []models.ReviewMember {
	if !rev.IsBranch() {
		return rev.Members
	}
	members := make([]models.ReviewMember, len(rev.Branches))
	for i, b := range rev.Branches {
		members[i] = models.ReviewMember{Pull: i + 1, Head: b.Head, Start: b.Start}
	}
	return members
}

// memberName is how Claude is told of a member of a review's stack: a pull
// request by its number, such as #12, or a branch by its name.
func memberName(rev models.Review, pull int) string {
	if rev.IsBranch() && pull >= 1 && pull <= len(rev.Branches) {
		return rev.Branches[pull-1].Branch
	}
	return "#" + strconv.Itoa(pull)
}

func ownerOf(tr *i18n.TranslationSet, rev models.Review, page models.PageRecord, c models.Conversation) *models.ReviewMember {
	members := stackOf(rev)
	switch len(members) {
	case 0:
		return nil
	case 1:
		return &members[0]
	}
	var lines map[string]map[string]models.LineMap
	if page.Stack != nil {
		lines = page.Stack.Lines
	}
	end := review.LineEntry(tr, lines, c.Path, c.Line, c.Side)
	for i := range members {
		if end.Pull != 0 && members[i].Pull == end.Pull {
			return &members[i]
		}
	}
	return nil
}

func placeShown(h Head, c models.Conversation) Head {
	h.Path = c.Path
	if c.Side == "RIGHT" {
		h.Lines = &[2]int{firstLine(c), c.Line}
	}
	return h
}

func placeTraced(tr *i18n.TranslationSet, h Head, lines map[string]map[string]models.LineMap, c models.Conversation) Head {
	end := review.LineEntry(tr, lines, c.Path, c.Line, c.Side)
	h.Path = end.Path
	if c.Side != "RIGHT" {
		return h
	}
	start := end
	if firstLine(c) != c.Line {
		start = review.LineEntry(tr, lines, c.Path, c.StartLine, "RIGHT")
	}
	from := end.Line
	if start.Pull == end.Pull && start.Path == end.Path {
		from = start.Line
	}
	h.Lines = &[2]int{from, end.Line}
	return h
}

func firstLine(c models.Conversation) int {
	side := c.StartSide
	if side == "" {
		side = c.Side
	}
	if c.StartLine != 0 && side == c.Side {
		return c.StartLine
	}
	return c.Line
}
