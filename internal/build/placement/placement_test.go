package placement

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/connorb645/lazyreview/internal/build/components"
	"github.com/connorb645/lazyreview/internal/i18n"
)

var byDefault = Grouping{MaxGroups: 8, SplitAbove: 8}

func config(t *testing.T, raw string) components.Config {
	t.Helper()
	var given []byte
	if raw != "" {
		given = []byte(raw)
	}
	c, err := components.Read(i18n.EnglishTranslationSet(), given)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func named(prefix string, n int, suffix string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%s%d%s", prefix, i, suffix)
	}
	return out
}

func folders(groups []FolderGroup) []string {
	var out []string
	for _, g := range groups {
		if g.Others {
			out = append(out, "<others>")
		} else {
			out = append(out, g.Folder)
		}
	}
	return out
}

func TestLiteralPatternsCoverTheirFolderAndWildcardsStayWithinOne(t *testing.T) {
	for _, c := range []struct {
		pattern, path string
		want          bool
	}{
		{"src/app", "src/app/a.ts", true}, {"src/app", "src/app", true}, {"src/app", "src/apple.ts", false},
		{"src/*.ts", "src/a.ts", true}, {"src/*.ts", "src/x/a.ts", false},
		{"src/**/*.ts", "src/a.ts", true}, {"src/**/*.ts", "src/x/y/a.ts", true},
		{"**", "anything/at/all", true}, {"a?c.md", "abc.md", true}, {"a?c.md", "a/c.md", false},
		{"docs/", "docs/a/b.md", true}, {"a.b/*", "a.b/c", true}, {"a.b/*", "axb/c", false},
	} {
		if got := NewPattern(c.pattern).Matches(c.path); got != c.want {
			t.Errorf("%s matches %s: %v", c.pattern, c.path, got)
		}
	}
	if !(NewPattern("src/app/Main.swift").Specificity > NewPattern("src/app/").Specificity && NewPattern("src/app/").Specificity > NewPattern("src/**").Specificity) {
		t.Error("an exact path is more specific than its folder, and a folder than a wildcard")
	}
	if got := NewPattern("src/app").Specificity; got != 7.5 {
		t.Errorf("a literal path without a slash counts half a character more: %v", got)
	}
}

func TestFolderGroupingKeepsASmallDiffTogetherAndSplitsABigOneByFolder(t *testing.T) {
	if got := folders(GroupByFolder([]string{"a/b/c.swift"}, byDefault)); !reflect.DeepEqual(got, []string{"a/b"}) {
		t.Errorf("one file groups as %v", got)
	}
	if got := folders(GroupByFolder([]string{"src/x/a.ts", "src/x/b.ts", "test/a.test.ts"}, byDefault)); !reflect.DeepEqual(got, []string{""}) {
		t.Errorf("a small diff groups as %v", got)
	}
	big := append(append(named("app/f", 20, ".ts"), named("lib/g", 12, ".ts")...), "README.md")
	var got [][3]any
	for _, g := range GroupByFolder(big, Grouping{MaxGroups: 8, SplitAbove: 8, Balanced: true}) {
		got = append(got, [3]any{g.Folder, len(g.Paths), g.Direct})
	}
	if want := [][3]any{{"", 1, true}, {"app", 20, false}, {"lib", 12, false}}; !reflect.DeepEqual(got, want) {
		t.Errorf("a big diff groups as %v", got)
	}
}

func TestBalancedGroupingDoesNotSplitAFolderToPeelOffAStrayFile(t *testing.T) {
	paths := append(named("big/main/f", 30, ".ts"), "big/other/one.ts")
	if got := folders(GroupByFolder(paths, Grouping{MaxGroups: 8, SplitAbove: 8, Balanced: true})); !reflect.DeepEqual(got, []string{"big"}) {
		t.Errorf("got %v", got)
	}
}

func TestAFolderOneSubfolderDominatesSplitsIntoThatSubfoldersPartsWithTheRestApart(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	paths := append(append(named("Modules/Search/Sources/f", 14, ".swift"), named("Modules/Basket/Sources/g", 8, ".swift")...), "App/App.xcodeproj/project.pbxproj", "App/App/Main.swift")
	var got [][2]any
	for _, g := range GroupByFolder(paths, Grouping{MaxGroups: 8, SplitAbove: 8, Balanced: true}) {
		folder := g.Folder
		if g.Others {
			folder = "<others>"
		}
		got = append(got, [2]any{folder, len(g.Paths)})
	}
	if want := [][2]any{{"Modules/Basket/Sources", 8}, {"Modules/Search/Sources", 14}, {"<others>", 2}}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
	var placed [][2]any
	for _, c := range PlaceFiles(tr, paths, config(t, ""), false, "app").Components {
		placed = append(placed, [2]any{c.Title, c.Size})
	}
	if want := [][2]any{{"Search/Sources", 14}, {"Basket/Sources", 8}, {"Other files", 2}}; !reflect.DeepEqual(placed, want) {
		t.Errorf("placed as %v", placed)
	}
	flat := append(named("src/f", 30, ".ts"), "README.md", "docs/a.md")
	if got := folders(GroupByFolder(flat, Grouping{MaxGroups: 8, SplitAbove: 8, Balanced: true})); !reflect.DeepEqual(got, []string{""}) {
		t.Errorf("a dominant folder with nothing to split into stays whole: %v", got)
	}
}

func TestBesideADominantFolderTheRestIsOneOtherFilesComponentEachFileOnce(t *testing.T) {
	var paths []string
	for _, part := range []string{"a", "b", "c"} {
		paths = append(paths, named("big/"+part+"/P", 27, ".swift")...)
	}
	paths = append(paths, named("big/small/S", 3, ".swift")...)
	for _, side := range []string{"x", "y", "z"} {
		paths = append(paths, named(side+"/H", 5, ".swift")...)
	}
	paths = append(paths, named("T", 5, ".swift")...)
	placed := PlaceFiles(i18n.EnglishTranslationSet(), paths, config(t, ""), false, "spread")
	var got []string
	held := map[string]bool{}
	for _, c := range placed.Components {
		got = append(got, fmt.Sprintf("%s %d", c.Title, c.Size))
		held[c.ID] = true
	}
	if want := []string{"a 27", "b 27", "c 27", "Other files 23"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("the components are %v", got)
	}
	for path, place := range placed.Places {
		if !held[place.Component] {
			t.Errorf("%s is placed in %s, which no component is", path, place.Component)
		}
	}
}

func TestASmallDiffWithNoConfigIsOneComponentNamedAfterTheRepository(t *testing.T) {
	placed := PlaceFiles(i18n.EnglishTranslationSet(), []string{"Sources/Core/A.swift", "Tests/CoreTests/ATests.swift"}, config(t, ""), false, "core")
	if len(placed.Components) != 1 || placed.Components[0].ID != "root" || placed.Components[0].Title != "core" {
		t.Fatalf("got %+v", placed.Components)
	}
	var titles []string
	for _, p := range placed.Components[0].Packages {
		titles = append(titles, p.Title)
	}
	if !reflect.DeepEqual(titles, []string{"Sources/Core", "Tests/CoreTests"}) {
		t.Errorf("a package of folders is titled as their path: %v", titles)
	}
}

func TestANameTakenFromAFolderIsTheFolderAsItIsOnDisk(t *testing.T) {
	var paths []string
	for _, n := range []string{"FrostWarning", "FrostAlarm"} {
		paths = append(paths, "Sources/Orchard/Features/SeasonFlows/HarvestPlanner/LateFrostWarnings/"+n+".swift")
	}
	for _, n := range []string{"Picker", "RotaBoard"} {
		paths = append(paths, "Sources/Orchard/Features/SeasonFlows/HarvestPlanner/PickingRota/"+n+".swift")
	}
	for _, n := range []string{"RareSeed", "SeedVault"} {
		paths = append(paths, "Sources/Orchard/Features/Seed Catalog/rare_seeds/"+n+".swift")
	}
	for _, n := range []string{"frost-map", "night-chart", "frost-page", "frost-banner"} {
		paths = append(paths, "web/orchard-site/frost-warnings/"+n+".ts")
	}
	tr := i18n.EnglishTranslationSet()
	placed := PlaceFiles(tr, paths, config(t, `{"components": [{"id": "orchard", "title": "Orchard", "paths": ["Sources/**"]}]}`), false, tr.DiagramRepositoryRoot)
	got := map[string][]string{}
	var order []string
	for _, c := range placed.Components {
		order = append(order, c.Title)
		for _, p := range c.Packages {
			got[c.Title] = append(got[c.Title], p.Title)
		}
	}
	want := map[string][]string{
		"Orchard":        {"SeasonFlows/HarvestPlanner/LateFrostWarnings", "SeasonFlows/HarvestPlanner/PickingRota", "Seed Catalog/rare_seeds"},
		"frost-warnings": {"frost-warnings"},
	}
	if !reflect.DeepEqual(order, []string{"Orchard", "frost-warnings"}) || !reflect.DeepEqual(got, want) {
		t.Errorf("got %v %v", order, got)
	}
}

func TestWithoutAConfigFilesAreGroupedByFolderIntoComponentsAndPackages(t *testing.T) {
	paths := []string{"Sources/Core/Log/A.swift", "Sources/Core/Log/B.swift", "Sources/Core/Log/E.swift", "Sources/Core/Net/C.swift", "Sources/Core/Net/D.swift",
		"Tests/CoreTests/ATests.swift", "Tests/CoreTests/BTests.swift", "Tests/CoreTests/CTests.swift", "Tests/CoreTests/DTests.swift"}
	tr := i18n.EnglishTranslationSet()
	placed := PlaceFiles(tr, paths, config(t, ""), false, tr.DiagramRepositoryRoot)
	var got [][3]string
	for _, c := range placed.Components {
		got = append(got, [3]string{c.ID, c.Title, c.Tag})
	}
	if want := [][3]string{{"sources-core", "Core", "Sources/Core/"}, {"tests-coretests", "CoreTests", "Tests/CoreTests/"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}
	core := placed.Components[0].Packages
	if len(core) != 2 || core[0].Title != "Log" || core[0].Path != "Sources/Core/Log/" || core[1].Title != "Net" || core[1].Path != "Sources/Core/Net/" {
		t.Errorf("the core packages are %+v", core)
	}
	if tests := placed.Components[1].Packages; len(tests) != 1 || tests[0].Title != "CoreTests" {
		t.Errorf("the test packages are %+v", tests)
	}
	if placed.Places["Sources/Core/Net/D.swift"].Package != core[1].ID {
		t.Errorf("a file is placed in %+v", placed.Places["Sources/Core/Net/D.swift"])
	}
}

func TestConfigPatternsPlaceFilesTheMostSpecificWinning(t *testing.T) {
	c := config(t, `{"components": [
		{"id": "app", "title": "App", "packages": [
			{"id": "screens", "title": "Screens", "paths": ["App/"]},
			{"id": "settings", "title": "Settings", "paths": ["App/SettingsScreen.swift"]}]},
		{"id": "rest", "title": "Everything else", "paths": ["**"]}]}`)
	tr := i18n.EnglishTranslationSet()
	placed := PlaceFiles(tr, []string{"App/Home.swift", "App/SettingsScreen.swift", "README.md", "tools/x/y.py"}, c, false, tr.DiagramRepositoryRoot)
	for path, want := range map[string]Place{
		"App/Home.swift": {"app", "screens"}, "App/SettingsScreen.swift": {"app", "settings"},
	} {
		if got := placed.Places[path]; got != want {
			t.Errorf("%s is placed in %+v", path, got)
		}
	}
	if placed.Places["README.md"].Component != "rest" || placed.Places["tools/x/y.py"].Component != "rest" {
		t.Errorf("the rest is placed in %+v", placed.Places)
	}
	if len(placed.Components) != 2 || placed.Components[0].ID != "app" || placed.Components[1].ID != "rest" {
		t.Errorf("got %+v", placed.Components)
	}
}

func TestFilesTheConfigDoesNotNameAreGroupedByFolderAfterTheNamedComponents(t *testing.T) {
	paths := append([]string{"App/a.ts"}, named("lib/x", 5, ".ts")...)
	tr := i18n.EnglishTranslationSet()
	placed := PlaceFiles(tr, paths, config(t, `{"components": [{"id": "app", "title": "App", "paths": ["App/"]}]}`), false, tr.DiagramRepositoryRoot)
	var got [][3]any
	for _, c := range placed.Components {
		got = append(got, [3]any{c.ID, c.Declared, c.Size})
	}
	if want := [][3]any{{"app", true, 1}, {"lib", false, 5}}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestContextFilesGoToContextComponentsAndTheRestToOneUnchangedCodeComponent(t *testing.T) {
	c := config(t, `{"components": [
		{"id": "core", "title": "Core", "paths": ["src/"]},
		{"id": "sdk", "title": "SDK around it", "context": true, "packages": [{"id": "net", "title": "Network", "paths": ["src/net/"]}]}]}`)
	tr := i18n.EnglishTranslationSet()
	placed := PlaceFiles(tr, []string{"src/net/Client.ts", "src/db/Store.ts"}, c, true, tr.DiagramRepositoryRoot)
	if got := placed.Places["src/net/Client.ts"]; got != (Place{"sdk", "net"}) {
		t.Errorf("a context file the config names is placed in %+v", got)
	}
	if got := placed.Places["src/db/Store.ts"].Component; got != ContextComponent {
		t.Errorf("a context file it does not is placed in %s", got)
	}
	var got [][3]any
	for _, c := range placed.Components {
		got = append(got, [3]any{c.ID, c.Title, c.Context})
	}
	if want := [][3]any{{"sdk", "SDK around it", true}, {"context", "Unchanged code", true}}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
	if placed.Components[1].Description != tr.DiagramContextDescription {
		t.Errorf("the unchanged code says %q", placed.Components[1].Description)
	}
}

func TestFoldersForHeadings(t *testing.T) {
	if got := CommonFolder([]string{"a/b/c.ts", "a/b/d/e.ts"}); got != "a/b/" {
		t.Errorf("got %q", got)
	}
	if got := CommonFolder([]string{"a.ts", "b/c.ts"}); got != "" {
		t.Errorf("got %q", got)
	}
	if got := CommonFolder(nil); got != "" {
		t.Errorf("no paths share %q", got)
	}
	if got := WhereFiles([]string{"Tests/A/x.swift", "Tests/A/y.swift", "Package.swift", "scripts/run.sh"}); got != "Tests/ · Package.swift · scripts/" {
		t.Errorf("got %q", got)
	}
	if got := WhereFiles([]string{"a/1", "b/1", "c/1", "d/1", "d/2"}); got != "d/ · a/ · b/ · …" {
		t.Errorf("more than three places end in an ellipsis: %q", got)
	}
}
