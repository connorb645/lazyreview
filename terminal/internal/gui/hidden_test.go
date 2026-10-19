package gui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

func panelTitles(a App) string {
	var titles []string
	for _, p := range a.Home.Panels {
		title := p.Title
		if p.Hidden {
			title += " (hidden)"
		}
		titles = append(titles, title)
	}
	return strings.Join(titles, ", ")
}

func barStatus(a App) string {
	return a.helpers.Home.Status(a.ContextTree)
}

func mainText(a App) string {
	var lines []string
	for _, l := range a.homeMainBox(a.helpers.Window.HomeLayout(a.ContextTree).Main) {
		lines = append(lines, strings.Trim(ansi.Strip(l), "│"))
	}
	return strings.Join(strings.Fields(strings.Join(lines[1:len(lines)-1], " ")), " ")
}

// The config hides octo/gadgets, and every repository of reviewer-home.
func TestTheHomeScreenLeavesOutTheRepositoriesTheConfigHidesAndCountsThem(t *testing.T) {
	a, _, _ := homeOn(t, "hidden", 100, 30)
	if got := panelTitles(a); got != "octo/widgets" {
		t.Fatalf("the panels: %s", got)
	}
	if a.Home.Panels[0].Number != 1 || selectedPull(a) != "octo/widgets#21" {
		t.Fatalf("the one left is the first: %d, %s", a.Home.Panels[0].Number, selectedPull(a))
	}
	if got := barStatus(a); got != "2 repositories hidden · I shows them" {
		t.Fatalf("the bar says %q", got)
	}
	screenIs(t, a, "home-hidden")
	for _, size := range [][2]int{{80, 24}, {161, 48}} {
		sized, _, _ := homeOn(t, "hidden", size[0], size[1])
		screenIs(t, sized, "home-hidden-"+utils.Itoa(size[0]))
	}
}

func TestShowHiddenShowsTheHiddenRepositoriesDimAfterTheOthersThenHidesThemAgain(t *testing.T) {
	a, _, _ := homeOn(t, "hidden", 100, 30)
	a = press(t, a, "I")
	if got := panelTitles(a); got != "octo/widgets, octo/gadgets (hidden), reviewer-home/notes (hidden)" {
		t.Fatalf("the panels: %s", got)
	}
	if got := barStatus(a); got != "Showing 2 hidden · I hides them" {
		t.Fatalf("the bar says %q", got)
	}
	a = press(t, a, "3")
	if !strings.Contains(mainText(a), "Hidden from the home screen by reviewer-home/* in the config. i shows it again.") {
		t.Fatalf("the main panel does not say what hides it: %s", mainText(a))
	}
	screenIs(t, a, "home-showing-hidden")
	for _, size := range [][2]int{{80, 24}, {161, 48}} {
		sized, _, _ := homeOn(t, "hidden", size[0], size[1])
		screenIs(t, press(t, sized, "I", "2"), "home-showing-hidden-"+utils.Itoa(size[0]))
	}
	a = press(t, a, "I")
	if got := panelTitles(a); got != "octo/widgets" || a.Home.Focus != 0 {
		t.Fatalf("I again hides them: %s, focus %d", got, a.Home.Focus)
	}
}

func TestShowHiddenSaysSoWhenNoneIsHidden(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	a = press(t, a, "I")
	if a.Home.ShowHidden || a.Notice != "No repository here is hidden" || barStatus(a) != "" {
		t.Fatalf("with none hidden: %v, %q, %q", a.Home.ShowHidden, a.Notice, barStatus(a))
	}
}

func TestHideOffersTheRepositoryOrEveryRepositoryOfItsOwnerAndShowHiddenThenShowsItAgain(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	a = press(t, a, "2", "i")
	if !a.Has(context.MenuKey) || a.Menu.Title != "Hide from the home screen" || len(a.Menu.Rows) != 2 ||
		a.Menu.Rows[0].Left != "octo/widgets" || a.Menu.Rows[1].Left != "octo/*" || a.Menu.Rows[1].Right != "Hide every repository of octo" {
		t.Fatalf("the menu: %+v", a.Menu.Menu)
	}
	screenIs(t, a, "home-hide-menu")
	a = press(t, a, "enter")
	if got := panelTitles(a); got != "octo/gadgets, reviewer-home/notes" {
		t.Fatalf("the panels: %s", got)
	}
	if a.Bad || a.Notice != "octo/widgets is hidden; I shows it" || a.Home.Focus != 1 {
		t.Fatalf("after hiding: %q, focus %d", a.Notice, a.Home.Focus)
	}
	a = press(t, a, "I", "3")
	if got := panelTitles(a); got != "octo/gadgets, reviewer-home/notes, octo/widgets (hidden)" || selectedPull(a) != "octo/widgets#21" {
		t.Fatalf("shown again, it is last: %s, at %s", got, selectedPull(a))
	}
	a = press(t, a, "i")
	if a.Has(context.MenuKey) || a.Notice != "octo/widgets shows again" {
		t.Fatalf("i on a hidden repository shows it again at once: menu %v, %q", a.Has(context.MenuKey), a.Notice)
	}
	if got := panelTitles(a); got != "octo/gadgets, octo/widgets, reviewer-home/notes" || a.Home.ShowHidden || selectedPull(a) != "octo/widgets#21" {
		t.Fatalf("with none hidden, the hidden ones no longer show: %s, %v, at %s", got, a.Home.ShowHidden, selectedPull(a))
	}
}

func TestAnOwnersEntryHidesEveryRepositoryOfItAndTheMenuSaysSoBeforeTakingItOff(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	a = press(t, a, "1", "i", "j", "enter")
	if got := panelTitles(a); got != "reviewer-home/notes" {
		t.Fatalf("the panels: %s", got)
	}
	if a.Notice != "Every repository of octo is hidden; I shows them" || strings.Contains(a.barLine(), "hidden ·") {
		t.Fatalf("after hiding, the bar says %q", ansi.Strip(a.barLine()))
	}
	a = press(t, a, "j")
	if !strings.HasSuffix(ansi.Strip(a.barLine()), " 2 repositories hidden · I shows them ") {
		t.Fatalf("once the notice goes, the bar counts them: %q", ansi.Strip(a.barLine()))
	}
	a = press(t, a, "I", "3", "i")
	if !a.Has(context.MenuKey) || a.Menu.Title != "Hidden by octo/*" || len(a.Menu.Rows) != 1 ||
		a.Menu.Rows[0].Right != "Take it off the list: every repository of octo shows again" {
		t.Fatalf("the menu: %+v", a.Menu.Menu)
	}
	screenIs(t, a, "home-hidden-by-owner")
	a = press(t, a, "enter")
	if got := panelTitles(a); got != "octo/gadgets, octo/widgets, reviewer-home/notes" || a.Notice != "Every repository of octo shows again" {
		t.Fatalf("after taking it off: %s, %q", got, a.Notice)
	}
}

func TestTakingAnOwnersEntryOffLeavesTheOthers(t *testing.T) {
	a, _, _ := homeOn(t, "hidden", 100, 30)
	a = press(t, a, "I", "3", "i", "enter")
	if got := panelTitles(a); got != "octo/widgets, reviewer-home/notes, octo/gadgets (hidden)" || a.Notice != "Every repository of reviewer-home shows again" {
		t.Fatalf("after taking reviewer-home/* off: %s, %q", got, a.Notice)
	}
}

func TestAChangeTheFileRefusesSaysWhyAndChangesNothing(t *testing.T) {
	a, _, _ := homeOn(t, "", 100, 30)
	a = press(t, a, "3", "i", "enter")
	if got := panelTitles(a); got != "octo/gadgets, octo/widgets, reviewer-home/notes" {
		t.Fatalf("the panels: %s", got)
	}
	if !a.Bad || a.Notice != "Could not change the hidden repositories: /home/reviewer/Library/Application Support/lazyreview/config.yml was not changed: github is written in a way lazyreview cannot change line by line; change it by hand, as docs/Config.md shows" {
		t.Fatalf("the notice: %q", a.Notice)
	}
}

func TestWithEveryRepositoryHiddenTheHomeScreenSaysSo(t *testing.T) {
	a, _, _ := homeOn(t, "all hidden", 100, 30)
	if got := mainText(a); got != "The config hides every repository with open pull requests that involve reviewer and reviewer-home. I shows them. R reads GitHub again." {
		t.Fatalf("the main panel says: %s", got)
	}
	screenIs(t, a, "home-all-hidden")
	if got := panelTitles(press(t, a, "I")); got != "octo/gadgets (hidden), octo/widgets (hidden), reviewer-home/notes (hidden)" {
		t.Fatalf("I shows them: %s", got)
	}
}

func TestTheHideKeysCanBeChangedOrDisabled(t *testing.T) {
	cmn := handedOver(t, map[string]map[string][]string{"home": {"hideRepository": {"x"}, "showHidden": {}}})
	a, _ := openedIn(t, cmn, &changes{answers: map[string]answer{"GET /api/pulls": recorded(t)["GET /api/pulls#hidden"]}}, "", 100, 30)
	if a = press(t, a, "i"); a.Has(context.MenuKey) {
		t.Fatal("i still hides")
	}
	if a = press(t, a, "x"); !a.Has(context.MenuKey) || a.Menu.Title != "Hide from the home screen" {
		t.Fatalf("x hides: %+v", a.Menu.Menu)
	}
	if a = press(t, a, "esc", "I"); a.Home.ShowHidden {
		t.Fatal("I still shows the hidden ones")
	}
	if got := barStatus(a); got != "2 repositories hidden" {
		t.Fatalf("with no key to show them, the bar names none: %q", got)
	}
}

// Two repositories can each have a #9: reading the list again keeps the
// keys in the repository that had them.
func TestReadingAgainKeepsTheKeysInTheirRepositoryWhenAnotherHasTheSameNumber(t *testing.T) {
	kept := recorded(t)["GET /api/pulls"]
	body := regexp.MustCompile(`"number":\s*4,`).ReplaceAllString(string(kept.Body), `"number": 9,`)
	if body == string(kept.Body) {
		t.Fatal("the recorded list has no #4 to number again")
	}
	changed := &changes{answers: map[string]answer{"GET /api/pulls": {Status: 200, Body: []byte(body)}}}
	a, _ := openedOn(t, changed, "", 100, 30)
	a = press(t, a, "1", "j")
	if selectedPull(a) != "octo/gadgets#9" {
		t.Fatalf("at %s", selectedPull(a))
	}
	if a = press(t, a, "R"); selectedPull(a) != "octo/gadgets#9" || a.Home.Focus != 0 {
		t.Fatalf("read again, the keys went to %s, panel %d", selectedPull(a), a.Home.Focus)
	}
}
