package review

import "testing"

func TestALineMovesWithWhatTheDiffAddsAndRemovesAboveIt(t *testing.T) {
	diff := "diff --git a/f b/f\n--- a/f\n+++ b/f\n" +
		"@@ -1,3 +1,5 @@\n+// one\n+// two\n a\n b\n c\n" +
		"@@ -6,3 +8,2 @@\n f\n-g\n h\n" +
		"@@ -10,2 +11,2 @@\n j\n-k\n+K\n" +
		"@@ -14,2 +14,0 @@\n-n\n-o\n"
	hunks := ReadHunks(diff)
	for line, want := range map[int]int{
		1: 3, 2: 4, 3: 5,
		4: 6, 5: 7,
		6: 8, 7: 9, 8: 9,
		9: 10, 10: 11, 11: 12,
		12: 13, 13: 14,
		14: 14, 15: 14,
		16: 15, 30: 29,
	} {
		if got := MovedLine(hunks, line); got != want {
			t.Errorf("old line %d moved to %d, want %d", line, got, want)
		}
	}
	if got := MovedLine(nil, 7); got != 7 {
		t.Errorf("with no diff, line 7 moved to %d", got)
	}
}
