package ask

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/commands/claude"
	"github.com/connorb645/lazyreview/terminal/internal/commands/git"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const (
	unused     = 24 * time.Hour
	copyWait   = 10 * time.Minute
	keptSteps  = 200
	writeEvery = 400 * time.Millisecond
	stopWait   = 10 * time.Second
	stopPoll   = 50 * time.Millisecond
	stopFile   = "stop"
)

// Service keeps conversations with Claude on the lines of a review. Each
// question is kept at once, then Claude answers it, and the answer, or why
// there is none, is kept with the steps Claude took.
//
// Claude works as the user's own Claude Code, under the guard, in a
// throwaway copy of the code at the commit the lines belong to: one copy per
// conversation, made the first time from the review's own clone, or for a
// branch review from a clone lazyreview fetches the branch into from its
// repository. Copies no question used for a day are removed. Each
// conversation keeps one Claude Code session, so a follow-up goes on from
// what Claude read and found. While Claude works, the conversation holds
// the steps so far as its working steps.
type Service struct {
	tr       *i18n.TranslationSet
	store    *store.ReviewStore
	locks    *store.Locks
	claude   *claude.Claude
	copies   CodeCopies
	clones   *git.Clones
	answerer Answerer
	hook     func(folder string) string
	session  func() string
	now      func() string
}

// NewService keeps the conversations of the reviews in reviews, asking asker,
// copying the code through copies and clones, and having answerer answer
// each question; with no answerer this process answers it. A question and
// an answer are stamped with now.
func NewService(tr *i18n.TranslationSet, reviews *store.ReviewStore, locks *store.Locks, asker *claude.Claude, copies CodeCopies, clones *git.Clones, answerer Answerer, now func() string) *Service {
	s := &Service{tr: tr, store: reviews, locks: locks, claude: asker, copies: copies, clones: clones, answerer: answerer, hook: guardCommand, session: newSession, now: now}
	if s.copies == nil {
		s.copies = git.NewCopies(tr, nil, nil)
	}
	if s.clones == nil {
		s.clones = git.NewClones(tr, nil)
	}
	if s.answerer == nil {
		s.answerer = func(ref models.ReviewRef, id int) error { return s.Answer(context.Background(), ref, id) }
	}
	return s
}

// Start opens a conversation on lines as the app shows them, with the
// question text, and returns it once Claude answered. build is the build id
// of the page the window was drawn from; the question is refused once the
// review was loaded again since.
func (s *Service) Start(ref models.ReviewRef, at models.Lines, text, build string) (models.Conversation, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return models.Conversation{}, models.Refused(s.tr.AskFirst)
	}
	if at.Path == "" || (at.Side != "RIGHT" && at.Side != "LEFT") || at.Line < 1 {
		return models.Conversation{}, models.Refused(s.tr.AskNeedsLine)
	}
	var id int
	err := s.store.Change([]models.ReviewRef{ref}, func() ([]store.FileWrite, error) {
		if err := s.askable(ref, build); err != nil {
			return nil, err
		}
		threads, err := s.store.Threads(ref)
		if err != nil {
			return nil, err
		}
		for _, t := range threads {
			id = max(id, t.ID)
		}
		id++
		fresh := models.Conversation{ID: id, Path: at.Path, Side: at.Side, Line: at.Line, CreatedAt: s.now(), Messages: []models.Message{{From: "you", Text: text, At: s.now()}}}
		if at.StartLine != 0 {
			fresh.StartLine, fresh.StartSide = at.StartLine, at.StartSide
			if fresh.StartSide == "" {
				fresh.StartSide = at.Side
			}
		}
		write, err := s.store.ThreadWrite(ref, fresh)
		return []store.FileWrite{write}, err
	})
	if err != nil {
		return models.Conversation{}, err
	}
	return s.answered(ref, id)
}

// Reply adds the question text to conversation id, and returns it once
// Claude answered, as Start does.
func (s *Service) Reply(ref models.ReviewRef, id int, text, build string) (models.Conversation, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return models.Conversation{}, models.Refused(s.tr.AskFirst)
	}
	err := s.store.Change([]models.ReviewRef{ref}, func() ([]store.FileWrite, error) {
		if err := s.askable(ref, build); err != nil {
			return nil, err
		}
		c, err := s.thread(ref, id)
		if err != nil {
			return nil, err
		}
		if store.LockHeld(s.store.Home().Review(ref).Answering(id)) {
			return nil, models.Refused(s.tr.StillWorking)
		}
		c.Messages = append(c.Messages, models.Message{From: "you", Text: text, At: s.now()})
		write, err := s.store.ThreadWrite(ref, c)
		return []store.FileWrite{write}, err
	})
	if err != nil {
		return models.Conversation{}, err
	}
	return s.answered(ref, id)
}

// Stop stops Claude working on conversation id, wherever it works, and
// returns the conversation once it says so. When nothing works on it, as
// after an answer stopped partway, a conversation still waiting is told it
// was stopped, so it waits no longer.
func (s *Service) Stop(ref models.ReviewRef, id int) (models.Conversation, error) {
	answering := s.store.Home().Review(ref).Answering(id)
	if store.LockHeld(answering) {
		_ = os.WriteFile(filepath.Join(answering, stopFile), nil, 0o644)
		for deadline := time.Now().Add(stopWait); store.LockHeld(answering) && time.Now().Before(deadline); {
			time.Sleep(stopPoll)
		}
	}
	c, err := s.thread(ref, id)
	if err != nil || (!waiting(c) && c.Working == nil) {
		return c, err
	}
	return s.change(ref, id, func(c *models.Conversation) {
		if waiting(*c) {
			c.Messages = append(c.Messages, models.Message{From: "claude", Error: s.tr.ClaudeStopped, At: s.now()})
		}
		c.Working = nil
	})
}

// Answer has Claude answer conversation id's last question, and keeps the
// answer, or why there is none. It holds the conversation's answering lock
// while it works, and stops Claude when a stop asks it to through that
// lock, or when ctx ends.
func (s *Service) Answer(ctx context.Context, ref models.ReviewRef, id int) error {
	answering := s.store.Home().Review(ref).Answering(id)
	return s.locks.With([]string{answering}, 0, func() error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		var stopped atomic.Bool
		go func() {
			for ctx.Err() == nil {
				if _, err := os.Stat(filepath.Join(answering, stopFile)); err == nil {
					stopped.Store(true)
					cancel()
					return
				}
				select {
				case <-ctx.Done():
				case <-time.After(stopPoll):
				}
			}
		}()
		message := s.answer(ctx, ref, id)
		if message.Error != "" && (stopped.Load() || ctx.Err() != nil) {
			message.Error = s.tr.ClaudeStopped
		}
		_, err := s.change(ref, id, func(c *models.Conversation) {
			c.Messages = append(c.Messages, message)
			c.Working = nil
		})
		return err
	})
}

func (s *Service) answer(ctx context.Context, ref models.ReviewRef, id int) models.Message {
	var p *progress
	text, err := func() (string, error) {
		rev, err := s.store.Read(ref)
		if err != nil {
			return "", err
		}
		if rev == nil {
			return "", models.Refused(utils.ResolvePlaceholderString(s.tr.ReviewNone, map[string]string{"id": ref.ID()}))
		}
		read, err := s.store.ReadPage(ref)
		if err != nil {
			return "", err
		}
		c, err := s.thread(ref, id)
		if err != nil {
			return "", err
		}
		folder, err := s.copy(ctx, ref, *rev, *read.Page, c)
		if err != nil {
			return "", err
		}
		real := folder
		if r, err := filepath.EvalSymlinks(folder); err == nil {
			real = r
		}
		p = newProgress(func(h claude.Heard) string { return StepWords(s.tr, h.Tool, h.Input, real) }, func(steps []models.Step) {
			_, _ = s.change(ref, id, func(c *models.Conversation) { c.Working = &models.Working{Steps: steps} })
		})
		ask := func(session string, resume bool) (string, error) {
			q := QuestionFor(s.tr, *rev, *read.Page, c, folder)
			if resume {
				q = FollowUpFor(s.tr, c)
			}
			asked := claude.Asked{System: q.System, Prompt: q.Prompt, Folder: folder, Session: session, Resume: resume, Settings: GuardSettings(s.hook(folder), nil)}
			return s.claude.Ask(ctx, asked, p.heard)
		}
		session, err := s.sessionOf(ref, id, c.Session)
		if err != nil {
			return "", err
		}
		text, err := ask(session, c.Session != "")
		var lost claude.Unanswered
		if errors.As(err, &lost) && lost.NoSession {
			if session, err = s.sessionOf(ref, id, ""); err != nil {
				return "", err
			}
			text, err = ask(session, false)
		}
		return text, err
	}()
	message := models.Message{From: "claude"}
	if p != nil {
		p.settled()
		if steps := p.kept(); len(steps) > 0 {
			message.Steps = steps
		}
	}
	if err != nil {
		message.Error = err.Error()
	} else {
		message.Text = text
	}
	message.At = s.now()
	return message
}

func (s *Service) answered(ref models.ReviewRef, id int) (models.Conversation, error) {
	ran := s.answerer(ref, id)
	c, err := s.thread(ref, id)
	if err != nil || !waiting(c) {
		return c, err
	}
	if store.LockHeld(s.store.Home().Review(ref).Answering(id)) {
		return c, models.Refused(s.tr.StillWorking)
	}
	why := s.tr.ClaudeNothing
	if ran != nil {
		why = ran.Error()
	}
	return s.change(ref, id, func(c *models.Conversation) {
		if waiting(*c) {
			c.Messages = append(c.Messages, models.Message{From: "claude", Error: utils.ResolvePlaceholderString(s.tr.AnswerEnded, map[string]string{"why": why}), At: s.now()})
		}
		c.Working = nil
	})
}

func (s *Service) sessionOf(ref models.ReviewRef, id int, session string) (string, error) {
	if session != "" {
		return session, nil
	}
	fresh := s.session()
	_, err := s.change(ref, id, func(c *models.Conversation) { c.Session = fresh })
	return fresh, err
}

func (s *Service) copy(ctx context.Context, ref models.ReviewRef, rev models.Review, page models.PageRecord, c models.Conversation) (string, error) {
	home := s.store.Home()
	head := HeadAsked(s.tr, rev, page, c)
	plan := CopyBranches(s.tr, rev, page, c)
	folder, err := func() (string, error) {
		source := home.Clone(rev.Host, rev.Repo)
		if rev.IsBranch() {
			local, err := s.localClone(ctx, rev)
			if err != nil {
				return "", err
			}
			source = local
		}
		if err := os.MkdirAll(home.Copies(), 0o755); err != nil {
			return "", err
		}
		var folder string
		err := s.locks.With([]string{home.Copies() + ".lock"}, copyWait, func() error {
			made, err := s.copies.Prepare(ctx, home.Copy(ref, c.ID), git.CopyOf{Source: source, Commit: head.Commit, Branches: plan.Branches, Branch: plan.Branch})
			if err != nil {
				return err
			}
			folder = made
			s.copies.Prune(home.Copies(), unused, folder)
			return nil
		})
		return folder, err
	}()
	if err != nil {
		return "", models.Refused(utils.ResolvePlaceholderString(s.tr.NoCodebase, map[string]string{"why": err.Error()}))
	}
	return folder, nil
}

func (s *Service) localClone(ctx context.Context, rev models.Review) (string, error) {
	branch, err := models.ParseBranchRef(s.tr, rev.ID)
	if err != nil {
		return "", err
	}
	path := s.store.Home().LocalClone(branch.RepoKey)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	var clone string
	err = s.locks.With([]string{path + ".lock"}, copyWait, func() error {
		made, err := s.clones.CopyFrom(ctx, path, rev.Path, []string{rev.Head.Sha, rev.Base.Sha})
		clone = made
		return err
	})
	return clone, err
}

func (s *Service) askable(ref models.ReviewRef, build string) error {
	rev, err := s.store.Read(ref)
	if err != nil {
		return err
	}
	if rev == nil {
		return models.Refused(utils.ResolvePlaceholderString(s.tr.ReviewNone, map[string]string{"id": ref.ID()}))
	}
	current, err := s.store.PageBuild(ref)
	if err != nil {
		return err
	}
	if problem := store.BuildProblem(s.tr, current, build); problem != "" {
		return models.Refused(utils.ResolvePlaceholderString(s.tr.NotAsked, map[string]string{"why": problem}))
	}
	return nil
}

func (s *Service) change(ref models.ReviewRef, id int, edit func(*models.Conversation)) (models.Conversation, error) {
	var changed models.Conversation
	err := s.store.Change([]models.ReviewRef{ref}, func() ([]store.FileWrite, error) {
		c, err := s.thread(ref, id)
		if err != nil {
			return nil, err
		}
		edit(&c)
		changed = c
		write, err := s.store.ThreadWrite(ref, c)
		return []store.FileWrite{write}, err
	})
	return changed, err
}

func (s *Service) thread(ref models.ReviewRef, id int) (models.Conversation, error) {
	threads, err := s.store.Threads(ref)
	if err != nil {
		return models.Conversation{}, err
	}
	for _, t := range threads {
		if t.ID == id {
			return t, nil
		}
	}
	return models.Conversation{}, models.Refused(utils.ResolvePlaceholderString(s.tr.NoConversation, map[string]string{"id": strconv.Itoa(id), "review": ref.ID()}))
}

func waiting(c models.Conversation) bool {
	return len(c.Messages) > 0 && c.Messages[len(c.Messages)-1].From == "you"
}

func guardCommand(folder string) string {
	program, err := os.Executable()
	if err != nil {
		program = os.Args[0]
	}
	return quoted(program) + " guard " + quoted(folder)
}

func quoted(text string) string {
	return "'" + strings.ReplaceAll(text, "'", `'\''`) + "'"
}

func newSession() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
