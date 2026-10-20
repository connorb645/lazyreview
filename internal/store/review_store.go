package store

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

const changeWait = 30 * time.Second

var threadFile = regexp.MustCompile(`^\d+\.json$`)

// ReviewStore reads and writes the reviews kept under the data home.
// Reading never locks, and a review's page is read once per build and
// shared by every caller, who must not change it. A change goes through
// Change, which writes every file it returns or none.
type ReviewStore struct {
	tr    *i18n.TranslationSet
	home  DataHome
	locks *Locks
	mu    sync.Mutex
	pages map[string]PageRead
}

// NewReviewStore returns the store of the data home, changing reviews under
// locks.
func NewReviewStore(tr *i18n.TranslationSet, home DataHome, locks *Locks) *ReviewStore {
	return &ReviewStore{tr: tr, home: home, locks: locks, pages: map[string]PageRead{}}
}

// Home is the data home the store keeps.
func (s *ReviewStore) Home() DataHome {
	return s.home
}

// Read is the review ref names, or nil when there is none.
func (s *ReviewStore) Read(ref models.ReviewRef) (*models.Review, error) {
	var review models.Review
	found, err := s.readJSON(s.home.Review(ref).Record(), &review)
	if err != nil || !found {
		return nil, err
	}
	return &review, nil
}

// ReadPage is the review's page with its build id, or nil when it has none.
func (s *ReviewStore) ReadPage(ref models.ReviewRef) (*PageRead, error) {
	path := s.home.Review(ref).Page()
	text, found, err := utils.ReadText(path)
	if err != nil {
		return nil, s.unreadable(path, err)
	}
	if !found {
		return nil, nil
	}
	build := PageBuild(text)
	s.mu.Lock()
	kept, ok := s.pages[ref.ID()]
	s.mu.Unlock()
	if ok && kept.Build == build {
		return &kept, nil
	}
	var page models.PageRecord
	if err := json.Unmarshal([]byte(text), &page); err != nil {
		return nil, s.unreadable(path, err)
	}
	read := PageRead{Build: build, Page: &page}
	s.mu.Lock()
	s.pages[ref.ID()] = read
	s.mu.Unlock()
	return &read, nil
}

// PageBuild is the build id of the review's page, or "" when it has none.
func (s *ReviewStore) PageBuild(ref models.ReviewRef) (string, error) {
	path := s.home.Review(ref).Page()
	text, found, err := utils.ReadText(path)
	if err != nil {
		return "", s.unreadable(path, err)
	}
	if !found {
		return "", nil
	}
	return PageBuild(text), nil
}

// ReadConfig is the components config the review was last loaded with, as
// its JSON text, or nil when it has none.
func (s *ReviewStore) ReadConfig(ref models.ReviewRef) ([]byte, error) {
	path := s.home.Review(ref).Config()
	text, found, err := utils.ReadText(path)
	if err != nil {
		return nil, s.unreadable(path, err)
	}
	if !found {
		return nil, nil
	}
	return []byte(text), nil
}

// ReadWritten is the review's written review, and false when it has none.
func (s *ReviewStore) ReadWritten(ref models.ReviewRef) (string, bool, error) {
	path := s.home.Review(ref).Written()
	text, found, err := utils.ReadText(path)
	if err != nil {
		return "", false, s.unreadable(path, err)
	}
	return text, found, nil
}

// Threads is every conversation on the review's lines, oldest first.
func (s *ReviewStore) Threads(ref models.ReviewRef) ([]models.Conversation, error) {
	folder := s.home.Review(ref).Threads()
	entries, err := os.ReadDir(folder)
	if errors.Is(err, fs.ErrNotExist) {
		return []models.Conversation{}, nil
	}
	if err != nil {
		return nil, s.unreadable(folder, err)
	}
	threads := []models.Conversation{}
	for _, e := range entries {
		if !threadFile.MatchString(e.Name()) {
			continue
		}
		var c models.Conversation
		found, err := s.readJSON(filepath.Join(folder, e.Name()), &c)
		if err != nil {
			return nil, err
		}
		if found {
			threads = append(threads, c)
		}
	}
	sort.SliceStable(threads, func(a, b int) bool { return threads[a].ID < threads[b].ID })
	return threads, nil
}

// GitHubThreads is the review threads on GitHub as the last load read them,
// none for a review that keeps none.
func (s *ReviewStore) GitHubThreads(ref models.ReviewRef) ([]models.GitHubThread, error) {
	threads := []models.GitHubThread{}
	if _, err := s.readJSON(s.home.Review(ref).GitHub(), &threads); err != nil {
		return nil, err
	}
	return threads, nil
}

// Mark is a short mark of how the review's files stand on disk, each file's
// name, size and time of change, which changes whenever anything of the
// review does, whoever changes it.
func (s *ReviewStore) Mark(ref models.ReviewRef) string {
	folder := s.home.Review(ref)
	var mark strings.Builder
	paths := []string{folder.Record(), folder.Page(), folder.Written(), folder.GitHub()}
	if entries, err := os.ReadDir(folder.Threads()); err == nil {
		for _, e := range entries {
			paths = append(paths, filepath.Join(folder.Threads(), e.Name()))
		}
	}
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			fmt.Fprintf(&mark, "%s %d %d\n", filepath.Base(path), info.Size(), info.ModTime().UnixNano())
		}
	}
	return mark.String()
}

// RecordWrite is the write that puts review in its record, as every
// lazyreview has written it: two spaces deep, with a final new line.
func (s *ReviewStore) RecordWrite(ref models.ReviewRef, review models.Review) (FileWrite, error) {
	text, err := indented(review)
	return FileWrite{Path: s.home.Review(ref).Record(), Text: text}, err
}

// ThreadWrite is the write that puts conversation in its file, as every
// lazyreview has written it: two spaces deep, with a final new line.
func (s *ReviewStore) ThreadWrite(ref models.ReviewRef, conversation models.Conversation) (FileWrite, error) {
	text, err := indented(conversation)
	return FileWrite{Path: s.home.Review(ref).Thread(conversation.ID), Text: text}, err
}

func indented(v any) (string, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(v)
	return out.String(), err
}

// Change locks the folders of every review in refs, runs work, and writes
// the files it returns, all or none: a failure partway puts back every file
// it had written, and each file is replaced whole.
func (s *ReviewStore) Change(refs []models.ReviewRef, work func() ([]FileWrite, error)) error {
	var locks []string
	for _, ref := range refs {
		lock := s.home.Review(ref).Lock()
		if err := os.MkdirAll(filepath.Dir(lock), 0o755); err != nil {
			return err
		}
		locks = append(locks, lock)
	}
	return s.locks.With(locks, changeWait, func() error {
		files, err := work()
		if err != nil {
			return err
		}
		return s.writeAll(files)
	})
}

func (s *ReviewStore) writeAll(files []FileWrite) error {
	type before struct {
		path   string
		text   string
		exists bool
	}
	var done []before
	for _, f := range files {
		text, exists, err := utils.ReadText(f.Path)
		if err == nil {
			err = replace(f)
		}
		if err != nil {
			for i := len(done) - 1; i >= 0; i-- {
				_ = replace(FileWrite{Path: done[i].path, Text: done[i].text, Remove: !done[i].exists})
			}
			return models.Refused(utils.ResolvePlaceholderString(s.tr.NotWritten, map[string]string{"why": err.Error()}))
		}
		done = append(done, before{f.Path, text, exists})
	}
	return nil
}

func replace(f FileWrite) error {
	if f.Remove {
		err := os.Remove(f.Path)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o755); err != nil {
		return err
	}
	return utils.WriteWhole(f.Path, f.Text, 0o644)
}

func (s *ReviewStore) readJSON(path string, into any) (bool, error) {
	text, found, err := utils.ReadText(path)
	if err != nil {
		return false, s.unreadable(path, err)
	}
	if !found {
		return false, nil
	}
	if err := json.Unmarshal([]byte(text), into); err != nil {
		return false, s.unreadable(path, err)
	}
	return true, nil
}

func (s *ReviewStore) unreadable(path string, err error) error {
	return models.Refused(utils.ResolvePlaceholderString(s.tr.Unreadable, map[string]string{"path": path, "why": err.Error()}))
}

// PageBuild is the build id of a page's text: a short hash of it, so a load
// that draws anything differently gives a new one, and a load that draws the
// same page keeps it.
func PageBuild(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])[:12]
}
