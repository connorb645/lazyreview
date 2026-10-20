package gui

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const internal = "github.com/connorb645/lazyreview/terminal/internal/"

var mayUse = map[string][]string{
	"app":          {"gui", "controllers", "helpers", "context", "presentation", "views", "types", "reviews", "logic", "store", "server", "github", "oscommands", "diagram", "shared"},
	"gui":          {"controllers", "helpers", "context", "presentation", "views", "types", "oscommands", "diagram", "shared"},
	"controllers":  {"helpers", "context", "presentation", "views", "types", "diagram", "shared"},
	"helpers":      {"context", "presentation", "views", "types", "oscommands", "diagram", "shared"},
	"context":      {"presentation", "views", "types", "diagram", "shared"},
	"presentation": {"views", "types", "diagram", "shared"},
	"views":        {"shared"},
	"types":        {"shared"},
	"diagram":      {"shared"},
	"reviews":      {"logic", "store", "server", "github", "shared"},
	"logic":        {"store", "github", "shared"},
	"store":        {"shared"},
	"server":       {"shared"},
	"github":       {"shared"},
	"oscommands":   {"shared"},
	"testsupport":  {"github", "shared"},
	"shared":       {},
}

var callsServer = []string{"app", "reviews", "server"}

func layerOf(pkg string) string {
	switch pkg {
	case "app", "gui", "diagram", "reviews", "store", "testsupport":
		return pkg
	case "queue", "review":
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
	case "commands/server":
		return "server"
	case "commands/github":
		return "github"
	case "commands/oscommands":
		return "oscommands"
	}
	return "shared"
}

// Each package imports only the layers below its own, as lazygit's
// dependency rule has them, the shared packages, such as the models, being
// below all, and only tests use testsupport. The gui reaches reviews only
// through the Reviews it is given; the reviews reach the queue's and a
// review's logic, which reach the store and GitHub. Of the server, only the
// app's start and the reviews hold a client.
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
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !slices.Contains(callsServer, layer) && (strings.Contains(string(source), "server.Client") || strings.Contains(string(source), "server.NewClient")) {
			t.Errorf("%s holds a client of the server, but the %s layer may use only its answers' types", path, layer)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
