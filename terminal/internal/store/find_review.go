package store

import (
	"os"
	"regexp"
	"strconv"

	"github.com/connorb645/lazyreview/terminal/internal/models"
)

var reviewNumber = regexp.MustCompile(`^\d+$`)

// Find is the review holding the pull request ref names, with its ref: a
// review of its repository, not moved, that holds it as a member, or else
// its own review, moved or not. The review is nil when there is none.
func (s *ReviewStore) Find(ref models.PullRef) (models.PullRef, *models.Review, error) {
	entries, err := os.ReadDir(s.home.Repository(ref.Host, ref.Repo))
	if err != nil && !os.IsNotExist(err) {
		return models.PullRef{}, nil, err
	}
	for _, e := range entries {
		if !e.IsDir() || !reviewNumber.MatchString(e.Name()) {
			continue
		}
		number, _ := strconv.Atoi(e.Name())
		held := ref.WithPull(number)
		rev, err := s.Read(held)
		if err != nil {
			return models.PullRef{}, nil, err
		}
		if rev == nil || rev.Moved != nil {
			continue
		}
		for _, m := range rev.Members {
			if m.Pull == ref.Pull {
				return held, rev, nil
			}
		}
	}
	rev, err := s.Read(ref)
	return ref, rev, err
}
