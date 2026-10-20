package testsupport

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Play makes the repository by each step of a recorded script in turn, as
// test/support/throwaway-repo.mjs does, so it makes the same commits.
func (r *ThrowawayRepo) Play(steps []ScriptStep) {
	r.t.Helper()
	for _, step := range steps {
		switch {
		case step.Commit != nil:
			var branch, message string
			var files map[string]*string
			for k, into := range []any{&branch, &files, &message} {
				if err := json.Unmarshal(step.Commit[k], into); err != nil {
					r.t.Fatal(err)
				}
			}
			r.onBranch(branch)
			writes := map[string]string{}
			for path, text := range files {
				if text == nil {
					if err := os.Remove(filepath.Join(r.Work, path)); err != nil {
						r.t.Fatal(err)
					}
					continue
				}
				writes[path] = *text
			}
			r.Commit(branch, writes, message)
		case step.Checkout != "":
			r.Checkout(step.Checkout)
		default:
			r.Git(step.Git...)
		}
	}
}

func (r *ThrowawayRepo) onBranch(branch string) {
	if r.Git("symbolic-ref", "--short", "HEAD") == branch {
		return
	}
	if r.Git("branch", "--list", branch) != "" {
		r.Git("checkout", "-q", branch)
	} else {
		r.Git("checkout", "-q", "-b", branch)
	}
}
