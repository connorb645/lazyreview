package gui

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/controllers"
	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// The keys each place takes, and what each does, from the bindings of the
// context with the keys in each: the home screen, the four panels, the three
// main views and the popups. A change to any key in any place shows. Each
// popup is also pressed with every key, to see it does what its bindings
// say.
func TestTheBindingsOfEachPlace(t *testing.T) {
	places := map[string]map[string]string{}
	home, _, _ := homeOn(t, "", 100, 30)
	places["home"] = placeBindings(home)
	a, _ := opened(t, stack, 100, 30)
	for n, name := range map[int]string{context.ReviewPanel: "review panel", context.FilesPanel: "files panel", context.NotesPanel: "notes panel", context.DiagramPanel: "diagram panel"} {
		places[name] = placeBindings(press(t, a, utils.Itoa(n)))
	}
	places["review view"] = placeBindings(press(t, a, "0"))
	places["diff"] = placeBindings(onBasket(t))
	places["diagram"] = placeBindings(press(t, a, "space", "m"))
	popups := map[string]context.ContextKey{
		"menu": context.MenuKey, "finder": context.FinderKey, "posting": context.PostingKey,
		"posting confirmation": context.PostingConfirmationKey, "confirmation": context.ConfirmationKey,
	}
	for name, key := range popups {
		places[name] = controllers.Sequences(a.controllers.Bindings(a.controllers.For(key)))
	}
	got, err := json.MarshalIndent(places, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", "bindings.json")
	if *update {
		if err := os.WriteFile(path, append(got, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no bindings yet; write them with go test ./... -update: %v", err)
	}
	if string(want) != string(got)+"\n" {
		t.Errorf("the bindings differ.\n--- kept\n%s\n--- now\n%s", want, got)
	}
	keys := probeKeys(t, places)
	for name, probe := range map[string]func(*testing.T, []string) map[string]string{
		"menu": menuKeys, "finder": finderKeys, "posting": postingKeys, "posting confirmation": postingConfirmationKeys, "confirmation": confirmationKeys,
	} {
		if pressed := probe(t, keys); !maps.Equal(pressed, places[name]) {
			t.Errorf("the %s's keys do %v, but its bindings say %v", name, pressed, places[name])
		}
	}
}

// A key the hand-over changes is the binding, and the help, the bar and a
// notice name it; an action the hand-over gives no key is unbound, and its
// help row and its hint are left out.
func TestTheHandOversKeysAreTheBindings(t *testing.T) {
	cmn := handedOver(t, map[string]map[string][]string{
		"universal":   {"quit": {"Q Q"}},
		"reviewPanel": {"highlight": {"V"}, "post": {}},
	})
	tr := cmn.Tr
	a, _ := openedIn(t, cmn, &changes{answers: map[string]answer{}}, stack, 100, 30)
	bound := placeBindings(a)
	if bound["V"] != "highlight" || bound["Q Q"] != "quit" || bound["v"] != "" || bound["Z Z"] != "" || bound["P"] != "" {
		t.Fatalf("the Review panel binds %v", bound)
	}
	filterIs(t, press(t, a, "V"), "[21]")
	if b := press(t, a, "v"); len(b.Filter) != 0 {
		t.Fatalf("v still highlights: %v", b.Filter)
	}
	if b := press(t, a, "P"); b.Has(context.PostingKey) || b.Notice != "" {
		t.Fatalf("P still posts: %q", b.Notice)
	}
	help := map[string]string{}
	for _, r := range press(t, a, "space", "?").Menu.Rows {
		help[r.Right] = r.Left
	}
	highlight := utils.ResolvePlaceholderString(tr.HelpHighlight, map[string]string{"move": "j k"})
	quit := utils.ResolvePlaceholderString(tr.HelpQuit, map[string]string{"command": ":"})
	if help[highlight] != "V" || help[quit] != "QQ" {
		t.Fatalf("the help labels the highlight %q and quitting %q", help[highlight], help[quit])
	}
	if keys, ok := help[tr.HelpPostPull]; ok {
		t.Fatalf("the help lists posting, with no key, as %q", keys)
	}
	if bar := a.View(); !strings.Contains(bar, tr.Highlight+": V") || strings.Contains(bar, tr.Post+":") {
		t.Fatalf("the bar reads:\n%s", bar)
	}
	if b := press(t, a, "q"); b.Notice != "QQ or :q quits" {
		t.Fatalf("q tells %q", b.Notice)
	}
}

func placeBindings(a App) map[string]string {
	return controllers.Sequences(a.controllers.Bindings(a.controllers.Place(a.ContextTree)))
}

// probeKeys are the keys tried on each popup: every key any place binds, and
// a few that none does.
func probeKeys(t *testing.T, places map[string]map[string]string) []string {
	t.Helper()
	seen := map[string]bool{"x": true, "y": true, "n": true, "backspace": true, "left": true, "right": true, "ctrl+n": true, "ctrl+p": true}
	for _, b := range places {
		for sequence := range b {
			seen[firstKey(sequence)] = true
		}
	}
	var names []string
	for name := range seen {
		if keyName(keyFor(name)) != name {
			t.Fatalf("the key %q is named %q", name, keyName(keyFor(name)))
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func firstKey(sequence string) string {
	for i, r := range sequence {
		if r == ' ' && i > 0 {
			return sequence[:i]
		}
	}
	return sequence
}

func keyFor(name string) tea.KeyMsg {
	named := map[string]tea.KeyType{
		"up": tea.KeyUp, "down": tea.KeyDown, "left": tea.KeyLeft, "right": tea.KeyRight, "backspace": tea.KeyBackspace,
		"ctrl+b": tea.KeyCtrlB, "ctrl+c": tea.KeyCtrlC, "ctrl+f": tea.KeyCtrlF, "ctrl+n": tea.KeyCtrlN, "ctrl+p": tea.KeyCtrlP,
	}
	if k, ok := named[name]; ok {
		return tea.KeyMsg{Type: k}
	}
	return keyMsg(name)
}

func pressed(a App, name string) App {
	next, _ := a.Update(keyFor(name))
	return next.(App)
}

// menuKeys opens a menu whose rows each open a place, with its cursor on the
// second row, and names each key by what it does to it.
func menuKeys(t *testing.T, keys []string) map[string]string {
	a, _ := opened(t, stack, 100, 30)
	a = a.with(a.helpers.Menu.OpenFinder(a.ContextTree))
	pn := a.Menu
	pn.Finding, pn.Rows = false, pn.All
	pn = pn.Move(1, a.helpers.Window.MenuRows(a.ContextTree))
	a = a.with(a.WithMenu(pn))
	a = a.with(a.Said("probe", false))
	closed := a.with(a.WithoutMenu())
	opening, _ := a.helpers.Menu.OpenRow(a.ContextTree)
	out := map[string]string{}
	for _, name := range keys {
		b := pressed(a, name)
		switch {
		case b.MenuOpen() && b.Menu.Cursor == pn.Move(1, a.helpers.Window.MenuRows(a.ContextTree)).Cursor:
			out[name] = "down"
		case b.MenuOpen() && b.Menu.Cursor == pn.Move(-1, a.helpers.Window.MenuRows(a.ContextTree)).Cursor:
			out[name] = "up"
		case !b.MenuOpen() && b.Keys.Typed() == "" && b.View() == closed.View():
			out[name] = "close"
		case !b.MenuOpen() && b.Keys.Typed() == "" && b.View() == a.with(opening).View():
			out[name] = "open"
		}
	}
	return out
}

// finderKeys opens the finder with two letters typed and its cursor on the
// second match, and names each key by what it does to it. Letters and
// Backspace type, and are not bindings.
func finderKeys(t *testing.T, keys []string) map[string]string {
	a, _ := opened(t, stack, 100, 30)
	a = a.with(a.helpers.Menu.OpenFinder(a.ContextTree))
	pn := a.Menu.Narrowed("es").Move(1, a.helpers.Window.MenuRows(a.ContextTree))
	a = a.with(a.WithMenu(pn))
	a = a.with(a.Said("probe", false))
	closed := a.with(a.WithoutMenu())
	opening, _ := a.helpers.Menu.OpenRow(a.ContextTree)
	out := map[string]string{}
	for _, name := range keys {
		b := pressed(a, name)
		switch {
		case b.MenuOpen() && b.Menu.Query == "es" && b.Menu.Cursor == pn.Move(1, a.helpers.Window.MenuRows(a.ContextTree)).Cursor:
			out[name] = "down"
		case b.MenuOpen() && b.Menu.Query == "es" && b.Menu.Cursor == pn.Move(-1, a.helpers.Window.MenuRows(a.ContextTree)).Cursor:
			out[name] = "up"
		case b.MenuOpen() && b.Menu.Query == "":
			out[name] = "clear"
		case !b.MenuOpen() && b.View() == closed.View():
			out[name] = "close"
		case !b.MenuOpen() && b.View() == a.with(opening).View():
			out[name] = "open"
		}
	}
	return out
}

// postingKeys opens posting's choice of three actions and names each key by
// what it does from each choice: the next, the previous, or one by place.
func postingKeys(t *testing.T, keys []string) map[string]string {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	a = a.with(a.helpers.Posting.Open(a.ContextTree, 30))
	a = a.with(a.Said("probe", false))
	if len(a.Posting.Actions) != 3 {
		t.Fatalf("posting offers %v", a.Posting.Actions)
	}
	closed := a.with(a.WithoutPosting())
	out := map[string]string{}
	for _, name := range keys {
		var after []int
		for choice := range a.Posting.Actions {
			from := a
			from.Posting.Choice = choice
			b := pressed(from, name)
			switch {
			case !b.Has(context.PostingKey) && b.View() == closed.View():
				after = append(after, -1)
			case b.Has(context.PostingConfirmationKey) && b.Posting.Choice == choice:
				after = append(after, -2)
			case b.Has(context.PostingKey) && !b.Has(context.PostingConfirmationKey):
				after = append(after, b.Posting.Choice)
			default:
				after = append(after, -3)
			}
		}
		switch after := fmtInts(after); after {
		case "1 2 2":
			out[name] = "down"
		case "0 0 1":
			out[name] = "up"
		case "0 0 0", "1 1 1", "2 2 2":
			out[name] = "choose " + string(rune('1'+after[0]-'0'))
		case "-1 -1 -1":
			out[name] = "close"
		case "-2 -2 -2":
			out[name] = "open"
		}
	}
	return out
}

func fmtInts(ns []int) string {
	out := ""
	for i, n := range ns {
		if i > 0 {
			out += " "
		}
		out += utils.Itoa(n)
	}
	return out
}

// postingConfirmationKeys opens what posting sends, and names each key by
// what it does: send it, or go back to the choice.
func postingConfirmationKeys(t *testing.T, keys []string) map[string]string {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	a = a.with(a.helpers.Posting.Open(a.ContextTree, 30))
	a.ContextTree = a.Opened(context.PostingConfirmationKey)
	out := map[string]string{}
	for _, name := range keys {
		b := pressed(a, name)
		switch {
		case !b.Has(context.PostingKey) && b.Notice == "Checking the pull request with GitHub, then sending…":
			out[name] = "post"
		case b.Has(context.PostingKey) && !b.Has(context.PostingConfirmationKey):
			out[name] = "back"
		}
	}
	return out
}

// confirmationKeys asks whether to drop a queued comment, and names the key
// that drops it; every other key keeps it.
func confirmationKeys(t *testing.T, keys []string) map[string]string {
	a := onBasket(t)
	a.ContextTree = a.Opened(context.ConfirmationKey)
	a.Confirmation = context.ConfirmationContext{Comment: 1, Pull: 21}
	out := map[string]string{}
	for _, name := range keys {
		if b := pressed(a, name); !b.Has(context.ConfirmationKey) && b.Notice == "Dropping…" {
			out[name] = "confirm"
		}
	}
	return out
}
