package app

import (
	"fmt"
	"strings"

	"review-desk/terminal/internal/server"
)

func notesOf(d reviewData, path string) []note {
	var out []note
	if q := d.state.Queue; q != nil {
		for _, m := range q.Members {
			for i := range m.Comments {
				c := &m.Comments[i]
				at := server.Lines{Path: c.Path, Side: c.Side, Line: c.Line, StartLine: c.StartLine, StartSide: c.StartSide}
				if c.Shown != nil {
					at = *c.Shown
				}
				if at.Path == path {
					out = append(out, note{kind: "queued", key: fmt.Sprintf("queued:%d", c.ID), at: &at, pull: m.Pull, comment: c})
				}
			}
		}
	}
	for i := range d.state.Threads {
		t := &d.state.Threads[i]
		if t.Path == path {
			at := server.Lines{Path: t.Path, Side: t.Side, Line: t.Line, StartLine: t.StartLine, StartSide: t.StartSide}
			out = append(out, note{kind: "conversation", key: fmt.Sprintf("conversation:%d", t.ID), at: &at, conversation: t})
		}
	}
	for i := range d.state.GitHub {
		t := &d.state.GitHub[i]
		if t.Path == path {
			at := t.At
			if t.File {
				at = nil
			}
			out = append(out, note{kind: "thread", key: "thread:" + t.ID, at: at, pull: t.Pull, thread: t})
		}
	}
	return out
}

func noteOf(d reviewData, path, key string) (note, bool) {
	for _, n := range notesOf(d, path) {
		if n.key == key {
			return n, true
		}
	}
	return note{}, false
}

// withNotes puts each note of the files shown under the row of its line, a
// note on an unchanged line under the run that hides it, and any other
// under its file's header.
func withNotes(d reviewData, v diffView, rows []diffRow) []diffRow {
	byPath := map[string][]note{}
	for _, r := range rows {
		if r.kind == "file" && !v.folded[r.path] {
			byPath[r.path] = notesOf(d, r.path)
		}
	}
	placed := map[string]bool{}
	var out []diffRow
	for _, r := range rows {
		out = append(out, r)
		if r.kind == "file" || r.kind == "hunk" {
			continue
		}
		for _, n := range byPath[r.path] {
			if !placed[n.key] && n.at != nil && rowHolds(r, *n.at, v) {
				placed[n.key] = true
				out = append(out, noteRows(d, v, r.path, n)...)
			}
		}
	}
	for path, notes := range byPath {
		var lost []diffRow
		for _, n := range notes {
			if !placed[n.key] {
				lost = append(lost, noteRows(d, v, path, n)...)
			}
		}
		if len(lost) == 0 {
			continue
		}
		for i, r := range out {
			if r.kind == "file" && r.path == path {
				out = append(out[:i+1], append(lost, out[i+1:]...)...)
				break
			}
		}
	}
	return out
}

func rowHolds(r diffRow, at server.Lines, v diffView) bool {
	if r.kind == "gap" {
		file := v.files[r.path]
		for _, g := range file.Gaps {
			if fmt.Sprintf("%s:%d", r.path, g.Before) != r.key {
				continue
			}
			if at.Side == "LEFT" {
				return at.Line >= g.From+g.Delta && at.Line <= g.To+g.Delta
			}
			return at.Line >= g.From && at.Line <= g.To
		}
		return false
	}
	if r.kind == "block" {
		b := r.block
		if !r.last {
			return false
		}
		if b.Left != nil && at.Side == "LEFT" {
			return at.Line >= b.Left.StartLine && at.Line <= b.Left.Line
		}
		return at.Side == b.Side && at.Line >= b.StartLine && at.Line <= b.Line
	}
	if r.line != nil {
		side, line, ok := r.at()
		return ok && side == at.Side && line == at.Line
	}
	return r.left != nil && at.Side == "LEFT" && r.left.Old == at.Line && r.left.Kind != "same" ||
		r.right != nil && at.Side == "RIGHT" && r.right.New == at.Line
}

func (v diffView) noteOpen(n note) bool {
	if open, ok := v.notes[n.key]; ok {
		return open
	}
	return n.kind == "queued"
}

func noteRows(d reviewData, v diffView, path string, n note) []diffRow {
	rows := []diffRow{{kind: "notehead", path: path, key: n.key}}
	if !v.noteOpen(n) {
		return rows
	}
	width := max(20, v.width-gutter-6)
	text := func(part, words string, message int) {
		for _, l := range wrap(words, width) {
			rows = append(rows, diffRow{kind: "notetext", path: path, key: n.key, text: l, part: part, message: message})
		}
	}
	switch n.kind {
	case "queued":
		text("body", n.comment.Body, -1)
		if n.comment.Stale != "" {
			text("stale", "Its line may have moved: "+n.comment.Stale+". Nothing sends it until it is queued again or dropped.", -1)
		}
	case "conversation":
		for i, m := range n.conversation.Messages {
			switch {
			case m.From == "you":
				text("you", "You: "+m.Text, i)
			case m.Error != "":
				text("error", "Claude did not answer: "+m.Error, i)
			default:
				text("claude", "Claude: "+m.Text, i)
			}
		}
		if last := n.conversation.Messages; len(last) > 0 && last[len(last)-1].From == "you" {
			text("waiting", "Waiting for Claude…", -1)
		}
	case "thread":
		for _, c := range n.thread.Comments {
			text("author", c.Author+" · "+when(c.At, v.loc), -1)
			text("body", c.Body, -1)
		}
		if n.thread.More > 0 {
			text("author", plural(n.thread.More, "more comment")+" on GitHub", -1)
		}
	}
	return rows
}

func noteWords(n note) string {
	lines := "the whole file"
	if n.at != nil {
		lines = linesWords(n.at.Side, n.at.Line, n.at.StartLine, n.at.StartSide)
	}
	switch n.kind {
	case "queued":
		return fmt.Sprintf("queued comment %d → #%d · %s", n.comment.ID, n.pull, lines)
	case "conversation":
		return "a conversation with Claude · " + lines
	}
	return fmt.Sprintf("a thread on #%d · %s", n.pull, lines)
}

func noteHead(d reviewData, n note, open bool) (string, string) {
	fold := "▸"
	if open {
		fold = "▾"
	}
	lines := ""
	if n.at != nil {
		lines = " · " + linesWords(n.at.Side, n.at.Line, n.at.StartLine, n.at.StartSide)
	}
	switch n.kind {
	case "queued":
		head := fmt.Sprintf("%s Queued %d → #%d%s", fold, n.comment.ID, n.pull, lines)
		if n.comment.Stale != "" {
			head += " · stale"
		}
		return head, "e edit  dd drop"
	case "conversation":
		head := fmt.Sprintf("%s Claude · %s%s", fold, plural(len(n.conversation.Messages), "message"), lines)
		if m := n.conversation.Messages; len(m) > 0 && m[len(m)-1].From == "you" {
			head += " · waiting for Claude"
		}
		return head, "a ask  za"
	}
	var people []string
	for _, c := range n.thread.Comments {
		if !contains(people, c.Author) {
			people = append(people, c.Author)
		}
	}
	head := fmt.Sprintf("%s %s · %s", fold, strings.Join(people, ", "), plural(len(n.thread.Comments)+n.thread.More, "comment"))
	if n.thread.Resolved {
		head += " · resolved"
	}
	if n.thread.Outdated {
		head += " · outdated"
	}
	if n.at == nil {
		head += " · the whole file"
	}
	return head, "gx GitHub  za"
}
