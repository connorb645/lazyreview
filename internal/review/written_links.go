package review

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/connorb645/lazyreview/internal/models"
)

var location = regexp.MustCompile(`^(.+?\.[A-Za-z0-9]+):(\d+)(?:[-–](\d+))?$`)

// WrittenLinks are the file:line links of a written review's section: each
// piece of inline code such as Basket.swift:12 or src/log.ts:31-40 that
// names a file of the review, where exactly one of its paths ends with that
// name, once each, in the order they first appear. Code inside a code block
// is no link.
func WrittenLinks(markdownText string, paths []string) []models.Link {
	source := []byte(markdownText)
	doc := markdown.Parser().Parse(text.NewReader(source))
	links := []models.Link{}
	seen := map[string]bool{}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		span, ok := n.(*ast.CodeSpan)
		if !entering || !ok {
			return ast.WalkContinue, nil
		}
		var code strings.Builder
		for c := span.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				code.Write(t.Segment.Value(source))
			}
		}
		named := strings.TrimSpace(code.String())
		m := location.FindStringSubmatch(named)
		if m == nil || seen[named] {
			return ast.WalkSkipChildren, nil
		}
		var hits []string
		for _, p := range paths {
			if p == m[1] || strings.HasSuffix(p, "/"+m[1]) {
				hits = append(hits, p)
			}
		}
		if len(hits) == 1 {
			line, _ := strconv.Atoi(m[2])
			seen[named] = true
			links = append(links, models.Link{Text: named, Path: hits[0], Line: line})
		}
		return ast.WalkSkipChildren, nil
	})
	return links
}
