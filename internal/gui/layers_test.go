package gui

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const internal = "github.com/connorb645/lazyreview/internal/"

var mayUse = map[string][]string{
	"app":          {"gui", "controllers", "helpers", "context", "presentation", "views", "types", "reviews", "logic", "store", "github", "claude", "git", "oscommands", "diagram", "shared"},
	"gui":          {"controllers", "helpers", "context", "presentation", "views", "types", "oscommands", "diagram", "shared"},
	"controllers":  {"helpers", "context", "presentation", "views", "types", "diagram", "shared"},
	"helpers":      {"context", "presentation", "views", "types", "oscommands", "diagram", "shared"},
	"context":      {"presentation", "views", "types", "diagram", "shared"},
	"presentation": {"views", "types", "diagram", "shared"},
	"views":        {"shared"},
	"types":        {"shared"},
	"diagram":      {"shared"},
	"reviews":      {"logic", "store", "github", "shared"},
	"logic":        {"store", "github", "claude", "git", "shared"},
	"store":        {"shared"},
	"github":       {"shared"},
	"claude":       {"shared"},
	"git":          {"shared"},
	"oscommands":   {"shared"},
	"testsupport":  {"github", "shared"},
	"shared":       {},
}

func layerOf(pkg string) string {
	switch pkg {
	case "app", "gui", "diagram", "reviews", "store", "testsupport":
		return pkg
	case "queue", "review", "ask", "home", "loader":
		return "logic"
	case "gui/controllers":
		return "controllers"
	case "gui/controllers/helpers":
		return "helpers"
	case "gui/context":
		return "context"
	case "gui/presentation", "gui/presentation/diagram":
		return "presentation"
	case "gui/views":
		return "views"
	case "gui/types":
		return "types"
	case "commands/github":
		return "github"
	case "commands/claude":
		return "claude"
	case "commands/git":
		return "git"
	case "commands/oscommands":
		return "oscommands"
	}
	if strings.HasPrefix(pkg, "build/") {
		return "logic"
	}
	return "shared"
}

// Each package imports only the layers below its own, as lazygit's
// dependency rule has them, the shared packages, such as the models, being
// below all, and only tests use testsupport. The gui reaches reviews only
// through the Reviews it is given; the reviews reach the logic of the queue,
// of a review, of asking Claude and of loading a review and building its
// page, which reach the store, GitHub, git and Claude Code.
func TestEachLayerImportsOnlyTheLayersBelowIt(t *testing.T) {
	root := ".."
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		dir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		layer := layerOf(filepath.ToSlash(dir))
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imported, _ := strconv.Unquote(spec.Path.Value)
			pkg, ok := strings.CutPrefix(imported, internal)
			if other := layerOf(pkg); ok && other != layer && !slices.Contains(mayUse[layer], other) {
				t.Errorf("%s imports %s, but the %s layer may not use the %s layer", path, pkg, layer, other)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
