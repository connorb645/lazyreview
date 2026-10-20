package unchanged

import (
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/connorb645/lazyreview/internal/build/languages"
	"github.com/connorb645/lazyreview/internal/build/model"
	"github.com/connorb645/lazyreview/internal/commands/git"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/utils"
)

const (
	maxFiles = 600
	jsSpace  = `[\t\n\v\f\r \x{00a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}]`
)

var plainName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Candidates finds the unchanged types the drawn nodes touch, looking both
// ways in the repository at head: types the changed code names, found by
// searching for their declarations, and unchanged code that names a
// changed type. Types named in include are found whatever links them. It
// reads only those files, and none of changedPaths, at most 600 of them.
func Candidates(repo *git.Repository, head string, drawn []*model.Node, changedPaths, include []string) (Found, error) {
	found := Found{Texts: map[string]*string{}}
	changed := map[string]bool{}
	for _, p := range changedPaths {
		changed[p] = true
	}
	var families []string
	for _, n := range drawn {
		if n.Family != "" && !slices.Contains(families, n.Family) {
			families = append(families, n.Family)
		}
	}
	outward := map[string][]string{}
	inward := map[string]int{}
	wanted := map[string][]string{}
	for _, family := range families {
		f := languages.Families[family]
		words := strings.Join(f.Keywords, "|")
		scan := regexp.MustCompile(`\b(?:` + words + `)` + jsSpace + `+([A-Za-z_][A-Za-z0-9_]*)`)
		hits, err := repo.Grep(head, "("+words+")[[:space:]]+[A-Za-z_]", f.Globs)
		if err != nil {
			return found, err
		}
		var declaredNames []string
		declared := map[string][]string{}
		for _, hit := range hits {
			if changed[hit.Path] {
				continue
			}
			for _, m := range scan.FindAllStringSubmatch(hit.Text, -1) {
				if _, seen := declared[m[1]]; !seen {
					declaredNames = append(declaredNames, m[1])
				}
				if !slices.Contains(declared[m[1]], hit.Path) {
					declared[m[1]] = append(declared[m[1]], hit.Path)
				}
			}
		}
		var own []*model.Node
		drawnNames := map[string]bool{}
		for _, n := range drawn {
			if n.Family == family {
				own = append(own, n)
				drawnNames[n.Name] = true
			}
		}
		var named []string
		for _, n := range own {
			text := []string{n.Code}
			for _, p := range n.Properties {
				text = append(text, p.Type)
			}
			text = append(append(text, n.Inherits...), n.Extends...)
			for _, word := range languages.Identifiers(strings.Join(text, " ")) {
				if _, held := declared[word]; held && !drawnNames[word] && !slices.Contains(named, word) {
					named = append(named, word)
				}
			}
		}
		for _, name := range include {
			if _, held := declared[name]; held && !slices.Contains(named, name) {
				named = append(named, name)
			}
		}
		for _, name := range named {
			for _, path := range declared[name] {
				if !slices.Contains(outward[path], name) {
					outward[path] = append(outward[path], name)
				}
			}
		}
		var existing []string
		for _, n := range own {
			if n.State == "changed" && n.Kind != "artifact" {
				existing = append(existing, n.Name)
			}
		}
		for _, n := range own {
			existing = append(existing, n.Extends...)
		}
		var unique []string
		for _, name := range existing {
			if !slices.Contains(unique, name) {
				unique = append(unique, name)
			}
		}
		if len(unique) > 0 {
			counts, err := repo.CountMentions(head, unique, f.Globs)
			if err != nil {
				return found, err
			}
			for path, count := range counts {
				if !changed[path] {
					inward[path] = count
				}
			}
		}
		wanted[family] = unique
	}

	var paths []string
	for path := range outward {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var rest []string
	for path := range inward {
		if _, held := outward[path]; !held {
			rest = append(rest, path)
		}
	}
	sort.Slice(rest, func(a, b int) bool {
		if inward[rest[a]] != inward[rest[b]] {
			return inward[rest[a]] > inward[rest[b]]
		}
		return utils.LocaleCompare(rest[a], rest[b]) < 0
	})
	for _, p := range rest {
		if len(paths) >= maxFiles {
			break
		}
		paths = append(paths, p)
	}
	paths = paths[:min(len(paths), maxFiles)]
	texts, err := repo.Read(head, paths)
	if err != nil {
		return found, err
	}
	found.Texts = texts
	for _, path := range paths {
		language := languages.LanguageFor(path)
		text := texts[path]
		if language == nil || text == nil {
			continue
		}
		read := language.Read(*text)
		names := outward[path]
		changedNames := wanted[language.Family]
		for _, d := range read.Declarations {
			if !d.Top || d.Kind == "extension" {
				continue
			}
			code := read.Masked[d.Span[0]:d.Span[1]]
			named := slices.Contains(names, d.Name)
			if !named && !slices.ContainsFunc(languages.Identifiers(code), func(w string) bool { return slices.Contains(changedNames, w) }) {
				continue
			}
			node := model.TypeNode(path, language, d, code, nil)
			node.State = "context"
			found.Nodes = append(found.Nodes, node)
		}
	}
	return found, nil
}

// Extensions finds, for the unchanged types drawn as context, the files at
// head that extend them: one context node per such file, holding every
// extension it declares of those types. A file in skip is drawn already
// and is left out, as is test code, which is never drawn as context.
func Extensions(tr *i18n.TranslationSet, repo *git.Repository, head string, types []*model.Node, skip map[string]bool) (Found, error) {
	found := Found{Texts: map[string]*string{}}
	var families []string
	for _, t := range types {
		if t.Family != "" && !slices.Contains(families, t.Family) {
			families = append(families, t.Family)
		}
	}
	for _, family := range families {
		var names []string
		for _, t := range types {
			if t.Family == family && !slices.Contains(names, t.Name) {
				names = append(names, t.Name)
			}
		}
		var words []string
		for _, n := range names {
			if plainName.MatchString(n) {
				words = append(words, n)
			}
		}
		if len(words) == 0 {
			continue
		}
		hits, err := repo.Grep(head, "extension[[:space:]]+("+strings.Join(words, "|")+")([^A-Za-z0-9_]|$)", languages.Families[family].Globs)
		if err != nil {
			return found, err
		}
		var paths []string
		for _, hit := range hits {
			if !skip[hit.Path] && !slices.Contains(paths, hit.Path) {
				paths = append(paths, hit.Path)
			}
		}
		sort.Strings(paths)
		texts, err := repo.Read(head, paths)
		if err != nil {
			return found, err
		}
		for _, path := range paths {
			language := languages.LanguageFor(path)
			text := texts[path]
			if language == nil || text == nil {
				continue
			}
			read := language.Read(*text)
			var extensions []languages.Declaration
			for _, d := range read.Declarations {
				extended, _, _ := strings.Cut(d.Name, ".")
				if d.Top && d.Kind == "extension" && slices.Contains(names, extended) {
					extensions = append(extensions, d)
				}
			}
			if len(extensions) == 0 || model.IsTestPath(path) {
				continue
			}
			codes := make([]string, len(extensions))
			node := model.TypeNode(path, language, extensions[0], "", nil)
			node.Kind, node.Extends, node.Inherits = "extension", []string{}, []string{}
			for i, d := range extensions {
				codes[i] = read.Masked[d.Span[0]:d.Span[1]]
				extended, _, _ := strings.Cut(d.Name, ".")
				if !slices.Contains(node.Extends, extended) {
					node.Extends = append(node.Extends, extended)
				}
				for _, s := range d.Inherits {
					if !slices.Contains(node.Inherits, s) {
						node.Inherits = append(node.Inherits, s)
					}
				}
			}
			node.Code = strings.Join(codes, "\n")
			node.Name = model.ExtensionName(tr, path, node.Extends)
			node.State = "context"
			found.Nodes = append(found.Nodes, node)
			found.Texts[path] = text
		}
	}
	return found, nil
}
