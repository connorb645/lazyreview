package gui

import (
	"github.com/connorb645/lazyreview/terminal/internal/i18n"
	"github.com/connorb645/lazyreview/terminal/internal/keys"
)

var everyView = keys.Bindings{
	":": "command", "space ?": "keys", "space v": "review", "g x": "github", "g X": "web",
	"q": "close", "esc": "close", "Z Z": "quit", "ctrl+c": "quit",
	"] p": "next pull", "[ p": "previous pull", "ctrl+o": "jump back", "tab": "jump forward",
	"space m": "diagram", "space f": "finder", "space h": "home", "R": "refresh",
	"0": "focus main", "+": "bigger", "_": "smaller",
}

var homeKeys = keys.Bindings{
	":": "command", "space ?": "keys", "space v": "review", "g x": "github", "g X": "web",
	"q": "close", "esc": "close", "Z Z": "quit", "ctrl+c": "quit", "R": "refresh", "+": "bigger", "_": "smaller",
	"j": "down", "down": "down", "k": "up", "up": "up", "g g": "top", "G": "bottom", "enter": "open",
	"h": "previous panel", "l": "next panel",
	"1": "repository 1", "2": "repository 2", "3": "repository 3", "4": "repository 4", "5": "repository 5",
	"6": "repository 6", "7": "repository 7", "8": "repository 8", "9": "repository 9",
}

var sideKeys = keys.Bindings{
	"j": "down", "down": "down", "k": "up", "up": "up", "g g": "top", "G": "bottom", "enter": "open",
	"h": "previous panel", "l": "next panel",
	"1": "focus review", "2": "focus files", "3": "focus notes", "4": "focus diagram",
}

var sidePanelKeys = [sideCount + 1]keys.Bindings{
	reviewPanel: {"P": "post", "v": "highlight"},
	filesPanel:  {"z a": "toggle fold"},
	notesPanel:  {"e": "edit", "d d": "drop", "P": "post"},
	diagramPanel: {
		"h": "left", "l": "right",
		"w": "next box", "b": "previous box", "-": "back",
		"g d": "follow", "g r": "references", "/": "search", "?": "search back", "n": "next match", "N": "previous match",
		"f": "labels", "space l": "links", "space p": "properties", "space d": "box diff", "space a": "all links",
	},
}

var reviewKeys = keys.Bindings{
	"j": "down", "down": "down", "k": "up", "up": "up", "g g": "top", "G": "bottom",
	"] ]": "next section", "[ [": "previous section", "z a": "fold", "enter": "open",
	"e": "edit", "d d": "drop", "P": "post",
}

var diffKeys = keys.Bindings{
	"j": "down", "down": "down", "k": "up", "up": "up", "ctrl+d": "half down", "ctrl+u": "half up", "ctrl+f": "page down", "ctrl+b": "page up",
	"g g": "top", "G": "bottom", "] c": "next hunk", "[ c": "previous hunk", "] f": "next file", "[ f": "previous file",
	"/": "search", "?": "search back", "n": "next match", "N": "previous match",
	"z a": "toggle fold", "z o": "open fold", "z c": "close fold", "z R": "open folds", "z M": "close folds",
	"enter": "open", "-": "back", "space s": "split", "space w": "wrap", "space o": "whole", "space r": "rendered", "g d": "show type",
	"] n": "next note", "[ n": "previous note", "V": "select", "o": "other end",
	"c": "comment", "a": "ask", "e": "edit", "d d": "drop",
}

var diagramKeys = keys.Bindings{
	"h": "left", "left": "left", "j": "down", "down": "down", "k": "up", "up": "up", "l": "right", "right": "right",
	"w": "next box", "b": "previous box", "g g": "top", "G": "bottom", "enter": "open", "-": "back",
	"g d": "follow", "g r": "references", "/": "search", "?": "search back", "n": "next match", "N": "previous match",
	"f": "labels", "space l": "links", "space p": "properties", "space d": "box diff", "space a": "all links",
}

func everyViewHelp(tr *i18n.TranslationSet) []helpRow {
	return []helpRow{
		{"1 2 3 4", tr.HelpFocusPanels},
		{"0", tr.HelpFocusMain},
		{"Esc", tr.HelpEscape},
		{"+ _", tr.HelpScreenMode},
		{":", tr.HelpCommandLine},
		{"Space ?", tr.HelpKeys},
		{"R :refresh", tr.HelpRefresh},
		{"Space v", tr.HelpReviewView},
		{"Space h", tr.HelpHome},
		{"Space m", tr.HelpDiagram},
		{"Space f", tr.HelpFinder},
		{"gx", tr.HelpGitHub},
		{"gX", tr.HelpWeb},
		{"]p [p", tr.HelpPullFilter},
		{"Ctrl-o Ctrl-i", tr.HelpJumps},
		{"q", tr.HelpClose},
		{"ZZ", tr.HelpQuit},
	}
}

func sideHelp(tr *i18n.TranslationSet) [sideCount + 1][]helpRow {
	return [sideCount + 1][]helpRow{
		reviewPanel: {
			{"j k gg G", tr.HelpPullsMove},
			{"h l", tr.HelpPanelsRound},
			{"Enter", tr.HelpOpenReviewView},
			{"v", tr.HelpHighlight},
			{"Esc", tr.HelpClearHighlight},
			{"P", tr.HelpPostPull},
		},
		filesPanel: {
			{"j k gg G", tr.HelpFilesMove},
			{"h l", tr.HelpPanels},
			{"Enter", tr.HelpOpenFile},
			{"za", tr.HelpFoldTree},
		},
		notesPanel: {
			{"j k gg G", tr.HelpNotesMove},
			{"h l", tr.HelpPanels},
			{"Enter", tr.HelpOpenNote},
			{"e dd", tr.HelpEditDrop},
			{"P", tr.HelpPostComment},
		},
		diagramPanel: {
			{"j k w b gg G", tr.HelpBoxesMove},
			{"h l", tr.HelpBoxesSideways},
			{"Enter", tr.HelpStepIn},
			{"-", tr.HelpStepOut},
			{"gd", tr.HelpFollowBox},
			{"gr", tr.HelpReferences},
			{"/ ? n N", tr.HelpFindBox},
			{tr.HelpLabelsKeys, tr.HelpLabels},
			{"Space l", tr.HelpBoxLinks},
			{"Space p", tr.HelpProperties},
			{"Space d", tr.HelpBoxDiff},
			{"Space a", tr.HelpAllLinks},
		},
	}
}

func reviewHelp(tr *i18n.TranslationSet) []helpRow {
	return []helpRow{
		{"j k", tr.HelpItemsMove},
		{"]] [[", tr.HelpSections},
		{"gg G", tr.HelpItemsEnds},
		{"Enter", tr.HelpOpenItem},
		{"za", tr.HelpFoldSection},
		{"e dd", tr.HelpEditDrop},
		{"P", tr.HelpPostAtCursor},
	}
}

func diffHelp(tr *i18n.TranslationSet) []helpRow {
	return []helpRow{
		{"j k", tr.HelpLinesMove},
		{"Ctrl-d Ctrl-u Ctrl-f Ctrl-b", tr.HelpPages},
		{"gg G {n}G", tr.HelpLineEnds},
		{"]c [c", tr.HelpHunks},
		{"]f [f", tr.HelpFiles},
		{"]n [n", tr.HelpNotes},
		{"/ ? n N", tr.HelpSearch},
		{"za zo zc", tr.HelpFold},
		{"zR zM", tr.HelpFoldAll},
		{"Enter", tr.HelpToggleFile},
		{"V", tr.HelpSelect},
		{"c", tr.HelpComment},
		{"a", tr.HelpAsk},
		{"e", tr.HelpEditAtCursor},
		{"dd", tr.HelpDropAtCursor},
		{"gd", tr.HelpShowType},
		{"-", tr.HelpBack},
		{"Space s", tr.HelpSplit},
		{"Space w", tr.HelpWrap},
		{"Space o", tr.HelpWhole},
		{"Space r", tr.HelpRendered},
	}
}

func diagramHelp(tr *i18n.TranslationSet) []helpRow {
	return []helpRow{
		{"h j k l", tr.HelpBoxesNearest},
		{"w b", tr.HelpBoxesOrder},
		{"gg G", tr.HelpBoxesEnds},
		{"Enter", tr.HelpStepIn},
		{"-", tr.HelpStepOut},
		{"gd", tr.HelpFollow},
		{"gr", tr.HelpReferences},
		{"/ ? n N", tr.HelpFindBox},
		{tr.HelpLabelsKeys, tr.HelpLabels},
		{"Space l", tr.HelpLinks},
		{"Space p", tr.HelpProperties},
		{"Space d", tr.HelpBoxDiff},
		{"Space a", tr.HelpAllLinks},
	}
}

func homeHelp(tr *i18n.TranslationSet) []helpRow {
	return []helpRow{
		{"j k", tr.HelpHomeMove},
		{"gg G", tr.HelpHomeEnds},
		{"h l", tr.HelpRepositories},
		{tr.HelpFirstNineKeys, tr.HelpFirstNine},
		{"Enter", tr.HelpOpenPull},
		{"R :refresh", tr.HelpReadAgain},
		{"gx", tr.HelpGitHubPull},
		{"gX", tr.HelpWebList},
		{"Space v", tr.HelpReviewOpen},
		{"+ _", tr.HelpListMode},
		{":", tr.HelpCommandQuit},
		{"ZZ", tr.HelpQuit},
	}
}

func homeMarks(tr *i18n.TranslationSet) []helpRow {
	return []helpRow{
		{"●", tr.MarkWaiting},
		{"◆", tr.MarkLoaded},
		{"┌ └", tr.MarkStack},
		{tr.Dim, tr.MarkDraft},
	}
}

func leaderWords(tr *i18n.TranslationSet) map[string]string {
	return map[string]string{
		"keys": tr.Keys, "review": tr.ReviewView, "home": tr.Home, "split": tr.Split, "wrap": tr.Wrap, "whole": tr.WholeFile, "rendered": tr.Rendered,
		"diagram": tr.Diagram, "finder": tr.Find, "links": tr.Links, "properties": tr.Properties, "box diff": tr.Diff, "all links": tr.EveryLink,
	}
}
