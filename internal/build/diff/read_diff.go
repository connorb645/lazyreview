package diff

import (
	"regexp"

	"github.com/connorb645/lazyreview/internal/build/languages"
	"github.com/connorb645/lazyreview/internal/commands/git"
	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
	"github.com/connorb645/lazyreview/internal/utils"
)

var (
	generatedFile = regexp.MustCompile(`(?:^|/)(?:package-lock\.json|yarn\.lock|pnpm-lock\.yaml|Podfile\.lock|Package\.resolved|Gemfile\.lock|poetry\.lock|Cargo\.lock|composer\.lock)$|\.(?:pbxproj|snap|min\.js|min\.css|svg)$`)
	dataFile      = regexp.MustCompile(`(?i)\.(?:json|jsonl|ndjson|csv|tsv|xml|plist|geojson|gpx)$`)
)

// ReadDiff reads the change from the merge base of base and head to head,
// or with exact straight from base, and the text of every file it touches:
// the head's text of each file it keeps, and the base's text only where the
// diagram needs it, for a deleted file and for changed code.
func ReadDiff(tr *i18n.TranslationSet, repo *git.Repository, base, head string, exact bool) (Read, error) {
	var read Read
	var err error
	if read.HeadCommit, err = repo.Resolve(head); err != nil {
		return read, err
	}
	if read.BaseCommit, err = repo.Resolve(base); err != nil {
		return read, err
	}
	read.MergeBase = read.BaseCommit
	if !exact {
		if read.MergeBase, err = repo.MergeBase(read.BaseCommit, read.HeadCommit); err != nil {
			return read, err
		}
	}
	text, err := repo.Diff(read.MergeBase, read.HeadCommit)
	if err != nil {
		return read, err
	}
	read.Chunks = SplitDiff(text)
	if len(read.Chunks) == 0 {
		return read, models.Refused(utils.ResolvePlaceholderString(tr.LoaderNoDifference, map[string]string{"base": base, "head": head}))
	}
	counts, err := repo.Counts(read.MergeBase, read.HeadCommit)
	if err != nil {
		return read, err
	}
	var headPaths, basePaths []string
	for _, c := range read.Chunks {
		if c.Status != "D" {
			headPaths = append(headPaths, c.Path)
		}
		if c.Status == "D" || ((c.Status == "M" || c.Status == "R") && languages.LanguageFor(c.Path) != nil) {
			basePaths = append(basePaths, c.OldPath)
		}
	}
	headTexts, err := repo.Read(read.HeadCommit, headPaths)
	if err != nil {
		return read, err
	}
	baseTexts, err := repo.Read(read.MergeBase, basePaths)
	if err != nil {
		return read, err
	}
	for _, c := range read.Chunks {
		count, counted := counts[c.Path]
		if !counted {
			count = git.FileCount{Binary: c.Binary}
		}
		changed := count.Adds + count.Dels
		file := File{
			Path: c.Path, OldPath: c.OldPath, Status: c.Status, Binary: c.Binary || count.Binary, Hunks: c.Hunks,
			Adds: count.Adds, Dels: count.Dels,
			Collapsed: generatedFile.MatchString(c.Path) || (dataFile.MatchString(c.Path) && changed > 60) || changed > 1500,
		}
		if c.Status != "D" {
			file.Text = headTexts[c.Path]
		}
		if c.OldPath != "" {
			file.BaseText = baseTexts[c.OldPath]
		}
		read.Files = append(read.Files, file)
	}
	return read, nil
}
