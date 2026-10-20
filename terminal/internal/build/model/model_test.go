package model

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/build/components"
	"github.com/connorb645/lazyreview/terminal/internal/build/diff"
	"github.com/connorb645/lazyreview/terminal/internal/build/languages"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
)

var diffBody = regexp.MustCompile(`^[@+\- ]|^\\|^$`)

func changed(t *testing.T, path string, baseText, text *string) diff.File {
	t.Helper()
	folder := t.TempDir()
	a, b := filepath.Join(folder, "a"), filepath.Join(folder, "b")
	for file, given := range map[string]*string{a: baseText, b: text} {
		content := ""
		if given != nil {
			content = *given
		}
		if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, _ := exec.Command("git", "diff", "--no-index", "--no-color", "-U3", a, b).Output()
	status, header := "M", "diff --git a/"+path+" b/"+path+"\n"
	switch {
	case baseText == nil:
		status, header = "A", header+"new file mode 100644\n"
	case text == nil:
		status, header = "D", header+"deleted file mode 100644\n"
	}
	var body []string
	for _, line := range strings.Split(string(out), "\n") {
		if diffBody.MatchString(line) && !strings.HasPrefix(line, "--- ") && !strings.HasPrefix(line, "+++ ") {
			body = append(body, line)
		}
	}
	chunk := diff.SplitDiff(header + strings.Join(body, "\n"))[0]
	if chunk.Status != status {
		t.Fatalf("%s reads as %s", path, chunk.Status)
	}
	return diff.File{Path: chunk.Path, OldPath: chunk.OldPath, Status: chunk.Status, Hunks: chunk.Hunks, Text: text, BaseText: baseText}
}

func text(s string) *string {
	return &s
}

const cacheBase = `protocol Store {}

final class Cache: Store {
    let size: Int
}

final class Untouched {
    let cache: Cache
}
`

const cacheHead = `protocol Store {}

final class Cache: Store {
    let size: Int
    let clock: Clock
}

final class Untouched {
    let cache: Cache
}

struct Fresh {
    let cache: Cache
}
`

func kindsAndStates(nodes []*Node) [][3]string {
	var out [][3]string
	for _, n := range nodes {
		out = append(out, [3]string{n.Name, n.Kind, n.State})
	}
	return out
}

func TestChangedTypesAreNewChangedOrUntouchedAndCodeWithNoTypesIsAFileBox(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	result := ReadChangedTypes(tr, []diff.File{
		changed(t, "Sources/Cache.swift", text(cacheBase), text(cacheHead)),
		changed(t, "Sources/Clock.swift", nil, text("protocol Clock {}")),
		changed(t, "README.md", text("old"), text("new")),
	})
	want := [][3]string{{"Cache", "class", "changed"}, {"Fresh", "struct", "new"}, {"Clock", "protocol", "new"}, {"README.md", "artifact", "changed"}}
	if got := kindsAndStates(result.Nodes); !reflect.DeepEqual(got, want) {
		t.Errorf("drawn %v", got)
	}
	if got := kindsAndStates(result.Untouched); !reflect.DeepEqual(got, [][3]string{{"Store", "protocol", "same"}, {"Untouched", "class", "same"}}) {
		t.Errorf("untouched %v", got)
	}
}

func TestAnExtensionJoinsTheChangedTypeItExtendsOrStandsForItsOwnFile(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	result := ReadChangedTypes(tr, []diff.File{
		changed(t, "Sources/Cache.swift", nil, text("final class Cache {}")),
		changed(t, "Sources/Cache+Store.swift", nil, text("extension Cache: Store {\n    func keep() {}\n}")),
		changed(t, "Sources/String+Trim.swift", nil, text("extension String {\n    var trimmed: String { self }\n}")),
	})
	cache := result.Nodes[0]
	if !reflect.DeepEqual(cache.Inherits, []string{"Store"}) || !reflect.DeepEqual(cache.ExtFiles, []string{"Sources/Cache+Store.swift"}) {
		t.Errorf("the extended type inherits %v from %v", cache.Inherits, cache.ExtFiles)
	}
	var names [][2]string
	for _, n := range result.Nodes {
		names = append(names, [2]string{n.Name, n.Kind})
	}
	if !reflect.DeepEqual(names, [][2]string{{"Cache", "class"}, {"String+Trim", "extension"}}) {
		t.Errorf("drawn %v", names)
	}
	if trim := result.Nodes[1]; trim.Line != 1 || !reflect.DeepEqual(trim.Extends, []string{"String"}) {
		t.Errorf("the box of extensions takes its first extension's line and what it extends: %+v", trim)
	}
}

func TestTestCodeIsKnownByItsPath(t *testing.T) {
	for path, want := range map[string]bool{
		"Tests/CoreTests/CacheTests.swift":   true,
		"app/src/test/java/x/CacheTest.java": true,
		"pkg/test_cache.py":                  true,
		"web/cache.spec.ts":                  true,
		"Sources/Core/Cache.swift":           false,
	} {
		if got := IsTestPath(path); got != want {
			t.Errorf("%s holds tests: %v", path, got)
		}
	}
}

func TestABoxOfExtensionsIsNamedForItsFileOrWhatItExtends(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for _, c := range []struct {
		path     string
		extended []string
		want     string
	}{
		{"Sources/String+Trim.swift", []string{"String"}, "String+Trim"},
		{"Sources/Cache.swift", []string{"Cache"}, "Cache extension"},
		{"Sources/Helpers.swift", []string{"Cache", "Store"}, "Cache · Helpers"},
	} {
		if got := ExtensionName(tr, c.path, c.extended); got != c.want {
			t.Errorf("%s extending %v is named %q", c.path, c.extended, got)
		}
	}
}

func node(id string, set func(*Node)) *Node {
	n := &Node{ID: id, Name: id, Kind: "class", File: "Sources/" + id + ".swift", Family: "swift", Inherits: []string{}, Properties: []models.Property{}, Extends: []string{}}
	if set != nil {
		set(n)
	}
	return n
}

func TestRelationsReadConformanceInheritanceHoldingUseTestsAndTheNearestTypeOfAName(t *testing.T) {
	nodes := []*Node{
		node("Store", func(n *Node) { n.Kind = "protocol" }),
		node("Base", nil),
		node("Cache", func(n *Node) {
			n.Inherits = []string{"Store", "Base"}
			n.Properties = []models.Property{{Name: "entries", Type: "[Entry]", Line: 2}, {Name: "spare", Type: "Entry?", Line: 3}}
			n.Code = "class Cache { func f() { Logger.log() } }"
		}),
		node("Entry", func(n *Node) { n.Kind = "struct" }),
		node("Logger", nil),
		node("Logger@Other", func(n *Node) { n.Name, n.File = "Logger", "Other/Logger.swift" }),
		node("CacheTests", func(n *Node) { n.Test, n.Code = true, "Cache()" }),
		node("Main", func(n *Node) { n.Family, n.Code = "jvm", "Cache()" }),
	}
	edges := FindRelations(i18n.EnglishTranslationSet(), nodes, nil)
	kinds := map[string]string{}
	for _, e := range edges {
		kinds[e.Source+">"+e.Target] = e.Kind
	}
	want := map[string]string{"Cache>Store": "realize", "Cache>Base": "inherit", "Cache>Entry": "has", "Cache>Logger": "use", "CacheTests>Cache": "test"}
	if !reflect.DeepEqual(kinds, want) {
		t.Errorf("got %v", kinds)
	}
	for _, e := range edges {
		if e.Kind == "has" && !reflect.DeepEqual(e.Properties, []string{"entries", "spare"}) {
			t.Errorf("a has link names the properties that hold its target: %v", e.Properties)
		}
	}
}

func TestImportsLinkScriptAndPythonFilesThatHaveNoTypes(t *testing.T) {
	nodes := []*Node{
		node("file:web/app.ts", func(n *Node) { n.Name, n.Kind, n.File, n.Family = "app.ts", "artifact", "web/app.ts", "js" }),
		node("file:web/util/format.ts", func(n *Node) { n.Name, n.Kind, n.File, n.Family = "format.ts", "artifact", "web/util/format.ts", "js" }),
		node("file:pkg/run.py", func(n *Node) { n.Name, n.Kind, n.File, n.Family = "run.py", "artifact", "pkg/run.py", "python" }),
		node("Parser", func(n *Node) { n.File, n.Family = "pkg/parse/core.py", "python" }),
	}
	imports := []FileImports{
		{Path: "web/app.ts", Imports: []languages.Import{{Spec: "./util/format", Names: []string{"format"}}}},
		{Path: "pkg/run.py", Imports: []languages.Import{{Spec: ".parse.core", Names: []string{"Parser"}, Python: true}}},
	}
	var got []string
	for _, e := range FindRelations(i18n.EnglishTranslationSet(), nodes, imports) {
		got = append(got, e.Source+">"+e.Target)
	}
	sort.Strings(got)
	if want := []string{"file:pkg/run.py>Parser", "file:web/app.ts>file:web/util/format.ts"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}

func names(nodes ...[]*Node) []string {
	var out []string
	for _, list := range nodes {
		for _, n := range list {
			out = append(out, n.Name)
		}
	}
	return out
}

func TestContextWithoutTheOptionCountsOnlyStructuralLinksToUntouchedTypes(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	drawn := []*Node{node("Cache", func(n *Node) { n.State, n.Inherits, n.Code = "changed", []string{"Store"}, "Helper()" })}
	untouched := []*Node{node("Store", func(n *Node) { n.Kind, n.State = "protocol", "same" }), node("Helper", func(n *Node) { n.State = "same" })}
	candidates := []*Node{node("Monitor", func(n *Node) { n.State, n.Code = "context", "let c = Cache()" })}
	same, context := ChooseContext(tr, drawn, untouched, candidates, ContextOptions{Max: 40})
	if !reflect.DeepEqual(names(same), []string{"Store"}) || len(context) != 0 {
		t.Errorf("without the option: %v and %v", names(same), names(context))
	}
	same, context = ChooseContext(tr, drawn, untouched, candidates, ContextOptions{Enabled: true, Max: 40})
	kept := names(same)
	sort.Strings(kept)
	if !reflect.DeepEqual(kept, []string{"Helper", "Store"}) || !reflect.DeepEqual(names(context), []string{"Monitor"}) {
		t.Errorf("with it: %v and %v", kept, names(context))
	}
	same, context = ChooseContext(tr, drawn, untouched, candidates, ContextOptions{Enabled: true, Max: 1, Include: []string{"Monitor"}, Exclude: []string{"Helper"}})
	if got := names(same, context); !reflect.DeepEqual(got, []string{"Monitor"}) {
		t.Errorf("capped: %v", got)
	}
	if !slices.Contains(names(untouched), "Store") || drawn[0].ID != "Cache" {
		t.Error("choosing changes none of the nodes given")
	}
}

func TestTheModelHasIDsPlacementNamedLinksGroupsAndArrowsBetweenComponents(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	nodes := []*Node{node("Payload", func(n *Node) { n.Kind, n.State, n.File = "protocol", "new", "Sources/Format/Payload.swift" })}
	for i := range 6 {
		nodes = append(nodes, node("Payload"+string(rune('0'+i)), func(n *Node) {
			n.Kind, n.State, n.Inherits, n.File = "struct", "new", []string{"Payload"}, "Sources/Format/Payloads.swift"
		}))
	}
	nodes = append(nodes,
		node("Reader", func(n *Node) { n.State, n.File, n.Code = "new", "Sources/Format/Reader.swift", "Payload0 Payload1" }),
		node("Screen", func(n *Node) { n.State, n.File, n.Code = "new", "App/Screen.swift", "Reader() Reader" }),
		node("Panel", func(n *Node) { n.State, n.File, n.Code = "new", "App/Panel.swift", "Reader()" }),
		node("Monitor", func(n *Node) { n.State, n.File, n.Code = "context", "Sources/Monitor.swift", "Reader()" }),
	)
	var files []diff.File
	for _, path := range []string{"Sources/Format/Payload.swift", "Sources/Format/Payloads.swift", "Sources/Format/Reader.swift", "App/Screen.swift", "App/Panel.swift"} {
		files = append(files, diff.File{Path: path, Status: "A", Adds: 3})
	}
	config, err := components.Read(tr, []byte(`{
		"components": [{"id": "sdk", "title": "SDK", "paths": ["Sources/"]}, {"id": "app", "title": "App", "color": "rose", "paths": ["App/"]}],
		"links": [{"from": "Screen", "to": "Reader", "label": "reads with"}],
		"summaries": [{"from": "app", "to": "sdk", "label": "shows what the SDK reads"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	model := BuildModel(tr, files, nodes, nil, config, tr.DiagramRepositoryRoot)
	var colours [][2]string
	for _, c := range model.Components {
		colours = append(colours, [2]string{c.ID, c.Color})
	}
	if !reflect.DeepEqual(colours, [][2]string{{"sdk", "blue"}, {"app", "rose"}, {"context", "grey"}}) {
		t.Errorf("components %v", colours)
	}
	if len(model.Groups) != 1 || model.Groups[0].Title != "6 types conforming to Payload" {
		t.Fatalf("groups %+v", model.Groups)
	}
	group := "group:" + model.Groups[0].ID
	if !slices.ContainsFunc(model.Edges, func(e models.Edge) bool {
		return e.Source == group && e.Target == "Payload" && e.Label == "each conforms to"
	}) {
		t.Error("the group has one arrow for the lot")
	}
	if slices.ContainsFunc(model.Edges, func(e models.Edge) bool { return e.Source == "Payload0" && e.Target == "Payload" }) {
		t.Error("a grouped type keeps its own arrow")
	}
	i := slices.IndexFunc(model.Edges, func(e models.Edge) bool { return e.Source == "Screen" && e.Target == "Reader" })
	if i < 0 || model.Edges[i].Label != "reads with" || !model.Edges[i].Key {
		t.Errorf("the named link is %+v", model.Edges)
	}
	var summary [][4]any
	componentOf := map[string]string{}
	for _, n := range model.Nodes {
		componentOf[n.ID] = n.Component
	}
	for _, s := range model.Summary {
		summary = append(summary, [4]any{s.Source, s.Target, s.Count, s.Label})
		if len(s.Links) != s.Count {
			t.Errorf("a summary names each link it counts: %+v", s)
		}
		for _, l := range s.Links {
			if componentOf[l[0]] != s.Source || componentOf[l[1]] != s.Target {
				t.Errorf("%v is not from %s to %s", l, s.Source, s.Target)
			}
		}
	}
	if want := [][4]any{{"app", "sdk", 2, "shows what the SDK reads"}, {"context", "sdk", 1, ""}}; !reflect.DeepEqual(summary, want) {
		t.Errorf("summary %v", summary)
	}
	if i := slices.IndexFunc(model.Nodes, func(n models.Node) bool { return n.ID == "Payload" }); model.Nodes[i].Stereotype != "«protocol»" {
		t.Errorf("the protocol's stereotype is %q", model.Nodes[i].Stereotype)
	}
	if want := []models.FileEntry{{Path: "Sources/Monitor.swift", Component: "context", Package: "top"}}; !reflect.DeepEqual(model.ContextFiles, want) {
		t.Errorf("context files %+v", model.ContextFiles)
	}
	for _, c := range model.Components {
		if len(c.Packages) != 1 || !c.Packages[0].Bare {
			t.Errorf("a lone package repeating its component is bare: %+v", c.Packages)
		}
	}
	if given := nodes[0]; given.ID != "Payload" || given.Group != "" || given.Component != "" {
		t.Error("building changes none of the nodes given")
	}
}

func TestTheModelGivesEachTypeAnIDOncePerNameAndFile(t *testing.T) {
	nodes := []*Node{
		node("A", func(n *Node) { n.State, n.File = "new", "x/A.swift" }),
		node("A", func(n *Node) { n.State, n.File = "new", "y/A.swift" }),
		node("readme", func(n *Node) { n.Name, n.Kind, n.State, n.File = "README.md", "artifact", "changed", "README.md" }),
		node("ext", func(n *Node) {
			n.Name, n.Kind, n.State, n.File, n.Line, n.Extends = "String+Trim", "extension", "new", "x/String+Trim.swift", unplaced, []string{"String"}
		}),
	}
	tr := i18n.EnglishTranslationSet()
	model := BuildModel(tr, []diff.File{{Path: "x/A.swift", Status: "A"}, {Path: "y/A.swift", Status: "A"}, {Path: "README.md", Status: "M"}, {Path: "x/String+Trim.swift", Status: "A"}}, nodes, nil, components.Config{}, "widgets")
	var got [][3]any
	for _, n := range model.Nodes {
		got = append(got, [3]any{n.ID, n.Stereotype, n.Line})
	}
	want := [][3]any{{"A@x/A.swift", "", 0}, {"A@y/A.swift", "", 0}, {"file:README.md", "«file»", 0}, {"ext:x/String+Trim.swift", "«extension» of String", 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}
