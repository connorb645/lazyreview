package presentation

import (
	"fmt"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/diagram"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// ItemWords say what an item of the Review view is, for the main panel's
// foot, a path among them cut from its start to fit a width.
func ItemWords(tr *i18n.TranslationSet, it Item, width int) string {
	switch it.Kind {
	case "section":
		return it.Title
	case "comment":
		return utils.ResolvePlaceholderString(tr.QueuedComment, map[string]string{"comment": utils.Itoa(it.Comment)})
	case "link", "thread", "conversation":
		if it.Place != nil {
			return utils.FitPathIn("", it.Place.Path, " "+LinesWords(tr, it.Place.Side, it.Place.Line, it.Place.StartLine, it.Place.StartSide), width)
		}
		return map[string]string{"link": tr.LinkNoun, "thread": tr.ThreadNoun, "conversation": tr.ConversationNoun}[it.Kind]
	case "component", "pull":
		return it.Title
	}
	return ""
}

// DiffRowWords say where a row of a diff is, for the main panel's foot: the
// note it belongs to, or its lines and what the server says of them.
func DiffRowWords(tr *i18n.TranslationSet, d types.ReviewData, r DiffRow) string {
	if n, ok := NoteOf(d, r.Path, r.Key); ok && (r.Kind == "notehead" || r.Kind == "notetext") {
		return NoteWords(tr, n)
	}
	if r.Kind == "block" {
		b, start := r.Block, 0
		if b.StartLine != b.Line {
			start = b.StartLine
		}
		return LinesWords(tr, b.Side, b.Line, start, "") + " · " + map[string]string{"add": tr.Added, "del": tr.Removed, "same": tr.Unchanged}[b.State]
	}
	side, line, ok := r.At()
	if !ok {
		return map[string]string{"file": tr.FileHeader, "hunk": tr.HunkHeader, "gap": tr.UnchangedLines, "note": ""}[r.Kind]
	}
	words := LinesWords(tr, side, line, 0, "")
	if w := r.Words(); w != "" {
		words += " · " + w
	}
	return words
}

// BoxWords say what a box of the diagram is, for the main panel's foot: its
// kind and details under the filter, or a type's file, state and pull
// requests.
func BoxWords(tr *i18n.TranslationSet, d types.ReviewData, l diagram.Level, card diagram.Card, filter types.PullFilter, width int) string {
	switch card.Kind {
	case "component":
		return card.Title + " · " + tr.ComponentNoun + " · " + CardDetail(tr, d, l, card, filter)
	case "package":
		return utils.FitPathIn("", card.Title, " · "+tr.PackageNoun+" · "+CardDetail(tr, d, l, card, filter), width)
	case "group":
		return card.Title + " · " + card.Detail
	}
	n, _ := NodeOf(d.Page, card.ID)
	kind := n.Kind
	if kind == "artifact" {
		kind = tr.FileNoun
	}
	tail := ""
	if state := diagram.StateWord(tr, n.State); state != "" {
		tail = " · " + state
		if len(n.Pulls) > 0 && d.Page.Stack != nil {
			pulls := make([]string, len(n.Pulls))
			for i, p := range n.Pulls {
				pulls[i] = fmt.Sprintf("#%d", p)
			}
			tail += " " + utils.ResolvePlaceholderString(tr.InPulls, map[string]string{"pulls": strings.Join(pulls, ", ")})
		}
	}
	return utils.FitPathIn(n.Name+" · "+kind+" · ", n.File, tail, width)
}

// DiagramTitle is the diagram's title for a level, and the package path that
// follows it on the types level.
func DiagramTitle(tr *i18n.TranslationSet, d types.ReviewData, l diagram.Level) (string, string) {
	out := tr.DiagramTitle
	if l.Component != "" {
		out += " · " + ComponentTitle(d, l.Component)
	}
	if l.Kind == "types" {
		return out + " › ", PackageTitle(d.Page, l.Component, l.Package)
	}
	return out, ""
}
