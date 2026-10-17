package gui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const cutLine = "# ------------------------ >8 ------------------------"

func (a App) noteAt() (presentation.Note, presentation.DiffRow, bool) {
	v := a.diff
	if a.screen != screenDiff || v.Cursor >= len(v.Rows) {
		return presentation.Note{}, presentation.DiffRow{}, false
	}
	r := v.Rows[v.Cursor]
	if r.Kind != "notehead" && r.Kind != "notetext" {
		return presentation.Note{}, r, false
	}
	n, ok := presentation.NoteOf(*a.review, r.Path, r.Key)
	return n, r, ok
}

func (a App) selected() (server.Lines, []string, bool) {
	v := a.diff
	if v.Cursor >= len(v.Rows) {
		return server.Lines{}, nil, false
	}
	from, to := v.Cursor, v.Cursor
	if v.Selecting {
		from, to = min(v.Anchor, v.Cursor), max(v.Anchor, v.Cursor)
	}
	var first, last server.Lines
	var firstBlock, seen *server.Block
	var code []string
	for i := from; i <= to; i++ {
		r := v.Rows[i]
		var start, end server.Lines
		switch side, line, ok := r.At(); {
		case r.Kind == "block":
			b := r.Block
			start = server.Lines{Path: r.Path, Side: b.Side, Line: b.StartLine}
			end = server.Lines{Path: r.Path, Side: b.Side, Line: b.Line}
			if b != seen {
				code = append(code, blockCode(b)...)
				seen = b
			}
			if firstBlock == nil {
				firstBlock = b
			}
		case ok:
			start = server.Lines{Path: r.Path, Side: side, Line: line}
			end = start
			code = append(code, codeLine(r))
		default:
			continue
		}
		if first.Path != "" && r.Path != first.Path {
			return server.Lines{}, nil, false
		}
		if first.Path == "" {
			first = start
		}
		last = end
	}
	if first.Path == "" {
		return server.Lines{}, nil, false
	}
	if first.Side != last.Side && last.Side == "LEFT" && firstBlock != nil && firstBlock.Left != nil {
		first.Side, first.Line = "LEFT", firstBlock.Left.StartLine
	}
	if first != last {
		last.StartLine, last.StartSide = first.Line, first.Side
	}
	return last, code, true
}

func blockCode(b *server.Block) []string {
	sign := map[string]string{"add": "+", "del": "-"}[b.State]
	if sign == "" {
		sign = " "
	}
	var out []string
	for i, text := range strings.Split(b.Markdown, "\n") {
		out = append(out, fmt.Sprintf("%4d %s %s", b.StartLine+i, sign, text))
	}
	return out
}

func codeLine(r presentation.DiffRow) string {
	l := r.Line
	if l == nil {
		l = r.Right
		if l == nil || l.Kind == "same" && r.Left != nil {
			l = r.Left
		}
	}
	sign := map[string]string{"add": "+", "del": "-"}[l.Kind]
	if sign == "" {
		sign = " "
	}
	n := l.New
	if l.Kind == "del" {
		n = l.Old
	}
	return fmt.Sprintf("%4d %s %s", n, sign, presentation.ExpandTabs(l.Text))
}

func linesKey(at server.Lines) string {
	return fmt.Sprintf("%s %s %d %s %d", at.Path, at.StartSide, at.StartLine, at.Side, at.Line)
}

func onWords(tr *i18n.TranslationSet, at server.Lines) string {
	return at.Path + ", " + presentation.LinesWords(tr, at.Side, at.Line, at.StartLine, at.StartSide)
}

func (a App) comment() (tea.Model, tea.Cmd) {
	tr := a.c.Tr
	if n, r, ok := a.noteAt(); ok {
		switch {
		case n.Kind == "conversation" && r.Part == "claude":
			text := n.Conversation.Messages[r.Message].Text
			about := utils.ResolvePlaceholderString(tr.CommentFromAnswer, map[string]string{"on": onWords(tr, *n.At)})
			e := editing{purpose: "comment", at: *n.At, text: text, key: "comment " + linesKey(*n.At), about: []string{about}}
			return a.say(tr.CheckingWhere, false), checkWhere(a.opts.Server, a.review.ID, a.review.Page.Build, e)
		case n.Kind == "queued":
			return a.say(tr.EEditsQueued, true), nil
		}
		return a.say(tr.CCommentsOrAnswer, true), nil
	}
	at, code, ok := a.selected()
	if !ok {
		return a.say(tr.CComments, true), nil
	}
	about := []string{utils.ResolvePlaceholderString(tr.CommentOn, map[string]string{"on": onWords(tr, at)})}
	e := editing{purpose: "comment", at: at, key: "comment " + linesKey(at), about: append(about, codeAbout(code)...)}
	a.diff.Selecting = false
	return a.say(tr.CheckingWhere, false), checkWhere(a.opts.Server, a.review.ID, a.review.Page.Build, e)
}

func codeAbout(code []string) []string {
	if len(code) == 0 {
		return nil
	}
	out := []string{""}
	for _, c := range code {
		out = append(out, "  "+c)
	}
	return out
}

func (a App) askClaude() (tea.Model, tea.Cmd) {
	tr := a.c.Tr
	if n, _, ok := a.noteAt(); ok && n.Kind == "conversation" {
		c := n.Conversation
		about := []string{utils.ResolvePlaceholderString(tr.FollowUpOn, map[string]string{"on": onWords(tr, *n.At)}), tr.ConversationSoFar, ""}
		for _, m := range c.Messages {
			who, text := tr.You, m.Text
			if m.From != "you" {
				who = tr.Claude
				if m.Error != "" {
					text = tr.DidNotAnswer + ": " + m.Error
				}
			}
			for _, l := range strings.Split(who+": "+text, "\n") {
				about = append(about, "  "+l)
			}
		}
		return a.edit(editing{purpose: "reply", conversation: c.ID, at: *n.At, key: fmt.Sprintf("reply %d", c.ID), about: about})
	}
	at, code, ok := a.selected()
	if !ok {
		return a.say(tr.AAsks, true), nil
	}
	a.diff.Selecting = false
	about := append([]string{utils.ResolvePlaceholderString(tr.QuestionAbout, map[string]string{"on": onWords(tr, at)}), tr.ClaudeAnswersHere}, codeAbout(code)...)
	return a.edit(editing{purpose: "ask", at: at, key: "ask " + linesKey(at), about: about})
}

func (a App) queuedAt() (server.Comment, int, bool) {
	id := 0
	if n, _, ok := a.noteAt(); ok && n.Kind == "queued" {
		id = n.Comment.ID
	}
	switch {
	case a.focus == notesPanel:
		if row, ok := a.sides[notesPanel].selected(); ok && row.Item.Kind == "comment" {
			id = row.Item.Comment
		}
	case a.focus == mainPanel && a.screen == screenReview && a.rv.view.cursor < len(a.rv.view.items):
		if it := a.rv.view.items[a.rv.view.cursor]; it.Kind == "comment" {
			id = it.Comment
		}
	}
	if id == 0 || a.review.State.Queue == nil {
		return server.Comment{}, 0, false
	}
	for _, m := range a.review.State.Queue.Members {
		for _, c := range m.Comments {
			if c.ID == id {
				return c, m.Pull, true
			}
		}
	}
	return server.Comment{}, 0, false
}

func (a App) editQueued() (tea.Model, tea.Cmd) {
	tr := a.c.Tr
	c, pull, ok := a.queuedAt()
	if !ok {
		return a.say(tr.EEditsAtCursor, true), nil
	}
	at := server.Lines{Path: c.Path, Side: c.Side, Line: c.Line, StartLine: c.StartLine, StartSide: c.StartSide}
	queued := utils.ResolvePlaceholderString(tr.QueuedCommentOn, map[string]string{"comment": utils.Itoa(c.ID), "on": onWords(tr, at), "pull": utils.Itoa(pull)})
	about := []string{queued, tr.SavingReplaces}
	return a.edit(editing{purpose: "edit", comment: c.ID, version: c.Version, at: at, pull: pull, text: c.Body, key: fmt.Sprintf("edit %d", c.ID), about: about})
}

func (a App) dropQueued() App {
	c, pull, ok := a.queuedAt()
	if !ok {
		return a.say(a.c.Tr.DdDrops, true)
	}
	a.confirming = &confirmation{comment: c.ID, pull: pull, version: c.Version}
	return a
}

func (a App) confirmKey(name string) (tea.Model, tea.Cmd) {
	c := *a.confirming
	a.confirming = nil
	if name != "y" {
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.KeptQueued, map[string]string{"comment": utils.Itoa(c.comment)}), false), nil
	}
	client, id, build := a.opts.Server, a.review.ID, a.review.Page.Build
	e := editing{purpose: "drop", comment: c.comment, pull: c.pull}
	return a.say(a.c.Tr.Dropping, false), func() tea.Msg {
		state, err := client.Drop(id, c.comment, c.version, build)
		return changed{e: e, state: &state, err: err}
	}
}

func (a App) edit(e editing) (tea.Model, tea.Cmd) {
	text := e.text
	if draft, ok := a.drafts[e.key]; ok {
		text = draft
	}
	tr := a.c.Tr
	file, err := os.CreateTemp("", "lazyreview-*.md")
	if err != nil {
		return a.say(tr.CouldNotMakeFile+": "+err.Error(), true), nil
	}
	var b strings.Builder
	b.WriteString(text)
	b.WriteString("\n\n" + cutLine + "\n# " + tr.NothingBelowLine + "\n")
	for _, l := range e.about {
		b.WriteString(strings.TrimRight("# "+l, " ") + "\n")
	}
	_, err = file.WriteString(b.String())
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return a.say(utils.ResolvePlaceholderString(tr.CouldNotWrite, map[string]string{"path": file.Name()})+": "+err.Error(), true), nil
	}
	cmd := exec.Command("sh", "-c", a.opts.Editor+` "$@"`, "editor", file.Name())
	a.editingNow = true
	return a, a.opts.Exec(cmd, func(err error) tea.Msg { return edited{e: e, file: file.Name(), err: err} })
}

func written(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	text := string(data)
	if i := strings.Index(text, "\n"+cutLine); i >= 0 {
		text = text[:i]
	} else if strings.HasPrefix(text, cutLine) {
		text = ""
	}
	return strings.TrimSpace(text), nil
}

func (a App) afterEditing(msg edited) (tea.Model, tea.Cmd) {
	a.editingNow = false
	tr := a.c.Tr
	text, readErr := written(msg.file)
	if err := os.Remove(msg.file); err != nil && readErr == nil {
		readErr = err
	}
	switch {
	case msg.err != nil:
		if text != "" {
			a.drafts = withDraft(a.drafts, msg.e.key, text)
		}
		return a.say(tr.EditorStopped+": "+msg.err.Error(), true), nil
	case readErr != nil:
		return a.say(tr.CouldNotReadWritten+": "+readErr.Error(), true), nil
	case text == "":
		a.drafts = withDraft(a.drafts, msg.e.key, "")
		return a.say(tr.NothingWritten, false), nil
	case msg.e.purpose == "edit" && text == msg.e.text:
		a.drafts = withDraft(a.drafts, msg.e.key, "")
		return a.say(utils.ResolvePlaceholderString(tr.QueuedAsItWas, map[string]string{"comment": utils.Itoa(msg.e.comment)}), false), nil
	}
	a.drafts = withDraft(a.drafts, msg.e.key, text)
	client, id, build, e := a.opts.Server, a.review.ID, a.review.Page.Build, msg.e
	switch e.purpose {
	case "comment":
		return a.say(tr.Queueing, false), func() tea.Msg {
			state, err := client.Queue(id, e.at, e.pull, text, build)
			return changed{e: e, state: &state, err: err}
		}
	case "edit":
		return a.say(tr.Saving, false), func() tea.Msg {
			state, err := client.Edit(id, e.comment, text, e.version, build)
			return changed{e: e, state: &state, err: err}
		}
	case "reply":
		return a.say(tr.AskingClaude, false), func() tea.Msg {
			c, err := client.Reply(id, e.conversation, text, build)
			return changed{e: e, conversation: &c, err: err}
		}
	}
	return a.say(tr.AskingClaude, false), func() tea.Msg {
		c, err := client.Ask(id, e.at, text, build)
		return changed{e: e, conversation: &c, err: err}
	}
}

func withDraft(drafts map[string]string, key, text string) map[string]string {
	out := map[string]string{}
	for k, v := range drafts {
		out[k] = v
	}
	if text == "" {
		delete(out, key)
	} else {
		out[key] = text
	}
	return out
}

func (a App) afterChange(msg changed) (tea.Model, tea.Cmd) {
	e := msg.e
	if msg.err != nil {
		var refusal server.Refusal
		if (e.purpose == "edit" || e.purpose == "drop") && errors.As(msg.err, &refusal) {
			return a.say(msg.err.Error(), true), loadState(a.opts.Server, a.review.ID)
		}
		return a.say(msg.err.Error(), true), nil
	}
	a.drafts = withDraft(a.drafts, e.key, "")
	if msg.conversation != nil {
		c := *msg.conversation
		a = a.withState(withConversation(a.review.State, c))
		a.diff.Notes = copyBools(a.diff.Notes)
		a.diff.Notes[fmt.Sprintf("conversation:%d", c.ID)] = true
		a = a.relaid()
		if last := c.Messages[len(c.Messages)-1]; last.Error != "" {
			return a.say(a.c.Tr.ClaudeSaysWhy, true), nil
		}
		return a.say(a.c.Tr.ClaudeAnswered, false), nil
	}
	a = a.withState(*msg.state)
	comment := map[string]string{"comment": utils.Itoa(e.comment)}
	switch e.purpose {
	case "edit":
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.SavedQueued, comment), false), nil
	case "drop":
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.DroppedQueued, comment), false), nil
	}
	return a.say(utils.ResolvePlaceholderString(a.c.Tr.QueuedAbout, map[string]string{"about": strings.TrimSuffix(e.about[0], ".")}), false), nil
}

func withConversation(st server.State, c server.Conversation) server.State {
	threads := append([]server.Conversation{}, st.Threads...)
	for i := range threads {
		if threads[i].ID == c.ID {
			threads[i] = c
			st.Threads = threads
			return st
		}
	}
	st.Threads = append(threads, c)
	return st
}

func (a App) withState(st server.State) App {
	d := *a.review
	d.State = st
	a.review = &d
	return a.relaid()
}

func (a App) afterCheck(msg checked) (tea.Model, tea.Cmd) {
	switch {
	case msg.err != nil:
		return a.say(msg.err.Error(), true), nil
	case msg.where.Refused != "":
		return a.say(msg.where.Refused, true), nil
	case len(msg.where.Choose) > 0:
		pending := msg.e
		pn := menu{Menu: presentation.Menu{Title: msg.where.Words}}
		for _, c := range msg.where.Choose {
			line := utils.ResolvePlaceholderString(a.c.Tr.LineAt, map[string]string{"line": utils.Itoa(c.Line)})
			pn.Rows = append(pn.Rows, presentation.MenuRow{Left: fmt.Sprintf("#%d", c.Pull), Right: c.Path + ", " + line, Open: &presentation.Item{Kind: "choose", Pull: c.Pull}})
		}
		a.menu, a.choosing = &pn, &pending
		return a.say("", false), nil
	}
	e := msg.e
	e.about = append([]string{e.about[0], msg.where.Words}, e.about[1:]...)
	return a.say("", false).edit(e)
}

func checkWhere(c *server.Client, id, build string, e editing) tea.Cmd {
	return func() tea.Msg {
		where, err := c.Where(id, e.at, e.pull, build)
		return checked{e: e, where: where, err: err}
	}
}

func (a App) whereNow() tea.Cmd {
	if a.screen != screenDiff || !a.diff.Selecting || a.review == nil {
		return nil
	}
	at, _, ok := a.selected()
	if !ok || linesKey(at) == a.where.key {
		return nil
	}
	client, id, build, key := a.opts.Server, a.review.ID, a.review.Page.Build, linesKey(at)
	return func() tea.Msg {
		where, err := client.Where(id, at, 0, build)
		words := where.Words
		switch {
		case err != nil:
			words = err.Error()
		case where.Refused != "":
			words = where.Refused
		}
		return whereShown{key: key, words: words}
	}
}

func loadState(c *server.Client, id string) tea.Cmd {
	return func() tea.Msg {
		state, err := c.State(id)
		return stateLoaded{state: state, err: err}
	}
}
