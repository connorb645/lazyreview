package loader

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/build/components"
	"github.com/connorb645/lazyreview/terminal/internal/build/page"
	"github.com/connorb645/lazyreview/terminal/internal/commands/git"
	"github.com/connorb645/lazyreview/terminal/internal/commands/github"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/queue"
	"github.com/connorb645/lazyreview/terminal/internal/review"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const (
	descriptionLimit = 2000
	cloneWait        = 15 * time.Minute
)

// Loader loads pull requests, or a stack of them, into reviews. A stack's
// review lives with its lowest pull request that has one, and the queues
// of other reviews of its pull requests move into it, those reviews then
// pointing at it. The review threads already on GitHub are kept as their
// own record, each on the line the app shows it on; when GitHub will not
// give them, the load goes on without them and a note says why. Each
// repository has one bare clone under the data home, which only one load
// at a time fetches into.
type Loader struct {
	tr     *i18n.TranslationSet
	store  *store.ReviewStore
	locks  *store.Locks
	clones *git.Clones
	github func(host, account string) *github.GitHub
	now    func() string
}

// NewLoader loads into store, fetching through clones and asking GitHub
// as gh gives it for a host and an account.
func NewLoader(tr *i18n.TranslationSet, reviews *store.ReviewStore, locks *store.Locks, clones *git.Clones, gh func(host, account string) *github.GitHub, now func() string) *Loader {
	return &Loader{tr: tr, store: reviews, locks: locks, clones: clones, github: gh, now: now}
}

// Load reads each pull request from GitHub, fetches their heads into the
// repository's clone, builds what the app draws, and writes the review.
// Loading again refreshes a review in place.
func (l *Loader) Load(ctx context.Context, r PullsRequest) (Loaded, error) {
	var loaded Loaded
	refs, err := l.readLinks(r.Links)
	if err != nil {
		return loaded, err
	}
	host, repo := refs[0].Host, refs[0].Repo
	pulls := make([]int, len(refs))
	for i, ref := range refs {
		pulls[i] = ref.Pull
	}
	anchor, absorbed, err := l.place(host, repo, pulls)
	if err != nil {
		return loaded, err
	}
	previous, err := l.store.Read(anchor)
	if err != nil {
		return loaded, err
	}
	account := r.Account
	if account == nil && previous != nil && previous.Account != nil && *previous.Account != "" {
		account = previous.Account
	}
	withContext := r.Context != nil && *r.Context
	if r.Context == nil && previous != nil {
		withContext = previous.Context
	}
	config := r.Config
	if config == nil {
		if config, err = l.store.ReadConfig(anchor); err != nil {
			return loaded, err
		}
	}
	checked, err := components.Read(l.tr, config)
	if err != nil {
		return loaded, err
	}
	gh := l.github(host, orNothing(account))

	var found []models.GitHubPull
	for _, pull := range pulls {
		f, err := gh.Pull(repo, pull)
		if err != nil {
			return loaded, err
		}
		found = append(found, f)
	}
	if len(found) > 1 {
		if err := l.checkChained(found, repo); err != nil {
			return loaded, err
		}
	}
	me, err := gh.Login()
	if err != nil {
		return loaded, err
	}
	var threads []models.GitHubThread
	var unread []string
	for _, f := range found {
		nodes, err := gh.ReviewThreads(repo, f.Number)
		if err != nil {
			unread = append(unread, utils.ResolvePlaceholderString(l.tr.LoaderThreadsUnread, map[string]string{"pull": strconv.Itoa(f.Number), "why": err.Error()}))
			continue
		}
		threads = append(threads, review.ReadThreads(f.Number, nodes)...)
	}
	leaving := map[int]models.ReviewMember{}
	if previous != nil && previous.Moved == nil {
		for _, m := range previous.Members {
			if !slices.Contains(pulls, m.Pull) {
				p, err := gh.Pull(repo, m.Pull)
				if err != nil {
					return loaded, err
				}
				leaving[m.Pull] = stateOf(p)
			}
		}
	}
	bottom := found[0]
	repository, err := gh.Repository(repo)
	if err != nil {
		return loaded, err
	}
	env, err := gh.Environment()
	if err != nil {
		return loaded, err
	}
	commits := append([]string{bottom.Base.Sha}, heads(found)...)
	clone, err := l.clone(ctx, host, repo, repository.CloneURL, bottom.Base.Ref, pulls, commits, env)
	if err != nil {
		return loaded, l.notFetched(ctx, host, repo, found, err)
	}
	members := make([]page.Member, len(found))
	for i, f := range found {
		members[i] = page.Member{Pull: f.Number, Head: f.Head.Sha}
	}
	title := bottom.Title
	if len(found) > 1 {
		title = utils.ResolvePlaceholderString(l.tr.LoaderStackTitle, map[string]string{"first": strconv.Itoa(pulls[0]), "last": strconv.Itoa(pulls[len(pulls)-1])})
	}
	built, err := page.Build(l.tr, git.NewRepository(l.tr, clone, nil), page.Request{
		Base: bottom.Base.Sha, Members: members, BaseBranch: bottom.Base.Ref, Title: title,
		Subject: page.PullsSubject(l.tr, pulls), Config: checked, Context: withContext, RootTitle: repo[strings.Index(repo, "/")+1:],
	})
	if err != nil {
		return loaded, err
	}
	fresh := make([]models.ReviewMember, len(found))
	for i, f := range found {
		description := []rune(f.Body)
		fresh[i] = stateOf(f)
		fresh[i].Pull, fresh[i].Head, fresh[i].Start, fresh[i].Title, fresh[i].URL, fresh[i].Author, fresh[i].Base = f.Number, f.Head.Sha, built.Starts[i], f.Title, f.HTMLURL, f.User.Login, f.Base.Ref
		fresh[i].Description = jsSlice(description, descriptionLimit)
	}

	changing := []models.ReviewRef{anchor}
	for _, a := range absorbed {
		changing = append(changing, a)
	}
	err = l.store.Change(changing, func() ([]store.FileWrite, error) {
		now := l.now()
		before, err := l.store.Read(anchor)
		if err != nil {
			return nil, err
		}
		var others []queue.Absorbed
		for _, a := range absorbed {
			other, err := l.store.Read(a)
			if err != nil {
				return nil, err
			}
			others = append(others, queue.Absorbed{Ref: a, Review: *other})
		}
		grouped, err := queue.Regroup(l.tr, queue.Regrouping{Previous: before, Fresh: fresh, Leaving: leaving, Absorbed: others, Anchor: anchor.ID(), Now: now})
		if err != nil {
			return nil, err
		}
		loaded.Ref = anchor
		loaded.Review = models.Review{
			ID: anchor.ID(), Host: host, Repo: repo, Account: account, Me: me, Title: built.Page.Model.Heading.Title,
			Base: models.Ref{Ref: bottom.Base.Ref, Sha: bottom.Base.Sha}, Context: withContext, LoadedAt: now,
			Notes: append(append([]string{}, built.Notes...), unread...), Members: grouped.Members, Earlier: grouped.Earlier,
		}
		loaded.Counts = built.Counts
		loaded.Notes = append(append([]string{}, loaded.Review.Notes...), grouped.Notes...)
		folder := l.store.Home().Review(anchor)
		pageText, err := compact(built.Page)
		if err != nil {
			return nil, err
		}
		threadsText, err := indented(review.PlaceThreads(orNone(threads), built.Page.Stack))
		if err != nil {
			return nil, err
		}
		record, err := l.store.RecordWrite(anchor, loaded.Review)
		if err != nil {
			return nil, err
		}
		files := []store.FileWrite{{Path: folder.Page(), Text: pageText}, {Path: folder.GitHub(), Text: threadsText}, record}
		if config != nil {
			text, err := indentedText(config)
			if err != nil {
				return nil, err
			}
			files = append(files, store.FileWrite{Path: folder.Config(), Text: text})
		}
		for _, m := range grouped.Moved {
			write, err := l.store.RecordWrite(m.Ref, m.Review)
			if err != nil {
				return nil, err
			}
			files = append(files, write)
		}
		return files, nil
	})
	return loaded, err
}

// Refresh loads a review again from GitHub, as its pull requests now
// stand. A stack's members that merged or closed leave it, unless none is
// open.
func (l *Loader) Refresh(ctx context.Context, ref models.PullRef) (Loaded, error) {
	rev, err := l.store.Read(ref)
	if err != nil {
		return Loaded{}, err
	}
	if rev == nil {
		return Loaded{}, models.Refused(utils.ResolvePlaceholderString(l.tr.ReviewNone, map[string]string{"id": ref.ID()}))
	}
	if rev.Moved != nil {
		return Loaded{}, models.Refused(utils.ResolvePlaceholderString(l.tr.LoaderMovedLoadThat, map[string]string{"to": rev.Moved.To}))
	}
	gh := l.github(rev.Host, orNothing(rev.Account))
	var open []models.ReviewMember
	for _, m := range rev.Members {
		p, err := gh.Pull(rev.Repo, m.Pull)
		if err != nil {
			return Loaded{}, err
		}
		if stateOf(p).State == "open" {
			open = append(open, m)
		}
	}
	keep := rev.Members
	if len(rev.Members) > 1 && len(open) > 0 {
		keep = open
	}
	links := make([]string, len(keep))
	for i, m := range keep {
		links[i] = ref.WithPull(m.Pull).ID()
	}
	return l.Load(ctx, PullsRequest{Links: links})
}

func (l *Loader) clone(ctx context.Context, host, repo, url, base string, pulls []int, commits, env []string) (string, error) {
	path := l.store.Home().Clone(host, repo)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	var prepared string
	err := l.locks.With([]string{path + ".lock"}, cloneWait, func() error {
		var err error
		prepared, err = l.clones.Prepare(ctx, path, url, base, pulls, commits, env)
		return err
	})
	return prepared, err
}

func (l *Loader) place(host, repo string, pulls []int) (models.PullRef, []models.PullRef, error) {
	all, err := l.store.InRepository(host, repo)
	if err != nil {
		return models.PullRef{}, nil, err
	}
	var holding []store.HeldReview
	for _, e := range all {
		if e.Review.Moved == nil && slices.ContainsFunc(e.Review.Members, func(m models.ReviewMember) bool { return slices.Contains(pulls, m.Pull) }) {
			holding = append(holding, e)
		}
	}
	anchor := models.PullRef{Host: host, Repo: repo, Pull: pulls[0]}
	found := false
	for _, p := range pulls {
		for _, e := range holding {
			if slices.ContainsFunc(e.Review.Members, func(m models.ReviewMember) bool { return m.Pull == p }) {
				anchor, found = e.Ref, true
				break
			}
		}
		if found {
			break
		}
	}
	var absorbed []models.PullRef
	for _, e := range holding {
		if e.Ref.ID() == anchor.ID() {
			continue
		}
		var others []string
		for _, m := range e.Review.Members {
			if !slices.Contains(pulls, m.Pull) {
				others = append(others, "#"+strconv.Itoa(m.Pull))
			}
		}
		if len(others) > 0 {
			return anchor, nil, models.Refused(utils.ResolvePlaceholderString(l.tr.LoaderAlsoReviews, map[string]string{"id": e.Ref.ID(), "pulls": strings.Join(others, ", ")}))
		}
		absorbed = append(absorbed, e.Ref)
	}
	return anchor, absorbed, nil
}

func (l *Loader) notFetched(ctx context.Context, host, repo string, found []models.GitHubPull, err error) error {
	missing := l.clones.Missing(ctx, l.store.Home().Clone(host, repo), append([]string{found[0].Base.Sha}, heads(found)...))
	var what []string
	for _, commit := range missing {
		if commit == found[0].Base.Sha {
			what = append(what, utils.ResolvePlaceholderString(l.tr.LoaderBaseAt, map[string]string{"base": found[0].Base.Ref, "sha": utils.Short(commit)}))
		}
		for _, f := range found {
			if f.Head.Sha == commit {
				what = append(what, utils.ResolvePlaceholderString(l.tr.LoaderPullHead, map[string]string{"pull": strconv.Itoa(f.Number), "head": utils.Short(commit)}))
			}
		}
	}
	if len(what) == 0 {
		what = []string{repo}
	}
	return models.Refused(utils.ResolvePlaceholderString(l.tr.LoaderNotFetched, map[string]string{"what": strings.Join(what, ", "), "repo": repo, "why": err.Error()}))
}

func heads(found []models.GitHubPull) []string {
	out := make([]string, len(found))
	for i, f := range found {
		out[i] = f.Head.Sha
	}
	return out
}

func (l *Loader) readLinks(links []string) ([]models.PullRef, error) {
	if len(links) == 0 {
		return nil, models.Refused(l.tr.LoaderGiveLinks)
	}
	refs := make([]models.PullRef, len(links))
	for i, link := range links {
		ref, err := models.ParsePullRef(l.tr, link)
		if err != nil {
			return nil, err
		}
		refs[i] = ref
	}
	seen := map[int]bool{}
	for _, r := range refs {
		if r.Host != refs[0].Host || r.Repo != refs[0].Repo {
			return nil, models.Refused(utils.ResolvePlaceholderString(l.tr.LoaderOtherRepository, map[string]string{"other": r.Host + "/" + r.Repo, "first": refs[0].Host + "/" + refs[0].Repo}))
		}
	}
	for _, r := range refs {
		if seen[r.Pull] {
			return nil, models.Refused(l.tr.LoaderNamedTwice)
		}
		seen[r.Pull] = true
	}
	return refs, nil
}

func (l *Loader) checkChained(found []models.GitHubPull, repo string) error {
	var problems []string
	for i := 1; i < len(found); i++ {
		below, above := found[i-1], found[i]
		if above.Base.Ref != below.Head.Ref || below.Head.Repo == nil || below.Head.Repo.FullName != repo {
			problems = append(problems, utils.ResolvePlaceholderString(l.tr.LoaderNotOnBelow, map[string]string{
				"above": strconv.Itoa(above.Number), "base": above.Base.Ref, "below": strconv.Itoa(below.Number), "head": below.Head.Ref,
			}))
		}
	}
	if len(problems) > 0 {
		return models.Refused(utils.ResolvePlaceholderString(l.tr.LoaderNotAStack, map[string]string{"why": strings.Join(problems, "; ")}))
	}
	return nil
}

func stateOf(p models.GitHubPull) models.ReviewMember {
	if p.MergedAt != nil {
		return models.ReviewMember{State: "merged", EndedAt: p.MergedAt}
	}
	return models.ReviewMember{State: p.State, EndedAt: p.ClosedAt}
}

func orNothing(account *string) string {
	if account == nil {
		return ""
	}
	return *account
}

func orNone[T any](list []T) []T {
	if list == nil {
		return []T{}
	}
	return list
}

func jsSlice(text []rune, limit int) string {
	units := 0
	for i, r := range text {
		width := 1
		if r > 0xFFFF {
			width = 2
		}
		if units+width > limit {
			return string(text[:i])
		}
		units += width
	}
	return string(text)
}

func compact(v any) (string, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(v)
	return strings.TrimSuffix(out.String(), "\n"), err
}

func indented(v any) (string, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(v)
	return out.String(), err
}

func indentedText(raw []byte) (string, error) {
	var out bytes.Buffer
	if err := json.Indent(&out, bytes.TrimSpace(raw), "", "  "); err != nil {
		return "", err
	}
	return out.String() + "\n", nil
}
