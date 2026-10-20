package ask

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/queue"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const (
	wholeFile = 400
	around    = 80
	listed    = 200
)

var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// Question is the one message that asks Claude about lines: the
// instructions added to Claude Code's own, and the message itself.
type Question struct {
	System string
	Prompt string
}

// QuestionFor is the question of a conversation whose last message is the
// question, for the copy of the code in folder. The message holds the pull
// request's title and description, or a branch's name and its base, where
// the copy is and its branches, the files the change touches, the file
// asked about and where the lines are, the file's part of the diff around
// them, the file's text around them, all of it when the file is short, and
// the conversation so far.
func QuestionFor(tr *i18n.TranslationSet, rev models.Review, page models.PageRecord, c models.Conversation, folder string) Question {
	file := fileOf(page, c.Path)
	owner := ownerOf(tr, rev, page, c)
	head := HeadAsked(tr, rev, page, c)
	top := 0
	if !rev.IsBranch() && len(rev.Members) > 1 {
		top = rev.Members[len(rev.Members)-1].Pull
	}
	say := utils.ResolvePlaceholderString
	parts := []string{about(tr, rev, owner)}
	plan := CopyBranches(tr, rev, page, c)
	after := tr.AskThisChange
	if head.Pull != 0 {
		after = "#" + strconv.Itoa(head.Pull)
	}
	parts = append(parts, say(tr.AskCodebase, map[string]string{"after": after, "folder": folder, "branch": plan.Branch, "base": plan.Base}))
	var files []string
	for _, f := range page.Model.Files {
		files = append(files, say(tr.AskChangedFile, map[string]string{"path": f.Path, "state": fileState(tr, &f)}))
	}
	changes := say(tr.AskChanges, map[string]string{"files": strings.Join(files[:min(len(files), listed)], "\n")})
	if len(files) > listed {
		changes += say(tr.AskChangesMore, map[string]string{"more": strconv.Itoa(len(files) - listed)})
	}
	parts = append(parts, changes)
	asked := say(tr.AskFile, map[string]string{"path": c.Path, "state": fileState(tr, file)})
	if file == nil || file.Status != "D" {
		asked += say(tr.AskFileAt, map[string]string{"at": filepath.Join(folder, head.Path)})
	}
	parts = append(parts, asked)
	lines := queue.LinesWords(tr, models.Target{Side: c.Side, Line: c.Line, StartLine: c.StartLine, StartSide: c.StartSide})
	if owner != nil && len(rev.Members) > 1 {
		parts = append(parts, say(tr.AskAboutChangedBy, map[string]string{"lines": lines, "pull": strconv.Itoa(owner.Pull)}))
	} else {
		parts = append(parts, say(tr.AskAbout, map[string]string{"lines": lines}))
	}
	if h := head.Lines; h != nil && (head.Path != c.Path || h[0] != firstLine(c) || h[1] != c.Line) {
		there := models.Target{Side: "RIGHT", Line: h[1]}
		if h[0] != h[1] {
			there.StartLine = h[0]
		}
		parts = append(parts, say(tr.AskThere, map[string]string{"pull": strconv.Itoa(head.Pull), "lines": queue.LinesWords(tr, there)}))
	}
	if file != nil {
		if hunk := hunkAround(page.Diff[c.Path], c); hunk != "" {
			parts = append(parts, say(tr.AskDiff, map[string]string{"hunk": hunk}))
		}
	}
	if text := headText(page, c, file); len(text) > 0 {
		of := ""
		if top != head.Pull {
			of = say(tr.AskHeadOf, map[string]string{"pull": strconv.Itoa(top)})
		}
		window := ""
		if from, to := lineRange(len(text), c); from != 1 || to != len(text) {
			window = say(tr.AskHeadWindow, map[string]string{"from": strconv.Itoa(from), "to": strconv.Itoa(to), "count": strconv.Itoa(len(text))})
		}
		parts = append(parts, say(tr.AskHead, map[string]string{"of": of, "window": window, "text": numbered(text, c)}))
	}
	var earlier []string
	for _, m := range c.Messages[:len(c.Messages)-1] {
		if m.Text == "" {
			continue
		}
		words := tr.AskTheyAsked
		if m.From == "claude" {
			words = tr.AskYouAnswered
		}
		earlier = append(earlier, say(words, map[string]string{"text": m.Text}))
	}
	if len(earlier) > 0 {
		parts = append(parts, say(tr.AskConversation, map[string]string{"messages": strings.Join(earlier, "\n\n")}))
	}
	parts = append(parts, say(tr.AskQuestion, map[string]string{"text": c.Messages[len(c.Messages)-1].Text}))
	return Question{System: tr.AskSystem, Prompt: strings.Join(parts, "\n\n")}
}

// FollowUpFor is the question that asks a conversation's last question in
// a Claude Code session that already holds the rest.
func FollowUpFor(tr *i18n.TranslationSet, c models.Conversation) Question {
	return Question{System: tr.AskSystem, Prompt: utils.ResolvePlaceholderString(tr.AskFollowUp, map[string]string{"text": c.Messages[len(c.Messages)-1].Text})}
}

func about(tr *i18n.TranslationSet, rev models.Review, owner *models.ReviewMember) string {
	say := utils.ResolvePlaceholderString
	switch {
	case rev.IsBranch():
		title := ""
		if rev.Title != rev.Head.Ref {
			title = say(tr.AskBranchTitle, map[string]string{"title": rev.Title})
		}
		return say(tr.AskBranch, map[string]string{"branch": rev.Head.Ref, "title": title, "base": rev.Base.Ref})
	case len(rev.Members) == 1:
		return describe(tr, rev.Members[0])
	}
	var members []string
	for _, m := range rev.Members {
		members = append(members, say(tr.AskStackMember, map[string]string{"pull": strconv.Itoa(m.Pull), "title": m.Title}))
	}
	stack := say(tr.AskStack, map[string]string{"members": strings.Join(members, "\n")})
	if owner != nil {
		stack += say(tr.AskStackOwner, map[string]string{"owner": describe(tr, *owner)})
	}
	return stack
}

func describe(tr *i18n.TranslationSet, m models.ReviewMember) string {
	words := utils.ResolvePlaceholderString(tr.AskPull, map[string]string{"pull": strconv.Itoa(m.Pull), "title": m.Title})
	if m.Description != "" {
		words += utils.ResolvePlaceholderString(tr.AskPullDescription, map[string]string{"description": m.Description})
	}
	return words
}

func fileOf(page models.PageRecord, path string) *models.FileEntry {
	for i := range page.Model.Files {
		if page.Model.Files[i].Path == path {
			return &page.Model.Files[i]
		}
	}
	return nil
}

func fileState(tr *i18n.TranslationSet, f *models.FileEntry) string {
	switch {
	case f == nil:
		return tr.AskUnchanged
	case f.Status == "A":
		return tr.AskAdded
	case f.Status == "D":
		return tr.AskDeleted
	case f.Status == "R":
		return utils.ResolvePlaceholderString(tr.AskRenamed, map[string]string{"from": f.OldPath})
	}
	return tr.AskChanged
}

func headText(page models.PageRecord, c models.Conversation, file *models.FileEntry) []string {
	if c.Side == "LEFT" {
		return nil
	}
	if text, ok := page.Sources[c.Path]; ok && text != nil {
		return text
	}
	if text, ok := page.Context[c.Path]; ok && text != nil {
		return text
	}
	return addedText(page.Diff[c.Path], file)
}

func addedText(diff string, file *models.FileEntry) []string {
	if file == nil || file.Status != "A" || diff == "" {
		return nil
	}
	var out []string
	for _, row := range strings.Split(diff, "\n") {
		if strings.HasPrefix(row, "+") && !strings.HasPrefix(row, "+++") {
			out = append(out, row[1:])
		}
	}
	return out
}

func hunkAround(diff string, c models.Conversation) string {
	var hunks []*diffHunk
	var current *diffHunk
	span := func(start, count string) [2]int {
		from, _ := strconv.Atoi(start)
		n := 1
		if count != "" {
			n, _ = strconv.Atoi(count)
		}
		return [2]int{from, from + n - 1}
	}
	for _, row := range strings.Split(diff, "\n") {
		if m := hunkHeader.FindStringSubmatch(row); m != nil {
			current = &diffHunk{rows: []string{row}, old: span(m[1], m[2]), now: span(m[3], m[4])}
			hunks = append(hunks, current)
		} else if current != nil {
			current.rows = append(current.rows, row)
		}
	}
	if len(hunks) == 0 {
		return ""
	}
	first := firstLine(c)
	distance := func(h *diffHunk) int {
		r := h.now
		if c.Side == "LEFT" {
			r = h.old
		}
		switch {
		case c.Line < r[0]:
			return r[0] - c.Line
		case first > r[1]:
			return first - r[1]
		}
		return 0
	}
	best := hunks[0]
	for _, h := range hunks[1:] {
		if distance(h) < distance(best) {
			best = h
		}
	}
	return strings.TrimSuffix(strings.Join(best.rows, "\n"), "\n")
}

func lineRange(count int, c models.Conversation) (int, int) {
	if count <= wholeFile {
		return 1, count
	}
	return max(1, firstLine(c)-around), min(count, c.Line+around)
}

func numbered(text []string, c models.Conversation) string {
	from, to := lineRange(len(text), c)
	if from > to {
		return ""
	}
	width := len(strconv.Itoa(to))
	rows := make([]string, 0, to-from+1)
	for i, row := range text[from-1 : to] {
		number := strconv.Itoa(from + i)
		rows = append(rows, strings.Repeat(" ", width-len(number))+number+" | "+row)
	}
	return strings.Join(rows, "\n")
}
