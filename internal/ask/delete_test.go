package ask

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/store"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

// breakingCopies removes copies as the copies given do, and breaks what
// goes along with the removal: break runs once the copy is aside, and the
// function it returns once what goes along has run.
type breakingCopies struct {
	CodeCopies
	breaks func() func()
}

func (b breakingCopies) Remove(folder string, along func() error) error {
	return b.CodeCopies.Remove(folder, func() error {
		mend := b.breaks()
		defer mend()
		return along()
	})
}

func (w *world) asked(ref models.ReviewRef, build string, questions ...string) {
	w.t.Helper()
	for _, q := range questions {
		if _, err := w.service.Start(ref, line("docs/guide.md", 3), q, build); err != nil {
			w.t.Fatal(err)
		}
	}
}

func (w *world) marked(ref models.ReviewRef, id int) string {
	w.t.Helper()
	mark := filepath.Join(w.home.Copy(ref, id), "left-by-claude.txt")
	if err := os.WriteFile(mark, []byte("conversation "+strconv.Itoa(id)+"\n"), 0o644); err != nil {
		w.t.Fatal(err)
	}
	return mark
}

func (w *world) ids(ref models.ReviewRef) []int {
	w.t.Helper()
	var ids []int
	for _, c := range w.threads(ref) {
		ids = append(ids, c.ID)
	}
	return ids
}

func names(t *testing.T, folder string) []string {
	t.Helper()
	entries, _ := os.ReadDir(folder)
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func TestDeletingAConversationTakesItsFileAndItsCopyAndLeavesTheRest(t *testing.T) {
	w := newWorld(t)
	ref, build := w.single()
	w.asked(ref, build, "Every?", "Why a list?")
	rev, _ := w.reviews.Read(ref)
	rev.Members[0].Comments = []models.Comment{{ID: 1, Path: "docs/guide.md", Line: 3, Side: "RIGHT", Body: w.threads(ref)[0].Messages[1].Text}}
	if err := w.reviews.Change([]models.ReviewRef{ref}, func() ([]store.FileWrite, error) {
		write, err := w.reviews.RecordWrite(ref, *rev)
		return []store.FileWrite{write}, err
	}); err != nil {
		t.Fatal(err)
	}
	record := testsupport.FolderState(t, filepath.Dir(w.home.Review(ref).Record()))["review.json"]
	w.marked(ref, 1)
	if err := w.service.Delete(ref, 1); err != nil {
		t.Fatal(err)
	}
	if got := w.ids(ref); !reflect.DeepEqual(got, []int{2}) {
		t.Errorf("the conversations left are %v", got)
	}
	if _, err := os.Stat(w.home.Review(ref).Thread(1)); err == nil {
		t.Error("the conversation's file stayed")
	}
	if got := names(t, filepath.Dir(w.home.Copy(ref, 1))); !reflect.DeepEqual(got, []string{"2"}) {
		t.Errorf("beside the other conversation's copy are %v", got)
	}
	if got := testsupport.FolderState(t, filepath.Dir(w.home.Review(ref).Record()))["review.json"]; got != record {
		t.Errorf("the review's record, with a comment made from an answer, changed to %s", got)
	}
	if err := w.service.Delete(ref, 1); err == nil || err.Error() != "not deleted: there is no conversation 1 on github.com/octo/widgets/30" {
		t.Errorf("deleting it again gave %v", err)
	}
	old, session := w.marked(ref, 2), w.threads(ref)[0].Session
	if err := w.service.Delete(ref, 2); err != nil {
		t.Fatal(err)
	}
	w.asked(ref, build, "Why a list?", "Again?")
	again := w.threads(ref)
	if len(again) != 2 || again[1].ID != 2 || again[1].Session == session {
		t.Fatalf("the questions after the deletes are %+v", again)
	}
	if _, err := os.Stat(old); err == nil {
		t.Error("the new conversation 2 answered in the copy the deleted one left")
	}
}

func TestDeletingAConversationClaudeIsAnsweringStopsTheAnswerFirst(t *testing.T) {
	w := newWorld(t)
	ref, build := w.single()
	w.fake.Answer(testsupport.FakeAnswer{Text: "Too late.", Steps: []testsupport.FakeStep{{Tool: "Bash", Input: map[string]any{"command": "npm test"}}}, Wait: 60_000})
	asking := make(chan models.Conversation)
	go func() {
		c, _ := w.service.Start(ref, line("docs/guide.md", 3), "Is it tested?", build)
		asking <- c
	}()
	for i := 0; i < 100; i++ {
		if threads := w.threads(ref); len(threads) > 0 && threads[0].Working != nil && len(threads[0].Working.Steps) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	created := w.threads(ref)[0].CreatedAt
	started := time.Now()
	if err := w.service.Delete(ref, 1); err != nil {
		t.Fatal(err)
	}
	if c := <-asking; c.ID != 1 || c.CreatedAt != created {
		t.Errorf("the question returned %+v, not the conversation deleted", c)
	}
	if took := time.Since(started); took > 5*time.Second {
		t.Errorf("the delete waited %s for the answer", took)
	}
	time.Sleep(200 * time.Millisecond)
	if got := names(t, w.home.Review(ref).Threads()); len(got) != 0 {
		t.Errorf("after the answer stopped, the conversations' folder holds %v", got)
	}
	if _, err := os.Stat(w.home.Copy(ref, 1)); err == nil {
		t.Error("the copy stayed")
	}
}

func TestADeleteWhenClaudeDoesNotStopTakesNothing(t *testing.T) {
	w := newWorld(t)
	ref, build := w.single()
	w.asked(ref, build, "Every?")
	before := testsupport.FolderState(t, w.home.Review(ref).Threads())
	mark := w.marked(ref, 1)
	answering := w.home.Review(ref).Answering(1)
	if err := os.Mkdir(answering, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(answering, "pid"), []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		t.Fatal(err)
	}
	w.service.stopping = 200 * time.Millisecond
	err := w.service.Delete(ref, 1)
	if err == nil || err.Error() != "not deleted: Claude is still answering it and did not stop" {
		t.Errorf("a delete Claude did not stop for gave %v", err)
	}
	if _, err := os.Stat(filepath.Join(answering, "stop")); err != nil {
		t.Error("Claude was not asked to stop")
	}
	if err := os.RemoveAll(answering); err != nil {
		t.Fatal(err)
	}
	if after := testsupport.FolderState(t, w.home.Review(ref).Threads()); !reflect.DeepEqual(after, before) {
		t.Errorf("the conversation changed to %v", after)
	}
	if _, err := os.Stat(mark); err != nil {
		t.Error("the copy went")
	}
}

func TestADeleteWhoseCopyCannotMoveAsideTakesNothing(t *testing.T) {
	w := newWorld(t)
	ref, build := w.single()
	w.asked(ref, build, "Every?")
	before := testsupport.FolderState(t, w.home.Review(ref).Threads())
	mark := w.marked(ref, 1)
	copies := filepath.Dir(w.home.Copy(ref, 1))
	if err := os.Chmod(copies, 0o500); err != nil {
		t.Fatal(err)
	}
	err := w.service.Delete(ref, 1)
	if err := os.Chmod(copies, 0o755); err != nil {
		t.Fatal(err)
	}
	if err == nil || !strings.HasPrefix(err.Error(), "not deleted: ") || !strings.HasSuffix(err.Error(), "permission denied") {
		t.Errorf("a copy that cannot move gave %v", err)
	}
	if after := testsupport.FolderState(t, w.home.Review(ref).Threads()); !reflect.DeepEqual(after, before) {
		t.Errorf("the conversation changed to %v", after)
	}
	if _, err := os.Stat(mark); err != nil || !reflect.DeepEqual(names(t, copies), []string{"1"}) {
		t.Errorf("the copies are %v", names(t, copies))
	}
}

func TestADeleteWhoseFileCannotBeRemovedPutsTheCopyBack(t *testing.T) {
	w := newWorld(t)
	ref, build := w.single()
	w.asked(ref, build, "Every?")
	before := testsupport.FolderState(t, w.home.Review(ref).Threads())
	mark := w.marked(ref, 1)
	threads := w.home.Review(ref).Threads()
	w.service.copies = breakingCopies{CodeCopies: w.service.copies, breaks: func() func() {
		if err := os.Chmod(threads, 0o500); err != nil {
			t.Fatal(err)
		}
		return func() { _ = os.Chmod(threads, 0o755) }
	}}
	err := w.service.Delete(ref, 1)
	if err == nil || !strings.HasPrefix(err.Error(), "not deleted: could not write the review, so every file is as it was: ") {
		t.Errorf("a file that cannot be removed gave %v", err)
	}
	if after := testsupport.FolderState(t, threads); !reflect.DeepEqual(after, before) {
		t.Errorf("the conversation changed to %v", after)
	}
	if _, err := os.Stat(mark); err != nil || !reflect.DeepEqual(names(t, filepath.Dir(w.home.Copy(ref, 1))), []string{"1"}) {
		t.Errorf("the copy was not put back: %v", names(t, filepath.Dir(w.home.Copy(ref, 1))))
	}
}
