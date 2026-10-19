package presentation

import (
	"slices"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/commands/server"
	"github.com/connorb645/lazyreview/terminal/internal/gui/types"
)

// bakery is a made-up review of two components and their context. The
// oven folder's own files sort on either side of its timer folder, so the
// tree's order is not the order of the paths.
func bakery() types.ReviewData {
	file := func(path, component string, pulls ...int) server.FileEntry {
		return server.FileEntry{Path: path, Component: component, Status: "M", Pulls: pulls}
	}
	return types.ReviewData{Page: server.Page{Model: server.Model{
		Components: []server.Component{{ID: "ovens"}, {ID: "orders"}, {ID: "context", Context: true}},
		Files: []server.FileEntry{
			file("src/ovens/heat/burner.ts", "ovens", 41),
			file("src/ovens/oven.ts", "ovens", 42),
			file("src/ovens/timer/timer.ts", "ovens", 41, 42),
			file("src/ovens/vent.ts", "ovens", 41),
			file("src/orders/queue.ts", "orders", 43),
		},
		ContextFiles: []server.FileEntry{{Path: "src/shared/clock.ts", Component: "context"}},
	}}}
}

func paths(files []server.FileEntry) []string {
	var out []string
	for _, f := range files {
		out = append(out, f.Path)
	}
	return out
}

func TestTheTreesFilesGoByComponentThenByFolder(t *testing.T) {
	want := []string{"src/ovens/heat/burner.ts", "src/ovens/oven.ts", "src/ovens/vent.ts", "src/ovens/timer/timer.ts", "src/orders/queue.ts", "src/shared/clock.ts"}
	if got := paths(TreeFiles(bakery(), nil)); !slices.Equal(got, want) {
		t.Fatalf("the tree's files are %v", got)
	}
}

func TestARowHoldsItsFileOrEveryFileUnderIt(t *testing.T) {
	d := bakery()
	for _, c := range []struct {
		name   string
		keys   []string
		filter types.PullFilter
		want   []string
	}{
		{"a file's row", []string{FileKey("src/ovens/timer/timer.ts")}, nil, []string{"src/ovens/timer/timer.ts"}},
		{"a folder's row", []string{DirKey("ovens", "src/ovens/oven.ts")}, nil, []string{"src/ovens/oven.ts", "src/ovens/vent.ts"}},
		{"a component's row", []string{ComponentKey("ovens")}, nil, []string{"src/ovens/heat/burner.ts", "src/ovens/oven.ts", "src/ovens/vent.ts", "src/ovens/timer/timer.ts"}},
		{"rows in the tree's order", []string{FileKey("src/orders/queue.ts"), FileKey("src/ovens/heat/burner.ts")}, nil, []string{"src/ovens/heat/burner.ts", "src/orders/queue.ts"}},
		{"a file under a row also chosen, once", []string{ComponentKey("ovens"), FileKey("src/ovens/vent.ts")}, nil, []string{"src/ovens/heat/burner.ts", "src/ovens/oven.ts", "src/ovens/vent.ts", "src/ovens/timer/timer.ts"}},
		{"rows of two components", []string{FileKey("src/orders/queue.ts"), ComponentKey("context")}, nil, []string{"src/orders/queue.ts", "src/shared/clock.ts"}},
		{"what the filter shows", []string{ComponentKey("ovens"), ComponentKey("context")}, types.PullFilter{42}, []string{"src/ovens/oven.ts", "src/ovens/timer/timer.ts", "src/shared/clock.ts"}},
		{"nothing the filter leaves out", []string{FileKey("src/orders/queue.ts")}, types.PullFilter{41}, nil},
	} {
		if got := paths(ChosenFiles(d, c.keys, c.filter)); !slices.Equal(got, c.want) {
			t.Errorf("%s holds %v, not %v", c.name, got, c.want)
		}
	}
}

func TestTheChosenRowsAreLitAndTheFilesOfAFolderWithNoRow(t *testing.T) {
	row := func(key, kind, path, component string) SideRow {
		return SideRow{Key: key, Item: Item{Kind: kind, Component: component, Place: &server.Lines{Path: path, Side: "file:"}}}
	}
	rows := []SideRow{
		row(ComponentKey("ovens"), "component", "src/ovens/oven.ts", "ovens"),
		row(FileKey("src/ovens/oven.ts"), "file", "src/ovens/oven.ts", "ovens"),
		row(FileKey("src/ovens/vent.ts"), "file", "src/ovens/vent.ts", "ovens"),
		row(ComponentKey("orders"), "component", "src/orders/queue.ts", "orders"),
		row(FileKey("src/orders/queue.ts"), "file", "src/orders/queue.ts", "orders"),
	}
	lit := func(keys ...string) []bool {
		var out []bool
		for _, r := range LitRows(rows, keys) {
			out = append(out, r.Lit)
		}
		return out
	}
	if got := lit(ComponentKey("orders")); !slices.Equal(got, []bool{false, false, false, true, false}) {
		t.Fatalf("a component's row is lit alone, not its files: %v", got)
	}
	if got := lit(DirKey("ovens", "src/ovens/oven.ts"), FileKey("src/orders/queue.ts")); !slices.Equal(got, []bool{false, true, true, false, true}) {
		t.Fatalf("a folder with no row lights its files: %v", got)
	}
}
