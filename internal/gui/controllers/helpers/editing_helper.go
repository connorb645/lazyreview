package helpers

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/internal/gui/context"
	"github.com/connorb645/lazyreview/internal/gui/presentation"
	"github.com/connorb645/lazyreview/internal/gui/types"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

const cutLine = "# ------------------------ >8 ------------------------"

// EditingHelper writes comments, questions and answers in the editor, and
// queues, edits and drops comments.
type EditingHelper struct {
	c *HelperCommon
}

// NoteAt is the note the diff's cursor is on, and its row, if it is on one.
func (self *EditingHelper) NoteAt(s context.ContextTree) (presentation.Note, presentation.DiffRow, bool) {
	v := s.Diff
	if s.Screen() != context.ScreenDiff || v.Cursor >= len(v.Rows) {
		return presentation.Note{}, presentation.DiffRow{}, false
	}
	r := v.Rows[v.Cursor]
	if r.Kind != "notehead" && r.Kind != "notetext" {
		return presentation.Note{}, r, false
	}
	n, ok := presentation.NoteOf(*s.Review, r.Path, r.Key)
	return n, r, ok
}

// Selected is the lines of one file the diff's cursor or selection is on,
// with their code, or false when they are in more than one file or none.
func (self *EditingHelper) Selected(s context.ContextTree) (models.Lines, []string, bool) {
	v := s.Diff
	if v.Cursor >= len(v.Rows) {
		return models.Lines{}, nil, false
	}
	from, to := v.Cursor, v.Cursor
	if v.Selecting {
		from, to = min(v.Anchor, v.Cursor), max(v.Anchor, v.Cursor)
	}
	var first, last models.Lines
	var firstBlock, seen *models.Block
	var code []string
	for i := from; i <= to; i++ {
		r := v.Rows[i]
		var start, end models.Lines
		switch side, line, ok := r.At(); {
		case r.Kind == "block":
			b := r.Block
			start = models.Lines{Path: r.Path, Side: b.Side, Line: b.StartLine}
			end = models.Lines{Path: r.Path, Side: b.Side, Line: b.Line}
			if b != seen {
				code = append(code, blockCode(b)...)
				seen = b
			}
			if firstBlock == nil {
				firstBlock = b
			}
		case ok:
			start = models.Lines{Path: r.Path, Side: side, Line: line}
			end = start
			code = append(code, codeLine(r))
		default:
			continue
		}
		if first.Path != "" && r.Path != first.Path {
			return models.Lines{}, nil, false
		}
		if first.Path == "" {
			first = start
		}
		last = end
	}
	if first.Path == "" {
		return models.Lines{}, nil, false
	}
	if first.Side != last.Side && last.Side == "LEFT" && firstBlock != nil && firstBlock.Left != nil {
		first.Side, first.Line = "LEFT", firstBlock.Left.StartLine
	}
	if first != last {
		last.StartLine, last.StartSide = first.Line, first.Side
	}
	return last, code, true
}

// LinesKey names lines of a file, as where the queue says they are
// commented is kept under.
func LinesKey(at models.Lines) string {
	return fmt.Sprintf("%s %s %d %s %d", at.Path, at.StartSide, at.StartLine, at.Side, at.Line)
}

// Comment is the state checking where a comment on the lines at the
// cursor would go, or on Claude's answer there, and the command that asks.
func (self *EditingHelper) Comment(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	tr := self.c.Tr
	if n, r, ok := self.NoteAt(s); ok {
		switch {
		case n.Kind == "conversation" && r.Part == "claude":
			text := n.Conversation.Messages[r.Message].Text
			about := utils.ResolvePlaceholderString(tr.CommentFromAnswer, map[string]string{"on": onWords(tr, *n.At)})
			e := types.Editing{Purpose: "comment", At: *n.At, Text: text, Key: "comment " + LinesKey(*n.At), About: []string{about}}
			return s.Said(tr.CheckingWhere, false), self.CheckWhere(s, e)
		case n.Kind == "queued":
			return s.Said(self.keyed(s, tr.EditsQueuedComment, "edit"), true), nil
		}
		return s.Said(self.keyed(s, tr.CommentsOnLinesOrAnswer, "comment"), true), nil
	}
	at, code, ok := self.Selected(s)
	if !ok {
		return s.Said(self.keyed(s, tr.CommentsOnLines, "comment"), true), nil
	}
	about := []string{utils.ResolvePlaceholderString(tr.CommentOn, map[string]string{"on": onWords(tr, at)})}
	e := types.Editing{Purpose: "comment", At: at, Key: "comment " + LinesKey(at), About: append(about, codeAbout(code)...)}
	s.Diff.Selecting = false
	return s.Said(tr.CheckingWhere, false), self.CheckWhere(s, e)
}

// Ask is the state with the editor open on a question to Claude about the
// lines at the cursor, or a follow-up to the conversation there.
func (self *EditingHelper) Ask(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	tr := self.c.Tr
	if n, _, ok := self.NoteAt(s); ok && n.Kind == "conversation" {
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
		return self.edit(s, types.Editing{Purpose: "reply", Conversation: c.ID, At: *n.At, Key: fmt.Sprintf("reply %d", c.ID), About: about})
	}
	at, code, ok := self.Selected(s)
	if !ok {
		return s.Said(self.keyed(s, tr.AsksAboutLines, "ask"), true), nil
	}
	s.Diff.Selecting = false
	about := append([]string{utils.ResolvePlaceholderString(tr.QuestionAbout, map[string]string{"on": onWords(tr, at)}), tr.ClaudeAnswersHere}, codeAbout(code)...)
	return self.edit(s, types.Editing{Purpose: "ask", At: at, Key: "ask " + LinesKey(at), About: about})
}

// Stop is the state asking Claude to stop answering the
// conversation at the cursor, or saying why it cannot, and the command that
// asks.
func (self *EditingHelper) Stop(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	tr := self.c.Tr
	c, ok := self.conversationAt(s)
	if !ok {
		return s.Said(self.keyed(s, tr.StopsClaude, "stop"), true), nil
	}
	if !presentation.Waiting(c) {
		return s.Said(tr.NotWorking, false), nil
	}
	client, id := self.c.Reviews, s.Review.ID
	e := types.Editing{Purpose: "stop", Conversation: c.ID}
	s.Stopped = c.ID
	return s.Said(tr.Stopping, false), func() tea.Msg {
		stopped, err := client.Stop(id, c.ID)
		return types.Changed{E: e, Conversation: &stopped, Err: err}
	}
}

func (self *EditingHelper) conversationAt(s context.ContextTree) (models.Conversation, bool) {
	id := 0
	if n, _, ok := self.NoteAt(s); ok && n.Kind == "conversation" {
		id = n.Conversation.ID
	}
	if s.Focus() == context.NotesPanel {
		if row, ok := s.Side(context.NotesPanel).Selected(); ok && row.Item.Kind == "conversation" {
			id = row.Item.Thread
		}
	}
	for _, c := range s.Review.State.Threads {
		if id != 0 && c.ID == id {
			return c, true
		}
	}
	return models.Conversation{}, false
}

// EditQueued is the state with the editor open on the queued comment at
// the cursor.
func (self *EditingHelper) EditQueued(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	tr := self.c.Tr
	c, pull, ok := self.queuedAt(s)
	if !ok {
		return s.Said(self.keyed(s, tr.EditsQueuedAtCursor, "edit"), true), nil
	}
	at := models.Lines{Path: c.Path, Side: c.Side, Line: c.Line, StartLine: c.StartLine, StartSide: c.StartSide}
	queued := utils.ResolvePlaceholderString(tr.QueuedCommentOn, map[string]string{"comment": utils.Itoa(c.ID), "on": onWords(tr, at), "pull": utils.Itoa(pull)})
	about := []string{queued, tr.SavingReplaces}
	return self.edit(s, types.Editing{Purpose: "edit", Comment: c.ID, Version: c.Version, At: at, Pull: pull, Text: c.Body, Key: fmt.Sprintf("edit %d", c.ID), About: about})
}

// DropQueued is the state asking whether to drop the queued comment at the
// cursor.
func (self *EditingHelper) DropQueued(s context.ContextTree) context.ContextTree {
	c, pull, ok := self.queuedAt(s)
	if !ok {
		return s.Said(self.keyed(s, self.c.Tr.DropsQueuedAtCursor, "drop"), true)
	}
	s = s.Opened(context.ConfirmationKey)
	s.Confirmation = context.ConfirmationContext{Comment: c.ID, Pull: pull, Version: c.Version}
	return s
}

// Dropped is the state dropping the comment the confirmation asked about,
// and the command that drops it.
func (self *EditingHelper) Dropped(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	c := s.Confirmation
	s = s.Closed(context.ConfirmationKey)
	client, id, build := self.c.Reviews, s.Review.ID, s.Review.Page.Build
	e := types.Editing{Purpose: "drop", Comment: c.Comment, Pull: c.Pull}
	return s.Said(self.c.Tr.Dropping, false), func() tea.Msg {
		state, err := client.Drop(id, c.Comment, c.Version, build)
		return types.Changed{E: e, State: &state, Err: err}
	}
}

// Kept is the state keeping the comment the confirmation asked about.
func (self *EditingHelper) Kept(s context.ContextTree) context.ContextTree {
	c := s.Confirmation
	s = s.Closed(context.ConfirmationKey)
	return s.Said(utils.ResolvePlaceholderString(self.c.Tr.KeptQueued, map[string]string{"comment": utils.Itoa(c.Comment)}), false)
}

// AfterEditing is the state once the editor closes: what was written
// queued, saved or asked, or kept as a draft when the editor failed.
func (self *EditingHelper) AfterEditing(s context.ContextTree, msg types.Edited) (context.ContextTree, tea.Cmd) {
	s.EditingNow = false
	tr := self.c.Tr
	text, readErr := written(msg.File)
	if err := os.Remove(msg.File); err != nil && readErr == nil {
		readErr = err
	}
	switch {
	case msg.Err != nil:
		if text != "" {
			s.Drafts = withDraft(s.Drafts, msg.E.Key, text)
		}
		return s.Said(tr.EditorStopped+": "+msg.Err.Error(), true), nil
	case readErr != nil:
		return s.Said(tr.CouldNotReadWritten+": "+readErr.Error(), true), nil
	case text == "":
		s.Drafts = withDraft(s.Drafts, msg.E.Key, "")
		return s.Said(tr.NothingWritten, false), nil
	case msg.E.Purpose == "edit" && text == msg.E.Text:
		s.Drafts = withDraft(s.Drafts, msg.E.Key, "")
		return s.Said(utils.ResolvePlaceholderString(tr.QueuedAsItWas, map[string]string{"comment": utils.Itoa(msg.E.Comment)}), false), nil
	}
	s.Drafts = withDraft(s.Drafts, msg.E.Key, text)
	client, id, build, e := self.c.Reviews, s.Review.ID, s.Review.Page.Build, msg.E
	switch e.Purpose {
	case "comment":
		return s.Said(tr.Queueing, false), func() tea.Msg {
			state, err := client.Queue(id, e.At, e.Pull, text, build)
			return types.Changed{E: e, State: &state, Err: err}
		}
	case "edit":
		return s.Said(tr.Saving, false), func() tea.Msg {
			state, err := client.Edit(id, e.Comment, text, e.Version, build)
			return types.Changed{E: e, State: &state, Err: err}
		}
	}
	s.Asking++
	if e.Purpose == "reply" {
		return s.Said("", false), func() tea.Msg {
			c, err := client.Reply(id, e.Conversation, text, build)
			return types.Changed{E: e, Conversation: &c, Err: err}
		}
	}
	return s.Said("", false), func() tea.Msg {
		c, err := client.Ask(id, e.At, text, build)
		return types.Changed{E: e, Conversation: &c, Err: err}
	}
}

// AfterChange is the state once a comment was queued, saved or dropped, or
// Claude answered, or why it did not.
func (self *EditingHelper) AfterChange(s context.ContextTree, msg types.Changed) (context.ContextTree, tea.Cmd) {
	e, h := msg.E, self.c.Helpers
	if e.Purpose == "ask" || e.Purpose == "reply" {
		s.Asking = max(0, s.Asking-1)
	}
	if msg.Err != nil {
		var refusal models.Refusal
		if (e.Purpose == "edit" || e.Purpose == "drop") && errors.As(msg.Err, &refusal) {
			return s.Said(msg.Err.Error(), true), h.Review.LoadState(s.Review.ID)
		}
		return s.Said(msg.Err.Error(), true), nil
	}
	s.Drafts = withDraft(s.Drafts, e.Key, "")
	if msg.Conversation != nil {
		c := *msg.Conversation
		s = h.Refresh.WithState(s, withConversation(s.Review.State, c))
		s.Diff.Notes = utils.CopyBools(s.Diff.Notes)
		s.Diff.Notes[fmt.Sprintf("conversation:%d", c.ID)] = true
		s = h.Refresh.Relaid(s)
		last := c.Messages[len(c.Messages)-1]
		if e.Purpose == "stop" || last.Error != "" && c.ID == s.Stopped {
			return s.Said(self.c.Tr.StoppedClaude, false), nil
		}
		if last.Error != "" {
			return s.Said(self.c.Tr.ClaudeSaysWhy, true), nil
		}
		return s.Said(self.c.Tr.ClaudeAnswered, false), nil
	}
	s = h.Refresh.WithState(s, *msg.State)
	comment := map[string]string{"comment": utils.Itoa(e.Comment)}
	switch e.Purpose {
	case "edit":
		return s.Said(utils.ResolvePlaceholderString(self.c.Tr.SavedQueued, comment), false), nil
	case "drop":
		return self.followed(s.Said(utils.ResolvePlaceholderString(self.c.Tr.DroppedQueued, comment), false))
	}
	return s.Said(utils.ResolvePlaceholderString(self.c.Tr.QueuedAbout, map[string]string{"about": strings.TrimSuffix(e.About[0], ".")}), false), nil
}

// AfterCheck is the state once the queue says where a comment would go:
// the editor open on it, a choice of pull requests, or why it cannot.
func (self *EditingHelper) AfterCheck(s context.ContextTree, msg types.Checked) (context.ContextTree, tea.Cmd) {
	switch {
	case msg.Err != nil:
		return s.Said(msg.Err.Error(), true), nil
	case msg.Where.Refused != "":
		return s.Said(msg.Where.Refused, true), nil
	case len(msg.Where.Choose) > 0:
		pending := msg.E
		pn := context.MenuContext{Menu: presentation.Menu{Title: msg.Where.Words}}
		for _, c := range msg.Where.Choose {
			line := utils.ResolvePlaceholderString(self.c.Tr.LineAt, map[string]string{"line": utils.Itoa(c.Line)})
			pn.Rows = append(pn.Rows, presentation.MenuRow{Left: fmt.Sprintf("#%d", c.Pull), Right: c.Path + ", " + line, Open: &presentation.Item{Kind: "choose", Pull: c.Pull}})
		}
		s = s.WithMenu(pn)
		s.Choosing = &pending
		return s.Said("", false), nil
	}
	e := msg.E
	e.About = append([]string{e.About[0], msg.Where.Words}, e.About[1:]...)
	return self.edit(s.Said("", false), e)
}

// CheckWhere is the command that asks the queue where a comment would go.
func (self *EditingHelper) CheckWhere(s context.ContextTree, e types.Editing) tea.Cmd {
	c, id, build := self.c.Reviews, s.Review.ID, s.Review.Page.Build
	return func() tea.Msg {
		where, err := c.Where(id, e.At, e.Pull, build)
		return types.Checked{E: e, Where: where, Err: err}
	}
}

// WhereNow is the command that asks the queue where a comment on the
// diff's selection would go, when the selection changed, or nothing.
func (self *EditingHelper) WhereNow(s context.ContextTree) tea.Cmd {
	if s.Screen() != context.ScreenDiff || !s.Diff.Selecting || s.Review == nil {
		return nil
	}
	at, _, ok := self.Selected(s)
	if !ok || LinesKey(at) == s.Where.Key {
		return nil
	}
	client, id, build, key := self.c.Reviews, s.Review.ID, s.Review.Page.Build, LinesKey(at)
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

func (self *EditingHelper) followed(s context.ContextTree) (context.ContextTree, tea.Cmd) {
	if s.Focus() != context.NotesPanel {
		return s, nil
	}
	return self.c.Helpers.Panels.Followed(s)
}

func (self *EditingHelper) keyed(s context.ContextTree, words, action string) string {
	return utils.ResolvePlaceholderString(words, map[string]string{action: self.c.KeyLabels(s.Place()).Of(action)})
}

func (self *EditingHelper) queuedAt(s context.ContextTree) (models.Comment, int, bool) {
	id := 0
	if n, _, ok := self.NoteAt(s); ok && n.Kind == "queued" {
		id = n.Comment.ID
	}
	switch {
	case s.Focus() == context.NotesPanel:
		if row, ok := s.Side(context.NotesPanel).Selected(); ok && row.Item.Kind == "comment" {
			id = row.Item.Comment
		}
	case s.Focus() == context.MainPanel && s.Screen() == context.ScreenReview && s.ReviewView.View.Cursor < len(s.ReviewView.View.Items):
		if it := s.ReviewView.View.Items[s.ReviewView.View.Cursor]; it.Kind == "comment" {
			id = it.Comment
		}
	}
	if id == 0 || s.Review.State.Queue == nil {
		return models.Comment{}, 0, false
	}
	for _, m := range s.Review.State.Queue.Members {
		for _, c := range m.Comments {
			if c.ID == id {
				return c, m.Pull, true
			}
		}
	}
	return models.Comment{}, 0, false
}

func (self *EditingHelper) edit(s context.ContextTree, e types.Editing) (context.ContextTree, tea.Cmd) {
	text := e.Text
	if draft, ok := s.Drafts[e.Key]; ok {
		text = draft
	}
	tr := self.c.Tr
	file, err := os.CreateTemp("", "lazyreview-*.md")
	if err != nil {
		return s.Said(tr.CouldNotMakeFile+": "+err.Error(), true), nil
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
		return s.Said(utils.ResolvePlaceholderString(tr.CouldNotWrite, map[string]string{"path": file.Name()})+": "+err.Error(), true), nil
	}
	cmd := exec.Command("sh", "-c", self.c.Editor+` "$@"`, "editor", file.Name())
	s.EditingNow = true
	return s, self.c.Exec(cmd, func(err error) tea.Msg { return types.Edited{E: e, File: file.Name(), Err: err} })
}

func blockCode(b *models.Block) []string {
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

func onWords(tr *i18n.TranslationSet, at models.Lines) string {
	return at.Path + ", " + presentation.LinesWords(tr, at.Side, at.Line, at.StartLine, at.StartSide)
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

func withConversation(st models.State, c models.Conversation) models.State {
	threads := append([]models.Conversation{}, st.Threads...)
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
