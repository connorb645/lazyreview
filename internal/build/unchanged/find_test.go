package unchanged

import (
	"reflect"
	"sort"
	"testing"

	"github.com/connorb645/lazyreview/internal/build/diff"
	"github.com/connorb645/lazyreview/internal/build/model"
	"github.com/connorb645/lazyreview/internal/commands/git"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/testsupport"
)

func readerChange(t *testing.T) (*git.Repository, diff.Read, model.ChangedTypes) {
	t.Helper()
	tr := i18n.EnglishTranslationSet()
	r := testsupport.NewThrowawayRepo(t)
	r.Commit("main", map[string]string{
		"Sources/Core/Store.swift":      "protocol Store {\n    func load() -> Data\n}\n",
		"Sources/Core/Disk.swift":       "final class Disk: Store {\n    func load() -> Data { Data() }\n}\n",
		"Sources/Core/Disk+Cache.swift": "extension Disk {\n    func cached() -> Data { load() }\n}\n",
		"Sources/Core/Ledger.swift":     "struct Ledger {\n    let rows: [String]\n}\n",
		"Sources/Core/Unrelated.swift":  "struct Unrelated {\n    let n: Int\n}\n",
		"Sources/App/Reader.swift":      "struct Reader {\n    let store: Store\n}\n",
		"Sources/App/Viewer.swift":      "struct Viewer {\n    let reader: Reader\n}\n",
		"Tests/DiskTests.swift":         "extension Disk {\n    func check() {}\n}\n",
	}, "base")
	r.Commit("feature", map[string]string{"Sources/App/Reader.swift": "struct Reader {\n    let store: Store\n    let disk: Disk\n}\n"}, "reader")
	repo := git.NewRepository(tr, r.Work, nil)
	read, err := diff.ReadDiff(tr, repo, "main", "feature", false)
	if err != nil {
		t.Fatal(err)
	}
	return repo, read, model.ReadChangedTypes(tr, read.Files)
}

func described(nodes []*model.Node) [][4]any {
	var out [][4]any
	for _, n := range nodes {
		out = append(out, [4]any{n.Name, n.Kind, n.File, n.State})
	}
	return out
}

func readPaths(texts map[string]*string) []string {
	var out []string
	for path := range texts {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func TestCandidatesAreTheTypesTheChangeNamesTheTypesThatNameItAndThoseIncluded(t *testing.T) {
	repo, read, changed := readerChange(t)
	found, err := Candidates(repo, read.HeadCommit, changed.Nodes, []string{"Sources/App/Reader.swift"}, []string{"Ledger"})
	if err != nil {
		t.Fatal(err)
	}
	want := [][4]any{
		{"Disk", "class", "Sources/Core/Disk.swift", "context"},
		{"Ledger", "struct", "Sources/Core/Ledger.swift", "context"},
		{"Store", "protocol", "Sources/Core/Store.swift", "context"},
		{"Viewer", "struct", "Sources/App/Viewer.swift", "context"},
	}
	if got := described(found.Nodes); !reflect.DeepEqual(got, want) {
		t.Errorf("found %v", got)
	}
	if got, want := readPaths(found.Texts), []string{"Sources/App/Viewer.swift", "Sources/Core/Disk.swift", "Sources/Core/Ledger.swift", "Sources/Core/Store.swift"}; !reflect.DeepEqual(got, want) {
		t.Errorf("read %v, and no changed or unrelated file", got)
	}
}

func TestCandidatesOfNothingDrawnReadNothing(t *testing.T) {
	repo, read, _ := readerChange(t)
	found, err := Candidates(repo, read.HeadCommit, nil, nil, nil)
	if err != nil || len(found.Nodes) != 0 || len(found.Texts) != 0 {
		t.Errorf("found %v, %v", described(found.Nodes), err)
	}
}

func TestExtensionsOfContextTypesAreFoundInFilesNotDrawnAndNotTests(t *testing.T) {
	repo, read, changed := readerChange(t)
	tr := i18n.EnglishTranslationSet()
	found, err := Candidates(repo, read.HeadCommit, changed.Nodes, []string{"Sources/App/Reader.swift"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	var disk []*model.Node
	for _, n := range found.Nodes {
		if n.Name == "Disk" {
			disk = append(disk, n)
		}
	}
	extended, err := Extensions(tr, repo, read.HeadCommit, disk, map[string]bool{"Sources/App/Reader.swift": true, "Sources/Core/Disk.swift": true})
	if err != nil {
		t.Fatal(err)
	}
	if got := described(extended.Nodes); !reflect.DeepEqual(got, [][4]any{{"Disk+Cache", "extension", "Sources/Core/Disk+Cache.swift", "context"}}) {
		t.Fatalf("found %v", got)
	}
	if n := extended.Nodes[0]; !reflect.DeepEqual(n.Extends, []string{"Disk"}) || len(n.Inherits) != 0 || n.Line != 1 {
		t.Errorf("the box extends %v, inherits %v, from line %d", n.Extends, n.Inherits, n.Line)
	}
	if got := readPaths(extended.Texts); !reflect.DeepEqual(got, []string{"Sources/Core/Disk+Cache.swift"}) {
		t.Errorf("kept the text of %v", got)
	}
	skipped, err := Extensions(tr, repo, read.HeadCommit, disk, map[string]bool{"Sources/Core/Disk+Cache.swift": true})
	if err != nil || len(skipped.Nodes) != 0 {
		t.Errorf("a file drawn already is left out: %v, %v", described(skipped.Nodes), err)
	}
}
