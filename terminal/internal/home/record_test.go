package home

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/commands/github"
	"github.com/connorb645/lazyreview/terminal/internal/config"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/store"
	"github.com/connorb645/lazyreview/terminal/internal/testsupport"
)

const (
	standInConfig = "/home/reviewer/Library/Application Support/lazyreview/config.yml"
	standInUser   = "/home/reviewer"
)

type recordedHomes struct {
	Lists []struct {
		Key     string                     `json:"key"`
		Config  *string                    `json:"config"`
		Routes  map[string]json.RawMessage `json:"routes"`
		Reviews []string                   `json:"reviews"`
	} `json:"lists"`
	Changes []struct {
		Key    string  `json:"key"`
		Before *string `json:"before"`
		After  *string `json:"after"`
	} `json:"changes"`
}

func recorded(t *testing.T) (recordedHomes, map[string]json.RawMessage) {
	t.Helper()
	var homes recordedHomes
	var answers map[string]struct {
		Body json.RawMessage `json:"body"`
	}
	for name, into := range map[string]any{"homes.json": &homes, "record.json": &answers} {
		text, err := os.ReadFile(testsupport.Testdata(name))
		if err == nil {
			err = json.Unmarshal(text, into)
		}
		if err != nil {
			t.Fatalf("could not read %s: %v", name, err)
		}
	}
	bodies := map[string]json.RawMessage{}
	for k, a := range answers {
		bodies[k] = a.Body
	}
	return homes, bodies
}

func same(t *testing.T, got any, recorded json.RawMessage, as any) bool {
	t.Helper()
	if err := json.Unmarshal(recorded, as); err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(got)
	b, _ := json.Marshal(as)
	var x, y any
	_ = json.Unmarshal(a, &x)
	_ = json.Unmarshal(b, &y)
	return reflect.DeepEqual(x, y)
}

// Each home screen list the server recorded is the one the Go code lists,
// from the same config, the same answers of GitHub and the same reviews.
func TestEachHomeScreenListIsTheOneTheServerRecorded(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	homes, bodies := recorded(t)
	if len(homes.Lists) != 8 {
		t.Fatalf("the record holds %d lists, not 8", len(homes.Lists))
	}
	for _, list := range homes.Lists {
		text := ""
		if list.Config != nil {
			text = *list.Config
		}
		reviews := store.NewReviewStore(tr, store.DataHome{Root: testsupport.StandInHome(t, list.Reviews...)}, store.NewLocks(tr))
		gh := &testsupport.FakeGh{Routes: list.Routes}
		read := func() (config.Read, error) { return config.ConfigFrom(tr, text, standInConfig, standInUser) }
		pulls, err := NewOpenPulls(tr, reviews, read, func(host, account string) *github.GitHub { return github.New(tr, host, account, gh.Run) }).List()
		if err != nil {
			t.Errorf("%s: %v", list.Key, err)
			continue
		}
		if !same(t, pulls, bodies[list.Key], &models.Pulls{}) {
			got, _ := json.Marshal(pulls)
			t.Errorf("%s lists\n%s\nnot\n%s", list.Key, got, bodies[list.Key])
		}
	}
}

// Each change to the hidden repositories the server recorded edits the
// config file in Go to the same text, and answers the same.
func TestEachChangeToTheHiddenRepositoriesIsTheOneTheServerRecorded(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	homes, bodies := recorded(t)
	if len(homes.Changes) != 6 {
		t.Fatalf("the record holds %d changes, not 6", len(homes.Changes))
	}
	for _, change := range homes.Changes {
		path := filepath.Join(t.TempDir(), "config.yml")
		if change.Before != nil {
			if err := os.WriteFile(path, []byte(*change.Before), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		read := func() (config.Read, error) {
			text, _ := os.ReadFile(path)
			return config.ConfigFrom(tr, string(text), path, standInUser)
		}
		var body struct {
			Hide  string   `json:"hide"`
			Show  []string `json:"show"`
			Repos []string `json:"repos"`
		}
		_, fields, _ := strings.Cut(change.Key, " {")
		if err := json.Unmarshal([]byte("{"+fields), &body); err != nil {
			t.Fatal(err)
		}
		hidden, err := NewHiddenRepositories(tr, read).Change(body.Hide, body.Show, body.Repos)
		var refused struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(bodies[change.Key], &refused)
		if refused.Error != "" || err != nil {
			if err == nil || strings.Replace(err.Error(), path, standInConfig, 1) != refused.Error {
				t.Errorf("%s gave %v, not the refusal %q", change.Key, err, refused.Error)
			}
			continue
		}
		hidden.Config = strings.Replace(hidden.Config, path, standInConfig, 1)
		after, _ := os.ReadFile(path)
		if !same(t, hidden, bodies[change.Key], &models.Hidden{}) || change.After == nil || string(after) != *change.After {
			t.Errorf("%s answers %+v and leaves\n%s\nnot\n%s", change.Key, hidden, after, *change.After)
		}
	}
}
