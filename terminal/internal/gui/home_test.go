package gui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/views"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// homeOn opens the app with no review, on the home screen, with the server
// giving one of the recorded lists of open pull requests: "" for the whole
// list, or a variant such as "none".
func homeOn(t *testing.T, variant string, width, height int) (App, *[]string, *changes) {
	t.Helper()
	changed := &changes{answers: map[string]answer{}}
	if variant != "" {
		kept, ok := recorded(t)["GET /api/pulls#"+variant]
		if !ok {
			t.Fatalf("no recorded list %q", variant)
		}
		changed.set("GET /api/pulls", kept)
	}
	a, visited := openedOn(t, changed, "", width, height, func(o *Options) { o.Home = "/home/reviewer" })
	return a, visited, changed
}

func selectedPull(a App) string {
	repo, _, m, ok := a.Home.Pull()
	if !ok {
		return ""
	}
	return repo.Name + "#" + utils.Itoa(m.Number)
}

func TestTheHomeScreenGivesEachRepositoryAPanelOfItsPullRequests(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	if a.Screen() != context.ScreenHome || a.Review != nil {
		t.Fatalf("with no review it opens on screen %d", a.Screen())
	}
	var titles []string
	for _, p := range a.Home.Panels {
		titles = append(titles, p.Title)
	}
	if strings.Join(titles, ", ") != "octo/gadgets, octo/widgets, reviewer-home/notes" {
		t.Fatalf("the panels: %s", strings.Join(titles, ", "))
	}
	if got := selectedPull(a); got != "octo/gadgets#7" {
		t.Fatalf("it starts on the first pull request, not %s", got)
	}
	screenIs(t, a, "home")
	for _, size := range [][2]int{{80, 24}, {161, 48}} {
		sized, _, _ := homeOn(t, "", size[0], size[1])
		screenIs(t, press(t, sized, "j", "j"), "home-"+utils.Itoa(size[0]))
	}
}

func TestMovingOnTheHomeScreenRunsThroughEveryRepository(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	steps := []struct{ keys, at string }{
		{"j", "octo/gadgets#9"}, {"j", "octo/widgets#21"}, {"j", "octo/widgets#22"}, {"j", "octo/widgets#30"},
		{"j", "reviewer-home/notes#3"}, {"j", "reviewer-home/notes#4"}, {"j", "reviewer-home/notes#4"},
		{"k", "reviewer-home/notes#3"}, {"k", "octo/widgets#30"}, {"gg", "octo/gadgets#7"}, {"G", "reviewer-home/notes#4"},
		{"2", "octo/widgets#30"}, {"gg", "octo/gadgets#7"}, {"3", "reviewer-home/notes#4"}, {"9", "reviewer-home/notes#4"},
	}
	for _, s := range steps {
		a = press(t, a, strings.Split(s.keys, "")...)
		if got := selectedPull(a); got != s.at {
			t.Fatalf("%s went to %s, not %s", s.keys, got, s.at)
		}
	}
	if a.Home.Focus != 2 {
		t.Fatalf("the panel with the keys follows the pull request: %d", a.Home.Focus)
	}
}

func TestHAndLGoToTheRepositoryAboveOrBelowAndRoundAtTheEnds(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	steps := []struct{ keys, at string }{
		{"h", "reviewer-home/notes#3"}, {"l", "octo/gadgets#7"}, {"l", "octo/widgets#21"}, {"j", "octo/widgets#22"},
		{"l", "reviewer-home/notes#3"}, {"h", "octo/widgets#22"},
	}
	for _, s := range steps {
		a = press(t, a, s.keys)
		if got := selectedPull(a); got != s.at {
			t.Fatalf("%s went to %s, not %s", s.keys, got, s.at)
		}
	}
	if a.Home.Focus != 1 {
		t.Fatalf("the panel with the keys is the repository chosen: %d", a.Home.Focus)
	}
}

func TestTheMainPanelSaysWhatInvolvesYouAndWhatEnterDoes(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	a = press(t, a, "2", "j")
	screenIs(t, a, "home-stack")
	a = press(t, a, "j")
	screenIs(t, a, "home-not-loaded")
	if a.controllers.Hints(a.ContextTree)[0] != [2]string{"Enter", "Load and open"} {
		t.Fatalf("the bar says Enter loads it: %v", a.controllers.Hints(a.ContextTree)[0])
	}
}

func TestEnterOpensTheStacksReviewAndSpaceHComesBack(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	a = press(t, a, "2", "j", "enter")
	if a.Screen() != context.ScreenReview || a.Review == nil || a.Review.ID != "github.com/octo/widgets/21" {
		t.Fatalf("Enter on #22 opens the stack's review: screen %d, %v", a.Screen(), a.Review)
	}
	a = press(t, a, "2", "j")
	left := selectedKey(a, context.FilesPanel)
	a = press(t, a, "space", "h")
	if a.Screen() != context.ScreenHome || selectedPull(a) != "octo/widgets#22" {
		t.Fatalf("Space h goes home, where it was: screen %d at %s", a.Screen(), selectedPull(a))
	}
	a = press(t, a, "k", "enter")
	if a.Screen() != context.ScreenDiff || a.Focus() != context.FilesPanel || selectedKey(a, context.FilesPanel) != left {
		t.Fatalf("Enter on the open review goes back where it was: screen %d, panel %d, %s", a.Screen(), a.Focus(), selectedKey(a, context.FilesPanel))
	}
	a = press(t, a, "space", "h", "space", "v")
	if a.Screen() != context.ScreenReview || a.Focus() != context.MainPanel {
		t.Fatalf("Space v from home is the Review view of the review open: screen %d, panel %d", a.Screen(), a.Focus())
	}
}

func TestEnterLoadsAPullRequestNoReviewHoldsThenOpensIt(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	a = press(t, a, "2", "j", "j")
	next, cmd := a.Update(keyMsg("enter"))
	loading := next.(App)
	if loading.Home.Loading != "octo/widgets#30" || loading.helpers.Home.Status(loading.ContextTree) != "Loading octo/widgets #30 from GitHub…" {
		t.Fatalf("Enter starts loading #30: %q, %q", loading.Home.Loading, loading.helpers.Home.Status(loading.ContextTree))
	}
	screenIs(t, loading, "home-loading")
	again := press(t, loading, "enter")
	if again.Notice != "One load at a time: octo/widgets #30 is loading" {
		t.Fatalf("a second load waits: %q", again.Notice)
	}
	a = drive(t, loading, cmd)
	if a.Screen() != context.ScreenReview || a.Review == nil || a.Review.ID != "github.com/octo/widgets/30" {
		t.Fatalf("once loaded, it opens: screen %d, %v", a.Screen(), a.Review)
	}
	a = press(t, a, "space", "h")
	if _, u, m, _ := a.Home.Pull(); u.Review != "github.com/octo/widgets/30" || m.Review != "github.com/octo/widgets/30" {
		t.Fatalf("the list marks it loaded: %q", u.Review)
	}
}

func TestALoadThatFailsSaysWhyAndStaysHome(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	a = press(t, a, "enter")
	if a.Screen() != context.ScreenHome || a.Home.Loading != "" || !a.Bad ||
		a.Notice != "Could not load octo/gadgets #7: GitHub refused GET repos/octo/gadgets/pulls/7: HTTP 404: Not Found (fake gh has no GET repos/octo/gadgets/pulls/7)" {
		t.Fatalf("a failed load: screen %d, %q", a.Screen(), a.Notice)
	}
}

func TestADraftInTheReviewOpenKeepsAnotherFromOpening(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	a = press(t, a, "2", "enter")
	a.Drafts = map[string]string{"comment x": "Half written."}
	a = press(t, a, "space", "h", "j", "j", "enter")
	if a.Screen() != context.ScreenHome || a.Home.Loading != "" || a.Notice != "Your draft in Stack #21 to #22 is not sent yet: Space v goes back to it" {
		t.Fatalf("a draft keeps the review open: screen %d, %q", a.Screen(), a.Notice)
	}
}

func TestTheHomeScreenSaysWhenThereIsNothingToList(t *testing.T) {
	cases := []struct{ variant, screen, hint string }{
		{"none", "home-none", "No open pull request involves reviewer"},
		{"no settings", "home-unset", "No GitHub accounts are set"},
		{"unreachable", "home-unreachable", "GitHub could not be read as reviewer:"},
		{"one unreachable", "home-one-unreachable", "GitHub could not be read as reviewer-home:"},
		{"one repository", "home-one-repository", "Keep the notes in plain text"},
	}
	for _, c := range cases {
		a, _, _ := homeOn(t, c.variant, 100, 30)
		if !strings.Contains(a.View(), c.hint) {
			t.Fatalf("%s does not say %q", c.variant, c.hint)
		}
		screenIs(t, a, c.screen)
		if c.variant == "one unreachable" {
			screenIs(t, press(t, a, "j"), c.screen+"-list")
		}
	}
	small, _, _ := homeOn(t, "unreachable", 80, 24)
	screenIs(t, small, "home-unreachable-80")
}

func TestRReadsGitHubAgainAndKeepsTheListMeanwhile(t *testing.T) {
	a, _, changed := homeOn(t, "", 100, 30)
	a = press(t, a, "2", "j")
	changed.set("GET /api/pulls", recorded(t)["GET /api/pulls#one unreachable"])
	next, cmd := a.Update(keyMsg("R"))
	reading := next.(App)
	if !reading.Home.Reading || reading.helpers.Home.Status(reading.ContextTree) != "Reading GitHub…" || selectedPull(reading) != "octo/widgets#22" {
		t.Fatalf("R reads again and keeps the list: %v, %q", reading.Home.Reading, reading.helpers.Home.Status(reading.ContextTree))
	}
	a = drive(t, reading, cmd)
	if a.Home.Reading || selectedPull(a) != "octo/widgets#22" || a.Home.Panels[0].Title != "Not read from GitHub" {
		t.Fatalf("the list read again keeps the place: %s, %s", selectedPull(a), a.Home.Panels[0].Title)
	}
	changed.set("GET /api/pulls", answer{Status: 409, Body: []byte(`{"error":"settings.json needs \"accounts\""}`)})
	a = press(t, a, "R")
	if !a.Bad || a.Notice != `Could not read GitHub again: settings.json needs "accounts"` || selectedPull(a) != "octo/widgets#22" {
		t.Fatalf("a read that fails keeps the list and says why: %q", a.Notice)
	}
}

func TestAListThatCannotBeReadSaysWhy(t *testing.T) {
	changed := &changes{answers: map[string]answer{}}
	changed.set("GET /api/pulls", answer{Status: 409, Body: []byte(`{"error":"settings.json gives account 1 no GitHub login"}`)})
	a, _ := openedOn(t, changed, "", 100, 30)
	if a.Home.Failed != "settings.json gives account 1 no GitHub login" {
		t.Fatalf("the failure: %q", a.Home.Failed)
	}
	screenIs(t, a, "home-failed")
}

func TestTheHomeScreensOtherKeys(t *testing.T) {
	a, visited, _ := homeOn(t, "", 100, 30)
	a = press(t, a, "g", "x")
	if len(*visited) != 1 || (*visited)[0] != "https://github.com/octo/gadgets/pull/7" {
		t.Fatalf("gx opens the pull request: %v", *visited)
	}
	a = press(t, a, "q")
	if a.Notice != "ZZ or :q quits" {
		t.Fatalf("q on the home screen: %q", a.Notice)
	}
	a = press(t, a, "space", "v")
	if a.Notice != "No review is open yet: Enter opens one" {
		t.Fatalf("Space v with no review: %q", a.Notice)
	}
	screenIs(t, press(t, a, "space", "?"), "home-keys")
	a = press(t, a, "+")
	if l := a.helpers.Window.HomeLayout(a.ContextTree); l.Panels[0].W != 66 || l.Main.W != 34 || a.Mode != context.HalfScreen {
		t.Fatalf("+ gives the list two thirds: %d and %d", l.Panels[0].W, l.Main.W)
	}
	a = press(t, a, "+")
	if l := a.helpers.Window.HomeLayout(a.ContextTree); l.Panels[0].W != 100 || l.Main.Shown() {
		t.Fatalf("+ again gives the list the screen: %d", l.Panels[0].W)
	}
	screenIs(t, a, "home-full")
}

func TestAClickChoosesAPullRequestAndADoubleClickOpensIt(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	y := a.helpers.Window.HomeLayout(a.ContextTree).Panels[1].Y + 2
	a = send(t, a, tea.MouseMsg{X: 5, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if selectedPull(a) != "octo/widgets#22" {
		t.Fatalf("a click chose %s", selectedPull(a))
	}
	a = send(t, a, tea.MouseMsg{X: 5, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if a.Screen() != context.ScreenReview || a.Review == nil || a.Review.ID != "github.com/octo/widgets/21" {
		t.Fatalf("a double click opens it: screen %d, opening %q, %q", a.Screen(), a.Opening, a.Failed)
	}
}

func TestAFoldedPanelIsOneRowAsWideAsItsPanel(t *testing.T) {
	p := testCommon().Theme
	for _, w := range []int{20, 40, 80} {
		strip := views.Framed(nil, views.Extent{}, w, 1, "[3]─octo/widgets", " ● 1", "1 of 3", false, p)
		if len(strip) != 1 || ansi.StringWidth(strip[0]) != w {
			t.Fatalf("at %d columns the strip is %q", w, strip)
		}
	}
	if got := views.Framed(nil, views.Extent{}, 40, 1, "[3]─octo/widgets", " ● 1", "1 of 3", false, p)[0]; got != "──[3]─octo/widgets ● 1──────────1 of 3──" {
		t.Fatalf("the strip reads %q", got)
	}
}
