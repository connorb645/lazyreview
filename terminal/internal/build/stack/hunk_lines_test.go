package stack

import (
	"reflect"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/models"
)

func diffText(lines ...string) string {
	return strings.Join(lines, "\n")
}

func TestHunkLinesReadBothSidesOfEachHunk(t *testing.T) {
	hunks := ReadHunkLines(diffText(
		"diff --git a/notes.md b/notes.md",
		"--- a/notes.md",
		"+++ b/notes.md",
		"@@ -2,4 +2,4 @@",
		" note 2",
		"-note 3",
		"+note three",
		"+note 3a",
		" note 4",
		"-note 5",
		`\ No newline at end of file`,
	))
	if len(hunks) != 1 {
		t.Fatalf("got %d hunks", len(hunks))
	}
	h := hunks[0]
	if !reflect.DeepEqual(h.Added, map[int]bool{3: true, 4: true}) || !reflect.DeepEqual(h.Removed, map[int]bool{3: true, 5: true}) {
		t.Errorf("added %v, removed %v", h.Added, h.Removed)
	}
	if !reflect.DeepEqual(h.RemovedAt, map[int]int{3: 3, 5: 6}) {
		t.Errorf("removed at %v", h.RemovedAt)
	}
	if !reflect.DeepEqual(h.NewToOld, map[int]int{2: 2, 5: 4}) || !reflect.DeepEqual(h.OldToNew, map[int]int{2: 2, 4: 5}) {
		t.Errorf("unchanged %v and %v", h.NewToOld, h.OldToNew)
	}
}

func TestALineStepsAcrossHunksBothWays(t *testing.T) {
	hunks := ReadHunkLines(diffText(
		"@@ -3,0 +4,2 @@",
		"+added 4",
		"+added 5",
		"@@ -10,2 +11,0 @@",
		"-removed 10",
		"-removed 11",
		"@@ -20,3 +20,3 @@",
		" kept 20",
		"-old 21",
		"+new 21",
		" kept 22",
	))
	cases := []struct {
		name string
		got  LineStep
		want LineStep
	}{
		{"a new line before every hunk keeps its number", TowardOld(hunks, 2), LineStep{Line: 2}},
		{"a new line the diff adds", TowardOld(hunks, 5), LineStep{Added: true}},
		{"a new line after an addition", TowardOld(hunks, 6), LineStep{Line: 4}},
		{"a new line just before lines were removed", TowardOld(hunks, 11), LineStep{Line: 9}},
		{"a new line where lines were removed", TowardOld(hunks, 12), LineStep{Line: 12}},
		{"a new line after a removal", TowardOld(hunks, 13), LineStep{Line: 13}},
		{"a new line a hunk shows unchanged", TowardOld(hunks, 22), LineStep{Shown: true, Line: 22}},
		{"a new line the last hunk changes", TowardOld(hunks, 21), LineStep{Added: true}},
		{"a new line after every hunk", TowardOld(hunks, 30), LineStep{Line: 30}},
		{"an old line before every hunk", TowardNew(hunks, 1), LineStep{Line: 1}},
		{"an old line next to an addition", TowardNew(hunks, 3), LineStep{Line: 3}},
		{"an old line after an addition", TowardNew(hunks, 4), LineStep{Line: 6}},
		{"an old line the diff removes", TowardNew(hunks, 11), LineStep{Removed: true}},
		{"an old line after a removal", TowardNew(hunks, 12), LineStep{Line: 12}},
		{"an old line a hunk shows unchanged", TowardNew(hunks, 20), LineStep{Shown: true, Line: 20}},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, c.got, c.want)
		}
	}
}

func TestOnlyALineAnotherPullRequestOwnsInsideTheSpanCounts(t *testing.T) {
	hunks := ReadHunkLines(diffText(
		"@@ -4,4 +4,4 @@",
		" kept 4",
		"+added 5",
		"-removed 5",
		" kept 6",
		"+added 7",
		"-removed 7",
	))
	owner := func(pull, line int) models.LineEntry {
		return models.LineEntry{Owner: &models.LinePlace{Pull: pull, Line: line}}
	}
	lines := map[string]models.LineMap{
		"RIGHT": {"5": owner(11, 5), "7": models.LineEntry{None: "taken by no pull request"}},
		"LEFT":  {"5": owner(12, 5), "7": owner(13, 7)},
	}
	if got := ChangedInSpan(hunks, lines, 5, 6); !reflect.DeepEqual(got, map[int]bool{11: true, 12: true}) {
		t.Errorf("from line 5 to 6: %v", got)
	}
	if got := ChangedInSpan(hunks, lines, 6, 7); len(got) != 0 {
		t.Errorf("from line 6 to 7: %v", got)
	}
	if got := ChangedInSpan(hunks, lines, 6, 8); !reflect.DeepEqual(got, map[int]bool{13: true}) {
		t.Errorf("from line 6 to 8: %v", got)
	}
	if got := ChangedInSpan(hunks, nil, 1, 10); len(got) != 0 {
		t.Errorf("a file with no entries: %v", got)
	}
}
