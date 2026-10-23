package stack

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/build/diff"
	"github.com/connorb645/lazyreview/internal/i18n"
)

// A stack of branches is read as a stack of pull requests is, its members
// numbered by their place in it, and says what it warns of by branch.
func TestAStackOfBranchesNamesItsBranchesInWhatItSays(t *testing.T) {
	text, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "stacks.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []recordedStack
	if err := json.Unmarshal(text, &cases); err != nil {
		t.Fatal(err)
	}
	var c recordedStack
	for _, each := range cases {
		if strings.HasPrefix(each.Name, "out of step: the member below takes back a line") {
			c = each
		}
	}
	if c.Name == "" {
		t.Fatal("no recorded stack has a line no member's own diff adds")
	}
	tr := i18n.EnglishTranslationSet()
	repo := rebuilt(t, c)
	baseCommit, err := repo.Resolve(c.Base)
	if err != nil {
		t.Fatal(err)
	}
	combinedBase, err := repo.MergeBase(baseCommit, c.Top)
	if err != nil {
		t.Fatal(err)
	}
	members := []Member{{Pull: 1, Branch: "stack/below", Head: c.Members[0].Head}, {Pull: 2, Branch: "stack/above", Head: c.Members[1].Head}}
	read, err := ReadSteps(tr, repo, combinedBase, baseCommit, c.Top, members)
	if err != nil {
		t.Fatal(err)
	}
	want := "stack/above is not built on stack/below's head: stack/below has commits stack/above does not hold. Until stack/above is rebased, its own diff also shows the older stack/below it holds."
	if len(read.Warnings) != 1 || read.Warnings[0] != want {
		t.Errorf("the stack warns %v", read.Warnings)
	}
	if read.Steps[0].Branch != "stack/below" || read.Steps[1].Branch != "stack/above" || read.Steps[1].Pull != 2 {
		t.Errorf("the steps are %+v and %+v", read.Steps[0], read.Steps[1])
	}
	combined, err := repo.Diff(combinedBase, c.Top)
	if err != nil {
		t.Fatal(err)
	}
	mapped := MapLines(tr, read.Steps, diff.SplitDiff(combined), []int{1, 2})
	untraced := 0
	for _, sides := range mapped.Lines {
		for _, side := range sides {
			for _, entry := range side {
				if entry.None == "" {
					continue
				}
				untraced++
				if entry.None != tr.StackBranchNotAdded && entry.None != tr.StackBranchNotRemoved {
					t.Errorf("an untraced line says %q", entry.None)
				}
			}
		}
	}
	if untraced == 0 {
		t.Error("no line is untraced")
	}
}
