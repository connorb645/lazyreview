package review

import (
	"fmt"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/models"
)

const twoHunks = "diff --git a/a.ts b/a.ts\n--- a/a.ts\n+++ b/a.ts\n@@ -1,4 +1,4 @@ top\n one\n-two\n+TWO\n three\n\n@@ -9 +9,2 @@\n-x\n+y\n+z\n\\ No newline at end of file\n"

func TestAFilesDiffIsReadHunkByHunkRowByRowWithEachRowsNumbersOnEachSide(t *testing.T) {
	hunks := ReadHunks(twoHunks)
	if len(hunks) != 2 || hunks[0].Header != "@@ -1,4 +1,4 @@ top" || hunks[0].OldCount != 4 || hunks[0].NewCount != 4 || hunks[1].OldCount != 1 {
		t.Fatalf("the hunks are %+v", hunks)
	}
	var rows []string
	for _, r := range hunks[0].Rows {
		rows = append(rows, fmt.Sprintf("%s %d %d %q", r.Kind, r.Old, r.New, r.Text))
	}
	if strings.Join(rows, ", ") != `same 1 1 "one", del 2 0 "two", add 0 2 "TWO", same 3 3 "three", same 4 4 ""` {
		t.Errorf("an empty row inside a hunk is not an empty unchanged line: %v", rows)
	}
	var kinds []string
	for _, r := range hunks[1].Rows {
		kinds = append(kinds, r.Kind)
	}
	if strings.Join(kinds, " ") != "del add add note" {
		t.Errorf("the empty row after the last hunk is a line: %v", kinds)
	}
	if len(ReadHunks("")) != 0 {
		t.Errorf("no diff gives hunks")
	}
}

func TestACommentQuotesTheCodeOfItsLinesFromThePage(t *testing.T) {
	page := &models.PageRecord{
		Model:   models.Model{Files: []models.FileEntry{{Path: "a.ts", Status: "M"}, {Path: "new.ts", Status: "A"}}},
		Diff:    map[string]string{"a.ts": twoHunks, "new.ts": "@@ -0,0 +1,2 @@\n+first\n+second\n"},
		Sources: map[string][]string{"a.ts": {"one", "TWO", "three", "", "five", "six", "seven", "eight", "y", "z"}},
		Context: map[string][]string{"c.ts": {"class C {}", "}"}},
	}
	quote := func(path, side string, line, start int, startSide string) string {
		return QuotedCode(page, models.Lines{Path: path, Side: side, Line: line, StartLine: start, StartSide: startSide})
	}
	for _, c := range []struct{ got, want, why string }{
		{quote("a.ts", "RIGHT", 3, 1, ""), "one\nTWO\nthree", "the removed row between is not on the new side"},
		{quote("a.ts", "LEFT", 2, 0, ""), "two", "a removed line"},
		{quote("a.ts", "RIGHT", 2, 2, "LEFT"), "two\nTWO", "from a removed line to an added one, both"},
		{quote("a.ts", "RIGHT", 6, 5, ""), "five\nsix", "a line outside the hunks comes from the whole text"},
		{quote("a.ts", "LEFT", 6, 0, ""), "", "the page holds no whole text of the old side"},
		{quote("new.ts", "RIGHT", 2, 0, ""), "second", "an added file's line"},
		{quote("c.ts", "RIGHT", 1, 0, ""), "class C {}", "an unchanged file of the context quotes its own code"},
	} {
		if c.got != c.want {
			t.Errorf("%s: %q, not %q", c.why, c.got, c.want)
		}
	}
}
