package config

import (
	"strings"

	"github.com/connorb645/lazyreview/terminal/internal/i18n"
)

func moves() []keyDefault {
	return []keyDefault{{"down", []string{"j", "<down>"}}, {"up", []string{"k", "<up>"}}, {"top", "gg"}, {"bottom", "G"}}
}

func pages() []keyDefault {
	return []keyDefault{{"halfDown", "<c-d>"}, {"halfUp", "<c-u>"}, {"pageDown", "<c-f>"}, {"pageUp", "<c-b>"}}
}

func boxes() []keyDefault {
	return []keyDefault{
		{"nextBox", "w"}, {"previousBox", "b"}, {"top", "gg"}, {"bottom", "G"}, {"open", "<enter>"}, {"back", "-"}, {"follow", "gd"}, {"references", "gr"},
		{"search", "/"}, {"searchBack", "?"}, {"nextMatch", "n"}, {"previousMatch", "N"}, {"labels", "f"}, {"links", "<space>l"}, {"properties", "<space>p"},
		{"boxDiff", "<space>d"}, {"allLinks", "<space>a"},
	}
}

func joined(parts ...[]keyDefault) []keyDefault {
	var out []keyDefault
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func keybindings(tr *i18n.TranslationSet) *Section {
	var panelBoxes []keyDefault
	for _, b := range boxes() {
		if b.action != "top" && b.action != "bottom" && b.action != "open" {
			panelBoxes = append(panelBoxes, b)
		}
	}
	return &Section{About: strings.Split(tr.ConfigAboutKeys, "\n"), Entries: []Entry{
		{Name: "universal", Section: keys(tr.ConfigKeysUniversal,
			keyDefault{"command", ":"}, keyDefault{"keys", "<space>?"}, keyDefault{"review", "<space>v"}, keyDefault{"github", "gx"},
			keyDefault{"close", []string{"q", "<esc>"}}, keyDefault{"quit", []string{"ZZ", "<c-c>"}}, keyDefault{"refresh", "R"}, keyDefault{"bigger", "+"},
			keyDefault{"smaller", "_"})},
		{Name: "home", Section: keys(tr.ConfigKeysHome, joined(moves(), pages(), []keyDefault{
			{"open", "<enter>"}, {"previousPanel", "h"}, {"nextPanel", "l"}, {"jumpToRepository", []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}},
			{"hideRepository", "i"}, {"showHidden", "I"},
		})...)},
		{Name: "reviewScreen", Section: keys(tr.ConfigKeysReviewScreen,
			keyDefault{"nextPull", "]p"}, keyDefault{"previousPull", "[p"}, keyDefault{"jumpBack", "<c-o>"}, keyDefault{"jumpForward", "<c-i>"},
			keyDefault{"diagram", "<space>m"}, keyDefault{"finder", "<space>f"}, keyDefault{"home", "<space>h"}, keyDefault{"focusMain", "0"})},
		{Name: "panels", Section: keys(tr.ConfigKeysPanels, joined(moves(), pages(), []keyDefault{
			{"open", "<enter>"}, {"previousPanel", "h"}, {"nextPanel", "l"}, {"jumpToBlock", []string{"1", "2", "3", "4"}},
		})...)},
		{Name: "reviewPanel", Section: keys("", keyDefault{"post", "P"}, keyDefault{"highlight", "v"})},
		{Name: "filesPanel", Section: keys("", keyDefault{"toggleFold", "za"}, keyDefault{"select", "v"})},
		{Name: "notesPanel", Section: keys("", keyDefault{"edit", "e"}, keyDefault{"drop", "dd"}, keyDefault{"post", "P"}, keyDefault{"stop", "x"})},
		{Name: "diagramPanel", Section: keys("", panelBoxes...)},
		{Name: "reviewView", Section: keys(tr.ConfigKeysReviewView, joined(moves(), pages(), []keyDefault{
			{"nextSection", "]]"}, {"previousSection", "[["}, {"fold", "za"}, {"open", "<enter>"}, {"edit", "e"}, {"drop", "dd"}, {"post", "P"}, {"drawings", "gX"},
		})...)},
		{Name: "diff", Section: keys("",
			keyDefault{"down", []string{"j", "<down>"}}, keyDefault{"up", []string{"k", "<up>"}}, keyDefault{"halfDown", "<c-d>"}, keyDefault{"halfUp", "<c-u>"},
			keyDefault{"pageDown", "<c-f>"}, keyDefault{"pageUp", "<c-b>"}, keyDefault{"top", "gg"}, keyDefault{"bottom", "G"},
			keyDefault{"scrollLeft", []string{"h", "<left>"}}, keyDefault{"scrollRight", []string{"l", "<right>"}},
			keyDefault{"nextHunk", "]c"}, keyDefault{"previousHunk", "[c"}, keyDefault{"nextFile", "]f"}, keyDefault{"previousFile", "[f"}, keyDefault{"search", "/"},
			keyDefault{"searchBack", "?"}, keyDefault{"nextMatch", "n"}, keyDefault{"previousMatch", "N"},
			keyDefault{"toggleFold", "za"}, keyDefault{"openFold", "zo"}, keyDefault{"closeFold", "zc"}, keyDefault{"openFolds", "zR"}, keyDefault{"closeFolds", "zM"},
			keyDefault{"open", "<enter>"}, keyDefault{"back", "-"}, keyDefault{"split", "<space>s"}, keyDefault{"wrap", "<space>w"},
			keyDefault{"whole", "<space>o"}, keyDefault{"rendered", "<space>r"}, keyDefault{"showType", "gd"}, keyDefault{"nextNote", "]n"}, keyDefault{"previousNote", "[n"},
			keyDefault{"select", "V"}, keyDefault{"otherEnd", "o"}, keyDefault{"comment", "c"}, keyDefault{"ask", "a"}, keyDefault{"edit", "e"}, keyDefault{"drop", "dd"},
			keyDefault{"stop", "x"})},
		{Name: "diagram", Section: keys(tr.ConfigKeysDiagram, joined([]keyDefault{
			{"left", []string{"h", "<left>"}}, {"down", []string{"j", "<down>"}}, {"up", []string{"k", "<up>"}}, {"right", []string{"l", "<right>"}},
		}, boxes())...)},
		{Name: "menu", Section: keys(tr.ConfigKeysMenu, joined(moves(), pages(), []keyDefault{{"open", "<enter>"}, {"close", []string{"<esc>", "q"}}})...)},
		{Name: "finder", Section: keys("", keyDefault{"down", []string{"<c-n>", "<down>"}}, keyDefault{"up", []string{"<c-p>", "<up>"}}, keyDefault{"open", "<enter>"},
			keyDefault{"close", "<esc>"}, keyDefault{"clear", "<c-u>"})},
		{Name: "posting", Section: keys("", keyDefault{"down", []string{"j", "<down>"}}, keyDefault{"up", []string{"k", "<up>"}}, keyDefault{"choose", []string{"1", "2", "3"}},
			keyDefault{"open", "<enter>"}, keyDefault{"close", []string{"<esc>", "n", "q"}}, keyDefault{"post", "y"}, keyDefault{"back", []string{"n", "<esc>", "q"}})},
		{Name: "confirmation", Section: keys("", keyDefault{"confirm", "y"})},
	}}
}
