package store

import "os"

// LockHeld says whether the lock at path is held now by a process that is
// still running.
func LockHeld(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return holderAlive(path)
}
