// Package store keeps lazyreview's files under its data home: each review's
// folder, its record, its page and its conversations, each file written
// whole, and the locks that let one process at a time change a review.
package store

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/models"
)

// DataHome is the folder lazyreview keeps its files under:
//
//	reviews/<host>/<owner>/<repo>/<pull>/  one folder per pull request review
//	reviews/local/<repo key>/<branch>/      one folder per branch review
//	clones/<host>/<owner>/<repo>.git       a bare clone per repository
//	clones/local/<repo key>.git            a bare clone per repository on
//	                                       this machine, for Claude's copies
//	copies/<review id>/<conversation>/     a throwaway copy of the code that
//	                                       Claude answers a conversation in
//	settings.json                          an earlier version's settings,
//	                                       no longer read
//
// It only names paths, and reads and writes nothing.
type DataHome struct {
	Root string
}

// Reviews is the folder holding every review.
func (h DataHome) Reviews() string {
	return filepath.Join(h.Root, "reviews")
}

// Review is the folder of the review a ref names.
func (h DataHome) Review(ref models.ReviewRef) ReviewFolder {
	return ReviewFolder{Path: filepath.Join(append([]string{h.Reviews()}, strings.Split(ref.ID(), "/")...)...)}
}

// Clone is the bare clone of repo, owner/name, on host.
func (h DataHome) Clone(host, repo string) string {
	return filepath.Join(h.Root, "clones", host, repo+".git")
}

// LocalClone is the bare clone of a repository on this machine, by its
// branch ref's repository key.
func (h DataHome) LocalClone(repoKey string) string {
	return filepath.Join(h.Root, "clones", "local", repoKey+".git")
}

// Copies is the folder holding every throwaway copy of the code.
func (h DataHome) Copies() string {
	return filepath.Join(h.Root, "copies")
}

// Copy is the throwaway copy of the code for conversation n of the review
// a ref names.
func (h DataHome) Copy(ref models.ReviewRef, n int) string {
	return filepath.Join(append(append([]string{h.Copies()}, strings.Split(ref.ID(), "/")...), strconv.Itoa(n))...)
}

// Repository is the folder holding the reviews of repo, owner/name, on host.
func (h DataHome) Repository(host, repo string) string {
	return filepath.Join(append([]string{h.Reviews(), host}, strings.Split(repo, "/")...)...)
}

// SettingsFile is settings.json, which an earlier version read and the
// config replaced.
func (h DataHome) SettingsFile() string {
	return filepath.Join(h.Root, "settings.json")
}
