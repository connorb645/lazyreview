package ask

import (
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/review"
)

// Head is the commit whose code Claude reads to answer, on a stack the pull
// request it heads, else 0, and where the lines asked about are in that
// code: the file's path, and its first and last lines, or no lines for
// removed ones.
type Head struct {
	Commit string
	Pull   int
	Path   string
	Lines  *[2]int
}

// HeadAsked is the head for a conversation: the head of the pull request
// that changed its lines, or of the stack's top when none did, or a
// branch's commit as it was loaded.
func HeadAsked(tr *i18n.TranslationSet, rev models.Review, page models.PageRecord, c models.Conversation) Head {
	switch {
	case rev.IsBranch():
		return placeShown(Head{Commit: rev.Head.Sha}, c)
	case len(rev.Members) == 1:
		return placeShown(Head{Commit: rev.Members[0].Head}, c)
	}
	if owner := ownerOf(tr, rev, page, c); owner != nil {
		return placeTraced(tr, Head{Commit: owner.Head, Pull: owner.Pull}, page.Stack.Lines, c)
	}
	top := rev.Members[len(rev.Members)-1]
	return placeShown(Head{Commit: top.Head, Pull: top.Pull}, c)
}

func ownerOf(tr *i18n.TranslationSet, rev models.Review, page models.PageRecord, c models.Conversation) *models.ReviewMember {
	if rev.IsBranch() {
		return nil
	}
	if len(rev.Members) == 1 {
		return &rev.Members[0]
	}
	var lines map[string]map[string]models.LineMap
	if page.Stack != nil {
		lines = page.Stack.Lines
	}
	end := review.LineEntry(tr, lines, c.Path, c.Line, c.Side)
	for i := range rev.Members {
		if end.Pull != 0 && rev.Members[i].Pull == end.Pull {
			return &rev.Members[i]
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
