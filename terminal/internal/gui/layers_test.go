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
	"app":          {"gui", "controllers", "helpers", "context", "presentation", "views", "types", "server", "oscommands", "diagram", "shared"},
	"gui":          {"controllers", "helpers", "context", "presentation", "views", "types", "server", "oscommands", "diagram", "shared"},
	"controllers":  {"helpers", "context", "presentation", "views", "types", "server", "diagram", "shared"},
	"helpers":      {"context", "presentation", "views", "types", "server", "oscommands", "diagram", "shared"},
	"context":      {"presentation", "views", "types", "server", "diagram", "shared"},
	"presentation": {"views", "types", "server", "diagram", "shared"},
	"views":        {"shared"},
	"types":        {"server", "shared"},
	"diagram":      {"server", "shared"},
	"server":       {"shared"},
	"oscommands":   {"shared"},
	"shared":       {},
}

var callsServer = []string{"app", "gui", "helpers", "server"}

func layerOf(pkg string) string {
	switch pkg {
	case "app", "gui", "diagram":
		return pkg
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
	case "commands/oscommands":
		return "oscommands"
	}
	return "shared"
}

// Each package imports only the layers below its own, as lazygit's
// dependency rule has them, the shared packages being below all. Of the
// server, only the helpers, the gui and the app's start hold a client; the
// layers under them use only its answers' types.
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
