package review

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

var (
	writtenParts   = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n?(.*)$`)
	headerField    = regexp.MustCompile(`^([a-z]+):\s*(.*)$`)
	headCommit     = regexp.MustCompile(`(?i)^[0-9a-f]{7,40}$`)
	pullSeparators = regexp.MustCompile(`[\s,]+`)
	fenceMarker    = regexp.MustCompile("^(```+|~~~+)")
	sectionHeading = regexp.MustCompile(`^##\s+(.+?)\s*#*\s*$`)
	idBreaks       = regexp.MustCompile(`[^a-z0-9]+`)
	digits         = regexp.MustCompile(`^\d+$`)
	headerFields   = map[string]bool{"pull": true, "stack": true, "head": true, "title": true}
)

// WrittenRead is a written review as read: the pull requests it names,
// bottom first, the head it read, its title, and its ## sections.
type WrittenRead struct {
	Pulls    []int
	Head     string
	Title    string
	Sections []models.Section
}

// ReadWrittenReview reads the Markdown an agent or a person attaches to a
// review. It starts with a header between --- lines naming what it read,
// pull: <number> or stack: <numbers, bottom first>, head: <commit> and
// maybe title, then its sections as ## headings, each with something in
// it. It fails naming every problem at once.
func ReadWrittenReview(tr *i18n.TranslationSet, text string) (WrittenRead, error) {
	text = strings.TrimPrefix(strings.ReplaceAll(text, "\r\n", "\n"), "\ufeff")
	m := writtenParts.FindStringSubmatch(text)
	if m == nil {
		return WrittenRead{}, models.Refused(tr.WrittenNoHeader)
	}
	var problems []string
	header := map[string]string{}
	for _, line := range strings.Split(m[1], "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		field := headerField.FindStringSubmatch(line)
		if field == nil {
			problems = append(problems, utils.ResolvePlaceholderString(tr.WrittenHeaderLine, map[string]string{"line": line}))
			continue
		}
		if !headerFields[field[1]] {
			problems = append(problems, utils.ResolvePlaceholderString(tr.WrittenUnknownField, map[string]string{"field": field[1]}))
		}
		header[field[1]] = strings.TrimSpace(field[2])
	}
	if !headCommit.MatchString(header["head"]) {
		problems = append(problems, tr.WrittenNeedsHead)
	}
	if (header["pull"] != "") == (header["stack"] != "") {
		problems = append(problems, tr.WrittenPullOrStack)
	}
	named := header["pull"]
	if named == "" {
		named = header["stack"]
	}
	var pulls []string
	for _, p := range pullSeparators.Split(named, -1) {
		if p != "" {
			pulls = append(pulls, strings.TrimPrefix(p, "#"))
		}
	}
	if header["pull"] != "" && (len(pulls) != 1 || !digits.MatchString(pulls[0])) {
		problems = append(problems, tr.WrittenPullNumber)
	}
	if header["stack"] != "" && (len(pulls) < 2 || !allDigits(pulls)) {
		problems = append(problems, tr.WrittenStackNumbers)
	}

	type section struct {
		title string
		lines []string
	}
	var sections []*section
	var current *section
	fence := ""
	for _, line := range strings.Split(m[2], "\n") {
		if marker := fenceMarker.FindStringSubmatch(line); marker != nil {
			switch {
			case fence == "":
				fence = marker[1][:1]
			case strings.HasPrefix(line, fence):
				fence = ""
			}
		}
		if heading := sectionHeading.FindStringSubmatch(line); fence == "" && !strings.HasPrefix(line, "###") && heading != nil {
			current = &section{title: heading[1]}
			sections = append(sections, current)
		} else if current != nil {
			current.lines = append(current.lines, line)
		} else if strings.TrimSpace(line) != "" {
			problems = append(problems, tr.WrittenTextBefore)
		}
	}
	if len(sections) == 0 {
		problems = append(problems, tr.WrittenNoSection)
	}
	for _, s := range sections {
		if strings.TrimSpace(strings.Join(s.lines, "\n")) == "" {
			problems = append(problems, utils.ResolvePlaceholderString(tr.WrittenEmptySection, map[string]string{"title": s.title}))
		}
	}
	if len(problems) > 0 {
		return WrittenRead{}, models.Refused(utils.ResolvePlaceholderString(tr.WrittenProblems, map[string]string{"problems": Listed(utils.Plural(len(problems), tr.ProblemNoun), problems)}))
	}
	read := WrittenRead{Head: strings.ToLower(header["head"]), Title: header["title"]}
	for _, p := range pulls {
		n, _ := strconv.Atoi(p)
		read.Pulls = append(read.Pulls, n)
	}
	ids := map[string]bool{}
	for _, s := range sections {
		read.Sections = append(read.Sections, models.Section{ID: uniqueID(s.title, ids), Title: s.title, Markdown: strings.TrimSpace(strings.Join(s.lines, "\n"))})
	}
	return read, nil
}

// Listed is a count of problems, then each on a line of its own.
func Listed(count string, problems []string) string {
	var out strings.Builder
	out.WriteString(count + ":")
	for _, p := range problems {
		out.WriteString("\n  - " + p)
	}
	return out.String()
}

func allDigits(parts []string) bool {
	for _, p := range parts {
		if !digits.MatchString(p) {
			return false
		}
	}
	return true
}

func uniqueID(title string, ids map[string]bool) string {
	base := strings.Trim(idBreaks.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if base == "" {
		base = "section"
	}
	id := base
	for k := 2; ids[id]; k++ {
		id = base + "-" + strconv.Itoa(k)
	}
	ids[id] = true
	return id
}
