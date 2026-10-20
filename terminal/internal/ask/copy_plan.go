package ask

import (
	"strconv"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
)

// CopyPlan is what the copy of the code for a conversation holds: a branch
// for each name, at its commit, the one checked out, at the head's commit,
// and the base's.
type CopyPlan struct {
	Branches map[string]string
	Branch   string
	Base     string
}

// CopyBranches is the plan for a conversation's copy: the base branch, and
// each pull request's head as pull/<number>, or the branch.
func CopyBranches(tr *i18n.TranslationSet, rev models.Review, page models.PageRecord, c models.Conversation) CopyPlan {
	if rev.IsBranch() {
		return CopyPlan{Branches: map[string]string{rev.Base.Ref: rev.Base.Sha, rev.Head.Ref: rev.Head.Sha}, Branch: rev.Head.Ref, Base: rev.Base.Ref}
	}
	head := HeadAsked(tr, rev, page, c)
	branches := map[string]string{rev.Base.Ref: rev.Base.Sha}
	for _, m := range rev.Members {
		branches["pull/"+strconv.Itoa(m.Pull)] = m.Head
	}
	pull := head.Pull
	if pull == 0 {
		pull = rev.Members[0].Pull
	}
	return CopyPlan{Branches: branches, Branch: "pull/" + strconv.Itoa(pull), Base: rev.Base.Ref}
}
