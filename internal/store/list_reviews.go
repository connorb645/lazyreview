package store

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

// List is every review the data home keeps, of branches and of pull
// requests, moved or not, the newest load first.
func (s *ReviewStore) List() ([]KeptReview, error) {
	branches, err := s.branchReviews()
	if err != nil {
		return nil, err
	}
	pulls, err := s.pullReviews()
	if err != nil {
		return nil, err
	}
	kept := append(branches, pulls...)
	sort.SliceStable(kept, func(i, j int) bool {
		return utils.LocaleCompare(kept[i].Review.LoadedAt, kept[j].Review.LoadedAt) > 0
	})
	return kept, nil
}

func (s *ReviewStore) branchReviews() ([]KeptReview, error) {
	keys, err := folders(s.home.BranchReviews())
	if err != nil {
		return nil, err
	}
	var kept []KeptReview
	for _, key := range keys {
		branches, err := folders(filepath.Join(s.home.BranchReviews(), key))
		if err != nil {
			return nil, err
		}
		for _, branch := range branches {
			ref, err := models.ParseBranchRef(s.tr, "local/"+key+"/"+branch)
			if err != nil {
				return nil, err
			}
			rev, err := s.Read(ref)
			if err != nil {
				return nil, err
			}
			if rev != nil {
				kept = append(kept, KeptReview{Ref: ref, Review: *rev})
			}
		}
	}
	return kept, nil
}

func (s *ReviewStore) pullReviews() ([]KeptReview, error) {
	hosts, err := folders(s.home.Reviews())
	if err != nil {
		return nil, err
	}
	var kept []KeptReview
	for _, host := range hosts {
		if filepath.Join(s.home.Reviews(), host) == s.home.BranchReviews() {
			continue
		}
		owners, err := folders(filepath.Join(s.home.Reviews(), host))
		if err != nil {
			return nil, err
		}
		for _, owner := range owners {
			names, err := folders(filepath.Join(s.home.Reviews(), host, owner))
			if err != nil {
				return nil, err
			}
			for _, name := range names {
				held, err := s.InRepository(host, owner+"/"+name)
				if err != nil {
					return nil, err
				}
				for _, h := range held {
					kept = append(kept, KeptReview{Ref: h.Ref, Review: h.Review})
				}
			}
		}
	}
	return kept, nil
}

func folders(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
