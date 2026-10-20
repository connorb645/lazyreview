package presentation

import (
	"html"
	"strings"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

const drawingsStyle = `:root { color-scheme: light dark; }
body { margin: 2rem auto; max-width: 64rem; padding: 0 1rem; font: 16px/1.5 system-ui, sans-serif; color: #17152f; background: #ffffff; }
h1 { font-size: 1.4rem; }
h2 { font-size: 1.1rem; margin-top: 2.5rem; }
svg { display: block; max-width: 100%; height: auto; }
@media (prefers-color-scheme: dark) { body { color: #f1f0fb; background: #15132a; } }`

// DrawingsPage is a page for the browser that shows a written review's
// drawings, each inline SVG of its sections under the section's title, as
// a terminal cannot. Its policy lets the page load nothing and run no
// script. ok is false when no section holds a drawing.
func DrawingsPage(tr *i18n.TranslationSet, title string, written models.Written) (page string, ok bool) {
	heading := html.EscapeString(utils.ResolvePlaceholderString(tr.DrawingsOf, map[string]string{"title": title}))
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString("<meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'none'; style-src 'unsafe-inline'; img-src data:\">\n")
	b.WriteString("<title>" + heading + "</title>\n<style>\n" + drawingsStyle + "\n</style>\n</head>\n<body>\n<h1>" + heading + "</h1>\n")
	for _, s := range written.Sections {
		drawings := inlineSVG.FindAllString(s.Markdown, -1)
		if len(drawings) == 0 {
			continue
		}
		ok = true
		b.WriteString("<h2>" + html.EscapeString(s.Title) + "</h2>\n")
		for _, d := range drawings {
			b.WriteString(d + "\n")
		}
	}
	b.WriteString("</body>\n</html>\n")
	return b.String(), ok
}
