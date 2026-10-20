package review

import (
	"strings"

	"github.com/connorb645/lazyreview/internal/models"
)

const hunkTail = 4

// ReadThreads reads one pull request's review threads, as GitHub's GraphQL
// API gives them, into one record each, as the app shows them: the line
// is 0 for an outdated thread, File marks a thread on the whole file, the
// comments come oldest first, More counts the comments GitHub holds beyond
// them, and Hunk is the end of the diff the first comment was made on.
func ReadThreads(pull int, nodes []models.GitHubReviewThread) []models.GitHubThread {
	threads := make([]models.GitHubThread, len(nodes))
	for i, node := range nodes {
		t := models.GitHubThread{
			ID: node.ID, Pull: pull, Path: node.Path, Side: node.DiffSide,
			Line: orZero(node.Line), StartLine: orZero(node.StartLine), OriginalLine: orZero(node.OriginalLine),
			Resolved: node.IsResolved, Outdated: node.IsOutdated, File: node.SubjectType == "FILE",
			Comments: []models.GitHubThreadComment{}, More: max(0, node.Comments.TotalCount-len(node.Comments.Nodes)),
		}
		if t.StartLine != 0 {
			t.StartSide = node.DiffSide
			if node.StartDiffSide != nil && *node.StartDiffSide != "" {
				t.StartSide = *node.StartDiffSide
			}
		}
		if node.ResolvedBy != nil {
			t.ResolvedBy = node.ResolvedBy.Login
		}
		for _, c := range node.Comments.Nodes {
			comment := models.GitHubThreadComment{ID: c.DatabaseID, Author: "ghost", Body: c.Body, At: c.CreatedAt, URL: c.URL}
			if c.Author != nil {
				if c.Author.Login != "" {
					comment.Author = c.Author.Login
				}
				comment.Bot = c.Author.Typename == "Bot"
			}
			t.Comments = append(t.Comments, comment)
		}
		if len(node.Comments.Nodes) > 0 && node.Comments.Nodes[0].DiffHunk != "" {
			lines := strings.Split(node.Comments.Nodes[0].DiffHunk, "\n")
			t.Hunk = strings.Join(lines[max(0, len(lines)-hunkTail):], "\n")
		}
		threads[i] = t
	}
	return threads
}

func orZero(n *int) int {
	if n == nil {
		return 0
	}
	return *n
}
