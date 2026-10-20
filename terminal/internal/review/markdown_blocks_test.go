package review

import (
	"fmt"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/models"
)

func textLines(text string) []string {
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// changed is a changed file whose whole text the page holds, its diff one
// hunk that removes every old line and adds every new one.
func changed(before, after string) []models.Block {
	b, a := textLines(before), textLines(after)
	var rows []models.Row
	for _, text := range b {
		rows = append(rows, models.Row{Kind: "del", Text: text})
	}
	for _, text := range a {
		rows = append(rows, models.Row{Kind: "add", Text: text})
	}
	hunk := models.Hunk{OldStart: 1, OldCount: len(b), NewStart: 1, NewCount: len(a), Rows: rows}
	return MarkdownBlocks(models.FileEntry{Status: "M"}, []models.Hunk{hunk}, a).Blocks
}

func states(blocks []models.Block) string {
	var out []string
	for _, b := range blocks {
		out = append(out, b.State+" "+b.Markdown)
	}
	return strings.Join(out, " | ")
}

func findBlock(t *testing.T, blocks []models.Block, text string) models.Block {
	t.Helper()
	for _, b := range blocks {
		if strings.Contains(b.Markdown, text) {
			return b
		}
	}
	t.Fatalf("no block holds %q", text)
	return models.Block{}
}

func at(b models.Block) string {
	left := "none"
	if b.Left != nil {
		left = fmt.Sprintf("%d-%d", b.Left.StartLine, b.Left.Line)
	}
	return fmt.Sprintf("%s %s %d-%d, before %s", b.State, b.Side, b.StartLine, b.Line, left)
}

func TestTheRenderedViewMarksEachAddedOrRemovedBlockAndAChangedListItemByItem(t *testing.T) {
	blocks := changed("# Guide\n\nOne.\n\n- a\n- b\n", "# Guide\n\nOne.\n\nTwo.\n\n- a\n- c\n")
	if got := states(blocks); got != "same # Guide | same One. | add Two. | same - a | del - b | add - c" {
		t.Errorf("the blocks are %s", got)
	}
	grown := changed("Intro.\n\n- a\n- b\n", "Intro, longer.\n\n- a\n- b\n- c\n\n1. one\n")
	if got := states(grown); got != "del Intro. | add Intro, longer. | same - a | same - b | add - c | add 1. one" {
		t.Errorf("a list is not paired with the one list of its kind beside a list of another kind: %s", got)
	}
}

func TestEachBlockAndListItemNamesTheSourceLinesItWasReadFrom(t *testing.T) {
	blocks := changed("# Guide\n\nOne\nline.\n\n- a\n- b\n\n```\ncode\n```\n", "# Guide\n\nOne\nline.\n\nTwo.\n\n- a\n- c\n\n```\ncode\n```\n")
	for text, want := range map[string]string{
		"Guide":     "same RIGHT 1-1, before 1-1",
		"One\nline": "same RIGHT 3-4, before 3-4",
		"Two":       "add RIGHT 6-6, before none",
		"- a":       "same RIGHT 8-8, before 6-6",
		"- b":       "del LEFT 7-7, before none",
		"code":      "same RIGHT 11-13, before 9-11",
	} {
		if got := at(findBlock(t, blocks, text)); got != want {
			t.Errorf("%q is at %s, not %s", text, got, want)
		}
	}
	defined := changed("[x]: https://example.com\n\nOld.\n", "[x]: https://example.com\n\nNew.\n")
	if got := at(findBlock(t, defined, "New")); got != "add RIGHT 3-3, before none" {
		t.Errorf("a link definition, which draws nothing, takes a block's lines: %s", got)
	}
}

func TestAnAddedFileIsReadWholeAnOrderedListKeepsEachItemsNumberAndAPartialFileShowsItsChangedParts(t *testing.T) {
	var rows []models.Row
	for _, text := range textLines("First.\n\n1. x\n1. y\n") {
		rows = append(rows, models.Row{Kind: "add", Text: text})
	}
	added := MarkdownBlocks(models.FileEntry{Status: "A"}, []models.Hunk{{Rows: rows}}, nil)
	var got []string
	for _, b := range added.Blocks {
		got = append(got, fmt.Sprintf("%s %s %d", b.State, b.Markdown, b.StartLine))
	}
	if strings.Join(got, " | ") != "add First. 1 | add 1. x 3 | add 2. y 4" {
		t.Errorf("an added file reads as %v", got)
	}
	parts := MarkdownBlocks(models.FileEntry{Status: "M"}, []models.Hunk{
		{OldStart: 40, NewStart: 42, Rows: []models.Row{{Kind: "del", Text: "Old."}, {Kind: "add", Text: "New."}}},
		{OldStart: 90, NewStart: 92, Rows: []models.Row{{Kind: "add", Text: "More."}}},
	}, nil)
	got = nil
	for _, b := range parts.Blocks {
		got = append(got, fmt.Sprintf("%s %s %d %d", b.State, b.Side, b.StartLine, b.Part))
	}
	if !parts.Partial || strings.Join(got, " | ") != "del LEFT 40 0 | add RIGHT 42 0 | add RIGHT 92 1" {
		t.Errorf("a file without its whole text reads as %v, partial %v", got, parts.Partial)
	}
}
