package home

import (
	"errors"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/connorb645/lazyreview/terminal/internal/commands/github"
	"github.com/connorb645/lazyreview/terminal/internal/config"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const (
	open = "is:pr is:open archived:false"
	host = "github.com"
)

// OpenPulls lists the open pull requests that involve the config's
// accounts. A pull request involves an account when its review is asked of
// the account, or of one of its teams, while it is ready for review; when
// the account opened it; or when the account commented on it or reviewed
// it. A pull request in a stack is listed with the whole stack, bottom
// first, as one unit, even its members that involve no account; any other
// is a unit alone. A repository is read as the first account that found
// something in it, and its units come by their lowest number.
type OpenPulls struct {
	tr     *i18n.TranslationSet
	store  *store.ReviewStore
	config func() (config.Read, error)
	github func(host, account string) *github.GitHub
}

// NewOpenPulls lists from the config read by read at each list, asking
// GitHub as each account by its own token through gh, and naming the
// review of reviews that holds each pull request.
func NewOpenPulls(tr *i18n.TranslationSet, reviews *store.ReviewStore, read func() (config.Read, error), gh func(host, account string) *github.GitHub) *OpenPulls {
	return &OpenPulls{tr: tr, store: reviews, config: read, github: gh}
}

// List is the config file's path, each account with why GitHub could not be
// read as it, if it could not, and each repository with pull requests that
// involve an account, by name, with the entries of the config's hidden
// repositories that hide it. While the data home still holds
// settings.json, which the config replaced, it fails, saying how to move it.
func (o *OpenPulls) List() (models.Pulls, error) {
	read, err := o.config()
	if err != nil {
		return models.Pulls{}, err
	}
	settings := o.store.Home().SettingsFile()
	if _, err := os.Stat(settings); err == nil {
		return models.Pulls{}, errors.New(utils.ResolvePlaceholderString(o.tr.ConfigNotMoved, map[string]string{"settings": settings, "config": read.Path}))
	}
	gh := read.Values["github"].(map[string]any)
	accounts := gh["accounts"].([]config.Account)
	hidden := gh["hiddenRepositories"].([]string)
	found := make([]accountFound, len(accounts))
	var wait sync.WaitGroup
	for i, account := range accounts {
		wait.Add(1)
		go func() {
			defer wait.Done()
			repos, err := o.read(account)
			found[i] = accountFound{login: account.Login, err: err, repos: repos}
		}()
	}
	wait.Wait()
	byName := map[string]*foundRepo{}
	var names []string
	for _, f := range found {
		for _, repo := range f.repos {
			kept, held := byName[repo.name]
			if !held {
				byName[repo.name] = repo
				names = append(names, repo.name)
				continue
			}
			for _, number := range repo.numbers {
				if _, had := kept.involved[number]; !had {
					kept.numbers = append(kept.numbers, number)
				}
				kept.involved[number] = kept.involved[number].joined(repo.involved[number])
			}
		}
	}
	sort.SliceStable(names, func(a, b int) bool { return beforeByName(names[a], names[b]) })
	out := models.Pulls{Config: read.Path, Accounts: []models.AccountRead{}, Repos: []models.PullRepo{}}
	for _, f := range found {
		account := models.AccountRead{Login: f.login}
		if f.err != nil {
			account.Error = f.err.Error()
		}
		out.Accounts = append(out.Accounts, account)
	}
	for _, name := range names {
		repo := byName[name]
		units, err := o.units(repo)
		if err != nil {
			return models.Pulls{}, err
		}
		out.Repos = append(out.Repos, models.PullRepo{Host: host, Name: name, Account: repo.account, HiddenBy: HiddenBy(hidden, name), Units: units})
	}
	return out, nil
}

func (o *OpenPulls) read(account config.Account) ([]*foundRepo, error) {
	gh := o.github(host, account.Login)
	searches := []search{{open + " draft:false user-review-requested:@me", func(i *involved) { i.asked = append(i.asked, account.Login) }}}
	for _, team := range account.Teams {
		query := open + " draft:false team-review-requested:" + team.Name
		if team.Since != nil {
			query += " created:>=" + *team.Since
		}
		searches = append(searches, search{query, func(i *involved) { i.asked = append(i.asked, team.Name) }})
	}
	searches = append(searches,
		search{open + " author:@me", func(i *involved) { i.yours = true }},
		search{open + " commenter:@me", func(i *involved) { i.commented = true }},
		search{open + " reviewed-by:@me", func(i *involved) { i.commented = true }},
	)
	items := make([][]models.GitHubIssue, len(searches))
	failures := make([]error, len(searches))
	var wait sync.WaitGroup
	for i, s := range searches {
		wait.Add(1)
		go func() {
			defer wait.Done()
			items[i], failures[i] = gh.Search(s.query)
		}()
	}
	wait.Wait()
	for _, f := range failures {
		if f != nil {
			return nil, f
		}
	}
	var repos []*foundRepo
	byName := map[string]*foundRepo{}
	for i, found := range items {
		for _, item := range found {
			parts := strings.Split(item.RepositoryURL, "/")
			name := strings.Join(parts[max(0, len(parts)-2):], "/")
			repo, held := byName[name]
			if !held {
				repo = &foundRepo{name: name, account: account.Login, involved: map[int]involved{}}
				byName[name] = repo
				repos = append(repos, repo)
			}
			if _, had := repo.involved[item.Number]; !had {
				repo.numbers = append(repo.numbers, item.Number)
				repo.involved[item.Number] = involved{asked: []string{}}
			}
			marked := repo.involved[item.Number]
			searches[i].mark(&marked)
			repo.involved[item.Number] = marked
		}
	}
	failures = make([]error, len(repos))
	for i, repo := range repos {
		wait.Add(1)
		go func() {
			defer wait.Done()
			repo.pulls, failures[i] = gh.OpenPulls(repo.name)
		}()
	}
	wait.Wait()
	for _, f := range failures {
		if f != nil {
			return nil, f
		}
	}
	return repos, nil
}

func (o *OpenPulls) units(repo *foundRepo) ([]models.Unit, error) {
	openNow := map[int]bool{}
	for _, p := range repo.pulls {
		openNow[p.Number] = true
	}
	grouped := Stacks(repo.pulls, repo.name)
	numbers := append([]int{}, repo.numbers...)
	sort.Ints(numbers)
	var keys []string
	units := map[string][]models.OpenPull{}
	for _, number := range numbers {
		if !openNow[number] {
			continue
		}
		members := grouped[number]
		var parts []string
		for _, m := range members {
			parts = append(parts, strconv.Itoa(m.Number))
		}
		key := strings.Join(parts, "+")
		if _, held := units[key]; held {
			continue
		}
		for _, m := range members {
			pull, err := o.pull(repo, m)
			if err != nil {
				return nil, err
			}
			units[key] = append(units[key], pull)
		}
		keys = append(keys, key)
	}
	sort.SliceStable(keys, func(a, b int) bool { return units[keys[a]][0].Number < units[keys[b]][0].Number })
	out := []models.Unit{}
	for _, key := range keys {
		members := units[key]
		review := members[0].Review
		for _, m := range members {
			if m.Review != review {
				review = ""
			}
		}
		out = append(out, models.Unit{Review: review, Members: members})
	}
	return out, nil
}

func (o *OpenPulls) pull(repo *foundRepo, p models.GitHubPull) (models.OpenPull, error) {
	how, held := repo.involved[p.Number]
	if !held {
		how = involved{asked: []string{}}
	}
	ref, err := models.NewPullRef(o.tr, host, repo.name, p.Number)
	if err != nil {
		return models.OpenPull{}, err
	}
	at, rev, err := o.store.Find(ref)
	if err != nil {
		return models.OpenPull{}, err
	}
	review := ""
	if rev != nil && rev.Moved == nil {
		review = at.ID()
	}
	description := strings.ReplaceAll(strings.ReplaceAll(p.Body, "\r\n", "\n"), "\r", "\n")
	return models.OpenPull{Number: p.Number, Title: p.Title, Author: p.User.Login, URL: p.HTMLURL, Draft: p.Draft, Base: p.Base.Ref, Head: p.Head.Ref,
		Description: description, Asked: how.asked, Yours: how.yours, Commented: how.commented, Review: review}, nil
}
