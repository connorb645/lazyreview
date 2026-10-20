package review

import (
	"regexp"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
)

func TestAWrittenReviewIsReadIntoWhatItReadAndItsSectionsInOrder(t *testing.T) {
	read, err := ReadWrittenReview(i18n.EnglishTranslationSet(), "---\nstack: 12, 13\nhead: 9d01c2aa\ntitle: Search and basket\n---\n## What it does\nAdds search.\n\n```md\n## not a heading\n```\n## Risks\n### Smaller\nNone.\n## Risks\nAgain.\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Pulls) != 2 || read.Pulls[0] != 12 || read.Pulls[1] != 13 || read.Head != "9d01c2aa" || read.Title != "Search and basket" {
		t.Fatalf("it read %+v", read)
	}
	var ids []string
	for _, s := range read.Sections {
		ids = append(ids, s.ID+" "+s.Title)
	}
	if len(ids) != 3 || ids[0] != "what-it-does What it does" || ids[1] != "risks Risks" || ids[2] != "risks-2 Risks" {
		t.Fatalf("the sections are %v", ids)
	}
	if !regexp.MustCompile(`## not a heading`).MatchString(read.Sections[0].Markdown) {
		t.Fatalf("a heading in a code block began a section")
	}
}

func TestAWrittenReviewWithProblemsNamesEveryOne(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for text, want := range map[string]string{
		"## Only\nx\n": `must start with a header`,
		"---\npull: 12\nstack: 12, 13\nurl: x\n---\nintro\n## Empty\n": `problems:[\s\S]*unknown field "url"[\s\S]*needs "head"[\s\S]*not both[\s\S]*text before the first ## section heading`,
		"---\npull: 12\nhead: abcdef1\n---\n## Empty\n\n":              `"## Empty" is empty`,
	} {
		if _, err := ReadWrittenReview(tr, text); err == nil || !regexp.MustCompile(want).MatchString(err.Error()) {
			t.Errorf("%q gives %v", text, err)
		}
	}
}

func TestAWrittenReviewLinksEachFileLineInInlineCodeToTheOneFileWhosePathEndsWithIt(t *testing.T) {
	markdown := "See `Basket.swift:12`, `log.ts:3-4` and `Basket.swift:12` again, not `x.ts:1`, nor `nowhere.ts:2`.\n\n```\nBasket.swift:2\n```\n\n- in a list, `Sources/Basket/Basket.swift:5`"
	paths := []string{"Sources/Basket/Basket.swift", "src/log.ts", "a/x.ts", "b/x.ts"}
	got := WrittenLinks(markdown, paths)
	want := []models.Link{
		{Text: "Basket.swift:12", Path: "Sources/Basket/Basket.swift", Line: 12},
		{Text: "log.ts:3-4", Path: "src/log.ts", Line: 3},
		{Text: "Sources/Basket/Basket.swift:5", Path: "Sources/Basket/Basket.swift", Line: 5},
	}
	if len(got) != len(want) {
		t.Fatalf("the links are %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("link %d is %+v, not %+v", i, got[i], want[i])
		}
	}
	if len(WrittenLinks("", paths)) != 0 {
		t.Errorf("no text has links")
	}
}

func TestAWrittenReviewMustMatchTheReviewItIsAttachedTo(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	rev := models.Review{Members: []models.ReviewMember{{Pull: 12, Head: "3f2a9c1d0e77"}, {Pull: 13, Head: "9d01c2aa45bb"}}}
	read := func(text string) WrittenRead {
		written, err := ReadWrittenReview(tr, text)
		if err != nil {
			t.Fatal(err)
		}
		return written
	}
	if err := CheckWrittenReview(tr, read("---\nstack: 12, 13\nhead: 9d01c2aa\n---\n## Notes\nx\n"), rev); err != nil {
		t.Fatalf("a written review of the stack's top head is refused: %v", err)
	}
	if err := CheckWrittenReview(tr, read("---\npull: 12\nhead: abcdef1\n---\n## Notes\nx\n"), rev); err == nil || err.Error() != "the written review names #12, but the review shows #12, #13" {
		t.Fatalf("a written review of other pull requests says %v", err)
	}
	if err := CheckWrittenReview(tr, read("---\nstack: 12, 13\nhead: 1111111\n---\n## Notes\nx\n"), rev); err == nil || err.Error() != "the written review read 1111111, but the review shows head 9d01c2aa45bb; review the head the app shows" {
		t.Fatalf("a written review of an older head says %v", err)
	}
}
