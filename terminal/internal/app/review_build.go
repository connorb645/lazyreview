package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"review-desk/terminal/internal/server"
)

const preview = 220

type builder struct {
	lines []line
	items []item
	width int
	p     palette
}

func (b *builder) text(texts ...string) {
	for _, t := range texts {
		b.lines = append(b.lines, line{text: t, item: -1})
	}
}

func (b *builder) prose(indent string, style lipgloss.Style, text string) {
	for _, l := range wrap(text, b.width-len(indent)-1) {
		b.text(indent + style.Render(l))
	}
}

func (b *builder) add(it item, texts ...string) {
	b.items = append(b.items, it)
	for _, t := range texts {
		b.lines = append(b.lines, line{text: t, item: len(b.items) - 1})
	}
}

func buildReview(d reviewData, folded map[string]bool, width int, p palette, md *markdown, loc *time.Location) ([]line, []item) {
	b := &builder{width: width, p: p}
	reviewHeader(b, d)
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
			b.prose(" ", p.Warn, fmt.Sprintf("This written review read head %s; the pull request has moved on since, so it may not match the diff.", short(w.Head)))
		}
		for _, s := range w.Sections {
			s := s
			section("written:"+s.ID, s.Title, plural(len(s.Links), "link"), func() {
				for _, l := range md.lines(withoutDiagrams(s.Markdown), width-2) {
					b.text(" " + l)
				}
				for _, link := range s.Links {
					place := server.Lines{Path: link.Path, Side: "RIGHT", Line: link.Line}
					b.add(item{kind: "link", section: "written:" + s.ID, place: &place}, "   "+p.Accent.Render("→ "+link.Text)+p.Muted.Render(fmt.Sprintf("  %s line %d", link.Path, link.Line)))
				}
			})
		}
	} else if d.state.Queue != nil {
		b.text("")
		b.prose(" ", p.Muted, "No written review is attached. The diagram and the diffs hold the change.")
	} else {
		b.text("")
		b.prose(" ", p.Muted, "This branch has no pull request yet, so its review is read only.")
	}
	if q := d.state.Queue; q != nil {
		count := 0
		for _, m := range q.Members {
			count += len(m.Comments)
		}
		section("queue", "Queued comments", plural(count, "comment"), func() { queueLines(b, d, loc) })
	}
	if len(d.state.GitHub) > 0 {
		open := 0
		for _, t := range d.state.GitHub {
			if !t.Resolved {
				open++
			}
		}
		section("github", "On GitHub", plural(len(d.state.GitHub), "thread"), func() {
			b.prose(" ", p.Muted, fmt.Sprintf("%s, %d unresolved, as GitHub had them when this review was loaded.", plural(len(d.state.GitHub), "review thread"), open))
			for _, t := range d.state.GitHub {
				githubLines(b, d, t)
			}
		})
	}
	if len(d.state.Threads) > 0 {
		section("conversations", "Conversations", plural(len(d.state.Threads), "conversation"), func() {
			for _, t := range d.state.Threads {
				conversationLines(b, t, loc)
			}
		})
	}
	section("change", "The change", plural(len(d.page.Model.Components), "component"), func() {
		for _, c := range d.page.Model.Components {
			b.add(item{kind: "component", title: c.Title, section: "change", component: c.ID}, "   "+p.componentColor(c.Color).Render("■ ")+p.Ink.Render(c.Title)+"  "+p.Muted.Render(componentWords(d, c)))
		}
	})
	return b.lines, b.items
}

func reviewHeader(b *builder, d reviewData) {
	p, s := b.p, d.state
	title := s.Title
	if s.Written != nil && s.Written.Title != "" {
		title = s.Written.Title
	}
	totals := changeTotals(p, d.page.Model.Files)
	switch {
	case s.Kind == "branch":
		b.prose(" ", p.Muted, fmt.Sprintf("Branch %s in %s, with no pull request yet", s.Head.Ref, s.Repo))
		b.prose(" ", p.Bold, title)
		b.text(" " + p.Muted.Render(fmt.Sprintf("Base %s · Head %s · ", s.Base.Ref, short(s.Head.Sha))) + totals)
	case d.page.Stack != nil:
		b.prose(" ", p.Muted, fmt.Sprintf("A stack of %d pull requests, each built on the one before, reviewed together", len(s.Members)))
		b.prose(" ", p.Bold, title)
		for _, m := range s.Members {
			own := stackMember(d.page.Stack, m.Pull)
			meta := fmt.Sprintf("by %s · on %s · head %s · %s ", m.Author, m.Base, short(m.Head), plural(own.Files, "file"))
			state := ""
			if m.State != "" && m.State != "open" {
				state = p.Del.Render(" · " + m.State)
			}
			head := "  " + p.pull(m.Color).Render(fmt.Sprintf("#%d", m.Pull)) + "  " + p.Ink.Render(m.Title)
			tail := p.Muted.Render(meta) + p.Add.Render(fmt.Sprintf("+%d", own.Adds)) + " " + p.Del.Render(fmt.Sprintf("−%d", own.Dels)) + state
			if ansi.StringWidth(head)+2+ansi.StringWidth(tail) <= b.width {
				b.text(head + "  " + tail)
				continue
			}
			b.text(head, "       "+tail)
		}
		top := s.Members[len(s.Members)-1]
		b.text(" " + p.Muted.Render(fmt.Sprintf("Base %s · Head %s · Combined: ", s.Base.Ref, short(top.Head))) + totals)
		for _, w := range d.page.Stack.Warnings {
			b.prose(" ", p.Warn, w)
		}
	default:
		m := s.Members[0]
		state := ""
		if m.State != "" && m.State != "open" {
			state = " · " + m.State
		}
		b.text(" " + p.Muted.Render(fmt.Sprintf("Pull request #%d by %s%s", m.Pull, m.Author, state)))
		b.prose(" ", p.Bold, title)
		b.text(" " + p.Muted.Render(fmt.Sprintf("Base %s · Head %s · ", m.Base, short(m.Head))) + totals)
	}
	if s.Moved != nil {
		b.prose(" ", p.Warn, fmt.Sprintf("This review moved: its pull requests are reviewed together on %s. Its queued comments moved there, and nothing can be sent from here.", s.Moved.To))
	}
	for _, n := range s.Notes {
		for _, l := range wrap(n, b.width-2) {
			b.text(" " + p.Warn.Render(l))
		}
	}
}

func queueLines(b *builder, d reviewData, loc *time.Location) {
	p, q := b.p, d.state.Queue
	if q.Moved != nil {
		b.prose(" ", p.Muted, "This review moved to "+q.Moved.To+". Its queued comments moved there.")
		return
	}
	stacked := d.page.Stack != nil
	if stacked {
		b.prose(" ", p.Muted, "A comment goes on the pull request that last changed its line, shown as → #number.")
	}
	groups := append(append([]server.QueueMember{}, q.Members...), q.Earlier...)
	for i, m := range groups {
		left := i >= len(q.Members)
		color := memberColor(d.state, m.Pull)
		if stacked {
			b.text("")
			b.add(item{kind: "pull", section: "queue", title: fmt.Sprintf("#%d %s", m.Pull, m.Title), pull: m.Pull, url: m.URL}, "  "+p.pull(color).Render(fmt.Sprintf("#%d", m.Pull))+"  "+p.Ink.Render(m.Title))
		}
		if len(m.Comments) == 0 {
			b.text("   " + p.Muted.Render("No comments queued."))
		}
		for _, c := range m.Comments {
			commentLines(b, d, m, c, color, stacked)
		}
		switch {
		case m.State != "" && m.State != "open":
			at := ""
			if m.EndedAt != nil {
				at = " on " + when(*m.EndedAt, loc)
			}
			b.prose("   ", p.Muted, fmt.Sprintf("#%d was %s%s. Nothing can be sent to it.", m.Pull, m.State, at))
		case left:
			b.prose("   ", p.Muted, fmt.Sprintf("#%d left this stack. Nothing can be sent to it from here.", m.Pull))
		}
		for _, posted := range m.Posted {
			what := map[string]string{"comment": "Posted " + plural(posted.Count, "comment"), "approve-with-comments": "Approved with " + plural(posted.Count, "comment"), "approve": "Approved"}[posted.Action]
			b.prose("   ", p.Muted, fmt.Sprintf("%s, %s: %s", what, when(posted.At, loc), posted.Review))
		}
	}
}

func commentLines(b *builder, d reviewData, m server.QueueMember, c server.Comment, color int, stacked bool) {
	p := b.p
	head := "   " + p.Accent.Render(fmt.Sprintf("%d", c.ID)) + " "
	if stacked {
		head += p.pull(color).Render(fmt.Sprintf("→ #%d", m.Pull)) + " "
	}
	head += p.Ink.Render(c.Path + ", " + linesWords(c.Side, c.Line, c.StartLine, c.StartSide))
	place := server.Lines{Path: c.Path, Side: c.Side, Line: c.Line, StartLine: c.StartLine, StartSide: c.StartSide}
	if c.Shown != nil {
		place = *c.Shown
		if c.Shown.Path != c.Path || c.Shown.Line != c.Line {
			head += p.Muted.Render(fmt.Sprintf("  shown at line %d", c.Shown.Line))
		}
	}
	texts := []string{head}
	if c.Code != "" {
		for _, l := range strings.Split(c.Code, "\n") {
			texts = append(texts, "     "+p.Muted.Render("│ "+l))
		}
	}
	for _, l := range wrap(c.Body, b.width-6) {
		texts = append(texts, "     "+p.Ink.Render(l))
	}
	if c.Stale != "" {
		for _, l := range wrap("Its line may have moved: "+c.Stale+". Nothing sends it until it is queued again or dropped.", b.width-6) {
			texts = append(texts, "     "+p.Warn.Render(l))
		}
	}
	b.add(item{kind: "comment", section: "queue", pull: m.Pull, comment: c.ID, url: m.URL, place: &place}, texts...)
}

func githubLines(b *builder, d reviewData, t server.GitHubThread) {
	p := b.p
	where := t.Path
	switch {
	case t.File:
		where += ", the whole file"
	case t.At != nil:
		where += ":" + strings.TrimPrefix(linesWords(t.At.Side, t.At.Line, t.At.StartLine, t.At.StartSide), "line ")
	case t.OriginalLine > 0:
		where += fmt.Sprintf(", was line %d", t.OriginalLine)
	}
	var people []string
	for _, c := range t.Comments {
		if !contains(people, c.Author) {
			people = append(people, c.Author)
		}
	}
	meta := strings.Join(people, ", ") + " · " + plural(len(t.Comments)+t.More, "comment")
	if t.Resolved {
		meta += " · resolved"
	}
	if t.Outdated {
		meta += " · outdated"
	}
	head := "   " + p.Accent.Render(where) + "  "
	if d.page.Stack != nil {
		head += p.pull(memberColor(d.state, t.Pull)).Render(fmt.Sprintf("#%d", t.Pull)) + " "
	}
	texts := []string{head + p.Muted.Render(meta)}
	if len(t.Comments) > 0 {
		for _, l := range clipped(plainText(t.Comments[0].Body), b.width-6, 2) {
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
	p := b.p
	lines := fmt.Sprint(t.Line)
	if t.StartLine > 0 {
		lines = fmt.Sprintf("%d-%d", t.StartLine, t.Line)
	}
	texts := []string{"   " + p.Accent.Render(t.Path+":"+lines) + "  " + p.Muted.Render(when(t.CreatedAt, loc)+" · "+plural(len(t.Messages), "message"))}
	if len(t.Messages) > 0 {
		for _, l := range clipped(t.Messages[0].Text, b.width-6, 2) {
			texts = append(texts, "     "+p.Bold.Render(l))
		}
		last := t.Messages[len(t.Messages)-1]
		answer := "Waiting for Claude…"
		switch {
		case last.From == "claude" && last.Error != "":
			answer = "No answer: " + last.Error
		case last.From == "claude":
			answer = plainText(last.Text)
		}
		for _, l := range clipped(answer, b.width-6, 2) {
			texts = append(texts, "     "+p.Muted.Render(l))
		}
	}
	place := server.Lines{Path: t.Path, Side: t.Side, Line: t.Line, StartLine: t.StartLine, StartSide: t.StartSide}
	b.add(item{kind: "conversation", section: "conversations", place: &place}, texts...)
}

func changeTotals(p palette, files []server.FileEntry) string {
	adds, dels := 0, 0
	for _, f := range files {
		adds += f.Adds
		dels += f.Dels
	}
	return p.Muted.Render(plural(len(files), "file")+" ") + p.Add.Render(fmt.Sprintf("+%d", adds)) + " " + p.Del.Render(fmt.Sprintf("−%d", dels))
}

func componentWords(d reviewData, c server.Component) string {
	if c.Context {
		n := 0
		for _, f := range d.page.Model.ContextFiles {
			if f.Component == c.ID {
				n++
			}
		}
		return plural(n, "file") + " · unchanged"
	}
	n, adds, dels := 0, 0, 0
	for _, f := range d.page.Model.Files {
		if f.Component == c.ID {
			n, adds, dels = n+1, adds+f.Adds, dels+f.Dels
		}
	}
	return fmt.Sprintf("%s +%d −%d", plural(n, "file"), adds, dels)
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
