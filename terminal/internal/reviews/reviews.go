// Package reviews is what the terminal app reads and changes reviews
// through. It reads a review's state, its page and each file from the data
// home, keeps its queue and its conversations with Claude, itself; the
// server still loads reviews and lists the open pull requests.
package reviews

import (
	"context"
	"errors"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/ask"
	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/queue"
	"github.com/connorb645/lazyreview/terminal/internal/review"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const watchEvery = 250 * time.Millisecond

// Reviews reads and changes the reviews of one data home.
type Reviews struct {
	tr     *i18n.TranslationSet
	store  *store.ReviewStore
	queue  *queue.Service
	asking *ask.Service
	server *server.Client
}

// New returns the reviews kept in store, whose queues queue changes and
// whose conversations asking keeps, asking the server for what it still
// does.
func New(tr *i18n.TranslationSet, reviews *store.ReviewStore, q *queue.Service, asking *ask.Service, s *server.Client) *Reviews {
	return &Reviews{tr: tr, store: reviews, queue: q, asking: asking, server: s}
}

// State reads a review's state: its pull requests, its queue, its
// conversations, the threads on GitHub and its written review.
func (r *Reviews) State(id string) (models.State, error) {
	ref, err := models.ParseReviewRef(r.tr, id)
	if err != nil {
		return models.State{}, refusal(err)
	}
	state, err := State(r.tr, r.store, ref)
	if err != nil {
		return models.State{}, refusal(err)
	}
	if state == nil {
		return models.State{}, models.Refused(r.tr.NoSuchReview)
	}
	return *state, nil
}

// Page reads what a review draws: its build, its model and its layout.
func (r *Reviews) Page(id string) (models.Page, error) {
	read, err := r.page(id)
	if err != nil {
		return models.Page{}, err
	}
	return review.PageView(read.Build, read.Page), nil
}

// File reads one file of a review ready to draw, for the page build the app
// drew from: its hunks of rows, the unchanged runs between them and its
// whole text.
func (r *Reviews) File(id, path, build string) (models.File, error) {
	read, err := r.page(id)
	if err != nil {
		return models.File{}, err
	}
	if problem := store.BuildProblem(r.tr, read.Build, build); problem != "" {
		return models.File{}, models.Refused(utils.ResolvePlaceholderString(r.tr.NoFile, map[string]string{"why": problem}))
	}
	file, ok := review.FileView(r.tr, read.Page, path)
	if !ok {
		return models.File{}, models.Refused(utils.ResolvePlaceholderString(r.tr.NotAFileOfReview, map[string]string{"path": path}))
	}
	return file, nil
}

// Where says where a comment on lines as the app shows them would go, for
// the page build the app drew from; pull names one pull request of a stack
// where several can take it, or is 0.
func (r *Reviews) Where(id string, at models.Lines, pull int, build string) (models.Where, error) {
	ref, err := models.ParseReviewRef(r.tr, id)
	if err != nil {
		return models.Where{}, refusal(err)
	}
	where, err := r.queue.Where(ref, at, pull, build)
	return where, refusal(err)
}

// Queue queues a comment on lines as the app shows them, and returns the
// state as it then stands.
func (r *Reviews) Queue(id string, at models.Lines, pull int, body, build string) (models.State, error) {
	return r.changed(id, func(ref models.ReviewRef) error { return r.queue.Add(ref, at, pull, body, build) })
}

// Edit replaces a queued comment's text, naming the version it was shown
// at, and returns the state as it then stands.
func (r *Reviews) Edit(id string, comment int, body, version, build string) (models.State, error) {
	return r.changed(id, func(ref models.ReviewRef) error { return r.queue.Edit(ref, comment, body, version, build) })
}

// Drop drops a queued comment, naming the version it was shown at, and
// returns the state as it then stands.
func (r *Reviews) Drop(id string, comment int, version, build string) (models.State, error) {
	return r.changed(id, func(ref models.ReviewRef) error { return r.queue.Drop(ref, comment, version, build) })
}

// Post sends one action offered for a pull request, naming the set of
// comments it was shown with, and returns the words for what it did and
// the state as it then stands.
func (r *Reviews) Post(id string, pull int, action, set, build string) (string, models.State, error) {
	var posting queue.Posting
	state, err := r.changed(id, func(ref models.ReviewRef) error {
		var err error
		posting, err = r.queue.Post(ref, pull, action, set, build)
		return err
	})
	return posting.Message, state, err
}

// Ask starts a conversation with Claude on lines as the app shows them, and
// returns it once Claude has answered, or said why not.
func (r *Reviews) Ask(id string, at models.Lines, text, build string) (models.Conversation, error) {
	return r.conversed(id, func(ref models.ReviewRef) (models.Conversation, error) { return r.asking.Start(ref, at, text, build) })
}

// Reply asks a follow-up in a conversation, and returns it once Claude has
// answered, or said why not.
func (r *Reviews) Reply(id string, conversation int, text, build string) (models.Conversation, error) {
	return r.conversed(id, func(ref models.ReviewRef) (models.Conversation, error) {
		return r.asking.Reply(ref, conversation, text, build)
	})
}

// Stop stops Claude answering in a conversation, and returns it once it
// says so.
func (r *Reviews) Stop(id string, conversation int) (models.Conversation, error) {
	return r.conversed(id, func(ref models.ReviewRef) (models.Conversation, error) { return r.asking.Stop(ref, conversation) })
}

// Refresh loads a review again: a pull request's from GitHub, or a branch's
// from its repository, and returns the notes on what it found.
func (r *Reviews) Refresh(id string) ([]string, error) {
	return r.server.Refresh(id)
}

// Pulls reads the open pull requests that involve the accounts in the
// config, as GitHub has them now.
func (r *Reviews) Pulls() (models.Pulls, error) {
	return r.server.Pulls()
}

// ChangeHidden puts an entry on the config's hidden repositories, or takes
// entries off it, and returns what now hides each of repos.
func (r *Reviews) ChangeHidden(hide string, show, repos []string) (models.Hidden, error) {
	return r.server.ChangeHidden(hide, show, repos)
}

// Load loads pull requests from GitHub into a review, a stack bottom first,
// as the GitHub account named, and returns the review's id and the notes on
// what it found.
func (r *Reviews) Load(links []string, account string) (string, []string, error) {
	return r.server.Load(links, account)
}

// Events sends each change notice of a review to notices, the first naming
// both the state's version and the page's build, then each naming only what
// changed, until ctx ends. It watches the review's folder, so it hears a
// change made by this app, by another or by the command alike.
func (r *Reviews) Events(ctx context.Context, id string, notices chan<- models.Notice) error {
	ref, err := models.ParseReviewRef(r.tr, id)
	if err != nil {
		return refusal(err)
	}
	ticker := time.NewTicker(watchEvery)
	defer ticker.Stop()
	var last models.Notice
	seen := ""
	for first := true; ; first = false {
		if mark := r.store.Mark(ref); first || mark != seen {
			seen = mark
			now, err := r.notice(ref)
			if err != nil {
				return err
			}
			notice := now
			if !first {
				notice = models.Notice{}
				if now.State != last.State {
					notice.State = now.State
				}
				if now.Build != last.Build {
					notice.Build = now.Build
				}
			}
			last = now
			if notice != (models.Notice{}) {
				select {
				case notices <- notice:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (r *Reviews) notice(ref models.ReviewRef) (models.Notice, error) {
	state, err := State(r.tr, r.store, ref)
	if err != nil {
		return models.Notice{}, err
	}
	build, err := r.store.PageBuild(ref)
	if err != nil {
		return models.Notice{}, err
	}
	notice := models.Notice{Build: build}
	if state != nil {
		notice.State = state.Version
	}
	return notice, nil
}

func (r *Reviews) page(id string) (*store.PageRead, error) {
	ref, err := models.ParseReviewRef(r.tr, id)
	if err != nil {
		return nil, refusal(err)
	}
	read, err := r.store.ReadPage(ref)
	if err != nil {
		return nil, refusal(err)
	}
	if read == nil {
		return nil, models.Refused(r.tr.NoSuchReview)
	}
	return read, nil
}

func (r *Reviews) changed(id string, change func(models.ReviewRef) error) (models.State, error) {
	ref, err := models.ParseReviewRef(r.tr, id)
	if err != nil {
		return models.State{}, refusal(err)
	}
	if err := change(ref); err != nil {
		return models.State{}, refusal(err)
	}
	return r.State(id)
}

func (r *Reviews) conversed(id string, change func(models.ReviewRef) (models.Conversation, error)) (models.Conversation, error) {
	ref, err := models.ParseReviewRef(r.tr, id)
	if err != nil {
		return models.Conversation{}, refusal(err)
	}
	c, err := change(ref)
	return c, refusal(err)
}

// refusal is err as the app's refusal: every way a review cannot be read or
// changed is told in the same words to the person, as the server told it.
func refusal(err error) error {
	if err == nil {
		return nil
	}
	var refused models.Refusal
	if errors.As(err, &refused) {
		return refused
	}
	return models.Refused(err.Error())
}
