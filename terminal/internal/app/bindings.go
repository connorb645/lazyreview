package app

import "review-desk/terminal/internal/keys"

var everyView = keys.Bindings{
	":": "command", "space ?": "keys", "g h": "reviews", "space v": "review", "g x": "github", "g X": "web",
	"q": "close", "esc": "close", "Z Z": "quit", "ctrl+c": "quit",
	"] p": "next pull", "[ p": "previous pull", "ctrl+o": "jump back", "tab": "jump forward",
	"space m": "diagram", "space f": "finder",
}

var listKeys = keys.Bindings{
	"j": "down", "down": "down", "k": "up", "up": "up", "g g": "top", "G": "bottom", "enter": "open",
	"/": "find", "n": "next match", "N": "previous match",
}

var reviewKeys = keys.Bindings{
	"j": "down", "down": "down", "k": "up", "up": "up", "g g": "top", "G": "bottom",
	"] ]": "next section", "[ [": "previous section", "z a": "fold", "enter": "open",
}

var diffKeys = keys.Bindings{
	"j": "down", "down": "down", "k": "up", "up": "up", "ctrl+d": "half down", "ctrl+u": "half up", "ctrl+f": "page down", "ctrl+b": "page up",
	"g g": "top", "G": "bottom", "] c": "next hunk", "[ c": "previous hunk", "] f": "next file", "[ f": "previous file",
	"/": "search", "?": "search back", "n": "next match", "N": "previous match",
	"z a": "toggle fold", "z o": "open fold", "z c": "close fold", "z R": "open folds", "z M": "close folds",
	"enter": "open", "-": "back", "space s": "split", "space w": "wrap", "space o": "whole", "g d": "show type",
}

var diagramKeys = keys.Bindings{
	"h": "left", "left": "left", "j": "down", "down": "down", "k": "up", "up": "up", "l": "right", "right": "right",
	"w": "next box", "b": "previous box", "g g": "top", "G": "bottom", "enter": "open", "-": "back",
	"g d": "follow", "g r": "references", "/": "search", "?": "search back", "n": "next match", "N": "previous match",
	"f": "labels", "space l": "links", "space p": "properties", "space d": "box diff", "space a": "all links",
}

var everyViewHelp = []helpRow{
	{":", "the command line: :q quits, :q! quits and drops drafts"},
	{"Space ?", "lists the keys of this view"},
	{"Space v", "the Review view"},
	{"Space m", "the diagram"},
	{"Space f", "finds a file, type or component by name"},
	{"gh", "every review"},
	{"gx", "opens the pull request, or the thread at the cursor, on GitHub"},
	{"gX", "opens this place in the web app"},
	{"]p [p", "the next or previous pull request filter on a stack: All, then each one"},
	{"Ctrl-o Ctrl-i", "back and forward through jumps"},
	{"q Esc", "closes the top panel or list"},
	{"ZZ", "quits, as :q does"},
}

var listHelp = []helpRow{
	{"j k gg G", "move"},
	{"Enter", "opens the review"},
	{"/ n N", "find"},
}

var reviewHelp = []helpRow{
	{"j k", "the next or previous item: a section, a queued comment, a conversation or a link"},
	{"]] [[", "the next or previous section"},
	{"gg G", "the first or last item"},
	{"Enter", "opens the item in its diff; a component opens its diff"},
	{"za", "folds or unfolds a section"},
}

var diffHelp = []helpRow{
	{"j k", "the next or previous line"},
	{"Ctrl-d Ctrl-u Ctrl-f Ctrl-b", "half a page and a page, down and up"},
	{"gg G {n}G", "the top, the bottom, or line n of this file"},
	{"]c [c", "the next or previous hunk"},
	{"]f [f", "the next or previous file"},
	{"/ ? n N", "search the text, forward or back"},
	{"za zo zc", "toggle, open or close the fold at the cursor: a file or a run of unchanged lines"},
	{"zR zM", "open every fold, close every fold"},
	{"Enter", "on a file's header, toggles the file"},
	{"gd", "shows the type at the cursor on the diagram; on a file's header, lists its types"},
	{"-", "goes back to the diagram or the Review view"},
	{"Space s", "split or unified"},
	{"Space w", "wraps lines, or stops"},
	{"Space o", "the whole file, or back to the diff"},
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

// leaderWords say what each key after the leader does, for the hint line.
var leaderWords = map[string]string{
	"keys": "keys", "review": "Review view", "split": "split", "wrap": "wrap", "whole": "whole file",
	"diagram": "diagram", "finder": "find", "links": "links", "properties": "properties", "box diff": "diff", "all links": "every link",
}
