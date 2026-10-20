package reviews

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/queue"
	"github.com/connorb645/lazyreview/internal/review"
	"github.com/connorb645/lazyreview/internal/store"
)

// State is what the app shows of one review apart from its page: its pull
// requests, its queue as the buttons need it, its conversations, the review
// threads on GitHub and its written review, or nil when there is no review
// ref. A written review is marked stale when it read another head than the
// review now shows, and each of its sections carries its file:line links.
// A branch review gives its branch, its base and its conversations, with no
// queue: it takes no comments and sends nothing. Each state carries its
// version, a short hash of the rest of it, which the change notices name.
func State(tr *i18n.TranslationSet, reviews *store.ReviewStore, ref models.ReviewRef) (*models.State, error) {
	rev, err := reviews.Read(ref)
	if err != nil || rev == nil {
		return nil, err
	}
	threads, err := reviews.Threads(ref)
	if err != nil {
		return nil, err
	}
	state := models.State{ID: rev.ID, Kind: rev.Kind, Repo: rev.Repo, Title: rev.Title, Base: rev.Base, Head: rev.Head, LoadedAt: rev.LoadedAt,
		Notes: orEmpty(rev.Notes), Threads: threads, GitHub: []models.GitHubThread{}}
	if rev.IsBranch() {
		return versioned(state), nil
	}
	state.Host, state.Me, state.Account, state.Moved = rev.Host, rev.Me, rev.Account, rev.Moved
	state.Members, state.Earlier = summaries(rev.Members), summaries(rev.Earlier)
	q := queue.View(tr, *rev)
	state.Queue = &q
	if state.GitHub, err = reviews.GitHubThreads(ref); err != nil {
		return nil, err
	}
	text, found, err := reviews.ReadWritten(ref)
	if err != nil {
		return nil, err
	}
	if found {
		if state.Written, err = written(tr, reviews, ref, *rev, text); err != nil {
			return nil, err
		}
	}
	return versioned(state), nil
}

func written(tr *i18n.TranslationSet, reviews *store.ReviewStore, ref models.ReviewRef, rev models.Review, text string) (*models.Written, error) {
	read, err := review.ReadWrittenReview(tr, text)
	if err != nil {
		return nil, err
	}
	drawn, err := reviews.ReadPage(ref)
	if err != nil {
		return nil, err
	}
	var paths []string
	if drawn != nil {
		for _, f := range append(append([]models.FileEntry{}, drawn.Page.Model.Files...), drawn.Page.Model.ContextFiles...) {
			paths = append(paths, f.Path)
		}
	}
	head := ""
	if len(rev.Members) > 0 {
		head = rev.Members[len(rev.Members)-1].Head
	}
	w := models.Written{Title: read.Title, Head: read.Head, Stale: !strings.HasPrefix(head, read.Head)}
	for _, s := range read.Sections {
		s.Links = review.WrittenLinks(s.Markdown, paths)
		w.Sections = append(w.Sections, s)
	}
	return &w, nil
}

func summaries(members []models.ReviewMember) []models.Member {
	out := []models.Member{}
	for _, m := range members {
		out = append(out, models.Member{Pull: m.Pull, Head: m.Head, Title: m.Title, URL: m.URL, Author: m.Author, Base: m.Base, State: m.State, EndedAt: m.EndedAt, Color: m.Color})
	}
	return out
}

func versioned(state models.State) *models.State {
	state.Version = ""
	text, _ := json.Marshal(state)
	sum := sha256.Sum256(text)
	state.Version = hex.EncodeToString(sum[:])[:12]
	return &state
}

func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
