package presentation

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/internal/diagram"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/theme"
	"github.com/connorb645/lazyreview/internal/utils"
)

const preview = 220

// BuildReview draws the Review view: the review's header, its written
// sections and their links, the queued comments, the threads on GitHub, the
// conversations with Claude and the change's components, each section folded
// or not, its words naming keys as labels label them. It gives the lines
// and the items they stand for.
func BuildReview(tr *i18n.TranslationSet, th theme.Theme, md *Markdown, labels KeyLabels, loc *time.Location, d types.ReviewData, filter types.PullFilter, folded map[string]bool, width int) ([]Line, []Item) {
	b := &builder{width: width, tr: tr, th: th, labels: labels}
	reviewHeader(b, d, filter)
	section := func(id, title, summary string, body func()) {
		b.text("")
		mark := "▾ "
		if folded[id] {
			mark = "▸ "
		}
		head := " " + th.Muted.Render(mark) + th.Bold.Render(title)
		if folded[id] && summary != "" {
			head += th.Muted.Render(" · " + summary)
		}
		b.add(Item{Kind: "section", Title: title, Section: id}, head)
		if !folded[id] {
			body()
		}
	}
	if w := d.State.Written; w != nil {
		if w.Stale {
			b.text("")
			b.prose(" ", th.Warning, utils.ResolvePlaceholderString(tr.WrittenStale, map[string]string{"head": utils.Short(w.Head)}))
		}
		for _, s := range w.Sections {
			s := s
			section("written:"+s.ID, s.Title, utils.Plural(len(s.Links), tr.LinkNoun), func() {
				for _, l := range md.Lines(withoutDiagrams(tr, labels, s.Markdown), width-2) {
					b.text(" " + l)
				}
				for _, link := range s.Links {
					place := models.Lines{Path: link.Path, Side: "RIGHT", Line: link.Line}
					where := "  " + link.Path + " " + utils.ResolvePlaceholderString(tr.LineAt, map[string]string{"line": utils.Itoa(link.Line)})
					b.add(Item{Kind: "link", Section: "written:" + s.ID, Place: &place}, "   "+th.Accent.Render("→ "+link.Text)+th.Muted.Render(where))
				}
			})
		}
	} else if d.State.Queue != nil {
		b.text("")
		b.prose(" ", th.Muted, tr.NoWrittenReview)
	} else {
		b.text("")
		b.prose(" ", th.Muted, tr.BranchReadOnly)
	}
	if q := d.State.Queue; q != nil {
		count := 0
		for _, m := range q.Members {
			if filter.Shows(m.Pull) {
				count += len(m.Comments)
			}
		}
		section("queue", tr.QueuedComments, utils.Plural(count, tr.CommentNoun), func() { queueLines(b, d, filter, loc) })
	}
	var threads []models.GitHubThread
	open := 0
	for _, t := range d.State.GitHub {
		if filter.Shows(t.Pull) {
			threads = append(threads, t)
			if !t.Resolved {
				open++
			}
		}
	}
	if len(threads) > 0 {
		section("github", tr.OnGitHubHeading, utils.Plural(len(threads), tr.ThreadNoun), func() {
			b.prose(" ", th.Muted, utils.ResolvePlaceholderString(tr.GitHubThreadsLoaded, map[string]string{"threads": utils.Plural(len(threads), tr.ReviewThreadNoun), "open": utils.Itoa(open)}))
			for _, t := range threads {
				githubLines(b, d, t)
			}
		})
	}
	var asked []models.Conversation
	for _, t := range d.State.Threads {
		if ShowsFile(filter, d, t.Path) {
			asked = append(asked, t)
		}
	}
	if len(asked) > 0 {
		section("conversations", tr.Conversations, utils.Plural(len(asked), tr.ConversationNoun), func() {
			for _, t := range asked {
				conversationLines(b, t, loc)
			}
		})
	}
	var components []models.Component
	for _, c := range d.Page.Model.Components {
		if len(DiffFiles(d, c.ID, filter)) > 0 {
			components = append(components, c)
		}
	}
	section("change", tr.TheChange, utils.Plural(len(components), tr.ComponentNoun), func() {
		for _, c := range components {
			b.add(Item{Kind: "component", Title: c.Title, Section: "change", Component: c.ID}, "   "+th.Component(c.Color).Render("■ ")+th.Text.Render(c.Title)+"  "+th.Muted.Render(diagram.ComponentDetail(tr, d.Page, c, filter.ShowsAny, filter.Lines)))
		}
	})
	return b.lines, b.items
}

func reviewHeader(b *builder, d types.ReviewData, filter types.PullFilter) {
	th, s, tr := b.th, d.State, b.tr
	title := s.Title
	if s.Written != nil && s.Written.Title != "" {
		title = s.Written.Title
	}
	totals := changeTotals(tr, th, d.Page.Model.Files)
	base := func(ref string) string { return utils.ResolvePlaceholderString(tr.Base, map[string]string{"ref": ref}) }
	head := func(sha string) string {
		return utils.ResolvePlaceholderString(tr.Head, map[string]string{"sha": utils.Short(sha)})
	}
	switch {
	case s.Kind == "branch" && (d.Page.Stack == nil || len(s.Members) == 0):
		b.prose(" ", th.Muted, utils.ResolvePlaceholderString(tr.BranchHeader, map[string]string{"branch": s.Head.Ref, "repo": s.Repo}))
		b.prose(" ", th.Bold, title)
		b.parts(" ", th.Muted.Render(base(s.Base.Ref)), th.Muted.Render(head(s.Head.Sha)), totals)
	case d.Page.Stack != nil:
		words := utils.ResolvePlaceholderString(tr.StackHeader, map[string]string{"count": utils.Itoa(len(s.Members))})
		if s.Kind == "branch" {
			words = utils.ResolvePlaceholderString(tr.BranchStackHeader, map[string]string{"count": utils.Itoa(len(s.Members)), "repo": s.Repo})
		}
		b.prose(" ", th.Muted, words)
		b.prose(" ", th.Bold, title)
		widest := 0
		for _, m := range s.Members {
			widest = max(widest, ansi.StringWidth(m.Branch))
		}
		for _, m := range s.Members {
			own := StackMember(d.Page.Stack, m.Pull)
			number, name, added, removed := th.PullRequest(m.Color), th.Text, th.Added, th.Removed
			if !filter.Shows(m.Pull) {
				number, name, added, removed = th.Faint, th.Faint, th.DimAdded, th.DimRemoved
			}
			parts := []string{th.Muted.Render(utils.ResolvePlaceholderString(tr.MemberHead, map[string]string{"sha": utils.Short(m.Head)})),
				th.Muted.Render(utils.Plural(own.Files, tr.FileNoun)+" ") + added.Render(fmt.Sprintf("+%d", own.Adds)) + " " + removed.Render(fmt.Sprintf("−%d", own.Dels))}
			head, indent := "  "+number.Render(MemberName(d, m.Pull))+strings.Repeat(" ", widest-ansi.StringWidth(m.Branch)), "    "
			if m.Branch == "" {
				parts = append([]string{th.Muted.Render(utils.ResolvePlaceholderString(tr.ByAuthor, map[string]string{"author": m.Author})),
					th.Muted.Render(utils.ResolvePlaceholderString(tr.OnBase, map[string]string{"base": m.Base}))}, parts...)
				head, indent = head+"  "+name.Render(m.Title), "       "
			}
			if m.State != "" && m.State != "open" {
				parts = append(parts, th.Removed.Render(m.State))
			}
			if tail := strings.Join(parts, th.Muted.Render(" · ")); ansi.StringWidth(head)+2+ansi.StringWidth(tail) <= b.width {
				b.text(head + "  " + tail)
				continue
			}
			b.text(head)
			b.parts(indent, parts...)
		}
		top := s.Members[len(s.Members)-1]
		b.parts(" ", th.Muted.Render(base(s.Base.Ref)), th.Muted.Render(head(top.Head)), th.Muted.Render(tr.Combined+" ")+totals)
		for _, w := range d.Page.Stack.Warnings {
			b.prose(" ", th.Warning, w)
		}
		if len(filter) > 0 {
			b.prose(" ", th.Accent, utils.ResolvePlaceholderString(tr.ShowingOnly, map[string]string{"pulls": FilterWords(tr, d, filter), "close": b.labels.Nth("close", 1)}))
		}
	default:
		m := s.Members[0]
		state := ""
		if m.State != "" && m.State != "open" {
			state = " · " + m.State
		}
		b.text(" " + th.Muted.Render(utils.ResolvePlaceholderString(tr.PullHeader, map[string]string{"pull": utils.Itoa(m.Pull), "author": m.Author})+state))
		b.prose(" ", th.Bold, title)
		b.parts(" ", th.Muted.Render(base(m.Base)), th.Muted.Render(head(m.Head)), totals)
	}
	if s.Moved != nil {
		b.prose(" ", th.Warning, utils.ResolvePlaceholderString(tr.ReviewMoved, map[string]string{"to": s.Moved.To}))
	}
	for _, n := range s.Notes {
		if d.Page.Stack != nil && slices.Contains(d.Page.Stack.Warnings, n) {
			continue
		}
		for _, l := range utils.Wrap(n, b.width-2) {
			b.text(" " + th.Warning.Render(l))
		}
	}
}

func queueLines(b *builder, d types.ReviewData, filter types.PullFilter, loc *time.Location) {
	th, q, tr := b.th, d.State.Queue, b.tr
	if q.Moved != nil {
		b.prose(" ", th.Muted, utils.ResolvePlaceholderString(tr.QueueMoved, map[string]string{"to": q.Moved.To}))
		return
	}
	stacked := d.Page.Stack != nil
	if stacked {
		b.prose(" ", th.Muted, tr.CommentGoesOn)
	}
	groups := append(append([]models.QueueMember{}, q.Members...), q.Earlier...)
	for i, m := range groups {
		if !filter.Shows(m.Pull) {
			continue
		}
		left := i >= len(q.Members)
		color := MemberColor(d.State, m.Pull)
		if stacked {
			b.text("")
			b.add(Item{Kind: "pull", Section: "queue", Title: fmt.Sprintf("#%d %s", m.Pull, m.Title), Pull: m.Pull, URL: m.URL}, "  "+th.PullRequest(color).Render(fmt.Sprintf("#%d", m.Pull))+"  "+th.Text.Render(m.Title))
		}
		if len(m.Comments) == 0 {
			b.text("   " + th.Muted.Render(tr.NoCommentsQueued))
		}
		for _, c := range m.Comments {
			commentLines(b, d, m, c, color, stacked)
		}
		switch {
		case m.State != "" && m.State != "open":
			at := ""
			if m.EndedAt != nil {
				at = " " + utils.ResolvePlaceholderString(tr.OnDate, map[string]string{"date": utils.When(*m.EndedAt, loc)})
			}
			b.prose("   ", th.Muted, utils.ResolvePlaceholderString(tr.PullEnded, map[string]string{"pull": utils.Itoa(m.Pull), "state": m.State, "at": at}))
		case left:
			b.prose("   ", th.Muted, utils.ResolvePlaceholderString(tr.PullLeftStack, map[string]string{"pull": utils.Itoa(m.Pull)}))
		}
		for _, posted := range m.Posted {
			comments := utils.Plural(posted.Count, tr.CommentNoun)
			what := map[string]string{"comment": utils.ResolvePlaceholderString(tr.PostedComments, map[string]string{"comments": comments}), "approve-with-comments": utils.ResolvePlaceholderString(tr.ApprovedWith, map[string]string{"comments": comments}), "approve": tr.Approved}[posted.Action]
			b.prose("   ", th.Muted, fmt.Sprintf("%s, %s: %s", what, utils.When(posted.At, loc), posted.Review))
		}
	}
}

func commentLines(b *builder, d types.ReviewData, m models.QueueMember, c models.Comment, color int, stacked bool) {
	th, tr := b.th, b.tr
	head := "   " + th.Accent.Render(fmt.Sprintf("%d", c.ID)) + " "
	if stacked {
		head += th.PullRequest(color).Render(fmt.Sprintf("→ #%d", m.Pull)) + " "
	}
	head += th.Text.Render(c.Path + ", " + LinesWords(tr, c.Side, c.Line, c.StartLine, c.StartSide))
	place := models.Lines{Path: c.Path, Side: c.Side, Line: c.Line, StartLine: c.StartLine, StartSide: c.StartSide}
	if c.Shown != nil {
		place = *c.Shown
		if c.Shown.Path != c.Path || c.Shown.Line != c.Line {
			head += th.Muted.Render("  " + utils.ResolvePlaceholderString(tr.ShownAtLine, map[string]string{"line": utils.Itoa(c.Shown.Line)}))
		}
	}
	texts := []string{head}
	if c.Code != "" {
		for _, l := range strings.Split(c.Code, "\n") {
			texts = append(texts, "     "+th.Muted.Render("│ "+l))
		}
	}
	for _, l := range utils.Wrap(c.Body, b.width-6) {
		texts = append(texts, "     "+th.Text.Render(l))
	}
	if c.Stale != "" {
		for _, l := range utils.Wrap(utils.ResolvePlaceholderString(tr.MayHaveMoved, map[string]string{"reason": c.Stale}), b.width-6) {
			texts = append(texts, "     "+th.Warning.Render(l))
		}
	}
	b.add(Item{Kind: "comment", Section: "queue", Pull: m.Pull, Comment: c.ID, URL: m.URL, Place: &place}, texts...)
}

func githubLines(b *builder, d types.ReviewData, t models.GitHubThread) {
	th, tr := b.th, b.tr
	where := t.Path
	switch {
	case t.File:
		where += ", " + tr.TheWholeFile
	case t.At != nil:
		lineWord := utils.ResolvePlaceholderString(tr.LineAt, map[string]string{"line": ""})
		where += ":" + strings.TrimPrefix(LinesWords(tr, t.At.Side, t.At.Line, t.At.StartLine, t.At.StartSide), lineWord)
	case t.OriginalLine > 0:
		where += ", " + utils.ResolvePlaceholderString(tr.WasLine, map[string]string{"line": utils.Itoa(t.OriginalLine)})
	}
	var people []string
	for _, c := range t.Comments {
		if !slices.Contains(people, c.Author) {
			people = append(people, c.Author)
		}
	}
	meta := strings.Join(people, ", ") + " · " + utils.Plural(len(t.Comments)+t.More, tr.CommentNoun)
	if t.Resolved {
		meta += " · " + tr.Resolved
	}
	if t.Outdated {
		meta += " · " + tr.Outdated
	}
	head := "   " + th.Accent.Render(where) + "  "
	if d.Page.Stack != nil {
		head += th.PullRequest(MemberColor(d.State, t.Pull)).Render(fmt.Sprintf("#%d", t.Pull)) + " "
	}
	texts := []string{head + th.Muted.Render(meta)}
	if len(t.Comments) > 0 {
		for _, l := range clipped(utils.PlainText(t.Comments[0].Body), b.width-6, 2) {
			texts = append(texts, "     "+th.Text.Render(l))
		}
	}
	it := Item{Kind: "thread", Section: "github", Pull: t.Pull, Place: t.At}
	if len(t.Comments) > 0 {
		it.URL = t.Comments[0].URL
	}
	b.add(it, texts...)
}

func conversationLines(b *builder, t models.Conversation, loc *time.Location) {
	th, tr := b.th, b.tr
	lines := fmt.Sprint(t.Line)
	if t.StartLine > 0 {
		lines = fmt.Sprintf("%d-%d", t.StartLine, t.Line)
	}
	texts := []string{"   " + th.Accent.Render(t.Path+":"+lines) + "  " + th.Muted.Render(utils.When(t.CreatedAt, loc)+" · "+utils.Plural(len(t.Messages), tr.MessageNoun))}
	if len(t.Messages) > 0 {
		for _, l := range clipped(t.Messages[0].Text, b.width-6, 2) {
			texts = append(texts, "     "+th.Bold.Render(l))
		}
		last := t.Messages[len(t.Messages)-1]
		answer := tr.WaitingForClaude
		switch {
		case last.From == "claude" && last.Error != "":
			answer = tr.NoAnswer + ": " + last.Error
		case last.From == "claude":
			answer = utils.PlainText(last.Text)
		}
		for _, l := range clipped(answer, b.width-6, 2) {
			texts = append(texts, "     "+th.Muted.Render(l))
		}
	}
	place := models.Lines{Path: t.Path, Side: t.Side, Line: t.Line, StartLine: t.StartLine, StartSide: t.StartSide}
	b.add(Item{Kind: "conversation", Section: "conversations", Thread: t.ID, Place: &place}, texts...)
}

func changeTotals(tr *i18n.TranslationSet, th theme.Theme, files []models.FileEntry) string {
	adds, dels := 0, 0
	for _, f := range files {
		adds += f.Adds
		dels += f.Dels
	}
	return th.Muted.Render(utils.Plural(len(files), tr.FileNoun)+" ") + th.Added.Render(fmt.Sprintf("+%d", adds)) + " " + th.Removed.Render(fmt.Sprintf("−%d", dels))
}
