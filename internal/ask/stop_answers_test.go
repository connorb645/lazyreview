package ask

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/internal/store"
)

func TestStoppingAsksEveryAnswerStillRunningToEndThroughItsLockAndWaitsForIt(t *testing.T) {
	root := t.TempDir()
	lock := func(path string, pid int) string {
		folder := filepath.Join(root, "reviews", path)
		if err := os.MkdirAll(folder, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "pid"), []byte(strconv.Itoa(pid)), 0o644); err != nil {
			t.Fatal(err)
		}
		return folder
	}
	running := lock("github.com/octo/widgets/21/threads/2.answering", os.Getpid())
	gone := lock("local/work-1a2b3c4d/fix~x/threads/1.answering", 1<<22+7)
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
			if _, err := os.Stat(filepath.Join(running, stopFile)); err == nil {
				_ = os.RemoveAll(running)
				return
			}
		}
	}()
	asked, err := StopAnswers(store.DataHome{Root: root}, 5*time.Second)
	<-ended
	if err != nil || asked != 1 {
		t.Fatalf("stopping asked %d answers, %v; want the one running", asked, err)
	}
	if _, err := os.Stat(running); !os.IsNotExist(err) {
		t.Fatal("the answer did not end")
	}
	if _, err := os.Stat(filepath.Join(gone, stopFile)); !os.IsNotExist(err) {
		t.Fatal("a lock whose holder is gone was asked to stop")
	}
	if asked, err := StopAnswers(store.DataHome{Root: filepath.Join(root, "none")}, time.Second); err != nil || asked != 0 {
		t.Fatalf("a data home with no reviews asked %d, %v", asked, err)
	}
}
