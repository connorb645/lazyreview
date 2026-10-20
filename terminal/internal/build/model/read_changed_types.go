package model

import (
	"regexp"
	"slices"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/build/diff"
	"github.com/connorb645/lazyreview/terminal/internal/build/languages"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var (
	testFolder = regexp.MustCompile(`(?:^|/)(?:Tests?|tests?|__tests__|spec|androidTest|testFixtures)/`)
	testType   = regexp.MustCompile(`(?:Tests?|Spec)\.(?:swift|kt|java)$`)
	testScript = regexp.MustCompile(`(?:^|/)test_[^/]*\.py$|_test\.py$|\.(?:test|spec)\.[cm]?[jt]sx?$`)
	extension  = regexp.MustCompile(`\.[^.]+$`)
)

// ReadChangedTypes reads the types each changed file declares, and which
// of them the change touches.
func ReadChangedTypes(tr *i18n.TranslationSet, files []diff.File) ChangedTypes {
	var out ChangedTypes
	var pending []pendingExtension
	var conformances []conformance
	for _, file := range files {
		language := languages.LanguageFor(file.Path)
		text := file.Text
		if file.Status == "D" {
			text = file.BaseText
		}
		if language == nil || file.Binary || text == nil {
			out.Nodes = append(out.Nodes, fileBox(file, nil, ""))
			continue
		}
		read := language.Read(*text)
		var tops []languages.Declaration
		for _, d := range read.Declarations {
			if d.Top {
				tops = append(tops, d)
			}
		}
		nested := nestedNames(read.Declarations)
		baseNames := map[string]bool{}
		if (file.Status == "M" || file.Status == "R") && file.BaseText != nil {
			for _, d := range language.Read(*file.BaseText).Declarations {
				if d.Top {
					baseNames[d.Name] = true
				}
			}
		}
		before := len(out.Nodes)
		for _, d := range tops {
			touched := file.Status == "A" || file.Status == "D" || diff.Touches(file.Hunks, d.Start, d.End)
			code := read.Masked[d.Span[0]:d.Span[1]]
			if d.Kind == "extension" {
				extended, _, _ := strings.Cut(d.Name, ".")
				if touched {
					pending = append(pending, pendingExtension{file: file, language: language, extended: extended, declaration: d, code: code})
				} else {
					conformances = append(conformances, conformance{extended: extended, family: language.Family, inherits: d.Inherits})
				}
				continue
			}
			node := TypeNode(file.Path, language, d, code, nested[d.Name])
			switch {
			case file.Status == "D":
				node.State = "removed"
			case touched && (file.Status == "A" || !baseNames[d.Name]):
				node.State = "new"
			case touched:
				node.State = "changed"
			default:
				node.State = "same"
				out.Untouched = append(out.Untouched, node)
				continue
			}
			out.Nodes = append(out.Nodes, node)
		}
		// A file box stands for code with no types; a file whose types the
		// change leaves alone keeps no code here, so it pulls in no context.
		if len(out.Nodes) == before && !slices.ContainsFunc(pending, func(p pendingExtension) bool { return p.file.Path == file.Path }) {
			masked := read.Masked
			if len(tops) > 0 {
				masked = ""
			}
			out.Nodes = append(out.Nodes, fileBox(file, language, masked))
		}
		if len(read.Imports) > 0 {
			out.Imports = append(out.Imports, FileImports{Path: file.Path, Imports: read.Imports})
		}
	}

	// A changed extension of a type drawn as changed is part of that type.
	// Any other changed extension stands for its file.
	byName := map[string]*Node{}
	for _, n := range out.Nodes {
		if n.Kind != "artifact" {
			byName[n.Family+":"+n.Name] = n
		}
	}
	extensionFiles := map[string]*Node{}
	var extensionOrder []*Node
	for _, p := range pending {
		// A type whose own file extends it is changed by that extension.
		if local := slices.IndexFunc(out.Untouched, func(u *Node) bool { return u.File == p.file.Path && u.Name == p.extended }); local >= 0 {
			promoted := out.Untouched[local]
			out.Untouched = slices.Delete(out.Untouched, local, local+1)
			promoted.State = "changed"
			out.Nodes = append(out.Nodes, promoted)
			byName[promoted.Family+":"+promoted.Name] = promoted
		}
		if owner := byName[p.language.Family+":"+p.extended]; owner != nil && owner.State != "removed" {
			owner.Code += "\n" + p.code
			owner.Inherits = union(owner.Inherits, p.declaration.Inherits)
			if p.file.Path != owner.File {
				owner.ExtFiles = union(owner.ExtFiles, []string{p.file.Path})
			}
			continue
		}
		node := extensionFiles[p.file.Path]
		if node == nil {
			node = extensionNode(p.file, p.language)
			extensionFiles[p.file.Path] = node
			extensionOrder = append(extensionOrder, node)
			out.Nodes = append(out.Nodes, node)
		}
		node.Extends = union(node.Extends, []string{p.extended})
		node.Code += "\n" + p.code
		node.Line = min(node.Line, p.declaration.Start)
	}
	for _, node := range extensionOrder {
		node.Name = ExtensionName(tr, node.File, node.Extends)
	}
	for _, c := range conformances {
		owner := byName[c.family+":"+c.extended]
		if owner == nil {
			if i := slices.IndexFunc(out.Untouched, func(u *Node) bool { return u.Family == c.family && u.Name == c.extended }); i >= 0 {
				owner = out.Untouched[i]
			}
		}
		if owner != nil {
			owner.Inherits = union(owner.Inherits, c.inherits)
		}
	}
	return out
}

// TypeNode is a node for the type d that the file at path declares, with
// its code, before it is given a state.
func TypeNode(path string, language *languages.Language, d languages.Declaration, code string, nested []string) *Node {
	properties := []models.Property{}
	for _, p := range d.Properties {
		property := models.Property{Name: p.Name, Line: p.Line, Inferred: p.Inferred}
		if p.Type != nil {
			property.Type = *p.Type
		}
		properties = append(properties, property)
	}
	label := ""
	if d.Label != nil {
		label = *d.Label
	}
	return &Node{
		Name: d.Name, Kind: d.Kind, Label: label, File: path, Line: d.Start, End: d.End,
		Family: language.Family, Inherits: d.Inherits, Properties: properties, Nested: orEmpty(nested),
		View: d.View, Test: d.Test || IsTestPath(path), Code: code, ExtFiles: []string{}, Extends: []string{},
	}
}

// ExtensionName is the name a box of extensions shows: its file's name, or
// what it extends.
func ExtensionName(tr *i18n.TranslationSet, path string, extended []string) string {
	stem := extension.ReplaceAllString(baseName(path), "")
	first := "undefined"
	if len(extended) > 0 {
		first = extended[0]
	}
	switch {
	case strings.Contains(stem, "+"):
		return stem
	case stem == first:
		return utils.ResolvePlaceholderString(tr.DiagramExtensionName, map[string]string{"name": stem})
	}
	return first + " · " + stem
}

// IsTestPath says whether a path holds test code, by the usual folder and
// file names.
func IsTestPath(path string) bool {
	return testFolder.MatchString(path) || testType.MatchString(path) || testScript.MatchString(path)
}

func fileBox(file diff.File, language *languages.Language, masked string) *Node {
	family := ""
	if language != nil {
		family = language.Family
	}
	state := "changed"
	switch file.Status {
	case "A":
		state = "new"
	case "D":
		state = "removed"
	}
	return &Node{
		Name: baseName(file.Path), Kind: "artifact", File: file.Path, Line: 1, Family: family,
		Inherits: []string{}, Properties: []models.Property{}, Nested: []string{}, Test: IsTestPath(file.Path),
		Code: masked, ExtFiles: []string{}, Extends: []string{}, State: state,
	}
}

func extensionNode(file diff.File, language *languages.Language) *Node {
	node := fileBox(file, language, "")
	node.Kind, node.Line = "extension", unplaced
	if file.Status != "A" {
		node.State = "changed"
	}
	return node
}

func nestedNames(declarations []languages.Declaration) map[string][]string {
	out := map[string][]string{}
	for _, d := range declarations {
		if d.Top || d.Kind == "extension" || d.Name == "CodingKeys" || d.Owner == "" || d.Owner == d.Name {
			continue
		}
		out[d.Owner] = append(out[d.Owner], d.Name)
	}
	return out
}

func union(a, b []string) []string {
	out := []string{}
	for _, s := range append(append([]string{}, a...), b...) {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

func orEmpty(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

func baseName(path string) string {
	return path[strings.LastIndex(path, "/")+1:]
}
