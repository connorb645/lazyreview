package land

import (
	"errors"
	"strings"
)

// Landing lands a branch on main as a fast-forward, on every remote and in
// one checkout, running git through its Run.
type Landing struct {
	repo string
	run  Run
}

// NewLanding works on the checkout at repo, running git through run, or
// git itself when run is nil.
func NewLanding(repo string, run Run) *Landing {
	if run == nil {
		run = RunGit
	}
	return &Landing{repo: repo, run: run}
}

// Plan fetches every remote, checks the checkout and each remote, and
// returns what landing branch takes, or an error of one plain line saying
// why it cannot land. The checkout must be on main with nothing
// uncommitted, the branch at one commit on every remote, and main, here
// and on each remote, behind it.
func (l *Landing) Plan(branch string) (Plan, error) {
	if branch == "" {
		return Plan{}, errors.New(noBranch)
	}
	if current := l.git("symbolic-ref", "--quiet", "--short", "HEAD").Out; current != "main" {
		return Plan{}, errors.New(notOnMain(current))
	}
	if l.git("--no-optional-locks", "status", "--porcelain").Out != "" {
		return Plan{}, errors.New(uncommitted)
	}
	names := lines(l.git("remote").Out)
	if len(names) == 0 {
		return Plan{}, errors.New(noRemote)
	}
	for _, name := range names {
		if fetched := l.git("fetch", "--prune", "--quiet", name); !fetched.OK {
			return Plan{}, errors.New(noFetch(name, reason(fetched)))
		}
	}

	remotes := make([]Remote, len(names))
	branches := make([]string, len(names))
	var lacking, mainless []string
	for i, name := range names {
		branches[i] = l.commit("refs/remotes/" + name + "/" + branch)
		remotes[i] = Remote{Name: name, Main: l.commit("refs/remotes/" + name + "/main")}
		if branches[i] == "" {
			lacking = append(lacking, name)
		}
		if remotes[i].Main == "" {
			mainless = append(mainless, name)
		}
	}
	if len(lacking) > 0 {
		return Plan{}, errors.New(missing(branch, list(lacking)))
	}
	commit := branches[0]
	for _, at := range branches {
		if at != commit {
			parts := make([]string, len(names))
			for i, name := range names {
				parts[i] = l.short(branches[i]) + " on " + name
			}
			return Plan{}, errors.New(apart(branch, strings.Join(parts, ", ")))
		}
	}
	if len(mainless) > 0 {
		return Plan{}, errors.New(noMain(list(mainless)))
	}
	here := l.commit("refs/heads/main")
	var ahead []string
	for _, r := range remotes {
		if !l.behind(r.Main, commit) {
			ahead = append(ahead, "on "+r.Name)
		}
	}
	if !l.behind(here, commit) {
		ahead = append(ahead, "here")
	}
	if len(ahead) > 0 {
		return Plan{}, errors.New(notForward(branch, list(ahead)))
	}
	for _, r := range remotes {
		if r.Main == commit {
			continue
		}
		if tried := l.git("push", "--dry-run", "--porcelain", r.Name, commit+":refs/heads/main"); !tried.OK {
			return Plan{}, errors.New(wouldRefuse(r.Name, reason(tried)))
		}
	}

	adds := lines(l.git("log", "--format=%h %s", here+".."+commit).Out)
	return Plan{Branch: branch, Commit: commit, Short: l.short(commit), Adds: adds, Remotes: remotes, Here: here}, nil
}

// Land pushes the plan's commit to main on each remote that lacks it, in
// turn, then moves main here to it, fast-forward only, and returns the
// line saying where main is. When a step fails after another remote took
// the push, the error's lines say which remotes are apart and the one
// command that finishes the job.
func (l *Landing) Land(plan Plan) (string, error) {
	var pushes []Remote
	for _, r := range plan.Remotes {
		if r.Main != plan.Commit {
			pushes = append(pushes, r)
		}
	}
	for i, remote := range pushes {
		pushed := l.git("push", "--porcelain", remote.Name, plan.Commit+":refs/heads/main")
		if pushed.OK {
			continue
		}
		left := pushes[i:]
		var took []string
		for _, r := range plan.Remotes {
			if !holds(left, r.Name) {
				took = append(took, r.Name)
			}
		}
		if len(took) == 0 {
			return "", errors.New(refused(remote.Name, reason(pushed)))
		}
		stillAt := make([]string, len(left))
		for j, r := range left {
			stillAt[j] = l.short(r.Main) + " on " + r.Name
		}
		return "", errors.New(strings.Join([]string{
			refusedPartway(remote.Name, reason(pushed)),
			remotesApart(plan.Short, list(took), strings.Join(stillAt, ", ")),
			finishWith(plan.Branch),
		}, "\n"))
	}
	if plan.Here != plan.Commit {
		if moved := l.git("merge", "--ff-only", "--quiet", plan.Commit); !moved.OK {
			return "", errors.New(notMoved(plan.Short, reason(moved)) + "\n" + finishWith(plan.Branch))
		}
	}
	where := make([]string, 0, len(plan.Remotes)+1)
	for _, r := range plan.Remotes {
		where = append(where, r.Name)
	}
	return landed(l.short(l.commit("refs/heads/main")), list(append(where, "here"))), nil
}

func (l *Landing) git(args ...string) Ran {
	return l.run(l.repo, args)
}

func (l *Landing) commit(ref string) string {
	if found := l.git("rev-parse", "--verify", "--quiet", ref+"^{commit}"); found.OK {
		return found.Out
	}
	return ""
}

func (l *Landing) short(commit string) string {
	return l.git("rev-parse", "--short", commit).Out
}

func (l *Landing) behind(main, commit string) bool {
	return l.git("merge-base", "--is-ancestor", main, commit).OK
}

func holds(remotes []Remote, name string) bool {
	for _, r := range remotes {
		if r.Name == name {
			return true
		}
	}
	return false
}

func lines(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
