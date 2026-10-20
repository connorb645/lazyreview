package queue

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const quote = 120

var hunkStart = regexp.MustCompile(`^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// Posted is the fields GitHub takes for comment.
func Posted(c models.Comment) PostedFields {
	f := PostedFields{Path: c.Path, Line: c.Line, Side: c.Side, Body: c.Body}
	if c.StartLine != 0 {
		f.StartLine, f.StartSide = c.StartLine, c.StartSide
		if f.StartSide == "" {
			f.StartSide = c.Side
		}
	}
	return f
}

// Sent is the comments action sends for member.
func Sent(member models.ReviewMember, action string) []models.Comment {
	if ActionOf[action].Comments {
		return member.Comments
	}
	return nil
}

// SetID is the id of action's set on member: a short hash of the
// repository, the pull request, its head, the action and the exact comments
// it sends, so it names one action on one pull request at one head with one
// set of comments, and nothing else.
func SetID(review models.Review, member models.ReviewMember, action string) string {
	comments := []PostedFields{}
	for _, c := range Sent(member, action) {
		comments = append(comments, Posted(c))
	}
	return shortHash(map[string]any{"repo": review.Repo, "pull": member.Pull, "head": member.Head, "action": action, "comments": comments})
}

// Offered is the actions offered for member: none that sends a stale
// comment, and no approval of one's own pull request.
func Offered(review models.Review, member models.ReviewMember) []string {
	ready := len(member.Comments) > 0
	for _, c := range member.Comments {
		if c.Stale != "" {
			ready = false
		}
	}
	approvable := !(member.Author != "" && member.Author == review.Me)
	var out []string
	for _, a := range Actions {
		if (ready || !ActionOf[a].Comments) && (approvable || ActionOf[a].Event != "APPROVE") {
			out = append(out, a)
		}
	}
	return out
}

// PressWords is the one sentence the app shows before a press of action
// posts for member.
func PressWords(tr *i18n.TranslationSet, review models.Review, member models.ReviewMember, action string) string {
	where := review.Repo + "#" + strconv.Itoa(member.Pull)
	count := len(member.Comments)
	these := tr.ThisCommentGoes
	if count != 1 {
		these = utils.ResolvePlaceholderString(tr.TheseCommentsGo, map[string]string{"comments": utils.Plural(count, tr.CommentNoun)})
	}
	switch action {
	case "comment":
		return utils.ResolvePlaceholderString(tr.PressComment, map[string]string{"these": capital(these), "where": where})
	case "approve-with-comments":
		return utils.ResolvePlaceholderString(tr.PressApproveWithComments, map[string]string{"these": these, "where": where})
	}
	words := utils.ResolvePlaceholderString(tr.PressApprove, map[string]string{"where": where})
	if count > 0 {
		stay := tr.Stay
		if count == 1 {
			stay = tr.Stays
		}
		words += utils.ResolvePlaceholderString(tr.PressApproveKeeps, map[string]string{"comments": utils.Plural(count, tr.CommentNoun), "stay": stay})
	}
	return words
}

// BodyProblems is what stops a comment's text being posted.
func BodyProblems(tr *i18n.TranslationSet, body string) []string {
	if strings.TrimSpace(body) == "" {
		return []string{tr.EmptyText}
	}
	return nil
}

// DiffPlaces is where each line of a file's patch sits, on each side. Rows
// before the first hunk, the header of a file's diff, are no lines.
func DiffPlaces(patch string) Places {
	places := Places{"RIGHT": {}, "LEFT": {}}
	hunk, old, now := -1, 0, 0
	var rows []string
	if patch != "" {
		rows = strings.Split(patch, "\n")
	}
	if len(rows) > 0 && rows[len(rows)-1] == "" {
		rows = rows[:len(rows)-1]
	}
	for index, row := range rows {
		at := Place{Hunk: hunk, Row: index}
		if m := hunkStart.FindStringSubmatch(row); m != nil {
			hunk++
			old, _ = strconv.Atoi(m[1])
			now, _ = strconv.Atoi(m[2])
			continue
		}
		if hunk < 0 {
			continue
		}
		switch {
		case strings.HasPrefix(row, "+"):
			places["RIGHT"][now] = at
			now++
		case strings.HasPrefix(row, "-"):
			places["LEFT"][old] = at
			old++
		case !strings.HasPrefix(row, `\`):
			places["RIGHT"][now] = at
			places["LEFT"][old] = at
			now++
			old++
		}
	}
	return places
}

// PlaceProblem is what stops GitHub taking a comment on a file whose diff
// has places, as words that follow the comment's name, or "" when nothing
// does. Its line must be a line of the diff, and a range's first line an
// earlier line of the same hunk.
func PlaceProblem(tr *i18n.TranslationSet, places Places, c models.Comment) string {
	end, ok := places[c.Side][c.Line]
	if !ok {
		return tr.NotOnDiff
	}
	if c.StartLine == 0 {
		return ""
	}
	side := c.StartSide
	if side == "" {
		side = c.Side
	}
	words := map[string]string{"line": strconv.Itoa(c.StartLine), "removed": ""}
	if side == "LEFT" {
		words["removed"] = tr.RemovedWord
	}
	start, ok := places[side][c.StartLine]
	switch {
	case !ok:
		return utils.ResolvePlaceholderString(tr.StartsOff, words)
	case start.Hunk != end.Hunk:
		return utils.ResolvePlaceholderString(tr.AcrossHunks, words)
	case start.Row >= end.Row:
		return utils.ResolvePlaceholderString(tr.StartsAfter, words)
	}
	return ""
}

// LiveMember is the live member for pull, or why there is none.
func LiveMember(tr *i18n.TranslationSet, review *models.Review, pull int) (*models.ReviewMember, error) {
	if review.Moved != nil {
		return nil, models.Refused(utils.ResolvePlaceholderString(tr.QueueReviewMoved, map[string]string{"to": review.Moved.To}))
	}
	for i := range review.Members {
		if review.Members[i].Pull == pull {
			return &review.Members[i], nil
		}
	}
	words := tr.NotInReview
	for _, m := range review.Earlier {
		if m.Pull == pull {
			words = tr.NotInStackAnyMore
		}
	}
	return nil, models.Refused(utils.ResolvePlaceholderString(words, map[string]string{"pull": strconv.Itoa(pull)}))
}

// NextID is the next free comment number across every member, live or not,
// and every review sent.
func NextID(review models.Review) int {
	most := 0
	for _, m := range append(append([]models.ReviewMember{}, review.Members...), review.Earlier...) {
		for _, c := range m.Comments {
			most = max(most, c.ID)
		}
		for _, p := range m.Posted {
			for _, c := range p.Comments {
				most = max(most, c.ID)
			}
		}
	}
	return most + 1
}

// CommentVersion is a short hash of a queued comment as it stands.
func CommentVersion(c models.Comment) string {
	c.Version = ""
	return shortHash(c)
}

// CheckVersion says, as refused words what did not happen, why a change
// naming version may not change comment: it changed since its window
// showed it, or the change names no version.
func CheckVersion(tr *i18n.TranslationSet, c models.Comment, version, refused string) error {
	if version == "" {
		return models.Refused(utils.ResolvePlaceholderString(refused, map[string]string{"why": utils.ResolvePlaceholderString(tr.NoVersion, map[string]string{"id": strconv.Itoa(c.ID)})}))
	}
	if version != CommentVersion(c) {
		body := c.Body
		if r := []rune(body); len(r) > quote {
			body = string(r[:quote]) + "…"
		}
		return models.Refused(utils.ResolvePlaceholderString(refused, map[string]string{"why": utils.ResolvePlaceholderString(tr.ChangedSince, map[string]string{"id": strconv.Itoa(c.ID), "body": body})}))
	}
	return nil
}

// FindComment is the queued comment numbered id, with the member holding
// it, or why there is none.
func FindComment(tr *i18n.TranslationSet, review *models.Review, id int) (*models.ReviewMember, int, error) {
	for _, members := range [][]models.ReviewMember{review.Members, review.Earlier} {
		for i := range members {
			for k, c := range members[i].Comments {
				if c.ID == id {
					return &members[i], k, nil
				}
			}
		}
	}
	return nil, 0, models.Refused(utils.ResolvePlaceholderString(tr.NoComment, map[string]string{"id": strconv.Itoa(id)}))
}

// View is what the app shows of each pull request's queue: its comments,
// each with its version, the reviews sent from it, and for each action on
// offer its set id and the words its press shows. A merged or closed pull
// request, one that left the stack, and a review that moved get no set ids,
// so no button.
func View(tr *i18n.TranslationSet, review models.Review) models.Queue {
	view := func(m models.ReviewMember, live bool) models.QueueMember {
		ended := m.State != "" && m.State != "open"
		state := m.State
		if state == "" {
			state = "open"
		}
		q := models.QueueMember{Pull: m.Pull, Head: m.Head, Title: m.Title, URL: m.URL, Author: m.Author, State: state, EndedAt: m.EndedAt,
			Sets: map[string]models.Set{}, Comments: []models.Comment{}, Posted: []models.Posted{}}
		if live && !ended && review.Moved == nil {
			for _, a := range Offered(review, m) {
				q.Sets[a] = models.Set{Set: SetID(review, m, a), Words: PressWords(tr, review, m, a)}
			}
		}
		for _, c := range m.Comments {
			c.Version = CommentVersion(c)
			q.Comments = append(q.Comments, c)
		}
		for _, p := range m.Posted {
			q.Posted = append(q.Posted, models.Posted{Set: p.Set, Action: p.Action, Review: p.Review, At: p.At, Count: len(p.Comments)})
		}
		return q
	}
	q := models.Queue{Repo: review.Repo, Members: []models.QueueMember{}, Earlier: []models.QueueMember{}, Moved: review.Moved}
	for _, m := range review.Members {
		q.Members = append(q.Members, view(m, true))
	}
	for _, m := range review.Earlier {
		q.Earlier = append(q.Earlier, view(m, false))
	}
	return q
}

func shortHash(v any) string {
	text, err := utils.StableJSON(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])[:12]
}

func capital(words string) string {
	if words == "" {
		return words
	}
	r := []rune(words)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}
