package loader

import (
	"path/filepath"

	"github.com/connorb645/lazyreview/terminal/internal/build/components"
	"github.com/connorb645/lazyreview/terminal/internal/build/page"
	"github.com/connorb645/lazyreview/terminal/internal/commands/git"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// BranchLoader loads a branch of a repository on this machine into a
// review, for a change that has no pull request yet, such as a branch
// about to be pushed. The review is kept by repository and branch, so
// loading the same branch again, from any worktree of the repository,
// updates it in place. A branch review is read only: it has no queue and
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

// Load reads the change from the merge base of the base and the head to
// the head, as a pull request would show it, straight from the repository,
// builds what the app draws, and writes the review.
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
	branch, isBranch, err := repo.BranchOf(r.Head)
	if err != nil {
		return loaded, err
	}
	if !isBranch {
		return loaded, models.Refused(utils.ResolvePlaceholderString(l.tr.LoaderNotABranch, map[string]string{"head": r.Head, "path": path}))
	}
	ref, err := models.BranchRefOf(l.tr, shared, branch)
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
	shown := branch
	if title != nil && *title != "" {
		shown = *title
	}
	name := models.RepositoryName(shared)
	built, err := page.Build(l.tr, repo, page.Request{
		Base: r.Base, Members: []page.Member{{Head: r.Head}}, BaseBranch: r.Base, Title: shown,
		Subject: page.BranchSubject(l.tr, branch), Config: checked, Context: withContext, RootTitle: name,
	})
	if err != nil {
		return loaded, err
	}
	err = l.store.Change([]models.ReviewRef{ref}, func() ([]store.FileWrite, error) {
		base, err := repo.Resolve(r.Base)
		if err != nil {
			return nil, err
		}
		loaded.Ref, loaded.Counts, loaded.Notes = ref, built.Counts, orNone(built.Notes)
		loaded.Review = models.Review{
			ID: ref.ID(), Kind: "branch", Repo: name, Path: path, Title: built.Page.Model.Heading.Title, GivenTitle: title,
			Base: models.Ref{Ref: r.Base, Sha: base}, Head: models.Ref{Ref: branch, Sha: built.Page.Model.Heading.Head}, Start: built.Starts[0],
			Context: withContext, LoadedAt: l.now(), Notes: loaded.Notes,
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

// Refresh loads a branch review again, from the folder and the base it was
// last loaded with.
func (l *BranchLoader) Refresh(ref models.BranchRef) (Loaded, error) {
	rev, err := l.store.Read(ref)
	if err != nil {
		return Loaded{}, err
	}
	if rev == nil {
		return Loaded{}, models.Refused(utils.ResolvePlaceholderString(l.tr.ReviewNone, map[string]string{"id": ref.ID()}))
	}
	return l.Load(BranchRequest{Repo: rev.Path, Base: rev.Base.Ref, Head: rev.Head.Ref})
}
