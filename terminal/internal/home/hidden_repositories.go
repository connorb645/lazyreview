package home

import (
	"errors"
	"sync"

	"github.com/connorb645/lazyreview/terminal/internal/config"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const hiddenKey = "github.hiddenRepositories"

// HiddenRepositories changes the repositories the home screen leaves out:
// the config's github.hiddenRepositories, read at each change and changed
// in the file in place. One change runs at a time.
type HiddenRepositories struct {
	tr     *i18n.TranslationSet
	config func() (config.Read, error)
	mu     sync.Mutex
}

// NewHiddenRepositories changes the list in the config read by read.
func NewHiddenRepositories(tr *i18n.TranslationSet, read func() (config.Read, error)) *HiddenRepositories {
	return &HiddenRepositories{tr: tr, config: read}
}

// Change puts the entry hide on the list, a repository as owner/name or
// every repository of an owner as owner/*, unless it is "", and takes each
// entry of show off it, in the file the config was read from. It returns
// the file's path, and each repository named in repos with the entries that
// now hide it.
func (h *HiddenRepositories) Change(hide string, show, repos []string) (models.Hidden, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	change := config.ListChange{Add: []string{}, Remove: show}
	if hide != "" {
		if !config.IsRepository(hide) {
			return models.Hidden{}, errors.New(utils.ResolvePlaceholderString(h.tr.ConfigNotAnEntry, map[string]string{"value": config.Shown(hide)}))
		}
		change.Add = []string{hide}
	}
	before, err := h.config()
	if err != nil {
		return models.Hidden{}, err
	}
	if err := config.SaveListChange(h.tr, before.Path, hiddenKey, change); err != nil {
		return models.Hidden{}, err
	}
	read, err := h.config()
	if err != nil {
		return models.Hidden{}, err
	}
	entries := read.Values["github"].(map[string]any)["hiddenRepositories"].([]string)
	out := models.Hidden{Config: read.Path, Repos: []models.HiddenRepo{}}
	for _, name := range repos {
		out.Repos = append(out.Repos, models.HiddenRepo{Name: name, HiddenBy: HiddenBy(entries, name)})
	}
	return out, nil
}
