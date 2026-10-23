package github_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/connorb645/lazyreview/internal/commands/github"
	"github.com/connorb645/lazyreview/internal/i18n"
)

func TestAnAccountsTokenIsLookedUpOnceWhenSearchesRunAtOnce(t *testing.T) {
	var lookups atomic.Int32
	run := func(args []string, _ []byte, _ []string) github.Ran {
		if len(args) > 1 && args[0] == "auth" && args[1] == "token" {
			lookups.Add(1)
			time.Sleep(20 * time.Millisecond)
			return github.Ran{Stdout: "token-sam\n"}
		}
		return github.Ran{}
	}
	g := github.New(i18n.EnglishTranslationSet(), "github.com", "sam", run)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if env, err := g.Environment(); err != nil || len(env) != 1 || env[0] != "GH_TOKEN=token-sam" {
				t.Errorf("the environment is %v, %v", env, err)
			}
		}()
	}
	wg.Wait()
	if n := lookups.Load(); n != 1 {
		t.Errorf("8 searches at once looked the token up %d times", n)
	}
}
