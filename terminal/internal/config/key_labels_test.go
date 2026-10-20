package config

import (
	"slices"
	"testing"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

const keysPath = "/home/lee/.config/lazyreview/config.yml"

func TestEveryFormOfKeyTurnsIntoTheNameTheTerminalAppsKeyReaderGivesIt(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for label, name := range map[string]string{
		"j": "j", "G": "G", "?": "?", ":": ":", "+": "+", "_": "_", "]": "]", "0": "0", "é": "é",
		"<space>": "space", "<enter>": "enter", "<esc>": "esc", "<tab>": "tab", "<up>": "up", "<down>": "down", "<left>": "left", "<right>": "right",
		"<c-o>": "ctrl+o", "<c-c>": "ctrl+c", "<c-i>": "tab",
		"gg": "g g", "ZZ": "Z Z", "]c": "] c", "[[": "[ [", "zR": "z R", "<space>v": "space v", "<space>?": "space ?", "g<space>": "g space", "<c-w>j": "ctrl+w j",
		"<disabled>": "",
	} {
		if why := LabelProblem(tr, label); why != "" {
			t.Errorf("%s is refused: %s", label, why)
		}
		if got := ReaderName(label); got != name {
			t.Errorf("%s reads as %q, want %q", label, got, name)
		}
	}
}

func TestAKeyThatIsNotOneIsRefusedSayingWhy(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	named := "is not a key; a named key is <space>, <enter>, <esc>, <tab>, <up>, <down>, <left>, <right>, or <c-x> for Ctrl-x"
	for label, why := range map[any]string{
		"":            "a key is not empty",
		"<":           "a < starts a named key, such as <space>, and needs its >",
		"<space":      "a < starts a named key, such as <space>, and needs its >",
		"<foo>":       "<foo> " + named,
		"<c-1>":       "<c-1> " + named,
		"<C-o>":       "<C-o> " + named,
		"g<disabled>": "<disabled> stands alone, for no key",
		3.0:           "a key is written as text",
	} {
		if got := LabelProblem(tr, label); got != why {
			t.Errorf("%v is refused as %q, want %q", label, got, why)
		}
	}
}

func TestAFileChangesOneActionsKeysAndKeepsTheRestAndTheTerminalAppIsHandedThemByName(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	read, err := ConfigFrom(tr, "keybinding:\n  universal:\n    quit: ['ZZ', '<c-q>']\n  diff:\n    nextHunk: ']h'\n    drop: '<disabled>'\n  home:\n    jumpToRepository: ['1', '2', '<disabled>', '4', '5', '6', '7', '8', '9']\n", keysPath, "/home/lee")
	if err != nil {
		t.Fatal(err)
	}
	universal := read.Values["keybinding"].(map[string]any)["universal"].(map[string]any)
	if quit, _ := universal["quit"].([]string); !slices.Equal(quit, []string{"ZZ", "<c-q>"}) {
		t.Fatalf("quit reads as %v", universal["quit"])
	}
	if closing, _ := universal["close"].([]string); len(closing) < 2 || closing[1] != "<esc>" {
		t.Fatalf("an action the file leaves out lost its keys: %v", universal["close"])
	}
	if next, previous := read.Sources["keybinding.diff.nextHunk"], read.Sources["keybinding.diff.previousHunk"]; next != "file" || previous != "default" {
		t.Fatalf("the sources are %s and %s", next, previous)
	}
	handed, err := TerminalPart(tr, read, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	keys := handed.Keybinding
	for _, handedKeys := range []struct{ got, want []string }{
		{keys.Universal.Quit, []string{"Z Z", "ctrl+q"}},
		{keys.Diff.NextHunk, []string{"] h"}},
		{keys.Diff.Drop, nil},
	} {
		if !slices.Equal(handedKeys.got, handedKeys.want) {
			t.Errorf("handed %v, want %v; a disabled action has no key", handedKeys.got, handedKeys.want)
		}
	}
	if want := []string{"1", "2", "", "4", "5", "6", "7", "8", "9"}; !slices.Equal(keys.Home.JumpToRepository, want) {
		t.Fatalf("a numbered list is handed as %v; it keeps its places", keys.Home.JumpToRepository)
	}
}

func TestKeysTheConfigCannotTakeAreRefusedNamingTheAction(t *testing.T) {
	tr := i18n.EnglishTranslationSet()
	for text, words := range map[string]string{
		"keybinding:\n  diff:\n    nextHunk: '<hunk>'\n":           `gives keybinding.diff.nextHunk "<hunk>", which is not a key or a list of keys: <hunk> is not a key; a named key is <space>, <enter>, <esc>, <tab>, <up>, <down>, <left>, <right>, or <c-x> for Ctrl-x`,
		"keybinding:\n  diff:\n    nextHunk: []\n":                 "gives keybinding.diff.nextHunk [], which is not a key or a list of keys: a key is not empty",
		"keybinding:\n  diff:\n    nextHunk: 3\n":                  "gives keybinding.diff.nextHunk 3, which is not a key or a list of keys: a key is written as text",
		"keybinding:\n  home:\n    jumpToRepository: ['1', '2']\n": `gives keybinding.home.jumpToRepository ["1","2"], which is not a list of 9 keys, one for each by its place`,
		"keybinding:\n  posting:\n    choose: '1'\n":               `gives keybinding.posting.choose "1", which is not a list of 3 keys, one for each by its place`,
		"keybinding:\n  diff:\n    nextHnk: ']h'\n":                "sets keybinding.diff.nextHnk, which is not a setting",
		"keybindings:\n  diff:\n    nextHunk: ']h'\n":              "sets keybindings, which is not a setting",
	} {
		_, err := ConfigFrom(tr, text, keysPath, "/home/lee")
		if err == nil || err.Error() != keysPath+" "+words {
			t.Errorf("%q is refused as %v", text, err)
		}
	}
}
