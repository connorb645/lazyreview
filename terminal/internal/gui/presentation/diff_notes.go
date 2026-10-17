package presentation

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// NotesOf are the notes on a file's lines: its queued comments, its
// conversations with Claude and its threads on GitHub.
func NotesOf(d types.ReviewData, path string) []Note {
	var out []Note
	if q := d.State.Queue; q != nil {
		for _, m := range q.Members {
			for i := range m.Comments {
				c := &m.Comments[i]
				at := server.Lines{Path: c.Path, Side: c.Side, Line: c.Line, StartLine: c.StartLine, StartSide: c.StartSide}
				if c.Shown != nil {
					at = *c.Shown
				}
				if at.Path == path {
					out = append(out, Note{Kind: "queued", Key: fmt.Sprintf("queued:%d", c.ID), At: &at, Pull: m.Pull, Comment: c})
				}
			}
		}
	}
	for i := range d.State.Threads {
		t := &d.State.Threads[i]
		if t.Path == path {
			at := server.Lines{Path: t.Path, Side: t.Side, Line: t.Line, StartLine: t.StartLine, StartSide: t.StartSide}
			out = append(out, Note{Kind: "conversation", Key: fmt.Sprintf("conversation:%d", t.ID), At: &at, Conversation: t})
		}
	}
	for i := range d.State.GitHub {
		t := &d.State.GitHub[i]
		if t.Path == path {
			at := t.At
			if t.File {
				at = nil
			}
			out = append(out, Note{Kind: "thread", Key: "thread:" + t.ID, At: at, Pull: t.Pull, Thread: t})
		}
	}
	return out
}

// NoteOf is the note on a file with a key, if it is still there.
func NoteOf(d types.ReviewData, path, key string) (Note, bool) {
	for _, n := range NotesOf(d, path) {
		if n.Key == key {
			return n, true
		}
	}
	return Note{}, false
}

func withNotes(tr *i18n.TranslationSet, loc *time.Location, d types.ReviewData, v DiffView, rows []DiffRow) []DiffRow {
	byPath := map[string][]Note{}
	for _, r := range rows {
		if r.Kind == "file" && !v.Folded[r.Path] {
			byPath[r.Path] = NotesOf(d, r.Path)
		}
	}
	placed := map[string]bool{}
	var out []DiffRow
	for _, r := range rows {
		out = append(out, r)
		if r.Kind == "file" || r.Kind == "hunk" {
			continue
		}
		for _, n := range byPath[r.Path] {
			if !placed[n.Key] && n.At != nil && rowHolds(r, *n.At, v) {
				placed[n.Key] = true
				out = append(out, noteRows(tr, loc, d, v, r.Path, n)...)
			}
		}
	}
	for path, notes := range byPath {
		var lost []DiffRow
		for _, n := range notes {
			if !placed[n.Key] {
				lost = append(lost, noteRows(tr, loc, d, v, path, n)...)
			}
		}
		if len(lost) == 0 {
			continue
		}
		for i, r := range out {
			if r.Kind == "file" && r.Path == path {
				out = append(out[:i+1], append(lost, out[i+1:]...)...)
				break
			}
		}
	}
	return out
}

func rowHolds(r DiffRow, at server.Lines, v DiffView) bool {
	if r.Kind == "gap" {
		file := v.Files[r.Path]
		for _, g := range file.Gaps {
			if fmt.Sprintf("%s:%d", r.Path, g.Before) != r.Key {
				continue
			}
			if at.Side == "LEFT" {
				return at.Line >= g.From+g.Delta && at.Line <= g.To+g.Delta
			}
			return at.Line >= g.From && at.Line <= g.To
		}
		return false
	}
	if r.Kind == "block" {
		b := r.Block
		if !r.Last {
			return false
		}
		if b.Left != nil && at.Side == "LEFT" {
			return at.Line >= b.Left.StartLine && at.Line <= b.Left.Line
		}
		return at.Side == b.Side && at.Line >= b.StartLine && at.Line <= b.Line
	}
	if r.Line != nil {
		side, line, ok := r.At()
		return ok && side == at.Side && line == at.Line
	}
	return r.Left != nil && at.Side == "LEFT" && r.Left.Old == at.Line && r.Left.Kind != "same" ||
		r.Right != nil && at.Side == "RIGHT" && r.Right.New == at.Line
}

// NoteOpen says whether a note shows its text: as it was last folded, or
// open for a queued comment and folded for anything else.
func (v DiffView) NoteOpen(n Note) bool {
	if open, ok := v.Notes[n.Key]; ok {
		return open
	}
	return n.Kind == "queued"
}

func noteRows(tr *i18n.TranslationSet, loc *time.Location, d types.ReviewData, v DiffView, path string, n Note) []DiffRow {
	rows := []DiffRow{{Kind: "notehead", Path: path, Key: n.Key}}
	if !v.NoteOpen(n) {
		return rows
	}
	width := max(20, v.Width-gutter-6)
	text := func(part, words string, message int) {
		for _, l := range utils.Wrap(words, width) {
			rows = append(rows, DiffRow{Kind: "notetext", Path: path, Key: n.Key, Text: l, Part: part, Message: message})
		}
	}
	switch n.Kind {
	case "queued":
		text("body", n.Comment.Body, -1)
		if n.Comment.Stale != "" {
			text("stale", utils.ResolvePlaceholderString(tr.MayHaveMoved, map[string]string{"reason": n.Comment.Stale}), -1)
		}
	case "conversation":
		for i, m := range n.Conversation.Messages {
			switch {
			case m.From == "you":
				text("you", utils.ResolvePlaceholderString(tr.YouSaid, map[string]string{"text": m.Text}), i)
			case m.Error != "":
				text("error", tr.ClaudeDidNotAnswer+": "+m.Error, i)
			default:
				text("claude", utils.ResolvePlaceholderString(tr.ClaudeSaid, map[string]string{"text": m.Text}), i)
			}
		}
		if last := n.Conversation.Messages; len(last) > 0 && last[len(last)-1].From == "you" {
			text("waiting", tr.WaitingForClaude, -1)
		}
	case "thread":
		for _, c := range n.Thread.Comments {
			text("author", c.Author+" · "+utils.When(c.At, loc), -1)
			text("body", c.Body, -1)
		}
		if n.Thread.More > 0 {
			text("author", utils.ResolvePlaceholderString(tr.MoreOnGitHub, map[string]string{"comments": utils.Plural(n.Thread.More, tr.MoreCommentNoun)}), -1)
		}
	}
	return rows
}

// NoteWords say what a note is and where, for the main panel's foot.
func NoteWords(tr *i18n.TranslationSet, n Note) string {
	lines := tr.TheWholeFile
	if n.At != nil {
		lines = LinesWords(tr, n.At.Side, n.At.Line, n.At.StartLine, n.At.StartSide)
	}
	switch n.Kind {
	case "queued":
		return utils.ResolvePlaceholderString(tr.QueuedComment, map[string]string{"comment": utils.Itoa(n.Comment.ID)}) + fmt.Sprintf(" → #%d · %s", n.Pull, lines)
	case "conversation":
		return tr.AConversation + " · " + lines
	}
	return utils.ResolvePlaceholderString(tr.AThreadOn, map[string]string{"pull": utils.Itoa(n.Pull)}) + " · " + lines
}

// NoteHead is a note's first row in a diff, and the keys that act on it,
// labelled from labels.
func NoteHead(tr *i18n.TranslationSet, labels KeyLabels, d types.ReviewData, n Note, open bool) (string, string) {
	fold := "▸"
	if open {
		fold = "▾"
	}
	keys := map[string]string{"edit": labels.Of("edit"), "drop": labels.Of("drop"), "ask": labels.Of("ask"), "fold": labels.Of("toggle fold"), "github": labels.Of("github")}
	lines := ""
	if n.At != nil {
		lines = " · " + LinesWords(tr, n.At.Side, n.At.Line, n.At.StartLine, n.At.StartSide)
	}
	switch n.Kind {
	case "queued":
		head := fold + " " + utils.ResolvePlaceholderString(tr.QueuedNumber, map[string]string{"comment": utils.Itoa(n.Comment.ID)}) + fmt.Sprintf(" → #%d%s", n.Pull, lines)
		if n.Comment.Stale != "" {
			head += " · " + tr.Stale
		}
		return head, utils.ResolvePlaceholderString(tr.QueuedNoteKeys, keys)
	case "conversation":
		head := fold + " " + tr.Claude + " · " + utils.Plural(len(n.Conversation.Messages), tr.MessageNoun) + lines
		if m := n.Conversation.Messages; len(m) > 0 && m[len(m)-1].From == "you" {
			head += " · " + tr.WaitingForClaudeWords
		}
		return head, utils.ResolvePlaceholderString(tr.ConversationNoteKeys, keys)
	}
	var people []string
	for _, c := range n.Thread.Comments {
		if !slices.Contains(people, c.Author) {
			people = append(people, c.Author)
		}
	}
	head := fmt.Sprintf("%s %s · %s", fold, strings.Join(people, ", "), utils.Plural(len(n.Thread.Comments)+n.Thread.More, tr.CommentNoun))
	if n.Thread.Resolved {
		head += " · " + tr.Resolved
	}
	if n.Thread.Outdated {
		head += " · " + tr.Outdated
	}
	if n.At == nil {
		head += " · " + tr.TheWholeFile
	}
	return head, utils.ResolvePlaceholderString(tr.ThreadNoteKeys, keys)
}
