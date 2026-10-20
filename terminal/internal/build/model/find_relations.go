package model

import (
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/build/languages"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
)

var (
	strength         = map[string]int{"inherit": 5, "realize": 5, "has": 4, "extend": 3, "use": 2}
	scriptExtensions = []string{".ts", ".tsx", ".mts", ".cts", ".js", ".jsx", ".mjs", ".cjs"}
	scriptModuleStem = regexp.MustCompile(`\.(?:[cm]?js|jsx)$`)
	interfaceKinds   = []string{"protocol", "interface"}
)

// FindRelations reads the links between the nodes from their code:
// realize, a type conforming to a protocol or implementing an interface;
// inherit, from a class, or a protocol from a protocol; has, a stored
// property holding the other type, naming those properties; extend, a file
// of extensions extending the other type; use, the code naming the other
// type, or importing the other file; and test, any link from a test. A
// name resolves within its language family only, to the nearest type of
// that name by folder. The imports link file boxes for code with no types
// of its own.
func FindRelations(tr *i18n.TranslationSet, nodes []*Node, imports []FileImports) []models.Edge {
	byName := map[string][]*Node{}
	for _, n := range nodes {
		if n.Kind == "artifact" || n.Kind == "extension" || n.Family == "" {
			continue
		}
		byName[n.Family+":"+n.Name] = append(byName[n.Family+":"+n.Name], n)
	}
	resolve := func(name string, from *Node) *Node {
		var candidates []*Node
		for _, c := range byName[from.Family+":"+name] {
			if c != from {
				candidates = append(candidates, c)
			}
		}
		return nearest(candidates, from)
	}

	var keys []string
	edges := map[string]models.Edge{}
	add := func(source, target *Node, kind, label string, properties []string) {
		if target == nil || source == target {
			return
		}
		key := source.ID + ">" + target.ID
		had, held := edges[key]
		if held && strength[had.Kind] >= strength[kind] {
			return
		}
		if !held {
			keys = append(keys, key)
		}
		edges[key] = models.Edge{Source: source.ID, Target: target.ID, Kind: kind, Label: label, Properties: properties}
	}

	for _, n := range nodes {
		if n.Family == "" {
			continue
		}
		for _, s := range n.Inherits {
			if t := resolve(s, n); t != nil {
				kind := "inherit"
				if slices.Contains(interfaceKinds, t.Kind) && !slices.Contains(interfaceKinds, n.Kind) {
					kind = "realize"
				}
				add(n, t, kind, "", nil)
			}
		}
		for _, e := range n.Extends {
			add(n, resolve(e, n), "extend", "", nil)
		}
		var holders []*Node
		held := map[*Node][]string{}
		for _, p := range n.Properties {
			for _, word := range languages.Identifiers(p.Type) {
				t := resolve(word, n)
				if t == nil {
					continue
				}
				if _, seen := held[t]; !seen {
					holders = append(holders, t)
					held[t] = []string{}
				}
				if !slices.Contains(held[t], p.Name) {
					held[t] = append(held[t], p.Name)
				}
			}
		}
		for _, t := range holders {
			label := ""
			if n.View && t.View {
				label = tr.DiagramShows
			}
			add(n, t, "has", label, held[t])
		}
		for _, word := range languages.Identifiers(n.Code) {
			if word == n.Name {
				continue
			}
			if t := resolve(word, n); t != nil {
				if n.View && t.View {
					add(n, t, "has", tr.DiagramShows, nil)
				} else {
					add(n, t, "use", "", nil)
				}
			}
		}
	}

	inFile := map[string][]*Node{}
	var paths []string
	for _, n := range nodes {
		if _, seen := inFile[n.File]; !seen {
			paths = append(paths, n.File)
		}
		inFile[n.File] = append(inFile[n.File], n)
	}
	for _, file := range imports {
		for _, i := range file.Imports {
			var target string
			if i.Python {
				target = pythonTarget(file.Path, i.Spec, i.Names, paths)
			} else {
				target = scriptTarget(file.Path, i.Spec, paths)
			}
			if target == "" || target == file.Path {
				continue
			}
			for _, s := range inFile[file.Path] {
				var words []string
				if s.Kind != "artifact" {
					words = languages.Identifiers(s.Code)
				}
				for _, t := range inFile[target] {
					named := i.Names == nil || slices.Contains(i.Names, t.Name)
					switch {
					case s.Kind == "artifact" && (t.Kind == "artifact" || named):
						add(s, t, "use", "", nil)
					case s.Kind != "artifact" && t.Kind == "artifact" && i.Names != nil && slices.ContainsFunc(i.Names, func(name string) bool { return slices.Contains(words, name) }):
						add(s, t, "use", "", nil)
					}
				}
			}
		}
	}

	tests := map[string]bool{}
	for _, n := range nodes {
		if n.Test {
			tests[n.ID] = true
		}
	}
	out := make([]models.Edge, len(keys))
	for i, key := range keys {
		out[i] = edges[key]
		if tests[out[i].Source] {
			out[i].Kind = "test"
		}
	}
	return out
}

func nearest(candidates []*Node, from *Node) *Node {
	if len(candidates) <= 1 {
		if len(candidates) == 0 {
			return nil
		}
		return candidates[0]
	}
	shared := func(c *Node) int {
		if c.File == from.File {
			return int(^uint(0) >> 1)
		}
		a, b := strings.Split(c.File, "/"), strings.Split(from.File, "/")
		k := 0
		for k < len(a) && k < len(b) && a[k] == b[k] {
			k++
		}
		return k
	}
	sorted := append([]*Node{}, candidates...)
	sort.SliceStable(sorted, func(x, y int) bool { return shared(sorted[x]) > shared(sorted[y]) })
	return sorted[0]
}

func normalize(parts []string) string {
	var out []string
	for _, p := range parts {
		switch p {
		case "", ".":
		case "..":
			if len(out) > 0 {
				out = out[:len(out)-1]
			}
		default:
			out = append(out, p)
		}
	}
	return strings.Join(out, "/")
}

func folderOf(path string) []string {
	parts := strings.Split(path, "/")
	return parts[:len(parts)-1]
}

func scriptTarget(from, spec string, paths []string) string {
	if !strings.HasPrefix(spec, ".") {
		return ""
	}
	base := normalize(append(folderOf(from), strings.Split(spec, "/")...))
	stem := scriptModuleStem.ReplaceAllString(base, "")
	tries := []string{base}
	for _, e := range scriptExtensions {
		tries = append(tries, stem+e)
	}
	for _, e := range scriptExtensions {
		tries = append(tries, base+"/index"+e)
	}
	for _, t := range tries {
		if slices.Contains(paths, t) {
			return t
		}
	}
	return ""
}

func pythonTarget(from, spec string, names []string, paths []string) string {
	dots := len(spec) - len(strings.TrimLeft(spec, "."))
	var module []string
	for _, part := range strings.Split(spec[dots:], ".") {
		if part != "" {
			module = append(module, part)
		}
	}
	modules := [][]string{module}
	for _, n := range names {
		modules = append(modules, append(append([]string{}, module...), n))
	}
	files := func(parts []string) []string {
		joined := strings.Join(parts, "/")
		return []string{joined + ".py", joined + "/__init__.py"}
	}
	if dots > 0 {
		folder := folderOf(from)
		end := len(folder) - (dots - 1)
		if end < 0 {
			end = max(0, len(folder)+end)
		}
		up := folder[:end]
		for i := len(modules) - 1; i >= 0; i-- {
			for _, f := range files(append(append([]string{}, up...), modules[i]...)) {
				if hit := normalize(strings.Split(f, "/")); slices.Contains(paths, hit) {
					return hit
				}
			}
		}
		return ""
	}
	for i := len(modules) - 1; i >= 0; i-- {
		if len(modules[i]) == 0 {
			continue
		}
		wanted := files(modules[i])
		for _, p := range paths {
			if slices.ContainsFunc(wanted, func(f string) bool { return p == f || strings.HasSuffix(p, "/"+f) }) {
				return p
			}
		}
	}
	return ""
}
