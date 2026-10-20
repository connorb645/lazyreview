package store

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
)

func scratch(t *testing.T) (*ReviewStore, string) {
	t.Helper()
	root := t.TempDir()
	tr := i18n.EnglishTranslationSet()
	return NewReviewStore(tr, DataHome{Root: root}, NewLocks(tr)), root
}

func TestAChangeWritesEveryFileOrNone(t *testing.T) {
	s, root := scratch(t)
	a := models.PullRef{Host: "github.com", Repo: "octo/widgets", Pull: 1}
	write := func(review models.Review, more ...FileWrite) error {
		return s.Change([]models.ReviewRef{a}, func() ([]FileWrite, error) {
			record, err := s.RecordWrite(a, review)
			return append([]FileWrite{record}, more...), err
		})
	}
	if err := write(models.Review{ID: a.ID(), LoadedAt: "x"}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(s.Home().Review(a).Record())
	blocked := filepath.Join(root, "blocked")
	if err := os.Mkdir(blocked, 0o500); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(blocked, 0o700)
	err := write(models.Review{ID: a.ID(), LoadedAt: "y"}, FileWrite{Path: filepath.Join(blocked, "deeper", "file.json"), Text: "{}"})
	if err == nil || !strings.Contains(err.Error(), "could not write the review, so every file is as it was") {
		t.Fatalf("a change that cannot write every file says %v", err)
	}
	if after, _ := os.ReadFile(s.Home().Review(a).Record()); string(after) != string(before) {
		t.Fatalf("the record changed to %s", after)
	}
}

func TestARecordIsWrittenTwoSpacesDeepAndReadsBack(t *testing.T) {
	s, _ := scratch(t)
	a := models.PullRef{Host: "github.com", Repo: "octo/widgets", Pull: 1}
	account := "lee"
	review := models.Review{ID: a.ID(), Host: "github.com", Repo: "octo/widgets", Account: &account, Me: "lee", Title: "Fix <the> guide",
		Members: []models.ReviewMember{{Pull: 1, Head: "abc", Comments: []models.Comment{{ID: 1, Path: "a.go", Line: 3, Side: "RIGHT", Body: "Why?"}}}}}
	write, err := s.RecordWrite(a, review)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(write.Text, "{\n  \"id\": \"github.com/octo/widgets/1\",\n  \"host\"") || !strings.HasSuffix(write.Text, "}\n") || !strings.Contains(write.Text, `"Fix <the> guide"`) {
		t.Fatalf("the record reads:\n%s", write.Text)
	}
	if strings.Contains(write.Text, `"kind"`) || strings.Contains(write.Text, `"start_line"`) || !strings.Contains(write.Text, `"posted": []`) {
		t.Fatalf("a pull request's record holds a field it never had, or lacks one:\n%s", write.Text)
	}
	if err := s.Change([]models.ReviewRef{a}, func() ([]FileWrite, error) { return []FileWrite{write}, nil }); err != nil {
		t.Fatal(err)
	}
	read, err := s.Read(a)
	if err != nil || read.Members[0].Comments[0].Body != "Why?" || *read.Account != "lee" {
		t.Fatalf("the record reads back as %+v, %v", read, err)
	}
}

func TestAReviewFileThatIsNotJSONIsNamedInTheError(t *testing.T) {
	s, _ := scratch(t)
	a := models.PullRef{Host: "github.com", Repo: "octo/widgets", Pull: 1}
	folder := s.Home().Review(a)
	if err := os.MkdirAll(folder.Path, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(folder.Record(), []byte(`{"id":`), 0o644)
	_ = os.WriteFile(folder.Page(), nil, 0o644)
	if _, err := s.Read(a); err == nil || !strings.HasPrefix(err.Error(), "could not read "+folder.Record()+": ") {
		t.Errorf("an unreadable record says %v", err)
	}
	if _, err := s.ReadPage(a); err == nil || !strings.HasPrefix(err.Error(), "could not read "+folder.Page()+": ") {
		t.Errorf("an unreadable page says %v", err)
	}
	if threads, err := s.Threads(a); err != nil || len(threads) != 0 {
		t.Errorf("a review with no conversations has %v, %v", threads, err)
	}
}

func TestLocksLetOneHolderInAtATimeAndALockWhoseHolderDiedIsTakenOver(t *testing.T) {
	root := t.TempDir()
	locks := NewLocks(i18n.EnglishTranslationSet())
	lock := filepath.Join(root, "one.lock")
	var mu sync.Mutex
	var order []string
	say := func(s string) {
		mu.Lock()
		defer mu.Unlock()
		order = append(order, s)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = locks.With([]string{lock}, time.Second, func() error { say("a in"); time.Sleep(30 * time.Millisecond); say("a out"); return nil })
	}()
	time.Sleep(5 * time.Millisecond)
	go func() {
		defer wg.Done()
		_ = locks.With([]string{lock}, time.Second, func() error { say("b in"); say("b out"); return nil })
	}()
	wg.Wait()
	if strings.Join(order, ", ") != "a in, a out, b in, b out" {
		t.Fatalf("the holders went in as %v", order)
	}
	if err := os.Mkdir(lock, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(lock, "pid"), []byte(strconv.Itoa(999999)), 0o644)
	if err := locks.With([]string{lock}, time.Second, func() error { return nil }); err != nil {
		t.Fatalf("a lock whose holder died is not taken over: %v", err)
	}
	if _, err := os.Stat(lock); !os.IsNotExist(err) {
		t.Fatalf("the lock is still there after its holder let it go")
	}
}

func TestALockHeldByALiveProcessElsewhereIsWaitedForThenRefused(t *testing.T) {
	root := t.TempDir()
	lock := filepath.Join(root, "held.lock")
	_ = os.Mkdir(lock, 0o755)
	_ = os.WriteFile(filepath.Join(lock, "pid"), []byte(strconv.Itoa(os.Getppid())), 0o644)
	err := NewLocks(i18n.EnglishTranslationSet()).With([]string{lock}, 120*time.Millisecond, func() error { return nil })
	if err == nil || err.Error() != "another process holds "+lock {
		t.Fatalf("a lock another live process holds gives %v", err)
	}
}
