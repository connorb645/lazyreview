package store

import (
	"path/filepath"
	"strconv"
)

// ReviewFolder is the files of one review, inside its folder:
//
//	review.json             the review: its pull requests, queued comments
//	                        and the reviews sent from it
//	page.json               what the app draws, built from the clone
//	written.md              a written review, if one was attached
//	threads/<n>.json        each conversation on a line
//	threads/<n>.answering   held while Claude answers conversation n
//	github.json             the review threads on GitHub, as the last load
//	                        read them
//	.lock                   held while the review changes
//
// It only names paths, and reads and writes nothing.
type ReviewFolder struct {
	Path string
}

// Record is review.json.
func (f ReviewFolder) Record() string { return filepath.Join(f.Path, "review.json") }

// Page is page.json.
func (f ReviewFolder) Page() string { return filepath.Join(f.Path, "page.json") }

// Written is written.md.
func (f ReviewFolder) Written() string { return filepath.Join(f.Path, "written.md") }

// Threads is the folder of conversations.
func (f ReviewFolder) Threads() string { return filepath.Join(f.Path, "threads") }

// Thread is conversation n's file.
func (f ReviewFolder) Thread(n int) string {
	return filepath.Join(f.Threads(), strconv.Itoa(n)+".json")
}

// Answering is the lock held while Claude answers conversation n, which a
// stop asks it to end through.
func (f ReviewFolder) Answering(n int) string {
	return filepath.Join(f.Threads(), strconv.Itoa(n)+".answering")
}

// GitHub is github.json.
func (f ReviewFolder) GitHub() string { return filepath.Join(f.Path, "github.json") }

// Lock is the lock held while the review changes.
func (f ReviewFolder) Lock() string { return filepath.Join(f.Path, ".lock") }
