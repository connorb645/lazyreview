package app

import "review-desk/terminal/internal/keys"

var everyView = keys.Bindings{
	":": "command", "space ?": "keys", "space v": "review", "g x": "github", "g X": "web",
	"q": "close", "esc": "close", "Z Z": "quit", "ctrl+c": "quit",
	"] p": "next pull", "[ p": "previous pull", "ctrl+o": "jump back", "tab": "jump forward",
	"space m": "diagram", "space f": "finder", "R": "refresh",
	"0": "focus main", "+": "bigger", "_": "smaller",
}

var sideKeys = keys.Bindings{
	"j": "down", "down": "down", "k": "up", "up": "up", "g g": "top", "G": "bottom", "enter": "open",
	"1": "focus review", "2": "focus files", "3": "focus notes", "4": "focus diagram",
}

var sidePanelKeys = [sideCount + 1]keys.Bindings{
	reviewPanel: {"P": "post"},
	filesPanel:  {"z a": "toggle fold"},
	notesPanel:  {"e": "edit", "d d": "drop", "P": "post"},
	diagramPanel: {
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

var everyViewHelp = []helpRow{
	{"1 2 3 4", "the Review, Files, Notes or Diagram panel, from a panel on the left"},
	{"0", "the main panel, on the right"},
	{"Esc", "closes the top popup; with none, back to the panel on the left that the main panel shows"},
	{"+ _", "more of the screen for the panel with the keys, then all of it; _ goes back"},
	{":", "the command line: :q quits, :q! quits and drops drafts"},
	{"Space ?", "lists the keys of this panel"},
	{"R :refresh", "loads the review again: from GitHub, or a branch from its repository"},
	{"Space v", "the Review view"},
	{"Space m", "the diagram"},
	{"Space f", "finds a file, type or component by name"},
	{"gx", "opens the pull request, or the thread at the cursor, on GitHub"},
	{"gX", "opens this place in the web app"},
	{"]p [p", "the next or previous pull request filter on a stack: All, then each one"},
	{"Ctrl-o Ctrl-i", "back and forward through jumps"},
	{"q", "closes the top popup"},
	{"ZZ", "quits, as :q does"},
}

var sideHelp = [sideCount + 1][]helpRow{
	reviewPanel: {
		{"j k gg G", "the next, previous, first or last pull request"},
		{"Enter", "the Review view, in the main panel"},
		{"P", "post: the actions offered for this pull request"},
	},
	filesPanel: {
		{"j k gg G", "the next, previous, first or last file; the main panel shows its diff"},
		{"Enter", "on a file, its diff in the main panel; on a component or a folder, folds it"},
		{"za", "folds or unfolds a component or a folder"},
	},
	notesPanel: {
		{"j k gg G", "the next, previous, first or last note; the main panel shows it in its diff"},
		{"Enter", "the note in its diff, in the main panel"},
		{"e dd", "edit or drop the queued comment"},
		{"P", "post: the actions offered for the comment's pull request"},
	},
	diagramPanel: {
		{"j k w b gg G", "the next, previous, first or last box, in reading order"},
		{"Enter", "steps in; on a type, opens its diff; on a group, lists its members"},
		{"-", "steps out"},
		{"gd", "follows a link of the box"},
		{"gr", "lists the boxes that link to it"},
		{"/ ? n N", "find a type or file, forward or back"},
		{"f then a label", "jumps to the box with that label"},
		{"Space l", "lists the box's links, in and out"},
		{"Space p", "lists its properties"},
		{"Space d", "opens its diff"},
		{"Space a", "draws every link faintly, or stops"},
	},
}

var reviewHelp = []helpRow{
	{"j k", "the next or previous item: a section, a queued comment, a conversation or a link"},
	{"]] [[", "the next or previous section"},
	{"gg G", "the first or last item"},
	{"Enter", "opens the item in its diff; a component opens its diff"},
	{"za", "folds or unfolds a section"},
	{"e dd", "edit or drop the queued comment"},
	{"P", "post: the actions offered for the pull request at the cursor"},
}

var diffHelp = []helpRow{
	{"j k", "the next or previous line"},
	{"Ctrl-d Ctrl-u Ctrl-f Ctrl-b", "half a page and a page, down and up"},
	{"gg G {n}G", "the top, the bottom, or line n of this file"},
	{"]c [c", "the next or previous hunk"},
	{"]f [f", "the next or previous file"},
	{"]n [n", "the next or previous note: a queued comment, a conversation or a colleague's thread"},
	{"/ ? n N", "search the text, forward or back"},
	{"za zo zc", "toggle, open or close the fold at the cursor: a file, a run of unchanged lines or a note"},
	{"zR zM", "open every fold, close every fold"},
	{"Enter", "on a file's header, toggles the file"},
	{"V", "selects lines; o goes to the other end; Esc stops"},
	{"c", "comments on the line or the selection; on Claude's answer, uses it as the comment"},
	{"a", "asks Claude about the line or the selection; in a conversation, asks a follow-up"},
	{"e", "edits the queued comment at the cursor"},
	{"dd", "drops the queued comment at the cursor, after y confirms"},
	{"gd", "shows the type at the cursor on the diagram; on a file's header, lists its types"},
	{"-", "goes back to the diagram or the Review view"},
	{"Space s", "split or unified"},
	{"Space w", "wraps lines, or stops"},
	{"Space o", "the whole file, or back to the diff"},
	{"Space r", "a Markdown file rendered, or back to the diff"},
}

var diagramHelp = []helpRow{
	{"h j k l", "focus the nearest box left, down, up or right"},
	{"w b", "the next or previous box in reading order"},
	{"gg G", "the first or last box"},
	{"Enter", "steps in; on a type, opens its diff; on a group, lists its members"},
	{"-", "steps out"},
	{"gd", "follows a link of the focused box"},
	{"gr", "lists the boxes that link to it"},
	{"/ ? n N", "find a type or file, forward or back"},
	{"f then a label", "jumps to the box with that label"},
	{"Space l", "lists the focused box's links, in and out"},
	{"Space p", "lists its properties"},
	{"Space d", "opens its diff"},
	{"Space a", "draws every link faintly, or stops"},
}

var leaderWords = map[string]string{
	"keys": "Keys", "review": "Review view", "split": "Split", "wrap": "Wrap", "whole": "Whole file", "rendered": "Rendered",
	"diagram": "Diagram", "finder": "Find", "links": "Links", "properties": "Properties", "box diff": "Diff", "all links": "Every link",
}
