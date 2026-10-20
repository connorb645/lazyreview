package reviews

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/commands/github"
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/models"
	"github.com/connorb645/lazyreview/terminal/internal/queue"
	"github.com/connorb645/lazyreview/terminal/internal/store"
)

const olderDiff = "diff --git a/a.ts b/a.ts\n--- a/a.ts\n+++ b/a.ts\n@@ -1,3 +1,4 @@\n one\n+two\n three\n four\n"

// olderFiles are reviews in the shapes earlier versions wrote, by path under
// the data home's reviews folder.
var olderFiles = map[string]string{
	// A pull request's review from before ranges had a side, before
	// conversations kept a session, and before the threads on GitHub had a
	// file of their own: they sat in the page, which then had no
	// properties on its types and no links on its arrows.
	"github.com/octo/old/5/review.json": `{
  "id": "github.com/octo/old/5", "host": "github.com", "repo": "octo/old", "account": null, "me": "reviewer", "title": "Old",
  "base": {"ref": "main", "sha": "aaaa"}, "context": true, "loaded_at": "2026-10-15T20:10:00.000Z", "notes": [], "moved": null,
  "members": [{"pull": 5, "head": "bbbbbbbbbb", "start": "aaaa", "title": "Old", "url": "https://github.com/octo/old/pull/5", "author": "sam",
    "base": "main", "description": "", "state": "open", "ended_at": null, "color": 1,
    "comments": [
      {"id": 1, "path": "a.ts", "line": 3, "side": "RIGHT", "body": "A range from before sides.", "code": "", "start_line": 2},
      {"id": 2, "path": "a.ts", "line": 2, "side": "RIGHT", "body": "One line.", "code": "two", "shown": {"path": "a.ts", "line": 2, "side": "RIGHT", "start_line": 1}}
    ],
    "posted": [{"set": "0123456789ab", "action": "comment", "review": "https://github.com/octo/old/pull/5#r1", "at": "2026-10-15T21:00:00.000Z",
      "comments": [{"id": 3, "path": "a.ts", "line": 3, "side": "RIGHT", "body": "Sent before sides.", "code": "", "start_line": 2}]}]}],
  "earlier": []
}
`,
	"github.com/octo/old/5/page.json": `{"model": {"heading": {"title": "Old", "context": [["Pull request ", false], ["#5", true]], "head": "bbbbbbbbbb", "base": "aaaa"},
  "components": [{"id": "root", "title": "old", "packages": [{"id": "top", "title": "old", "path": ""}]}], "areas": [],
  "files": [{"path": "a.ts", "oldPath": "a.ts", "status": "M", "adds": 1, "dels": 0, "binary": false, "collapsed": false, "component": "root", "package": "top"}],
  "contextFiles": [], "nodes": [{"id": "file:a.ts", "name": "a.ts", "kind": "artifact", "file": "a.ts", "line": 1, "component": "root", "package": "top", "state": "changed"}],
  "edges": [], "groups": [], "summary": [{"source": "root", "target": "root", "count": 1}]},
 "layout": {"components": {"root": {"x": 0, "y": 0, "w": 400, "h": 200}}, "boxes": {"file:a.ts": {"x": 10, "y": 10, "w": 100, "h": 60}},
  "pkgArrows": [{"comp": "root", "from": "top", "to": "top", "count": 1, "words": "", "countText": "1 link"}],
  "summaries": [{"source": "root", "target": "root", "both": false, "lines": ["→ 1 link"]}]},
 "diff": {"a.ts": ` + "\"" + strings.ReplaceAll(olderDiff, "\n", `\n`) + "\"" + `},
 "sources": {"a.ts": ["one", "two", "three", "four"]}, "context": {}, "stack": null,
 "threads": [{"id": "T1", "pull": 5, "path": "a.ts", "side": "RIGHT", "line": 2, "comments": []}]}`,
	"github.com/octo/old/5/threads/1.json": `{"id": 1, "path": "a.ts", "side": "RIGHT", "line": 2, "start_line": 1, "created_at": "2026-10-15T20:20:00.000Z", "code": "one\ntwo",
  "messages": [{"from": "you", "text": "Why two?", "at": "2026-10-15T20:20:00.000Z"}, {"from": "claude", "text": "To count.", "at": "2026-10-15T20:20:09.000Z"}]}`,
	"github.com/octo/old/5/threads/2.json": `{"id": 2, "path": "a.ts", "side": "RIGHT", "line": 3, "created_at": "2026-10-15T20:30:00.000Z",
  "messages": [{"from": "you", "text": "Left waiting by a server that stopped.", "at": "2026-10-15T20:30:00.000Z"}]}`,
	// A conversation left with Claude at work by a server that stopped,
	// after an answer that failed.
	"github.com/octo/old/5/threads/3.json": `{"id": 3, "path": "a.ts", "side": "RIGHT", "line": 4, "start_line": 3, "start_side": "RIGHT", "created_at": "2026-10-19T12:30:00.000Z", "session": "7a1d0c3e-0000-4000-8000-000000000001",
  "messages": [{"from": "you", "text": "And four?", "at": "2026-10-19T12:30:00.000Z"}, {"from": "claude", "error": "you stopped it", "steps": [{"words": "Read a.ts"}, {"words": "Ran a command", "refused": "it reaches outside the copy"}], "at": "2026-10-19T12:30:20.000Z"},
    {"from": "you", "text": "Again?", "at": "2026-10-19T12:31:00.000Z"}],
  "working": {"steps": [{"words": "Read a.ts"}]}}`,
	"github.com/octo/old/5/written.md": "---\npull: 5\nhead: bbbbbbb\n---\n## Notes\nSee `a.ts:2`.\n",
	// A review moved into a stack: it keeps its old fields, and its
	// comments went.
	"github.com/octo/old/6/review.json": `{"id": "github.com/octo/old/6", "host": "github.com", "repo": "octo/old", "account": "reviewer", "me": "reviewer", "title": "Moved",
  "base": {"ref": "main", "sha": "aaaa"}, "context": false, "loaded_at": "2026-10-16T10:00:00.000Z", "notes": [],
  "moved": {"to": "github.com/octo/old/5", "stack": [5, 6], "at": "2026-10-16T11:00:00.000Z"},
  "members": [{"pull": 6, "head": "cccccccccc", "start": "bbbb", "title": "Moved", "url": "https://github.com/octo/old/pull/6", "author": "sam", "base": "old",
    "description": "", "state": "open", "ended_at": null, "color": 2, "comments": [], "posted": []}], "earlier": []}`,
	"github.com/octo/old/6/page.json": `{"model": {"heading": {"title": "Moved", "context": [], "head": "cccc", "base": "bbbb"}, "components": [], "areas": [], "files": [], "contextFiles": [], "nodes": [], "edges": [], "groups": []},
 "layout": {"components": {}, "boxes": {}, "pkgArrows": [], "summaries": []}, "diff": {}, "sources": {}, "context": {}, "stack": null}`,
	// A branch's review, which has no pull requests and no queue.
	"local/old-0a1b2c3d/feature~x/review.json": `{"id": "local/old-0a1b2c3d/feature~x", "kind": "branch", "repo": "old", "path": "/work/old", "title": "feature/x", "given_title": null,
  "base": {"ref": "main", "sha": "aaaa"}, "head": {"ref": "feature/x", "sha": "dddd"}, "start": "aaaa", "context": false, "loaded_at": "2026-10-16T10:22:00.000Z", "notes": []}`,
	"local/old-0a1b2c3d/feature~x/page.json": `{"model": {"heading": {"title": "feature/x", "context": [], "head": "dddd", "base": "aaaa"}, "components": [], "areas": [],
  "files": [{"path": "a.ts", "oldPath": "a.ts", "status": "M", "adds": 1, "dels": 0, "binary": false, "collapsed": false, "component": "root", "package": "top"}],
  "contextFiles": [], "nodes": [], "edges": [], "groups": []}, "layout": {"components": {}, "boxes": {}, "pkgArrows": [], "summaries": []},
 "diff": {"a.ts": ` + "\"" + strings.ReplaceAll(olderDiff, "\n", `\n`) + "\"" + `}, "sources": {}, "context": {}, "stack": null}`,
}

func olderHome(t *testing.T) (*Reviews, string) {
	t.Helper()
	root := t.TempDir()
	for path, text := range olderFiles {
		full := filepath.Join(root, "reviews", filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tr := i18n.EnglishTranslationSet()
	reviews := store.NewReviewStore(tr, store.DataHome{Root: root}, store.NewLocks(tr))
	q := queue.NewService(tr, reviews, func(host, account string) *github.GitHub { return nil }, func() string { return "now" })
	return New(tr, reviews, q, nil, nil, nil, nil), root
}

func TestReviewsInTheShapesEarlierVersionsWroteStillOpen(t *testing.T) {
	r, _ := olderHome(t)
	state, err := r.State("github.com/octo/old/5")
	if err != nil {
		t.Fatal(err)
	}
	comments := state.Queue.Members[0].Comments
	if len(comments) != 2 || comments[0].StartLine != 2 || comments[0].StartSide != "" || comments[0].Version == "" {
		t.Errorf("the range queued before sides reads %+v", comments)
	}
	if sets := state.Queue.Members[0].Sets; len(sets) != 3 || state.Queue.Members[0].Posted[0].Action != "comment" {
		t.Errorf("the old queue offers %v and holds %+v", sets, state.Queue.Members[0].Posted)
	}
	if len(state.Threads) != 3 || state.Threads[0].Messages[1].Text != "To count." || state.Threads[1].Working != nil {
		t.Errorf("the conversations read %+v", state.Threads)
	}
	if left := state.Threads[2]; left.Working == nil || len(left.Messages) != 3 || left.Messages[1].Error != "you stopped it" || left.Messages[1].Steps[1].Refused != "it reaches outside the copy" {
		t.Errorf("the conversation left at work reads %+v", left)
	}
	posted := state.Queue.Members[0].Posted
	if len(posted) != 1 || posted[0].Count != 1 || state.Queue.Members[0].Comments[1].Shown == nil || state.Queue.Members[0].Comments[1].Shown.StartSide != "" {
		t.Errorf("the comment shown on a stack and the review sent before sides read %+v, %+v", state.Queue.Members[0].Comments[1], posted)
	}
	if len(state.GitHub) != 0 || state.Written == nil || state.Written.Stale || len(state.Written.Sections[0].Links) != 1 {
		t.Errorf("the threads on GitHub read %+v and the written review %+v", state.GitHub, state.Written)
	}
	page, err := r.Page("github.com/octo/old/5")
	if err != nil || len(page.Layout.Summaries) != 1 || page.Layout.Summaries[0].Ways != nil || page.Model.Nodes[0].Properties != nil {
		t.Errorf("the old page reads %+v, %v", page.Layout, err)
	}
	file, err := r.File("github.com/octo/old/5", "a.ts", page.Build)
	if err != nil || len(file.Hunks) != 1 || len(file.Whole) != 4 {
		t.Errorf("the old page's file reads %+v, %v", file, err)
	}

	moved, err := r.State("github.com/octo/old/6")
	if err != nil || moved.Moved == nil || moved.Moved.To != "github.com/octo/old/5" || len(moved.Queue.Members[0].Sets) != 0 {
		t.Errorf("the moved review reads %+v, %v", moved.Moved, err)
	}
	branch, err := r.State("local/old-0a1b2c3d/feature~x")
	if err != nil || branch.Kind != "branch" || branch.Queue != nil || branch.Head.Ref != "feature/x" {
		t.Errorf("the branch's review reads %+v, %v", branch, err)
	}
	where, err := r.Where("local/old-0a1b2c3d/feature~x", models.Lines{Path: "a.ts", Side: "RIGHT", Line: 2}, 0, page.Build)
	if err != nil || !strings.HasPrefix(where.Refused, "The review was loaded again") {
		t.Errorf("a comment on the branch, from another build, gives %+v, %v", where, err)
	}
}

func TestAChangeToAnOlderReviewKeepsEveryOtherCommentInTheShapeItHad(t *testing.T) {
	r, root := olderHome(t)
	state, _ := r.State("github.com/octo/old/5")
	page, _ := r.Page("github.com/octo/old/5")
	if _, err := r.Edit("github.com/octo/old/5", 2, "One line, reworded.", state.Queue.Members[0].Comments[1].Version, page.Build); err != nil {
		t.Fatal(err)
	}
	text, _ := os.ReadFile(filepath.Join(root, "reviews", "github.com", "octo", "old", "5", "review.json"))
	if strings.Contains(string(text), `"start_side"`) || !strings.Contains(string(text), `"start_line": 2`) || !strings.Contains(string(text), "One line, reworded.") ||
		!strings.Contains(string(text), `"account": null`) {
		t.Errorf("the review was written as:\n%s", text)
	}
	after, _ := r.State("github.com/octo/old/5")
	if after.Queue.Members[0].Comments[0].Version != state.Queue.Members[0].Comments[0].Version {
		t.Errorf("writing the review changed the version of a comment it did not change")
	}
	var written struct {
		Members []struct {
			Comments []map[string]any `json:"comments"`
			Posted   []struct {
				Comments []map[string]any `json:"comments"`
			} `json:"posted"`
		} `json:"members"`
	}
	if err := json.Unmarshal(text, &written); err != nil {
		t.Fatal(err)
	}
	shown, _ := written.Members[0].Comments[1]["shown"].(map[string]any)
	sent := written.Members[0].Posted[0].Comments[0]
	if len(shown) != 4 || shown["start_line"] != 1.0 || sent["start_line"] != 2.0 || sent["start_side"] != nil {
		t.Errorf("the comment shown on a stack reads %v and the comment sent before sides %v", shown, sent)
	}
}
