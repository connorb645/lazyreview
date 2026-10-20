package testsupport

import (
	"strconv"
	"strings"
)

// GitHubRoutes are the answers GitHub would give, as octo/widgets, to the
// account reviewer, about repo's pull requests as they now stand: the
// repository, with origin as its clone address; the account; the list of
// open pull requests; and for each pull request its details at its head,
// its files with their patches from the merge base, no reviews or review
// comments, and the review threads threads gives it, as GitHub's GraphQL
// API gives them.
func GitHubRoutes(repo *ThrowawayRepo, origin string, pulls []WorkbenchPull, threads map[int][]any) Routes {
	const name = "octo/widgets"
	var routes Routes
	routes.Set("GET repos/"+name, map[string]any{"full_name": name, "clone_url": origin})
	routes.Set("GET user", map[string]any{"login": "reviewer"})
	open := []any{}
	for _, p := range pulls {
		head, base := repo.Git("rev-parse", p.Head), repo.Git("rev-parse", p.Base)
		start := repo.Git("merge-base", base, head)
		number := strconv.Itoa(p.Number)
		detail := map[string]any{
			"number": p.Number, "title": p.Title, "body": p.Body, "html_url": "https://github.com/" + name + "/pull/" + number,
			"state": "open", "merged_at": nil, "closed_at": nil, "user": map[string]any{"login": p.Author}, "draft": false,
			"base": map[string]any{"ref": p.Base, "sha": base, "repo": map[string]any{"full_name": name}},
			"head": map[string]any{"ref": p.Head, "sha": head, "repo": map[string]any{"full_name": name}},
		}
		routes.Set("GET repos/"+name+"/pulls/"+number, detail)
		open = append(open, detail)
		routes.Set("GET repos/"+name+"/pulls/"+number+"/files?per_page=100&page=1", patches(repo.Git("diff", "--no-color", "--find-renames", start, head)))
		routes.Set("GET repos/"+name+"/pulls/"+number+"/reviews?per_page=100&page=1", []any{})
		routes.Set("GET repos/"+name+"/pulls/"+number+"/comments?per_page=100&page=1", []any{})
		nodes := threads[p.Number]
		if nodes == nil {
			nodes = []any{}
		}
		routes.Set("POST graphql "+name+"#"+number, map[string]any{"data": map[string]any{"repository": map[string]any{"pullRequest": map[string]any{
			"reviewThreads": map[string]any{"pageInfo": map[string]any{"hasNextPage": false, "endCursor": nil}, "nodes": nodes},
		}}}})
	}
	routes.Set("GET repos/"+name+"/pulls?state=open&per_page=100&page=1", open)
	return routes
}

func patches(diff string) []any {
	type file struct {
		name  string
		lines []string
	}
	var files []*file
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			parts := strings.Split(line, " b/")
			files = append(files, &file{name: parts[len(parts)-1]})
		case len(files) > 0:
			f := files[len(files)-1]
			if name, ok := strings.CutPrefix(line, "+++ b/"); ok {
				f.name = name
			}
			f.lines = append(f.lines, line)
		}
	}
	out := []any{}
	for _, f := range files {
		for len(f.lines) > 0 && f.lines[len(f.lines)-1] == "" {
			f.lines = f.lines[:len(f.lines)-1]
		}
		entry := map[string]any{"filename": f.name}
		for i, line := range f.lines {
			if strings.HasPrefix(line, "@@") {
				entry["patch"] = strings.Join(f.lines[i:], "\n")
				break
			}
		}
		out = append(out, entry)
	}
	return out
}
