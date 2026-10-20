package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const (
	lockPoll     = 50 * time.Millisecond
	unheldMaxAge = 10 * time.Second
)

// Locks holds a lock on a path while work runs, across processes and within
// one. A lock is a folder made at the path, holding the process id of its
// holder, as every lazyreview has made its locks; a lock whose holder has
// died, or that has stood without a holder for ten seconds, is taken over.
type Locks struct {
	tr   *i18n.TranslationSet
	mu   sync.Mutex
	held map[string]chan struct{}
}

// NewLocks returns the locks of one process.
func NewLocks(tr *i18n.TranslationSet) *Locks {
	return &Locks{tr: tr, held: map[string]chan struct{}{}}
}

// With takes each lock in sorted order, so two callers wanting the same
// locks never wait on each other, runs work, and lets the locks go whether
// it succeeds or fails. It waits up to wait for a held lock, then fails.
func (l *Locks) With(paths []string, wait time.Duration, work func() error) error {
	sorted := uniqueSorted(paths)
	deadline := time.Now().Add(wait)
	var taken []string
	defer func() {
		for i := len(taken) - 1; i >= 0; i-- {
			l.release(taken[i])
		}
	}()
	for _, path := range sorted {
		if err := l.take(path, deadline); err != nil {
			return err
		}
		taken = append(taken, path)
	}
	return work()
}

func (l *Locks) take(path string, deadline time.Time) error {
	for {
		l.mu.Lock()
		waiting, busy := l.held[path]
		if !busy {
			l.held[path] = make(chan struct{})
		}
		l.mu.Unlock()
		if !busy {
			break
		}
		select {
		case <-waiting:
		case <-time.After(time.Until(deadline)):
			return models.Refused(utils.ResolvePlaceholderString(l.tr.GaveUpWaiting, map[string]string{"path": path}))
		}
	}
	for {
		err := os.Mkdir(path, 0o755)
		if err == nil {
			if err := os.WriteFile(filepath.Join(path, "pid"), []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
				_ = os.RemoveAll(path)
				l.forget(path)
				return err
			}
			return nil
		}
		if !errors.Is(err, fs.ErrExist) {
			l.forget(path)
			return err
		}
		if !holderAlive(path) {
			_ = os.RemoveAll(path)
			continue
		}
		if time.Now().After(deadline) {
			l.forget(path)
			return models.Refused(utils.ResolvePlaceholderString(l.tr.HeldElsewhere, map[string]string{"path": path}))
		}
		time.Sleep(lockPoll)
	}
}

func (l *Locks) release(path string) {
	_ = os.RemoveAll(path)
	l.forget(path)
}

func (l *Locks) forget(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if done, ok := l.held[path]; ok {
		close(done)
		delete(l.held, path)
	}
}

func holderAlive(path string) bool {
	text, err := os.ReadFile(filepath.Join(path, "pid"))
	pid, _ := strconv.Atoi(strings.TrimSpace(string(text)))
	if err != nil || pid <= 0 {
		return youngerThan(path, unheldMaxAge)
	}
	err = syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

func youngerThan(path string, age time.Duration) bool {
	info, err := os.Stat(path)
	return err == nil && time.Since(info.ModTime()) < age
}

func uniqueSorted(paths []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}
