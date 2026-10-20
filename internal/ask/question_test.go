package ask

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/internal/i18n"
	"github.com/connorb645/lazyreview/internal/models"
)

func longFile() []string {
	out := make([]string, 900)
	for i := range out {
		out[i] = "line " + strconv.Itoa(i+1)
	}
	return out
}

func standInPage() models.PageRecord {
	return models.PageRecord{
		Model: models.Model{Files: []models.FileEntry{{Path: "src/a.ts", Status: "M", OldPath: "src/a.ts"}, {Path: "src/new.ts", Status: "A"}}},
		Diff: map[string]string{
			"src/a.ts":   "diff --git a/src/a.ts b/src/a.ts\n@@ -10,2 +10,3 @@ top\n line 10\n+added 11\n line 12\n@@ -500,2 +501,2 @@\n-old 500\n+line 501\n line 502\n",
			"src/new.ts": "diff --git a/src/new.ts b/src/new.ts\nnew file mode 100644\n@@ -0,0 +1,2 @@\n+export const a = 1;\n+export const b = 2;\n",
		},
		Sources: map[string][]string{"src/a.ts": longFile()},
		Context: map[string][]string{"src/other.ts": {"export class Other {}"}},
	}
}

var single = models.Review{Base: models.Ref{Ref: "main", Sha: strings.Repeat("f", 40)},
	Members: []models.ReviewMember{{Pull: 12, Head: strings.Repeat("e", 40), Title: "Tidy the parser", Description: "Makes the parser shorter."}}}

func asked(path, side string, line, startLine int, messages ...models.Message) models.Conversation {
	if len(messages) == 0 {
		messages = []models.Message{{From: "you", Text: "Q?"}}
	}
	return models.Conversation{Path: path, Side: side, Line: line, StartLine: startLine, Messages: messages}
}

func matches(t *testing.T, text, pattern, why string) {
	t.Helper()
	if !regexp.MustCompile(pattern).MatchString(text) {
		t.Errorf("%s: %q does not match %s", why, text, pattern)
	}
}

func TestAQuestionCarriesThePullRequestTheCodebaseTheHunkTheCodeNearItAndTheConversation(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	c := asked("src/a.ts", "RIGHT", 11, 0, models.Message{From: "you", Text: "Why add this?"}, models.Message{From: "claude", Text: "To log."},
		models.Message{From: "claude", Error: "timed out"}, models.Message{From: "you", Text: "Is it tested?"})
	q := QuestionFor(tr, single, standInPage(), c, "/code/c0ffee")
	if q.System != tr.AskSystem || !strings.Contains(q.System, "throwaway copy of the codebase") || !strings.Contains(q.System, "is refused, as nobody is there to approve it") {
		t.Errorf("the instructions are %q", q.System)
	}
	p := q.Prompt
	matches(t, p, `^Pull request #12: Tidy the parser\n\nIts description:\nMakes the parser shorter\.\n\nThe codebase as it stands after this change is in /code/c0ffee, your working folder, with the branch pull/12 checked out and the branch main at the base of the change\.\n\n`, "the start")
	matches(t, p, `The files this change touches:\nsrc/a\.ts \(changed\)\nsrc/new\.ts \(new\)\n\n`, "the files")
	matches(t, p, `File: src/a\.ts \(changed\), at /code/c0ffee/src/a\.ts\n\nThey ask about line 11\.\n\nThe diff of this file`, "the file")
	matches(t, p, "```diff\n@@ -10,2 \\+10,3 @@ top\n line 10\n\\+added 11\n line 12\n```", "the hunk")
	if strings.Contains(p, "old 500") || strings.Contains(p, "line 200") {
		t.Error("more than the hunk and the lines around the question went")
	}
	matches(t, p, "The file at the head, lines 1-91 of 900:\n```\n 1 \\| line 1\n", "the head")
	matches(t, p, `The conversation so far:\nThey asked: Why add this\?\n\nYou answered: To log\.\n\nTheir question:\nIs it tested\?$`, "the conversation")
}

func TestANewFileSendsItsWholeTextARemovedLineNoHeadTextAndAnUnchangedFileItsContext(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	ask := func(c models.Conversation) string { return QuestionFor(tr, single, standInPage(), c, "/code").Prompt }
	matches(t, ask(asked("src/new.ts", "RIGHT", 2, 1)), "They ask about lines 1-2\\.[\\s\\S]*The file at the head:\n```\n1 \\| export const a = 1;\n2 \\| export const b = 2;\n```", "a new file")
	removed := ask(asked("src/a.ts", "LEFT", 500, 0))
	matches(t, removed, `removed line 500`, "a removed line")
	matches(t, removed, `-old 500`, "a removed line's hunk")
	if strings.Contains(removed, "The file at the head") {
		t.Error("a removed line sent head text")
	}
	matches(t, ask(asked("src/other.ts", "RIGHT", 1, 0)), `File: src/other\.ts \(unchanged by this change\), at /code/src/other\.ts[\s\S]*1 \| export class Other \{\}`, "an unchanged file")
	across := asked("src/a.ts", "RIGHT", 11, 11)
	across.StartSide = "LEFT"
	matches(t, ask(across), `They ask about removed line 11 to line 11\.`, "a range across sides")
	matches(t, ask(across), `\+added 11`, "a range across sides' hunk")
}

func TestOnAStackTheCodeReadIsTheHeadOfThePullRequestThatChangedTheLines(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	low, top := strings.Repeat("a", 40), strings.Repeat("b", 40)
	stack := models.Review{Base: models.Ref{Ref: "main", Sha: strings.Repeat("c", 40)}, Members: []models.ReviewMember{{Pull: 21, Head: low, Title: "Add search"}, {Pull: 22, Head: top, Title: "Add basket"}}}
	stacked := standInPage()
	stacked.Stack = &models.StackRecord{Lines: map[string]map[string]models.LineMap{"src/a.ts": {
		"RIGHT": {"11": {Owner: &models.LinePlace{Pull: 21, Line: 9}}, "12": {Owner: &models.LinePlace{Pull: 21, Line: 10, Path: "src/old-a.ts"}},
			"13": {Candidates: []models.LinePlace{{Pull: 21, Line: 11}, {Pull: 22, Line: 13}}}},
		"LEFT": {"500": {Owner: &models.LinePlace{Pull: 22, Line: 500}}},
	}}}
	head := func(rev models.Review, page models.PageRecord, c models.Conversation) Head {
		return HeadAsked(tr, rev, page, c)
	}
	cases := []struct {
		got, want Head
		why       string
	}{
		{head(stack, stacked, asked("src/a.ts", "RIGHT", 11, 0)), Head{Commit: low, Pull: 21, Path: "src/a.ts", Lines: &[2]int{9, 9}}, "a line #21 changed"},
		{head(stack, stacked, asked("src/a.ts", "RIGHT", 12, 11)), Head{Commit: low, Pull: 21, Path: "src/old-a.ts", Lines: &[2]int{10, 10}}, "a range that moves file keeps its last line"},
		{head(stack, stacked, asked("src/a.ts", "RIGHT", 13, 0)), Head{Commit: top, Pull: 22, Path: "src/a.ts", Lines: &[2]int{13, 13}}, "a line no one pull request changed is read at the top"},
		{head(stack, stacked, asked("src/a.ts", "LEFT", 500, 0)), Head{Commit: top, Pull: 22, Path: "src/a.ts"}, "a removed line is at no head"},
		{head(models.Review{Base: stack.Base, Members: []models.ReviewMember{{Pull: 12, Head: low}}}, standInPage(), asked("src/a.ts", "RIGHT", 12, 11)), Head{Commit: low, Path: "src/a.ts", Lines: &[2]int{11, 12}}, "a single pull request"},
		{head(models.Review{Kind: "branch", Head: models.Ref{Sha: top}}, standInPage(), asked("src/a.ts", "RIGHT", 12, 11)), Head{Commit: top, Path: "src/a.ts", Lines: &[2]int{11, 12}}, "a branch"},
	}
	for _, c := range cases {
		if !reflect.DeepEqual(c.got, c.want) {
			t.Errorf("%s: %+v, not %+v", c.why, c.got, c.want)
		}
	}
	p := QuestionFor(tr, stack, stacked, asked("src/a.ts", "RIGHT", 11, 0), "/code").Prompt
	matches(t, p, `The codebase as it stands after #21 is in /code, your working folder, with the branch pull/21 checked out and the branch main at the base of the change\.`, "the codebase")
	if got := CopyBranches(tr, stack, stacked, asked("src/a.ts", "RIGHT", 11, 0)); !reflect.DeepEqual(got, CopyPlan{Branches: map[string]string{"main": strings.Repeat("c", 40), "pull/21": low, "pull/22": top}, Branch: "pull/21", Base: "main"}) {
		t.Errorf("the stack's copy holds %+v", got)
	}
	branch := models.Review{Kind: "branch", Base: models.Ref{Ref: "main", Sha: low}, Head: models.Ref{Ref: "fix/x", Sha: top}}
	if got := CopyBranches(tr, branch, standInPage(), asked("src/a.ts", "RIGHT", 11, 0)); !reflect.DeepEqual(got, CopyPlan{Branches: map[string]string{"main": low, "fix/x": top}, Branch: "fix/x", Base: "main"}) {
		t.Errorf("the branch's copy holds %+v", got)
	}
	matches(t, p, `File: src/a\.ts \(changed\), at /code/src/a\.ts\n\nThey ask about line 11, which #21 changed\.\n\nIn the codebase, after #21, they are line 9\.`, "where the lines are")
	matches(t, p, `The file at the head of #22, lines 1-91 of 900:`, "the head of the top")
	atTop := QuestionFor(tr, stack, stacked, asked("src/a.ts", "RIGHT", 13, 0), "/code").Prompt
	matches(t, atTop, `with the branch pull/22 checked out`, "a line at the top")
	if strings.Contains(atTop, "In the codebase, after") {
		t.Error("lines at the top were placed elsewhere")
	}
	matches(t, atTop, `The file at the head, lines`, "a line at the top's head")
}
