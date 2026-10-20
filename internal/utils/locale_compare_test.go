package utils

import (
	"slices"
	"testing"
)

func TestLocaleCompareSortsAsJavaScriptDoes(t *testing.T) {
	given := []string{"a", "A", "b", "B", "ab", "Ab", "aB", "a-b", "a_b", "a.b", "a/b", "ab1", "ab10", "ab2", "e", "f", "E", "Z", "z", "_", "-", "a b"}
	want := []string{"_", "-", "a", "A", "a b", "a_b", "a-b", "a.b", "a/b", "ab", "aB", "Ab", "ab1", "ab10", "ab2", "b", "B", "e", "E", "f", "z", "Z"}
	slices.SortFunc(given, LocaleCompare)
	if !slices.Equal(given, want) {
		t.Fatalf("got %q", given)
	}
	if LocaleCompare("Sources/", "sources") != 1 || LocaleCompare("same", "same") != 0 {
		t.Fatal("a longer text sorts after its start, and equal text is equal")
	}
}
