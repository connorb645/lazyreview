package home

import "github.com/connorb645/lazyreview/terminal/internal/models"

// Stacks finds the stacks among the open pull requests of repo, owner/name,
// as GitHub's list gives them: each one's number, with the pull requests it
// is reviewed with, bottom first. A stack is a chain of two or more, each
// based on the head branch of the one below, where every branch is in repo
// itself. A chain that forks, with two pull requests based on the same
// branch, is no stack: each of its pull requests is reviewed alone, as is
// one with nothing stacked on or under it.
func Stacks(pulls []models.GitHubPull, repo string) map[int][]models.GitHubPull {
	byHead := map[string]models.GitHubPull{}
	for _, p := range pulls {
		if p.Head.Repo != nil && p.Head.Repo.FullName == repo {
			byHead[p.Head.Ref] = p
		}
	}
	above := map[int][]models.GitHubPull{}
	for _, p := range pulls {
		if below, ok := byHead[p.Base.Ref]; ok && below.Number != p.Number {
			above[below.Number] = append(above[below.Number], p)
		}
	}
	out := map[int][]models.GitHubPull{}
	for _, p := range pulls {
		bottom := p
		seen := map[int]bool{p.Number: true}
		for {
			below, ok := byHead[bottom.Base.Ref]
			if !ok || seen[below.Number] {
				break
			}
			bottom = below
			seen[bottom.Number] = true
		}
		var group []models.GitHubPull
		in := map[int]bool{}
		for todo := []models.GitHubPull{bottom}; len(todo) > 0; {
			next := todo[0]
			todo = todo[1:]
			if in[next.Number] {
				continue
			}
			in[next.Number] = true
			group = append(group, next)
			todo = append(todo, above[next.Number]...)
		}
		forked := false
		for _, g := range group {
			forked = forked || len(above[g.Number]) > 1
		}
		if len(group) > 1 && !forked {
			out[p.Number] = group
		} else {
			out[p.Number] = []models.GitHubPull{p}
		}
	}
	return out
}
