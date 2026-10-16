package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/server"
)

const cutLine = "# ------------------------ >8 ------------------------"

func (a App) noteAt() (note, diffRow, bool) {
	v := a.diff
	if a.screen != screenDiff || v.cursor >= len(v.rows) {
		return note{}, diffRow{}, false
	}
	r := v.rows[v.cursor]
	if r.kind != "notehead" && r.kind != "notetext" {
		return note{}, r, false
	}
	n, ok := noteOf(*a.review, r.path, r.key)
	return n, r, ok
}

func (a App) selected() (server.Lines, []string, bool) {
	v := a.diff
	if v.cursor >= len(v.rows) {
		return server.Lines{}, nil, false
	}
	from, to := v.cursor, v.cursor
	if v.selecting {
		from, to = min(v.anchor, v.cursor), max(v.anchor, v.cursor)
	}
	var first, last server.Lines
	var firstBlock, seen *server.Block
	var code []string
	for i := from; i <= to; i++ {
		r := v.rows[i]
		var start, end server.Lines
		switch side, line, ok := r.at(); {
		case r.kind == "block":
			b := r.block
			start = server.Lines{Path: r.path, Side: b.Side, Line: b.StartLine}
			end = server.Lines{Path: r.path, Side: b.Side, Line: b.Line}
			if b != seen {
				code = append(code, blockCode(b)...)
				seen = b
			}
			if firstBlock == nil {
				firstBlock = b
			}
		case ok:
			start = server.Lines{Path: r.path, Side: side, Line: line}
			end = start
			code = append(code, codeLine(r))
		default:
			continue
		}
		if first.Path != "" && r.path != first.Path {
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

func codeLine(r diffRow) string {
	l := r.line
	if l == nil {
		l = r.right
		if l == nil || l.Kind == "same" && r.left != nil {
			l = r.left
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
	return fmt.Sprintf("%4d %s %s", n, sign, expandTabs(l.Text))
}

func linesKey(at server.Lines) string {
	return fmt.Sprintf("%s %s %d %s %d", at.Path, at.StartSide, at.StartLine, at.Side, at.Line)
}

func onWords(at server.Lines) string {
	return at.Path + ", " + linesWords(at.Side, at.Line, at.StartLine, at.StartSide)
}

func (a App) comment() (tea.Model, tea.Cmd) {
	if n, r, ok := a.noteAt(); ok {
		switch {
		case n.kind == "conversation" && r.part == "claude":
			text := n.conversation.Messages[r.message].Text
			e := editing{purpose: "comment", at: *n.at, text: text, key: "comment " + linesKey(*n.at), about: []string{"A comment on " + onWords(*n.at) + ", from Claude's answer."}}
			return a.say("Checking where it goes…", false), checkWhere(a.opts.Server, a.review.id, a.review.page.Build, e)
		case n.kind == "queued":
			return a.say("e edits a queued comment", true), nil
		}
		return a.say("c comments on a line or a selection, or on Claude's answer uses it", true), nil
	}
	at, code, ok := a.selected()
	if !ok {
		return a.say("c comments on a line or a selection", true), nil
	}
	about := []string{"A comment on " + onWords(at) + "."}
	e := editing{purpose: "comment", at: at, key: "comment " + linesKey(at), about: append(about, codeAbout(code)...)}
	a.diff.selecting = false
	return a.say("Checking where it goes…", false), checkWhere(a.opts.Server, a.review.id, a.review.page.Build, e)
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
	if n, _, ok := a.noteAt(); ok && n.kind == "conversation" {
		c := n.conversation
		about := []string{"A follow-up to Claude on " + onWords(*n.at) + ".", "The conversation so far:", ""}
		for _, m := range c.Messages {
			who, text := "You", m.Text
			if m.From != "you" {
				who = "Claude"
				if m.Error != "" {
					text = "did not answer: " + m.Error
				}
			}
			for _, l := range strings.Split(who+": "+text, "\n") {
				about = append(about, "  "+l)
			}
		}
		return a.edit(editing{purpose: "reply", conversation: c.ID, at: *n.at, key: fmt.Sprintf("reply %d", c.ID), about: about})
	}
	at, code, ok := a.selected()
	if !ok {
		return a.say("a asks Claude about a line or a selection, or in a conversation asks a follow-up", true), nil
	}
	a.diff.selecting = false
	about := append([]string{"A question to Claude about " + onWords(at) + ".", "Claude answers here, in the review. Nothing goes to GitHub."}, codeAbout(code)...)
	return a.edit(editing{purpose: "ask", at: at, key: "ask " + linesKey(at), about: about})
}

func (a App) queuedAt() (server.Comment, int, bool) {
	id := 0
	if n, _, ok := a.noteAt(); ok && n.kind == "queued" {
		id = n.comment.ID
	}
	switch {
	case a.focus == notesPanel:
		if row, ok := a.sides[notesPanel].selected(); ok && row.it.kind == "comment" {
			id = row.it.comment
		}
	case a.focus == mainPanel && a.screen == screenReview && a.rv.view.cursor < len(a.rv.view.items):
		if it := a.rv.view.items[a.rv.view.cursor]; it.kind == "comment" {
			id = it.comment
		}
	}
	if id == 0 || a.review.state.Queue == nil {
		return server.Comment{}, 0, false
	}
	for _, m := range a.review.state.Queue.Members {
		for _, c := range m.Comments {
			if c.ID == id {
				return c, m.Pull, true
			}
		}
	}
	return server.Comment{}, 0, false
}

func (a App) editQueued() (tea.Model, tea.Cmd) {
	c, pull, ok := a.queuedAt()
	if !ok {
		return a.say("e edits the queued comment at the cursor", true), nil
	}
	at := server.Lines{Path: c.Path, Side: c.Side, Line: c.Line, StartLine: c.StartLine, StartSide: c.StartSide}
	about := []string{fmt.Sprintf("Queued comment %d on %s, for #%d.", c.ID, onWords(at), pull), "Saving replaces it."}
	return a.edit(editing{purpose: "edit", comment: c.ID, version: c.Version, at: at, pull: pull, text: c.Body, key: fmt.Sprintf("edit %d", c.ID), about: about})
}

func (a App) dropQueued() App {
	c, pull, ok := a.queuedAt()
	if !ok {
		return a.say("dd drops the queued comment at the cursor", true)
	}
	a.confirming = &confirmation{comment: c.ID, pull: pull, version: c.Version}
	return a
}

func (a App) confirmKey(name string) (tea.Model, tea.Cmd) {
	c := *a.confirming
	a.confirming = nil
	if name != "y" {
		return a.say(fmt.Sprintf("Kept queued comment %d.", c.comment), false), nil
	}
	client, id, build := a.opts.Server, a.review.id, a.review.page.Build
	e := editing{purpose: "drop", comment: c.comment, pull: c.pull}
	return a.say("Dropping…", false), func() tea.Msg {
		state, err := client.Drop(id, c.comment, c.version, build)
		return changed{e: e, state: &state, err: err}
	}
}

func (a App) edit(e editing) (tea.Model, tea.Cmd) {
	text := e.text
	if draft, ok := a.drafts[e.key]; ok {
		text = draft
	}
	file, err := os.CreateTemp("", "lazyreview-*.md")
	if err != nil {
		return a.say("could not make a file to write in: "+err.Error(), true), nil
	}
	var b strings.Builder
	b.WriteString(text)
	b.WriteString("\n\n" + cutLine + "\n# Nothing below the line above is sent. Saving with no text sends nothing.\n")
	for _, l := range e.about {
		b.WriteString(strings.TrimRight("# "+l, " ") + "\n")
	}
	_, err = file.WriteString(b.String())
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return a.say("could not write "+file.Name()+": "+err.Error(), true), nil
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
	text, readErr := written(msg.file)
	if err := os.Remove(msg.file); err != nil && readErr == nil {
		readErr = err
	}
	switch {
	case msg.err != nil:
		if text != "" {
			a.drafts = withDraft(a.drafts, msg.e.key, text)
		}
		return a.say("the editor stopped: "+msg.err.Error(), true), nil
	case readErr != nil:
		return a.say("could not read what was written: "+readErr.Error(), true), nil
	case text == "":
		a.drafts = withDraft(a.drafts, msg.e.key, "")
		return a.say("Nothing was written, so nothing was sent.", false), nil
	case msg.e.purpose == "edit" && text == msg.e.text:
		a.drafts = withDraft(a.drafts, msg.e.key, "")
		return a.say(fmt.Sprintf("Queued comment %d is as it was.", msg.e.comment), false), nil
	}
	a.drafts = withDraft(a.drafts, msg.e.key, text)
	client, id, build, e := a.opts.Server, a.review.id, a.review.page.Build, msg.e
	switch e.purpose {
	case "comment":
		return a.say("Queueing…", false), func() tea.Msg {
			state, err := client.Queue(id, e.at, e.pull, text, build)
			return changed{e: e, state: &state, err: err}
		}
	case "edit":
		return a.say("Saving…", false), func() tea.Msg {
			state, err := client.Edit(id, e.comment, text, e.version, build)
			return changed{e: e, state: &state, err: err}
		}
	case "reply":
		return a.say("Asking Claude…", false), func() tea.Msg {
			c, err := client.Reply(id, e.conversation, text, build)
			return changed{e: e, conversation: &c, err: err}
		}
	}
	return a.say("Asking Claude…", false), func() tea.Msg {
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
			return a.say(msg.err.Error(), true), loadState(a.opts.Server, a.review.id)
		}
		return a.say(msg.err.Error(), true), nil
	}
	a.drafts = withDraft(a.drafts, e.key, "")
	if msg.conversation != nil {
		c := *msg.conversation
		a = a.withState(withConversation(a.review.state, c))
		a.diff.notes = copyBools(a.diff.notes)
		a.diff.notes[fmt.Sprintf("conversation:%d", c.ID)] = true
		a = a.relaid()
		if last := c.Messages[len(c.Messages)-1]; last.Error != "" {
			return a.say("Claude did not answer; the conversation says why.", true), nil
		}
		return a.say("Claude answered.", false), nil
	}
	a = a.withState(*msg.state)
	switch e.purpose {
	case "edit":
		return a.say(fmt.Sprintf("Saved queued comment %d.", e.comment), false), nil
	case "drop":
		return a.say(fmt.Sprintf("Dropped queued comment %d.", e.comment), false), nil
	}
	return a.say("Queued: "+strings.TrimSuffix(e.about[0], "."), false), nil
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
	d.state = st
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
		pn := menu{title: msg.where.Words}
		for _, c := range msg.where.Choose {
			pn.rows = append(pn.rows, menuRow{left: fmt.Sprintf("#%d", c.Pull), right: fmt.Sprintf("%s, line %d", c.Path, c.Line), open: &item{kind: "choose", pull: c.Pull}})
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
	if a.screen != screenDiff || !a.diff.selecting || a.review == nil {
		return nil
	}
	at, _, ok := a.selected()
	if !ok || linesKey(at) == a.where.key {
		return nil
	}
	client, id, build, key := a.opts.Server, a.review.id, a.review.page.Build, linesKey(at)
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
