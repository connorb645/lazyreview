package gui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/connorb645/lazyreview/terminal/internal/utils"
)

// The keys each place takes, and what each does: the home screen, the four
// panels, the three main views from their bindings, and the popups from what
// a press of each key does to them. A change to any key in any place shows.
func TestTheBindingsOfEachPlace(t *testing.T) {
	places := map[string]map[string]string{}
	home, _, _ := homeOn(t, "", 100, 30)
	places["home"] = home.bindings()
	a, _ := opened(t, stack, 100, 30)
	for n, name := range map[int]string{reviewPanel: "review panel", filesPanel: "files panel", notesPanel: "notes panel", diagramPanel: "diagram panel"} {
		places[name] = press(t, a, utils.Itoa(n)).bindings()
	}
	places["review view"] = press(t, a, "0").bindings()
	places["diff"] = onBasket(t).bindings()
	places["diagram"] = press(t, a, "space", "m").bindings()
	for name, probe := range map[string]func(*testing.T) map[string]string{
		"menu": menuKeys, "finder": finderKeys, "posting": postingKeys, "posting confirmation": postingConfirmationKeys, "confirmation": confirmationKeys,
	} {
		places[name] = probe(t)
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
}

// probeKeys are the keys tried on each popup: every key any place binds, and
// a few that none does.
func probeKeys(t *testing.T) []string {
	t.Helper()
	seen := map[string]bool{"x": true, "y": true, "n": true, "backspace": true, "left": true, "right": true, "ctrl+n": true, "ctrl+p": true}
	for _, b := range []map[string]string{homeKeys, sideKeys, reviewKeys, diffKeys, diagramKeys, everyView} {
		for sequence := range b {
			seen[firstKey(sequence)] = true
		}
	}
	for _, b := range sidePanelKeys {
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
func menuKeys(t *testing.T) map[string]string {
	a, _ := opened(t, stack, 100, 30)
	a = a.openFinder()
	pn := *a.menu
	pn.finding, pn.rows = false, pn.all
	pn = pn.move(1, a.menuRows())
	a.menu = &pn
	a = a.say("probe", false)
	closed := a
	closed.menu = nil
	opening, _ := a.openRow()
	out := map[string]string{}
	for _, name := range probeKeys(t) {
		b := pressed(a, name)
		switch {
		case b.menu != nil && b.menu.cursor == pn.move(1, a.menuRows()).cursor:
			out[name] = "down"
		case b.menu != nil && b.menu.cursor == pn.move(-1, a.menuRows()).cursor:
			out[name] = "up"
		case b.menu == nil && b.keys.Typed() == "" && b.View() == closed.View():
			out[name] = "close"
		case b.menu == nil && b.keys.Typed() == "" && b.View() == opening.(App).View():
			out[name] = "open"
		}
	}
	return out
}

// finderKeys opens the finder with two letters typed and its cursor on the
// second match, and names each key by what it does to it. Letters and
// Backspace type, and are not bindings.
func finderKeys(t *testing.T) map[string]string {
	a, _ := opened(t, stack, 100, 30)
	a = a.openFinder()
	pn := a.menu.narrowed("es").move(1, a.menuRows())
	a.menu = &pn
	a = a.say("probe", false)
	closed := a
	closed.menu = nil
	opening, _ := a.openRow()
	out := map[string]string{}
	for _, name := range probeKeys(t) {
		b := pressed(a, name)
		switch {
		case b.menu != nil && b.menu.query == "es" && b.menu.cursor == pn.move(1, a.menuRows()).cursor:
			out[name] = "down"
		case b.menu != nil && b.menu.query == "es" && b.menu.cursor == pn.move(-1, a.menuRows()).cursor:
			out[name] = "up"
		case b.menu != nil && b.menu.query == "":
			out[name] = "clear"
		case b.menu == nil && b.View() == closed.View():
			out[name] = "close"
		case b.menu == nil && b.View() == opening.(App).View():
			out[name] = "open"
		}
	}
	return out
}

// postingKeys opens posting's choice of three actions and names each key by
// what it does from each choice: the next, the previous, or one by place.
func postingKeys(t *testing.T) map[string]string {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	a = a.openPosting(30).say("probe", false)
	if len(a.posting.actions) != 3 {
		t.Fatalf("posting offers %v", a.posting.actions)
	}
	closed := a
	closed.posting = nil
	out := map[string]string{}
	for _, name := range probeKeys(t) {
		var after []int
		for choice := range a.posting.actions {
			from := a
			pn := *a.posting
			pn.choice = choice
			from.posting = &pn
			b := pressed(from, name)
			switch {
			case b.posting == nil && b.View() == closed.View():
				after = append(after, -1)
			case b.posting != nil && b.posting.confirming && b.posting.choice == choice:
				after = append(after, -2)
			case b.posting != nil && !b.posting.confirming:
				after = append(after, b.posting.choice)
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
func postingConfirmationKeys(t *testing.T) map[string]string {
	a, _ := opened(t, "github.com/octo/widgets/30", 100, 30)
	a = a.openPosting(30)
	pn := *a.posting
	pn.confirming = true
	a.posting = &pn
	out := map[string]string{}
	for _, name := range probeKeys(t) {
		b := pressed(a, name)
		switch {
		case b.posting == nil && b.notice == "Checking the pull request with GitHub, then sending…":
			out[name] = "post"
		case b.posting != nil && !b.posting.confirming:
			out[name] = "back"
		}
	}
	return out
}

// confirmationKeys asks whether to drop a queued comment, and names the key
// that drops it; every other key keeps it.
func confirmationKeys(t *testing.T) map[string]string {
	a := onBasket(t)
	a.confirming = &confirmation{comment: 1, pull: 21}
	out := map[string]string{}
	for _, name := range probeKeys(t) {
		if b := pressed(a, name); b.confirming == nil && b.notice == "Dropping…" {
			out[name] = "confirm"
		}
	}
	return out
}
