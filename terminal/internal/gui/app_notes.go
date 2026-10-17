package gui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

const cutLine = "# ------------------------ >8 ------------------------"

func (a App) noteAt() (presentation.Note, presentation.DiffRow, bool) {
	v := a.Diff
	if a.Screen() != context.ScreenDiff || v.Cursor >= len(v.Rows) {
		return presentation.Note{}, presentation.DiffRow{}, false
	}
	r := v.Rows[v.Cursor]
	if r.Kind != "notehead" && r.Kind != "notetext" {
		return presentation.Note{}, r, false
	}
	n, ok := presentation.NoteOf(*a.Review, r.Path, r.Key)
	return n, r, ok
}

func (a App) selected() (server.Lines, []string, bool) {
	v := a.Diff
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
			e := types.Editing{Purpose: "comment", At: *n.At, Text: text, Key: "comment " + linesKey(*n.At), About: []string{about}}
			return a.say(tr.CheckingWhere, false), checkWhere(a.opts.Server, a.Review.ID, a.Review.Page.Build, e)
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
	e := types.Editing{Purpose: "comment", At: at, Key: "comment " + linesKey(at), About: append(about, codeAbout(code)...)}
	a.Diff.Selecting = false
	return a.say(tr.CheckingWhere, false), checkWhere(a.opts.Server, a.Review.ID, a.Review.Page.Build, e)
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
		return a.edit(types.Editing{Purpose: "reply", Conversation: c.ID, At: *n.At, Key: fmt.Sprintf("reply %d", c.ID), About: about})
	}
	at, code, ok := a.selected()
	if !ok {
		return a.say(tr.AAsks, true), nil
	}
	a.Diff.Selecting = false
	about := append([]string{utils.ResolvePlaceholderString(tr.QuestionAbout, map[string]string{"on": onWords(tr, at)}), tr.ClaudeAnswersHere}, codeAbout(code)...)
	return a.edit(types.Editing{Purpose: "ask", At: at, Key: "ask " + linesKey(at), About: about})
}

func (a App) queuedAt() (server.Comment, int, bool) {
	id := 0
	if n, _, ok := a.noteAt(); ok && n.Kind == "queued" {
		id = n.Comment.ID
	}
	switch {
	case a.Focus() == context.NotesPanel:
		if row, ok := a.Side(context.NotesPanel).Selected(); ok && row.Item.Kind == "comment" {
			id = row.Item.Comment
		}
	case a.Focus() == context.MainPanel && a.Screen() == context.ScreenReview && a.ReviewView.View.Cursor < len(a.ReviewView.View.Items):
		if it := a.ReviewView.View.Items[a.ReviewView.View.Cursor]; it.Kind == "comment" {
			id = it.Comment
		}
	}
	if id == 0 || a.Review.State.Queue == nil {
		return server.Comment{}, 0, false
	}
	for _, m := range a.Review.State.Queue.Members {
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
	return a.edit(types.Editing{Purpose: "edit", Comment: c.ID, Version: c.Version, At: at, Pull: pull, Text: c.Body, Key: fmt.Sprintf("edit %d", c.ID), About: about})
}

func (a App) dropQueued() App {
	c, pull, ok := a.queuedAt()
	if !ok {
		return a.say(a.c.Tr.DdDrops, true)
	}
	a.ContextTree = a.Opened(context.ConfirmationKey)
	a.Confirmation = context.ConfirmationContext{Comment: c.ID, Pull: pull, Version: c.Version}
	return a
}

func (a App) confirmKey(name string) (tea.Model, tea.Cmd) {
	c := a.Confirmation
	a.ContextTree = a.Closed(context.ConfirmationKey)
	if name != "y" {
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.KeptQueued, map[string]string{"comment": utils.Itoa(c.Comment)}), false), nil
	}
	client, id, build := a.opts.Server, a.Review.ID, a.Review.Page.Build
	e := types.Editing{Purpose: "drop", Comment: c.Comment, Pull: c.Pull}
	return a.say(a.c.Tr.Dropping, false), func() tea.Msg {
		state, err := client.Drop(id, c.Comment, c.Version, build)
		return types.Changed{E: e, State: &state, Err: err}
	}
}

func (a App) edit(e types.Editing) (tea.Model, tea.Cmd) {
	text := e.Text
	if draft, ok := a.Drafts[e.Key]; ok {
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
	for _, l := range e.About {
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
	a.EditingNow = true
	return a, a.opts.Exec(cmd, func(err error) tea.Msg { return types.Edited{E: e, File: file.Name(), Err: err} })
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

func (a App) afterEditing(msg types.Edited) (tea.Model, tea.Cmd) {
	a.EditingNow = false
	tr := a.c.Tr
	text, readErr := written(msg.File)
	if err := os.Remove(msg.File); err != nil && readErr == nil {
		readErr = err
	}
	switch {
	case msg.Err != nil:
		if text != "" {
			a.Drafts = withDraft(a.Drafts, msg.E.Key, text)
		}
		return a.say(tr.EditorStopped+": "+msg.Err.Error(), true), nil
	case readErr != nil:
		return a.say(tr.CouldNotReadWritten+": "+readErr.Error(), true), nil
	case text == "":
		a.Drafts = withDraft(a.Drafts, msg.E.Key, "")
		return a.say(tr.NothingWritten, false), nil
	case msg.E.Purpose == "edit" && text == msg.E.Text:
		a.Drafts = withDraft(a.Drafts, msg.E.Key, "")
		return a.say(utils.ResolvePlaceholderString(tr.QueuedAsItWas, map[string]string{"comment": utils.Itoa(msg.E.Comment)}), false), nil
	}
	a.Drafts = withDraft(a.Drafts, msg.E.Key, text)
	client, id, build, e := a.opts.Server, a.Review.ID, a.Review.Page.Build, msg.E
	switch e.Purpose {
	case "comment":
		return a.say(tr.Queueing, false), func() tea.Msg {
			state, err := client.Queue(id, e.At, e.Pull, text, build)
			return types.Changed{E: e, State: &state, Err: err}
		}
	case "edit":
		return a.say(tr.Saving, false), func() tea.Msg {
			state, err := client.Edit(id, e.Comment, text, e.Version, build)
			return types.Changed{E: e, State: &state, Err: err}
		}
	case "reply":
		return a.say(tr.AskingClaude, false), func() tea.Msg {
			c, err := client.Reply(id, e.Conversation, text, build)
			return types.Changed{E: e, Conversation: &c, Err: err}
		}
	}
	return a.say(tr.AskingClaude, false), func() tea.Msg {
		c, err := client.Ask(id, e.At, text, build)
		return types.Changed{E: e, Conversation: &c, Err: err}
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

func (a App) afterChange(msg types.Changed) (tea.Model, tea.Cmd) {
	e := msg.E
	if msg.Err != nil {
		var refusal server.Refusal
		if (e.Purpose == "edit" || e.Purpose == "drop") && errors.As(msg.Err, &refusal) {
			return a.say(msg.Err.Error(), true), loadState(a.opts.Server, a.Review.ID)
		}
		return a.say(msg.Err.Error(), true), nil
	}
	a.Drafts = withDraft(a.Drafts, e.Key, "")
	if msg.Conversation != nil {
		c := *msg.Conversation
		a = a.withState(withConversation(a.Review.State, c))
		a.Diff.Notes = utils.CopyBools(a.Diff.Notes)
		a.Diff.Notes[fmt.Sprintf("conversation:%d", c.ID)] = true
		a = a.relaid()
		if last := c.Messages[len(c.Messages)-1]; last.Error != "" {
			return a.say(a.c.Tr.ClaudeSaysWhy, true), nil
		}
		return a.say(a.c.Tr.ClaudeAnswered, false), nil
	}
	a = a.withState(*msg.State)
	comment := map[string]string{"comment": utils.Itoa(e.Comment)}
	switch e.Purpose {
	case "edit":
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.SavedQueued, comment), false), nil
	case "drop":
		return a.say(utils.ResolvePlaceholderString(a.c.Tr.DroppedQueued, comment), false), nil
	}
	return a.say(utils.ResolvePlaceholderString(a.c.Tr.QueuedAbout, map[string]string{"about": strings.TrimSuffix(e.About[0], ".")}), false), nil
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
	d := *a.Review
	d.State = st
	a.Review = &d
	return a.relaid()
}

func (a App) afterCheck(msg types.Checked) (tea.Model, tea.Cmd) {
	switch {
	case msg.Err != nil:
		return a.say(msg.Err.Error(), true), nil
	case msg.Where.Refused != "":
		return a.say(msg.Where.Refused, true), nil
	case len(msg.Where.Choose) > 0:
		pending := msg.E
		pn := context.MenuContext{Menu: presentation.Menu{Title: msg.Where.Words}}
		for _, c := range msg.Where.Choose {
			line := utils.ResolvePlaceholderString(a.c.Tr.LineAt, map[string]string{"line": utils.Itoa(c.Line)})
			pn.Rows = append(pn.Rows, presentation.MenuRow{Left: fmt.Sprintf("#%d", c.Pull), Right: c.Path + ", " + line, Open: &presentation.Item{Kind: "choose", Pull: c.Pull}})
		}
		a = a.withMenu(pn)
		a.Choosing = &pending
		return a.say("", false), nil
	}
	e := msg.E
	e.About = append([]string{e.About[0], msg.Where.Words}, e.About[1:]...)
	return a.say("", false).edit(e)
}

func checkWhere(c *server.Client, id, build string, e types.Editing) tea.Cmd {
	return func() tea.Msg {
		where, err := c.Where(id, e.At, e.Pull, build)
		return types.Checked{E: e, Where: where, Err: err}
	}
}

func (a App) whereNow() tea.Cmd {
	if a.Screen() != context.ScreenDiff || !a.Diff.Selecting || a.Review == nil {
		return nil
	}
	at, _, ok := a.selected()
	if !ok || linesKey(at) == a.Where.Key {
		return nil
	}
	client, id, build, key := a.opts.Server, a.Review.ID, a.Review.Page.Build, linesKey(at)
	return func() tea.Msg {
		where, err := client.Where(id, at, 0, build)
		words := where.Words
		switch {
		case err != nil:
			words = err.Error()
		case where.Refused != "":
			words = where.Refused
		}
		return types.WhereShown{Key: key, Words: words}
	}
}

func loadState(c *server.Client, id string) tea.Cmd {
	return func() tea.Msg {
		state, err := c.State(id)
		return types.StateLoaded{State: state, Err: err}
	}
}
