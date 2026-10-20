package diff

import (
	"reflect"
	"strings"
	"testing"
)

const mixedDiff = `diff --git a/src/Old.swift b/src/Old.swift
index 1111111..2222222 100644
--- a/src/Old.swift
+++ b/src/Old.swift
@@ -1,4 +1,5 @@
 struct A {
-    let a: Int
+    let a: Int64
+    let b: Int
 }

@@ -10,3 +11,2 @@ struct B {
 one
-two
 three
diff --git a/src/New.swift b/src/New.swift
new file mode 100644
index 0000000..3333333
--- /dev/null
+++ b/src/New.swift
@@ -0,0 +1,2 @@
+struct New {}
+
diff --git a/gone.txt b/gone.txt
deleted file mode 100644
index 4444444..0000000
--- a/gone.txt
+++ /dev/null
@@ -1 +0,0 @@
-bye
diff --git a/before name.md b/after name.md
similarity index 100%
rename from before name.md
rename to after name.md
diff --git a/logo.png b/logo.png
index 5555555..6666666 100644
Binary files a/logo.png and b/logo.png differ
`

type pathsAndStatus struct {
	path, oldPath, status string
	binary                bool
}

func summaries(chunks []Chunk) []pathsAndStatus {
	var out []pathsAndStatus
	for _, c := range chunks {
		out = append(out, pathsAndStatus{c.Path, c.OldPath, c.Status, c.Binary})
	}
	return out
}

func TestSplitDiffReadsEachFilesStatusAndPaths(t *testing.T) {
	chunks := SplitDiff(mixedDiff)
	want := []pathsAndStatus{
		{"src/Old.swift", "src/Old.swift", "M", false},
		{"src/New.swift", "", "A", false},
		{"gone.txt", "gone.txt", "D", false},
		{"after name.md", "before name.md", "R", false},
		{"logo.png", "logo.png", "M", true},
	}
	if got := summaries(chunks); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	if !strings.HasPrefix(chunks[0].Text, "diff --git a/src/Old.swift") || !strings.HasSuffix(chunks[0].Text, " three") {
		t.Errorf("the first chunk's text is %q", chunks[0].Text)
	}
}

func TestEachHunkKeepsItsAddedLinesAndRemovalPointsOnTheNewSide(t *testing.T) {
	old := SplitDiff(mixedDiff)[0]
	if len(old.Hunks) != 2 {
		t.Fatalf("%d hunks", len(old.Hunks))
	}
	first, second := old.Hunks[0], old.Hunks[1]
	if !reflect.DeepEqual(first.Added, []int{2, 3}) || !reflect.DeepEqual(first.RemovedAt, []int{2}) {
		t.Errorf("the first hunk adds %v and removes at %v", first.Added, first.RemovedAt)
	}
	if len(second.Added) != 0 || !reflect.DeepEqual(second.RemovedAt, []int{12}) {
		t.Errorf("the second hunk adds %v and removes at %v", second.Added, second.RemovedAt)
	}
	if first.OldStart != 1 || first.OldLength != 4 || first.NewStart != 1 || first.NewLength != 5 || second.NewLength != 2 {
		t.Errorf("the hunks span %+v and %+v", first, second)
	}
	if gone := SplitDiff(mixedDiff)[2].Hunks[0]; gone.OldLength != 1 || gone.NewStart != 0 || gone.NewLength != 0 {
		t.Errorf("a hunk header with no length counts one line, and 0,0 none: %+v", gone)
	}
}

func TestASpanIsTouchedByAnAddedLineOrByLinesRemovedInsideIt(t *testing.T) {
	hunks := SplitDiff(mixedDiff)[0].Hunks
	for _, c := range []struct {
		start, end int
		want       bool
	}{{1, 4, true}, {11, 13, true}, {12, 13, false}, {20, 30, false}} {
		if got := Touches(hunks, c.start, c.end); got != c.want {
			t.Errorf("lines %d to %d touched: %v", c.start, c.end, got)
		}
	}
}

func TestQuotedPathsReadBackWithTheirEscapes(t *testing.T) {
	cafe := SplitDiff("diff --git \"a/caf\\303\\251.txt\" \"b/caf\\303\\251.txt\"\nindex 1..2 100644\n--- \"a/caf\\303\\251.txt\"\n+++ \"b/caf\\303\\251.txt\"\n@@ -1 +1 @@\n-a\n+b\n")[0]
	if cafe.Path != "café.txt" || cafe.OldPath != "café.txt" {
		t.Errorf("an octal escaped path reads as %q from %q", cafe.Path, cafe.OldPath)
	}
	quoted := SplitDiff("diff --git \"a/say \\\"hi\\\"\\tnow.md\" \"b/say \\\"hi\\\"\\tnow.md\"\nnew file mode 100644\nindex 0000000..1111111\n--- /dev/null\n+++ \"b/say \\\"hi\\\"\\tnow.md\"\n@@ -0,0 +1 @@\n+hi\n")[0]
	if quoted.Path != "say \"hi\"\tnow.md" || quoted.Status != "A" || quoted.OldPath != "" {
		t.Errorf("a path with quotes and a tab reads as %q", quoted.Path)
	}
	spaced := SplitDiff("diff --git a/my notes/a b.md b/my notes/a b.md\nold mode 100644\nnew mode 100755\n")[0]
	if spaced.Path != "my notes/a b.md" || spaced.OldPath != "my notes/a b.md" || spaced.Status != "M" || len(spaced.Hunks) != 0 {
		t.Errorf("a mode change of a path with spaces reads as %+v", spaced)
	}
	tabbed := SplitDiff("diff --git a/a b.txt b/a b.txt\nindex 1..2 100644\n--- a/a b.txt\t\n+++ b/a b.txt\t\n@@ -1 +1 @@\n-a\n+b\n")[0]
	if tabbed.Path != "a b.txt" {
		t.Errorf("a path git ends with a tab reads as %q", tabbed.Path)
	}
}

func TestRenamesCopiesAndLinesWithNoNewLineAtTheEndRead(t *testing.T) {
	chunks := SplitDiff(`diff --git a/src/Name.swift b/src/People/Name.swift
similarity index 80%
rename from src/Name.swift
rename to src/People/Name.swift
index 1111111..2222222 100644
--- a/src/Name.swift
+++ b/src/People/Name.swift
@@ -1,3 +1,4 @@
 struct Name {
     let first: String
+    let middle: String
 }
\ No newline at end of file
diff --git a/a.txt b/b.txt
similarity index 100%
copy from a.txt
copy to b.txt
`)
	want := []pathsAndStatus{{"src/People/Name.swift", "src/Name.swift", "R", false}, {"b.txt", "", "A", false}}
	if got := summaries(chunks); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	if hunk := chunks[0].Hunks[0]; !reflect.DeepEqual(hunk.Added, []int{3}) || len(hunk.RemovedAt) != 0 {
		t.Errorf("the rename's hunk adds %v and removes at %v", hunk.Added, hunk.RemovedAt)
	}
	if binary := SplitDiff("diff --git a/x.bin b/x.bin\nnew file mode 100644\nindex 0000000..1111111\nGIT binary patch\nliteral 3\nKcmZQzWB>pF\n")[0]; !binary.Binary || binary.Status != "A" {
		t.Errorf("a binary patch reads as %+v", binary)
	}
}

func TestADiffWithNothingInItHasNoChunks(t *testing.T) {
	if chunks := SplitDiff(""); len(chunks) != 0 {
		t.Errorf("got %+v", chunks)
	}
	if text := SplitDiff(mixedDiff + "\n\n")[4].Text; strings.HasSuffix(text, "\n") {
		t.Errorf("the last chunk keeps the empty lines after it: %q", text)
	}
}
