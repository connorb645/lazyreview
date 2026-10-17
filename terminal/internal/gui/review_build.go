package gui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const preview = 220

type builder struct {
	lines []line
	items []item
	width int
	tr    *i18n.TranslationSet
	p     palette
}

func (b *builder) text(texts ...string) {
	for _, t := range texts {
		b.lines = append(b.lines, line{text: t, item: -1})
	}
}

func (b *builder) prose(indent string, style lipgloss.Style, text string) {
	for _, l := range utils.Wrap(text, b.width-len(indent)-1) {
		b.text(indent + style.Render(l))
	}
}

func (b *builder) parts(indent string, parts ...string) {
	sep := b.p.Muted.Render(" · ")
	line := ""
	for _, part := range parts {
		switch {
		case line == "":
			line = part
		case ansi.StringWidth(indent+line+sep+part) <= b.width-1:
			line += sep + part
		default:
			b.text(indent + line)
			line = part
		}
	}
	b.text(indent + line)
}

func (b *builder) add(it item, texts ...string) {
	b.items = append(b.items, it)
	for _, t := range texts {
		b.lines = append(b.lines, line{text: t, item: len(b.items) - 1})
	}
}

func buildReview(tr *i18n.TranslationSet, d reviewData, filter pullFilter, folded map[string]bool, width int, p palette, md *markdown, loc *time.Location) ([]line, []item) {
	b := &builder{width: width, tr: tr, p: p}
	reviewHeader(b, d, filter)
	section := func(id, title, summary string, body func()) {
		b.text("")
		mark := "▾ "
		if folded[id] {
			mark = "▸ "
		}
		head := " " + p.Muted.Render(mark) + p.Bold.Render(title)
		if folded[id] && summary != "" {
			head += p.Muted.Render(" · " + summary)
		}
		b.add(item{kind: "section", title: title, section: id}, head)
		if !folded[id] {
			body()
		}
	}
	if w := d.state.Written; w != nil {
		if w.Stale {
			b.text("")
			b.prose(" ", p.Warn, utils.ResolvePlaceholderString(tr.WrittenStale, map[string]string{"head": utils.Short(w.Head)}))
		}
		for _, s := range w.Sections {
			s := s
			section("written:"+s.ID, s.Title, utils.Plural(len(s.Links), tr.LinkNoun), func() {
				for _, l := range md.lines(withoutDiagrams(tr, s.Markdown), width-2) {
					b.text(" " + l)
				}
				for _, link := range s.Links {
					place := server.Lines{Path: link.Path, Side: "RIGHT", Line: link.Line}
					where := "  " + link.Path + " " + utils.ResolvePlaceholderString(tr.LineAt, map[string]string{"line": utils.Itoa(link.Line)})
					b.add(item{kind: "link", section: "written:" + s.ID, place: &place}, "   "+p.Accent.Render("→ "+link.Text)+p.Muted.Render(where))
				}
			})
		}
	} else if d.state.Queue != nil {
		b.text("")
		b.prose(" ", p.Muted, tr.NoWrittenReview)
	} else {
		b.text("")
		b.prose(" ", p.Muted, tr.BranchReadOnly)
	}
	if q := d.state.Queue; q != nil {
		count := 0
		for _, m := range q.Members {
			if filter.shows(m.Pull) {
				count += len(m.Comments)
			}
		}
		section("queue", tr.QueuedComments, utils.Plural(count, tr.CommentNoun), func() { queueLines(b, d, filter, loc) })
	}
	var threads []server.GitHubThread
	open := 0
	for _, t := range d.state.GitHub {
		if filter.shows(t.Pull) {
			threads = append(threads, t)
			if !t.Resolved {
				open++
			}
		}
	}
	if len(threads) > 0 {
		section("github", tr.OnGitHubHeading, utils.Plural(len(threads), tr.ThreadNoun), func() {
			b.prose(" ", p.Muted, utils.ResolvePlaceholderString(tr.GitHubThreadsLoaded, map[string]string{"threads": utils.Plural(len(threads), tr.ReviewThreadNoun), "open": utils.Itoa(open)}))
			for _, t := range threads {
				githubLines(b, d, t)
			}
		})
	}
	var asked []server.Conversation
	for _, t := range d.state.Threads {
		if filter.showsFile(d, t.Path) {
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
	var components []server.Component
	for _, c := range d.page.Model.Components {
		if len(diffFiles(d, c.ID, filter)) > 0 {
			components = append(components, c)
		}
	}
	section("change", tr.TheChange, utils.Plural(len(components), tr.ComponentNoun), func() {
		for _, c := range components {
			b.add(item{kind: "component", title: c.Title, section: "change", component: c.ID}, "   "+p.componentColor(c.Color).Render("■ ")+p.Ink.Render(c.Title)+"  "+p.Muted.Render(componentWords(tr, d, c, filter)))
		}
	})
	return b.lines, b.items
}

func reviewHeader(b *builder, d reviewData, filter pullFilter) {
	p, s, tr := b.p, d.state, b.tr
	title := s.Title
	if s.Written != nil && s.Written.Title != "" {
		title = s.Written.Title
	}
	totals := changeTotals(tr, p, d.page.Model.Files)
	base := func(ref string) string { return utils.ResolvePlaceholderString(tr.Base, map[string]string{"ref": ref}) }
	head := func(sha string) string {
		return utils.ResolvePlaceholderString(tr.Head, map[string]string{"sha": utils.Short(sha)})
	}
	switch {
	case s.Kind == "branch":
		b.prose(" ", p.Muted, utils.ResolvePlaceholderString(tr.BranchHeader, map[string]string{"branch": s.Head.Ref, "repo": s.Repo}))
		b.prose(" ", p.Bold, title)
		b.parts(" ", p.Muted.Render(base(s.Base.Ref)), p.Muted.Render(head(s.Head.Sha)), totals)
	case d.page.Stack != nil:
		b.prose(" ", p.Muted, utils.ResolvePlaceholderString(tr.StackHeader, map[string]string{"count": utils.Itoa(len(s.Members))}))
		b.prose(" ", p.Bold, title)
		for _, m := range s.Members {
			own := stackMember(d.page.Stack, m.Pull)
			parts := []string{p.Muted.Render(utils.ResolvePlaceholderString(tr.ByAuthor, map[string]string{"author": m.Author})), p.Muted.Render(utils.ResolvePlaceholderString(tr.OnBase, map[string]string{"base": m.Base})),
				p.Muted.Render(utils.ResolvePlaceholderString(tr.MemberHead, map[string]string{"sha": utils.Short(m.Head)})),
				p.Muted.Render(utils.Plural(own.Files, tr.FileNoun)+" ") + p.Add.Render(fmt.Sprintf("+%d", own.Adds)) + " " + p.Del.Render(fmt.Sprintf("−%d", own.Dels))}
			if m.State != "" && m.State != "open" {
				parts = append(parts, p.Del.Render(m.State))
			}
			number, name := p.pull(m.Color), p.Ink
			if !filter.shows(m.Pull) {
				number, name = p.Faint, p.Faint
			}
			head := "  " + number.Render(fmt.Sprintf("#%d", m.Pull)) + "  " + name.Render(m.Title)
			if tail := strings.Join(parts, p.Muted.Render(" · ")); ansi.StringWidth(head)+2+ansi.StringWidth(tail) <= b.width {
				b.text(head + "  " + tail)
				continue
			}
			b.text(head)
			b.parts("       ", parts...)
		}
		top := s.Members[len(s.Members)-1]
		b.parts(" ", p.Muted.Render(base(s.Base.Ref)), p.Muted.Render(head(top.Head)), p.Muted.Render(tr.Combined+" ")+totals)
		for _, w := range d.page.Stack.Warnings {
			b.prose(" ", p.Warn, w)
		}
		if len(filter) > 0 {
			b.prose(" ", p.Accent, utils.ResolvePlaceholderString(tr.ShowingOnly, map[string]string{"pulls": filter.words(tr)}))
		}
	default:
		m := s.Members[0]
		state := ""
		if m.State != "" && m.State != "open" {
			state = " · " + m.State
		}
		b.text(" " + p.Muted.Render(utils.ResolvePlaceholderString(tr.PullHeader, map[string]string{"pull": utils.Itoa(m.Pull), "author": m.Author})+state))
		b.prose(" ", p.Bold, title)
		b.parts(" ", p.Muted.Render(base(m.Base)), p.Muted.Render(head(m.Head)), totals)
	}
	if s.Moved != nil {
		b.prose(" ", p.Warn, utils.ResolvePlaceholderString(tr.ReviewMoved, map[string]string{"to": s.Moved.To}))
	}
	for _, n := range s.Notes {
		for _, l := range utils.Wrap(n, b.width-2) {
			b.text(" " + p.Warn.Render(l))
		}
	}
}

func queueLines(b *builder, d reviewData, filter pullFilter, loc *time.Location) {
	p, q, tr := b.p, d.state.Queue, b.tr
	if q.Moved != nil {
		b.prose(" ", p.Muted, utils.ResolvePlaceholderString(tr.QueueMoved, map[string]string{"to": q.Moved.To}))
		return
	}
	stacked := d.page.Stack != nil
	if stacked {
		b.prose(" ", p.Muted, tr.CommentGoesOn)
	}
	groups := append(append([]server.QueueMember{}, q.Members...), q.Earlier...)
	for i, m := range groups {
		if !filter.shows(m.Pull) {
			continue
		}
		left := i >= len(q.Members)
		color := memberColor(d.state, m.Pull)
		if stacked {
			b.text("")
			b.add(item{kind: "pull", section: "queue", title: fmt.Sprintf("#%d %s", m.Pull, m.Title), pull: m.Pull, url: m.URL}, "  "+p.pull(color).Render(fmt.Sprintf("#%d", m.Pull))+"  "+p.Ink.Render(m.Title))
		}
		if len(m.Comments) == 0 {
			b.text("   " + p.Muted.Render(tr.NoCommentsQueued))
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
			b.prose("   ", p.Muted, utils.ResolvePlaceholderString(tr.PullEnded, map[string]string{"pull": utils.Itoa(m.Pull), "state": m.State, "at": at}))
		case left:
			b.prose("   ", p.Muted, utils.ResolvePlaceholderString(tr.PullLeftStack, map[string]string{"pull": utils.Itoa(m.Pull)}))
		}
		for _, posted := range m.Posted {
			comments := utils.Plural(posted.Count, tr.CommentNoun)
			what := map[string]string{"comment": utils.ResolvePlaceholderString(tr.PostedComments, map[string]string{"comments": comments}), "approve-with-comments": utils.ResolvePlaceholderString(tr.ApprovedWith, map[string]string{"comments": comments}), "approve": tr.Approved}[posted.Action]
			b.prose("   ", p.Muted, fmt.Sprintf("%s, %s: %s", what, utils.When(posted.At, loc), posted.Review))
		}
	}
}

func commentLines(b *builder, d reviewData, m server.QueueMember, c server.Comment, color int, stacked bool) {
	p, tr := b.p, b.tr
	head := "   " + p.Accent.Render(fmt.Sprintf("%d", c.ID)) + " "
	if stacked {
		head += p.pull(color).Render(fmt.Sprintf("→ #%d", m.Pull)) + " "
	}
	head += p.Ink.Render(c.Path + ", " + linesWords(tr, c.Side, c.Line, c.StartLine, c.StartSide))
	place := server.Lines{Path: c.Path, Side: c.Side, Line: c.Line, StartLine: c.StartLine, StartSide: c.StartSide}
	if c.Shown != nil {
		place = *c.Shown
		if c.Shown.Path != c.Path || c.Shown.Line != c.Line {
			head += p.Muted.Render("  " + utils.ResolvePlaceholderString(tr.ShownAtLine, map[string]string{"line": utils.Itoa(c.Shown.Line)}))
		}
	}
	texts := []string{head}
	if c.Code != "" {
		for _, l := range strings.Split(c.Code, "\n") {
			texts = append(texts, "     "+p.Muted.Render("│ "+l))
		}
	}
	for _, l := range utils.Wrap(c.Body, b.width-6) {
		texts = append(texts, "     "+p.Ink.Render(l))
	}
	if c.Stale != "" {
		for _, l := range utils.Wrap(utils.ResolvePlaceholderString(tr.MayHaveMoved, map[string]string{"reason": c.Stale}), b.width-6) {
			texts = append(texts, "     "+p.Warn.Render(l))
		}
	}
	b.add(item{kind: "comment", section: "queue", pull: m.Pull, comment: c.ID, url: m.URL, place: &place}, texts...)
}

func githubLines(b *builder, d reviewData, t server.GitHubThread) {
	p, tr := b.p, b.tr
	where := t.Path
	switch {
	case t.File:
		where += ", " + tr.TheWholeFile
	case t.At != nil:
		lineWord := utils.ResolvePlaceholderString(tr.LineAt, map[string]string{"line": ""})
		where += ":" + strings.TrimPrefix(linesWords(tr, t.At.Side, t.At.Line, t.At.StartLine, t.At.StartSide), lineWord)
	case t.OriginalLine > 0:
		where += ", " + utils.ResolvePlaceholderString(tr.WasLine, map[string]string{"line": utils.Itoa(t.OriginalLine)})
	}
	var people []string
	for _, c := range t.Comments {
		if !contains(people, c.Author) {
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
	head := "   " + p.Accent.Render(where) + "  "
	if d.page.Stack != nil {
		head += p.pull(memberColor(d.state, t.Pull)).Render(fmt.Sprintf("#%d", t.Pull)) + " "
	}
	texts := []string{head + p.Muted.Render(meta)}
	if len(t.Comments) > 0 {
		for _, l := range clipped(utils.PlainText(t.Comments[0].Body), b.width-6, 2) {
			texts = append(texts, "     "+p.Ink.Render(l))
		}
	}
	it := item{kind: "thread", section: "github", pull: t.Pull, place: t.At}
	if len(t.Comments) > 0 {
		it.url = t.Comments[0].URL
	}
	b.add(it, texts...)
}

func conversationLines(b *builder, t server.Conversation, loc *time.Location) {
	p, tr := b.p, b.tr
	lines := fmt.Sprint(t.Line)
	if t.StartLine > 0 {
		lines = fmt.Sprintf("%d-%d", t.StartLine, t.Line)
	}
	texts := []string{"   " + p.Accent.Render(t.Path+":"+lines) + "  " + p.Muted.Render(utils.When(t.CreatedAt, loc)+" · "+utils.Plural(len(t.Messages), tr.MessageNoun))}
	if len(t.Messages) > 0 {
		for _, l := range clipped(t.Messages[0].Text, b.width-6, 2) {
			texts = append(texts, "     "+p.Bold.Render(l))
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
			texts = append(texts, "     "+p.Muted.Render(l))
		}
	}
	place := server.Lines{Path: t.Path, Side: t.Side, Line: t.Line, StartLine: t.StartLine, StartSide: t.StartSide}
	b.add(item{kind: "conversation", section: "conversations", place: &place}, texts...)
}

func changeTotals(tr *i18n.TranslationSet, p palette, files []server.FileEntry) string {
	adds, dels := 0, 0
	for _, f := range files {
		adds += f.Adds
		dels += f.Dels
	}
	return p.Muted.Render(utils.Plural(len(files), tr.FileNoun)+" ") + p.Add.Render(fmt.Sprintf("+%d", adds)) + " " + p.Del.Render(fmt.Sprintf("−%d", dels))
}

func componentWords(tr *i18n.TranslationSet, d reviewData, c server.Component, filter pullFilter) string {
	if c.Context {
		n := 0
		for _, f := range d.page.Model.ContextFiles {
			if f.Component == c.ID {
				n++
			}
		}
		return utils.Plural(n, tr.FileNoun) + " · " + tr.Unchanged
	}
	n, adds, dels := 0, 0, 0
	for _, f := range d.page.Model.Files {
		if f.Component == c.ID && filter.showsAny(f.Pulls) {
			n, adds, dels = n+1, adds+f.Adds, dels+f.Dels
		}
	}
	return fmt.Sprintf("%s +%d −%d", utils.Plural(n, tr.FileNoun), adds, dels)
}

func stackMember(stack *server.PageStack, pull int) server.StackMember {
	for _, m := range stack.Members {
		if m.Pull == pull {
			return m
		}
	}
	return server.StackMember{Pull: pull}
}

func memberColor(s server.State, pull int) int {
	for _, m := range append(append([]server.Member{}, s.Members...), s.Earlier...) {
		if m.Pull == pull {
			return m.Color
		}
	}
	return 0
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
