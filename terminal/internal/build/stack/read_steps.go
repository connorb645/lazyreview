package stack

import (
	"github.com/connorb645/lazyreview/terminal/internal/build/diff"
	"github.com/connorb645/lazyreview/terminal/internal/commands/git"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// ReadSteps reads, through repo, the diffs that tie each of members, bottom
// first, to the stack's combined diff, from combinedBase to top. The bottom
// member starts where its head leaves baseCommit; each one above, where its
// head leaves the head of the one below, with a warning when that is not
// the head below.
func ReadSteps(tr *i18n.TranslationSet, repo *git.Repository, combinedBase, baseCommit, top string, members []Member) (Steps, error) {
	read := Steps{Warnings: []string{}}
	for _, m := range members {
		head, err := repo.Resolve(m.Head)
		if err != nil {
			return Steps{}, models.Refused(utils.ResolvePlaceholderString(tr.StackHeadMissing, map[string]string{"pull": itoa(m.Pull), "head": m.Head[:min(12, len(m.Head))]}))
		}
		read.Heads = append(read.Heads, head)
	}
	for i, m := range members {
		below := baseCommit
		if i > 0 {
			below = read.Heads[i-1]
		}
		start, err := repo.MergeBase(below, read.Heads[i])
		if err != nil {
			return Steps{}, err
		}
		if i > 0 && start != read.Heads[i-1] {
			read.Warnings = append(read.Warnings, utils.ResolvePlaceholderString(tr.StackNotBuiltOn, map[string]string{"pull": itoa(m.Pull), "below": itoa(members[i-1].Pull)}))
		}
		step := Step{Pull: m.Pull, Start: start}
		for _, d := range []struct {
			into     *StepDiff
			from, to string
		}{{&step.Own, start, read.Heads[i]}, {&step.FromTop, read.Heads[i], top}, {&step.FromBase, combinedBase, start}} {
			if *d.into, err = indexed(repo, d.from, d.to); err != nil {
				return Steps{}, err
			}
		}
		read.Steps = append(read.Steps, step)
	}
	for i, s := range read.Steps {
		counts, err := repo.Counts(s.Start, read.Heads[i])
		if err != nil {
			return Steps{}, err
		}
		commits, err := repo.CommitCount(s.Start, read.Heads[i])
		if err != nil {
			return Steps{}, err
		}
		count := Count{Pull: s.Pull, Files: len(counts), Commits: commits}
		for _, c := range counts {
			count.Adds += c.Adds
			count.Dels += c.Dels
		}
		read.Counts = append(read.Counts, count)
	}
	return read, nil
}

func indexed(repo *git.Repository, from, to string) (StepDiff, error) {
	out := StepDiff{ByNew: map[string]StepFile{}, ByOld: map[string]StepFile{}}
	if from == to {
		return out, nil
	}
	text, err := repo.Diff(from, to)
	if err != nil {
		return StepDiff{}, err
	}
	for _, chunk := range diff.SplitDiff(text) {
		file := StepFile{Path: chunk.Path, OldPath: chunk.OldPath, Status: chunk.Status, Hunks: ReadHunkLines(chunk.Text)}
		if chunk.Status == "D" {
			file.Path = ""
		}
		if file.Path != "" {
			out.ByNew[file.Path] = file
		}
		if file.OldPath != "" {
			out.ByOld[file.OldPath] = file
		}
	}
	return out, nil
}
