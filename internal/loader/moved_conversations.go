package loader

import (
	"slices"

	"github.com/connorb645/lazyreview/internal/build/diff"
	"github.com/connorb645/lazyreview/internal/commands/git"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/review"
)

func conversationPaths(threads []models.Conversation) []string {
	var paths []string
	for _, c := range threads {
		if !slices.Contains(paths, c.Path) {
			paths = append(paths, c.Path)
		}
	}
	return paths
}

func movedConversations(repo *git.Repository, before *models.PageRecord, after models.PageRecord, threads []models.Conversation) ([]models.Conversation, error) {
	if before == nil || len(threads) == 0 {
		return nil, nil
	}
	sides := map[string][2]string{
		"RIGHT": {before.Model.Heading.Head, after.Model.Heading.Head},
		"LEFT":  {before.Model.Heading.Base, after.Model.Heading.Base},
	}
	changed := map[string]map[string]diff.Chunk{}
	for side, commits := range sides {
		files, err := changedFiles(repo, commits[0], commits[1])
		if err != nil {
			return nil, err
		}
		changed[side] = files
	}
	var moved []models.Conversation
	for _, c := range threads {
		next := c
		next.Path, next.Line = movedPlace(changed[c.Side], c.Path, c.Line)
		if c.StartLine != 0 {
			startSide := c.StartSide
			if startSide == "" {
				startSide = c.Side
			}
			_, next.StartLine = movedPlace(changed[startSide], c.Path, c.StartLine)
			if startSide == c.Side && next.StartLine >= next.Line {
				next.StartLine, next.StartSide = 0, ""
			}
		}
		if next.Path != c.Path || next.Line != c.Line || next.StartLine != c.StartLine {
			moved = append(moved, next)
		}
	}
	return moved, nil
}

func changedFiles(repo *git.Repository, from, to string) (map[string]diff.Chunk, error) {
	files := map[string]diff.Chunk{}
	if from == "" || to == "" || from == to {
		return files, nil
	}
	if _, err := repo.Resolve(from); err != nil {
		return files, nil
	}
	text, err := repo.Diff(from, to)
	if err != nil {
		return nil, err
	}
	for _, c := range diff.SplitDiff(text) {
		old := c.OldPath
		if old == "" {
			old = c.Path
		}
		files[old] = c
	}
	return files, nil
}

func movedPlace(changed map[string]diff.Chunk, path string, line int) (string, int) {
	c, held := changed[path]
	if !held || c.Status == "D" {
		return path, line
	}
	return c.Path, review.MovedLine(review.ReadHunks(c.Text), line)
}
