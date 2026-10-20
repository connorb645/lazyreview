package land

import (
	"fmt"
	"strconv"
)

func finish(branch string) string {
	return "make land BRANCH=" + branch
}

var noBranch = "say which branch to land: " + finish("<branch>")

const (
	uncommitted    = "this checkout has uncommitted changes"
	noRemote       = "this repository has no remote to land on"
	stepsHeading   = "It will:"
	failedFallback = "git failed"
)

func notOnMain(current string) string {
	if current == "" {
		current = "no branch"
	}
	return "this checkout is on " + current + ", not main"
}

func noFetch(remote, why string) string {
	return "could not fetch " + remote + ": " + why
}

func missing(branch, where string) string {
	return branch + " is not on " + where
}

func apart(branch, at string) string {
	return branch + " is at different commits: " + at
}

func noMain(where string) string {
	return "there is no main on " + where
}

func notForward(branch, where string) string {
	return "main " + where + " has commits " + branch + " lacks, so landing it would not be a fast-forward"
}

func wouldRefuse(remote, why string) string {
	return remote + " would not take the push: " + why
}

func landing(branch, commit string, adds int) string {
	if adds == 0 {
		return fmt.Sprintf("Landing %s at %s, which main here is at already.", branch, commit)
	}
	brings := "1 commit"
	if adds != 1 {
		brings = strconv.Itoa(adds) + " commits"
	}
	return fmt.Sprintf("Landing %s at %s brings %s to main:", branch, commit, brings)
}

func push(commit, remote string) string {
	return "  push " + commit + " to main on " + remote + ", without force"
}

func move(commit string) string {
	return "  move main here to " + commit + ", fast-forward only"
}

func refused(remote, why string) string {
	return remote + " refused the push, so nothing has changed: " + why
}

func refusedPartway(remote, why string) string {
	return remote + " refused the push: " + why
}

func remotesApart(commit, took, left string) string {
	return "the remotes are apart: main is at " + commit + " on " + took + ", but still at " + left
}

func notMoved(commit, why string) string {
	return "the remotes have main at " + commit + ", but main here could not move to it: " + why
}

func finishWith(branch string) string {
	return "fix that, then finish with: " + finish(branch)
}

func landed(commit, where string) string {
	return "main is at " + commit + " on " + where + "."
}
