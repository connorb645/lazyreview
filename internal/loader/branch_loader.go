package loader

import (
	"path/filepath"
	"slices"

	"github.com/connorb645/lazyreview/internal/build/components"
	"github.com/connorb645/lazyreview/internal/build/page"
	"github.com/connorb645/lazyreview/internal/commands/git"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/queue"
	"github.com/connorb645/lazyreview/internal/store"
	"github.com/connorb645/lazyreview/internal/utils"
)

// BranchLoader loads a branch of a repository on this machine into a
// review, for a change that has no pull request yet, such as a branch
// about to be pushed, or a stack of such branches. The review is kept by
// repository and branch, a stack's by its bottom branch, so loading the
// same branch again, from any worktree of the repository, updates it in
// place, and a branch grows into a stack of it and the branches above at
// the same address. A branch review is read only: it has no queue and
// posts nothing.
type BranchLoader struct {
	tr    *i18n.TranslationSet
	store *store.ReviewStore
	now   func() string
}

// NewBranchLoader loads into store.
func NewBranchLoader(tr *i18n.TranslationSet, reviews *store.ReviewStore, now func() string) *BranchLoader {
	return &BranchLoader{tr: tr, store: reviews, now: now}
}

// Load reads the change from the merge base of the base and the top head
// to that head, as a pull request would show it, straight from the
// repository, builds what the app draws, and writes the review. Several
// heads are a stack: each branch's own diff is read from the branch below
// it, the bottom's from the base, and each branch keeps its colour from one
// load to the next.
func (l *BranchLoader) Load(r BranchRequest) (Loaded, error) {
	var loaded Loaded
	path, err := filepath.Abs(r.Repo)
	if err != nil {
		return loaded, err
	}
	repo := git.NewRepository(l.tr, path, nil)
	folder, err := repo.SharedFolder()
	if err != nil {
		return loaded, err
	}
	shared, err := filepath.EvalSymlinks(folder)
	if err != nil {
		return loaded, err
	}
	branches, err := l.branchesOf(repo, path, r.Heads)
	if err != nil {
		return loaded, err
	}
	ref, err := models.BranchRefOf(l.tr, shared, branches[0])
	if err != nil {
		return loaded, err
	}
	previous, err := l.store.Read(ref)
	if err != nil {
		return loaded, err
	}
	title := r.Title
	if title == nil && previous != nil {
		title = previous.GivenTitle
	}
	withContext := r.Context != nil && *r.Context
	if r.Context == nil && previous != nil {
		withContext = previous.Context
	}
	config := r.Config
	if config == nil {
		if config, err = l.store.ReadConfig(ref); err != nil {
			return loaded, err
		}
	}
	checked, err := components.Read(l.tr, config)
	if err != nil {
		return loaded, err
	}
	stacked := len(branches) > 1
	shown := branches[0]
	if stacked {
		shown = utils.ResolvePlaceholderString(l.tr.LoaderBranchesTitle, map[string]string{"first": branches[0], "last": branches[len(branches)-1]})
	}
	if title != nil && *title != "" {
		shown = *title
	}
	members := make([]page.Member, len(branches))
	for i, branch := range branches {
		members[i] = page.Member{Head: r.Heads[i]}
		if stacked {
			members[i].Pull, members[i].Branch = i+1, branch
		}
	}
	name := models.RepositoryName(shared)
	built, err := page.Build(l.tr, repo, page.Request{
		Base: r.Base, Members: members, BaseBranch: r.Base, Title: shown,
		Subject: page.BranchSubject(l.tr, branches), Config: checked, Context: withContext, RootTitle: name,
	})
	if err != nil {
		return loaded, err
	}
	err = l.store.Change([]models.ReviewRef{ref}, func() ([]store.FileWrite, error) {
		base, err := repo.Resolve(r.Base)
		if err != nil {
			return nil, err
		}
		before, err := l.store.Read(ref)
		if err != nil {
			return nil, err
		}
		loaded.Ref, loaded.Counts, loaded.Notes = ref, built.Counts, orNone(built.Notes)
		loaded.Review = models.Review{
			ID: ref.ID(), Kind: "branch", Repo: name, Path: path, Title: built.Page.Model.Heading.Title, GivenTitle: title,
			Base: models.Ref{Ref: r.Base, Sha: base}, Head: models.Ref{Ref: branches[len(branches)-1], Sha: built.Page.Model.Heading.Head}, Start: built.Page.Model.Heading.Base,
			Context: withContext, LoadedAt: l.now(), Notes: loaded.Notes,
		}
		if stacked {
			loaded.Review.Branches = stackOf(before, branches, built)
		}
		folder := l.store.Home().Review(ref)
		pageText, err := compact(built.Page)
		if err != nil {
			return nil, err
		}
		record, err := l.store.RecordWrite(ref, loaded.Review)
		if err != nil {
			return nil, err
		}
		files := []store.FileWrite{{Path: folder.Page(), Text: pageText}, record}
		if config != nil {
			text, err := indentedText(config)
			if err != nil {
				return nil, err
			}
			files = append(files, store.FileWrite{Path: folder.Config(), Text: text})
		}
		return files, nil
	})
	return loaded, err
}

// Refresh loads a branch review again, from the folder, the base and the
// branches it was last loaded with.
func (l *BranchLoader) Refresh(ref models.BranchRef) (Loaded, error) {
	rev, err := l.store.Read(ref)
	if err != nil {
		return Loaded{}, err
	}
	if rev == nil {
		return Loaded{}, models.Refused(utils.ResolvePlaceholderString(l.tr.ReviewNone, map[string]string{"id": ref.ID()}))
	}
	return l.Load(BranchRequest{Repo: rev.Path, Base: rev.Base.Ref, Heads: rev.BranchNames()})
}

func (l *BranchLoader) branchesOf(repo *git.Repository, path string, heads []string) ([]string, error) {
	if len(heads) == 0 {
		return nil, models.Refused(l.tr.LoaderGiveHead)
	}
	var branches []string
	for _, head := range heads {
		branch, isBranch, err := repo.BranchOf(head)
		if err != nil {
			return nil, err
		}
		if !isBranch {
			return nil, models.Refused(utils.ResolvePlaceholderString(l.tr.LoaderNotABranch, map[string]string{"head": head, "path": path}))
		}
		if slices.Contains(branches, branch) {
			return nil, models.Refused(utils.ResolvePlaceholderString(l.tr.LoaderBranchTwice, map[string]string{"branch": branch}))
		}
		branches = append(branches, branch)
	}
	return branches, nil
}

func stackOf(before *models.Review, branches []string, built page.Built) []models.BranchMember {
	had := map[string]int{}
	if before != nil {
		for _, b := range before.Branches {
			had[b.Branch] = b.Color
		}
	}
	given := make([]int, len(branches))
	for i, branch := range branches {
		given[i] = had[branch]
	}
	colours := queue.Coloured(given)
	out := make([]models.BranchMember, len(branches))
	for i, branch := range branches {
		out[i] = models.BranchMember{Branch: branch, Head: built.Page.Stack.Members[i].Head, Start: built.Starts[i], Color: colours[i]}
	}
	return out
}
