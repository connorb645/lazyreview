package ask

import (
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/connorb645/lazyreview/internal/store"
)

var answeringLock = regexp.MustCompile(`^\d+\.answering$`)

// StopAnswers asks every answer still running under the data home to end,
// as a stop from the app does, through the stop file in its answering
// lock, and returns how many it asked once each has ended, or wait has
// passed. It signals no process.
func StopAnswers(home store.DataHome, wait time.Duration) (int, error) {
	asked := running(home.Reviews())
	for _, lock := range asked {
		if err := os.WriteFile(filepath.Join(lock, stopFile), nil, 0o644); err != nil {
			return 0, err
		}
	}
	for deadline := time.Now().Add(wait); anyHeld(asked) && time.Now().Before(deadline); {
		time.Sleep(stopPoll)
	}
	return len(asked), nil
}

func running(folder string) []string {
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil
	}
	var locks []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join(folder, e.Name())
		if !answeringLock.MatchString(e.Name()) {
			locks = append(locks, running(path)...)
		} else if store.LockHeld(path) {
			locks = append(locks, path)
		}
	}
	return locks
}

func anyHeld(locks []string) bool {
	for _, lock := range locks {
		if store.LockHeld(lock) {
			return true
		}
	}
	return false
}
