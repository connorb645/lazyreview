package gui

import (
	"github.com/connorb645/lazyreview/terminal/internal/gui/context"
	"github.com/connorb645/lazyreview/terminal/internal/gui/presentation"
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

var sidePanelKeys = [context.SideCount + 1]keys.Bindings{
	context.ReviewPanel: {"P": "post", "v": "highlight"},
	context.FilesPanel:  {"z a": "toggle fold"},
	context.NotesPanel:  {"e": "edit", "d d": "drop", "P": "post"},
	context.DiagramPanel: {
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

func everyViewHelp(tr *i18n.TranslationSet) []presentation.HelpRow {
	return []presentation.HelpRow{
		{Keys: "1 2 3 4", Words: tr.HelpFocusPanels},
		{Keys: "0", Words: tr.HelpFocusMain},
		{Keys: "Esc", Words: tr.HelpEscape},
		{Keys: "+ _", Words: tr.HelpScreenMode},
		{Keys: ":", Words: tr.HelpCommandLine},
		{Keys: "Space ?", Words: tr.HelpKeys},
		{Keys: "R :refresh", Words: tr.HelpRefresh},
		{Keys: "Space v", Words: tr.HelpReviewView},
		{Keys: "Space h", Words: tr.HelpHome},
		{Keys: "Space m", Words: tr.HelpDiagram},
		{Keys: "Space f", Words: tr.HelpFinder},
		{Keys: "gx", Words: tr.HelpGitHub},
		{Keys: "gX", Words: tr.HelpWeb},
		{Keys: "]p [p", Words: tr.HelpPullFilter},
		{Keys: "Ctrl-o Ctrl-i", Words: tr.HelpJumps},
		{Keys: "q", Words: tr.HelpClose},
		{Keys: "ZZ", Words: tr.HelpQuit},
	}
}

func sideHelp(tr *i18n.TranslationSet) [context.SideCount + 1][]presentation.HelpRow {
	return [context.SideCount + 1][]presentation.HelpRow{
		context.ReviewPanel: {
			{Keys: "j k gg G", Words: tr.HelpPullsMove},
			{Keys: "h l", Words: tr.HelpPanelsRound},
			{Keys: "Enter", Words: tr.HelpOpenReviewView},
			{Keys: "v", Words: tr.HelpHighlight},
			{Keys: "Esc", Words: tr.HelpClearHighlight},
			{Keys: "P", Words: tr.HelpPostPull},
		},
		context.FilesPanel: {
			{Keys: "j k gg G", Words: tr.HelpFilesMove},
			{Keys: "h l", Words: tr.HelpPanels},
			{Keys: "Enter", Words: tr.HelpOpenFile},
			{Keys: "za", Words: tr.HelpFoldTree},
		},
		context.NotesPanel: {
			{Keys: "j k gg G", Words: tr.HelpNotesMove},
			{Keys: "h l", Words: tr.HelpPanels},
			{Keys: "Enter", Words: tr.HelpOpenNote},
			{Keys: "e dd", Words: tr.HelpEditDrop},
			{Keys: "P", Words: tr.HelpPostComment},
		},
		context.DiagramPanel: {
			{Keys: "j k w b gg G", Words: tr.HelpBoxesMove},
			{Keys: "h l", Words: tr.HelpBoxesSideways},
			{Keys: "Enter", Words: tr.HelpStepIn},
			{Keys: "-", Words: tr.HelpStepOut},
			{Keys: "gd", Words: tr.HelpFollowBox},
			{Keys: "gr", Words: tr.HelpReferences},
			{Keys: "/ ? n N", Words: tr.HelpFindBox},
			{Keys: tr.HelpLabelsKeys, Words: tr.HelpLabels},
			{Keys: "Space l", Words: tr.HelpBoxLinks},
			{Keys: "Space p", Words: tr.HelpProperties},
			{Keys: "Space d", Words: tr.HelpBoxDiff},
			{Keys: "Space a", Words: tr.HelpAllLinks},
		},
	}
}

func reviewHelp(tr *i18n.TranslationSet) []presentation.HelpRow {
	return []presentation.HelpRow{
		{Keys: "j k", Words: tr.HelpItemsMove},
		{Keys: "]] [[", Words: tr.HelpSections},
		{Keys: "gg G", Words: tr.HelpItemsEnds},
		{Keys: "Enter", Words: tr.HelpOpenItem},
		{Keys: "za", Words: tr.HelpFoldSection},
		{Keys: "e dd", Words: tr.HelpEditDrop},
		{Keys: "P", Words: tr.HelpPostAtCursor},
	}
}

func diffHelp(tr *i18n.TranslationSet) []presentation.HelpRow {
	return []presentation.HelpRow{
		{Keys: "j k", Words: tr.HelpLinesMove},
		{Keys: "Ctrl-d Ctrl-u Ctrl-f Ctrl-b", Words: tr.HelpPages},
		{Keys: "gg G {n}G", Words: tr.HelpLineEnds},
		{Keys: "]c [c", Words: tr.HelpHunks},
		{Keys: "]f [f", Words: tr.HelpFiles},
		{Keys: "]n [n", Words: tr.HelpNotes},
		{Keys: "/ ? n N", Words: tr.HelpSearch},
		{Keys: "za zo zc", Words: tr.HelpFold},
		{Keys: "zR zM", Words: tr.HelpFoldAll},
		{Keys: "Enter", Words: tr.HelpToggleFile},
		{Keys: "V", Words: tr.HelpSelect},
		{Keys: "c", Words: tr.HelpComment},
		{Keys: "a", Words: tr.HelpAsk},
		{Keys: "e", Words: tr.HelpEditAtCursor},
		{Keys: "dd", Words: tr.HelpDropAtCursor},
		{Keys: "gd", Words: tr.HelpShowType},
		{Keys: "-", Words: tr.HelpBack},
		{Keys: "Space s", Words: tr.HelpSplit},
		{Keys: "Space w", Words: tr.HelpWrap},
		{Keys: "Space o", Words: tr.HelpWhole},
		{Keys: "Space r", Words: tr.HelpRendered},
	}
}

func diagramHelp(tr *i18n.TranslationSet) []presentation.HelpRow {
	return []presentation.HelpRow{
		{Keys: "h j k l", Words: tr.HelpBoxesNearest},
		{Keys: "w b", Words: tr.HelpBoxesOrder},
		{Keys: "gg G", Words: tr.HelpBoxesEnds},
		{Keys: "Enter", Words: tr.HelpStepIn},
		{Keys: "-", Words: tr.HelpStepOut},
		{Keys: "gd", Words: tr.HelpFollow},
		{Keys: "gr", Words: tr.HelpReferences},
		{Keys: "/ ? n N", Words: tr.HelpFindBox},
		{Keys: tr.HelpLabelsKeys, Words: tr.HelpLabels},
		{Keys: "Space l", Words: tr.HelpLinks},
		{Keys: "Space p", Words: tr.HelpProperties},
		{Keys: "Space d", Words: tr.HelpBoxDiff},
		{Keys: "Space a", Words: tr.HelpAllLinks},
	}
}

func homeHelp(tr *i18n.TranslationSet) []presentation.HelpRow {
	return []presentation.HelpRow{
		{Keys: "j k", Words: tr.HelpHomeMove},
		{Keys: "gg G", Words: tr.HelpHomeEnds},
		{Keys: "h l", Words: tr.HelpRepositories},
		{Keys: tr.HelpFirstNineKeys, Words: tr.HelpFirstNine},
		{Keys: "Enter", Words: tr.HelpOpenPull},
		{Keys: "R :refresh", Words: tr.HelpReadAgain},
		{Keys: "gx", Words: tr.HelpGitHubPull},
		{Keys: "gX", Words: tr.HelpWebList},
		{Keys: "Space v", Words: tr.HelpReviewOpen},
		{Keys: "+ _", Words: tr.HelpListMode},
		{Keys: ":", Words: tr.HelpCommandQuit},
		{Keys: "ZZ", Words: tr.HelpQuit},
	}
}

func homeMarks(tr *i18n.TranslationSet) []presentation.HelpRow {
	return []presentation.HelpRow{
		{Keys: "●", Words: tr.MarkWaiting},
		{Keys: "◆", Words: tr.MarkLoaded},
		{Keys: "┌ └", Words: tr.MarkStack},
		{Keys: tr.Dim, Words: tr.MarkDraft},
	}
}

func leaderWords(tr *i18n.TranslationSet) map[string]string {
	return map[string]string{
		"keys": tr.Keys, "review": tr.ReviewView, "home": tr.Home, "split": tr.Split, "wrap": tr.Wrap, "whole": tr.WholeFile, "rendered": tr.Rendered,
		"diagram": tr.Diagram, "finder": tr.Find, "links": tr.Links, "properties": tr.Properties, "box diff": tr.Diff, "all links": tr.EveryLink,
	}
}
