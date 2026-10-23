package page

import (
	"slices"
	"strings"

	"github.com/connorb645/lazyreview/internal/build/diff"
	"github.com/connorb645/lazyreview/internal/build/layout"
	"github.com/connorb645/lazyreview/internal/build/model"
	"github.com/connorb645/lazyreview/internal/build/stack"
	"github.com/connorb645/lazyreview/internal/build/unchanged"
	"github.com/connorb645/lazyreview/internal/commands/git"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

// Build reads the change from the merge base of the request's base and its
// top member's head to that head, as a pull request shows it, and builds
// its page. One member is a single pull request or a branch; more are a
// stack of pull requests or of branches, each based on the one below, and
// the page also says which member changed each line and, for pull
// requests, where a comment on it goes. It only reads the repository.
func Build(tr *i18n.TranslationSet, repo *git.Repository, r Request) (Built, error) {
	var built Built
	pulls := make([]int, len(r.Members))
	for i, m := range r.Members {
		pulls[i] = m.Pull
	}
	read, err := diff.ReadDiff(tr, repo, r.Base, r.Members[len(r.Members)-1].Head, false)
	if err != nil {
		return built, err
	}
	var steps stack.Steps
	var mapped stack.Mapped
	stacked := len(r.Members) > 1
	if stacked {
		members := make([]stack.Member, len(r.Members))
		for i, m := range r.Members {
			members[i] = stack.Member{Pull: m.Pull, Branch: m.Branch, Head: m.Head}
		}
		if steps, err = stack.ReadSteps(tr, repo, read.MergeBase, read.BaseCommit, read.HeadCommit, members); err != nil {
			return built, err
		}
		mapped = stack.MapLines(tr, steps.Steps, read.Chunks, pulls)
		for i, f := range read.Files {
			owners := mapped.Files[f.Path]
			read.Files[i].Pulls, read.Files[i].ByPull = owners.Pulls, owners.ByPull
		}
	}

	changed := model.ReadChangedTypes(tr, read.Files)
	paths := make([]string, len(read.Files))
	for i, f := range read.Files {
		paths[i] = f.Path
	}
	found := unchanged.Found{Texts: map[string]*string{}}
	if r.Context {
		if found, err = unchanged.Candidates(repo, read.HeadCommit, changed.Nodes, paths, r.Config.ContextTypes.Include); err != nil {
			return built, err
		}
	}
	same, context := model.ChooseContext(tr, changed.Nodes, changed.Untouched, found.Nodes, model.ContextOptions{
		Enabled: r.Context, Max: r.Config.ContextTypes.Max, Include: r.Config.ContextTypes.Include, Exclude: r.Config.ContextTypes.Exclude,
	})
	extended := unchanged.Found{Texts: map[string]*string{}}
	if r.Context {
		skip := map[string]bool{}
		for _, p := range paths {
			skip[p] = true
		}
		for _, n := range context {
			skip[n.File] = true
		}
		if extended, err = unchanged.Extensions(tr, repo, read.HeadCommit, context, skip); err != nil {
			return built, err
		}
	}
	kept, err := keptFiles(repo, read.HeadCommit, r.Keep, paths, slices.Concat(changed.Nodes, same, context, extended.Nodes))
	if err != nil {
		return built, err
	}
	drawn := slices.Concat(changed.Nodes, same, context, extended.Nodes, kept.Nodes)
	if stacked {
		drawn = withPulls(drawn, read.Chunks, mapped, pulls)
	}
	m := model.BuildModel(tr, read.Files, drawn, changed.Imports, r.Config, r.RootTitle)
	m.Heading = models.Heading{Title: r.Title, Head: read.HeadCommit, Base: read.MergeBase}
	if r.Config.Title != "" {
		m.Heading.Title = r.Config.Title
	}
	if r.Config.Summary != "" {
		m.Heading.Context = []models.HeadingPart{{Text: r.Config.Summary}}
	} else {
		commits, err := repo.CommitCount(read.MergeBase, read.HeadCommit)
		if err != nil {
			return built, err
		}
		m.Heading.Context = DescribeChange(tr, r.Subject, r.BaseBranch, read.HeadCommit, read.MergeBase, commits)
	}

	sources, left := ChangedTexts(read.Files)
	built.Page = models.PageRecord{Model: m, Layout: layout.Of(tr, m), Diff: map[string]string{}, Sources: sources, Context: map[string][]string{}}
	for _, c := range read.Chunks {
		built.Page.Diff[c.Path] = c.Text
	}
	for _, f := range m.ContextFiles {
		text := found.Texts[f.Path]
		if text == nil || *text == "" {
			text = extended.Texts[f.Path]
		}
		if text == nil || *text == "" {
			text = kept.Texts[f.Path]
		}
		built.Page.Context[f.Path] = []string{}
		if text != nil && *text != "" {
			built.Page.Context[f.Path] = Lines(*text)
		}
	}
	if len(left) > 0 {
		built.Notes = append(built.Notes, utils.ResolvePlaceholderString(tr.LoaderLeftOut, map[string]string{
			"files": utils.Plural(len(left), tr.LoaderLargeFile), "paths": strings.Join(left, ", "),
		}))
	}
	built.Starts = []string{read.MergeBase}
	if stacked {
		record := &models.StackRecord{Lines: mapped.Lines, Warnings: orEmpty(steps.Warnings)}
		built.Starts = nil
		for i, s := range steps.Steps {
			c := steps.Counts[i]
			record.Members = append(record.Members, models.StackMember{Pull: s.Pull, Branch: s.Branch, Head: steps.Heads[i], Files: c.Files, Adds: c.Adds, Dels: c.Dels})
			built.Starts = append(built.Starts, s.Start)
		}
		built.Page.Stack = record
		built.Notes = append(built.Notes, steps.Warnings...)
	}
	built.Counts = Counts{Files: len(m.Files), Components: len(m.Components)}
	for _, f := range m.Files {
		built.Counts.Adds += f.Adds
		built.Counts.Dels += f.Dels
	}
	for _, n := range m.Nodes {
		if n.Kind != "artifact" {
			built.Counts.Types++
		}
	}
	return built, nil
}

func keptFiles(repo *git.Repository, head string, keep, changed []string, drawn []*model.Node) (unchanged.Found, error) {
	kept := unchanged.Found{Texts: map[string]*string{}}
	var missing []string
	for _, path := range keep {
		drawnAlready := slices.Contains(changed, path) || slices.ContainsFunc(drawn, func(n *model.Node) bool { return n.File == path })
		if !drawnAlready && !slices.Contains(missing, path) {
			missing = append(missing, path)
		}
	}
	texts, err := repo.Read(head, missing)
	if err != nil {
		return kept, err
	}
	for _, path := range missing {
		if texts[path] == nil {
			continue
		}
		kept.Texts[path] = texts[path]
		kept.Nodes = append(kept.Nodes, &model.Node{
			Name: path[strings.LastIndex(path, "/")+1:], Kind: "artifact", File: path, Line: 1, State: "context",
			Inherits: []string{}, Properties: []models.Property{}, Nested: []string{}, ExtFiles: []string{}, Extends: []string{},
		})
	}
	return kept, nil
}

func withPulls(drawn []*model.Node, chunks []diff.Chunk, mapped stack.Mapped, pulls []int) []*model.Node {
	hunks := map[string][]stack.HunkLines{}
	for _, c := range chunks {
		hunks[c.Path] = stack.ReadHunkLines(c.Text)
	}
	out := make([]*model.Node, len(drawn))
	for i, n := range drawn {
		copied := *n
		out[i] = &copied
		copied.Pulls = []int{}
		if n.State == "same" || n.State == "context" {
			continue
		}
		found := map[int]bool{}
		if n.Kind == "artifact" || n.Kind == "extension" || n.State == "removed" || n.End == 0 {
			for _, p := range mapped.Files[n.File].Pulls {
				found[p] = true
			}
		} else if lines, held := mapped.Lines[n.File]; held {
			found = stack.ChangedInSpan(hunks[n.File], lines, n.Line, n.End)
		}
		for _, path := range n.ExtFiles {
			for _, p := range mapped.Files[path].Pulls {
				found[p] = true
			}
		}
		for _, p := range pulls {
			if found[p] {
				copied.Pulls = append(copied.Pulls, p)
			}
		}
	}
	return out
}

func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}
