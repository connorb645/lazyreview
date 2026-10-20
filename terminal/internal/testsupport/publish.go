package testsupport

import (
	"os"
	"path/filepath"
	"strconv"
)

// Publish pushes every branch to the repository's bare origin, made beside
// it the first time, points refs/pull/<n>/head at each pull request's
// branch, as GitHub serves a pull request's head, and returns the origin's
// path, the address a clone fetches from.
func (r *ThrowawayRepo) Publish(pulls map[int]string) string {
	r.t.Helper()
	origin := filepath.Join(filepath.Dir(r.Work), "origin.git")
	if _, err := os.Stat(origin); err != nil {
		RunGit(r.t, r.Work, "init", "-q", "--bare", origin)
	}
	RunGit(r.t, r.Work, "push", "-q", "--force", origin, "--all")
	for number, branch := range pulls {
		RunGit(r.t, origin, "update-ref", "refs/pull/"+strconv.Itoa(number)+"/head", r.Git("rev-parse", branch))
	}
	return origin
}
