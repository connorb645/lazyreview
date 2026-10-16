package app

import "review-desk/terminal/internal/keys"

var everyView = keys.Bindings{
	":": "command", "space ?": "keys", "g h": "reviews", "space v": "review", "g x": "github", "g X": "web",
	"q": "close", "esc": "close", "Z Z": "quit", "ctrl+c": "quit",
}

var listKeys = keys.Bindings{
	"j": "down", "down": "down", "k": "up", "up": "up", "g g": "top", "G": "bottom", "enter": "open",
	"/": "find", "n": "next match", "N": "previous match",
}

var reviewKeys = keys.Bindings{
	"j": "down", "down": "down", "k": "up", "up": "up", "g g": "top", "G": "bottom",
	"] ]": "next section", "[ [": "previous section", "z a": "fold",
}

var everyViewHelp = []helpRow{
	{":", "the command line: :q quits, :q! quits and drops drafts"},
	{"Space ?", "lists the keys of this view"},
	{"Space v", "the Review view"},
	{"gh", "every review"},
	{"gx", "opens the pull request, or the thread at the cursor, on GitHub"},
	{"gX", "opens this place in the web app"},
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
	{"za", "folds or unfolds a section"},
}

// leaderWords say what each key after the leader does, for the hint line.
var leaderWords = map[string]string{"keys": "keys", "review": "Review view"}
