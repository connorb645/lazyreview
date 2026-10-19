// Package i18n holds every string the terminal app shows, in English.
package i18n

// TranslationSet is every string the terminal app shows. A string with
// values names them as placeholders, such as {{.path}}, which
// utils.ResolvePlaceholderString fills.
type TranslationSet struct {
	ServerFileFlag       string
	UserConfigFlag       string
	Usage                string
	TerminalFailed       string
	UserConfigUnreadable string
	MoreThanOneValue     string

	ServerNotRunning        string
	CouldNotRead            string
	CouldNotListen          string
	ServerDidNotAnswer      string
	ServerAnsweredListening string
	CouldNotReadNotice      string
	StreamBroke             string
	StreamEnded             string
	CouldNotAsk             string
	CouldNotWriteChange     string
	CouldNotSendChange      string
	CouldNotReadAnswer      string
	ServerAnswered          string

	LinkHolds        string
	LinkConformsTo   string
	LinkInheritsFrom string
	LinkExtends      string
	LinkUses         string
	LinkTests        string
	GroupOf          string
	InPlace          string
	FileNoun         string
	TypeNoun         string
	Unchanged        string
	New              string
	Changed          string
	Removed          string

	CouldNotReadReviewAgain string
	StoppedHearing          string
	NoReviewOpenYet         string
	ReviewNotOpenYet        string
	HowToQuit               string
	NoPullRequestOnGitHub   string
	CouldNotOpen            string
	Opened                  string
	EveryPanel              string
	HomeTitle               string
	Marks                   string
	KeysTitle               string
	NotACommand             string
	OpeningReviewAgain      string
	LoadingReviewAgain      string

	HelpFocusPanels    string
	HelpFocusMain      string
	HelpEscape         string
	HelpScreenMode     string
	HelpCommandLine    string
	HelpKeys           string
	HelpRefresh        string
	HelpReviewView     string
	HelpHome           string
	HelpDiagram        string
	HelpFinder         string
	HelpGitHub         string
	HelpWeb            string
	HelpPullFilter     string
	HelpJumps          string
	HelpClose          string
	HelpQuit           string
	HelpPullsMove      string
	HelpPanelsRound    string
	HelpOpenReviewView string
	HelpHighlight      string
	HelpClearHighlight string
	HelpPostPull       string
	HelpFilesMove      string
	HelpPanels         string
	HelpOpenFile       string
	HelpFoldTree       string
	HelpNotesMove      string
	HelpOpenNote       string
	HelpEditDrop       string
	HelpPostComment    string
	HelpBoxesMove      string
	HelpPanelsToReview string
	HelpOpenDiagram    string
	HelpStepIn         string
	HelpStepOut        string
	HelpFollowBox      string
	HelpReferences     string
	HelpFindBox        string
	HelpLabelsKeys     string
	HelpLabels         string
	HelpBoxLinks       string
	HelpProperties     string
	HelpBoxDiff        string
	HelpAllLinks       string
	HelpItemsMove      string
	HelpSections       string
	HelpItemsEnds      string
	HelpOpenItem       string
	HelpFoldSection    string
	HelpPostAtCursor   string
	HelpLinesMove      string
	HelpPages          string
	HelpLineEnds       string
	HelpHunks          string
	HelpFiles          string
	HelpNotes          string
	HelpSearch         string
	HelpFold           string
	HelpFoldAll        string
	HelpToggleFile     string
	HelpSelect         string
	HelpComment        string
	HelpAsk            string
	HelpEditAtCursor   string
	HelpDropAtCursor   string
	HelpShowType       string
	HelpBack           string
	HelpSplit          string
	HelpWrap           string
	HelpWhole          string
	HelpRendered       string
	HelpBoxesNearest   string
	HelpBoxesOrder     string
	HelpBoxesEnds      string
	HelpFollow         string
	HelpLinks          string
	HelpHomeMove       string
	HelpHomeEnds       string
	HelpRepositories   string
	HelpFirstNineKeys  string
	HelpFirstNine      string
	HelpOpenPull       string
	HelpReadAgain      string
	HelpGitHubPull     string
	HelpWebList        string
	HelpReviewOpen     string
	HelpListMode       string
	HelpCommandQuit    string
	MarkWaiting        string
	MarkLoaded         string
	MarkStack          string
	Dim                string
	MarkDraft          string
	Keys               string
	ReviewView         string
	Home               string
	Split              string
	Wrap               string
	WholeFile          string
	Rendered           string
	Diagram            string
	Find               string
	Links              string
	Properties         string
	Diff               string
	EveryLink          string

	Visual             string
	OfTotal            string
	All                string
	Loading            string
	TryAgainOrQuit     string
	NothingToDraw      string
	MainTitle          string
	ReviewTitle        string
	FilesTitle         string
	NotesTitle         string
	DiagramTitle       string
	DiffTitle          string
	DropQuestion       string
	DropTitle          string
	DropFooter         string
	Drop               string
	AnyOtherKey        string
	Keep               string
	TypeKey            string
	Narrow             string
	Move               string
	Go                 string
	Close              string
	TryAgain           string
	Quit               string
	ALabelKey          string
	JumpToBox          string
	Stop               string
	Fold               string
	PullRequest        string
	Panels             string
	Open               string
	Edit               string
	Post               string
	In                 string
	Out                string
	Follow             string
	GrowShrink         string
	Review             string
	GitHub             string
	Highlight          string
	List               string
	Comment            string
	AskClaude          string
	OtherEnd           string
	EditDrop           string
	Sections           string
	SelectionInOneFile string
	Notes              string
	Back               string
	CommentWithAnswer  string
	FollowUp           string
	OnGitHub           string
	Hunks              string
	Select             string
	Search             string
	Added              string
	FileHeader         string
	HunkHeader         string
	UnchangedLines     string
	QueuedComment      string
	LinkNoun           string
	ThreadNoun         string
	ConversationNoun   string
	NotInDiff          string

	CommandLine      string
	WebApp           string
	Bigger           string
	Smaller          string
	NextPull         string
	PreviousPull     string
	JumpBack         string
	JumpForward      string
	FocusMain        string
	Down             string
	Up               string
	Top              string
	Bottom           string
	PreviousPanel    string
	NextPanel        string
	Left             string
	Right            string
	NextBox          string
	PreviousBox      string
	References       string
	SearchBack       string
	NextMatch        string
	PreviousMatch    string
	NextSection      string
	PreviousSection  string
	HalfPageDown     string
	HalfPageUp       string
	PageDown         string
	PageUp           string
	NextHunk         string
	PreviousHunk     string
	NextFile         string
	PreviousFile     string
	OpenFold         string
	CloseFold        string
	OpenFolds        string
	CloseFolds       string
	ToggleFile       string
	ShowType         string
	NextNote         string
	PreviousNote     string
	RepositoryNumber string
	Clear            string
	ChooseNumber     string
	QuitCommandKeys  string
	HelpRefreshKeys  string
	HelpLineEndsKeys string

	Stale            string
	QueuedHeading    string
	Claude           string
	Waiting          string
	OnGitHubHeading  string
	Resolved         string
	Outdated         string
	NothingOnYet     string
	NothingQueuedYet string
	Components       string
	ListAnd          string

	LineAt          string
	RemovedLineAt   string
	LinesAt         string
	RemovedLinesAt  string
	LineToLine      string
	DiagramNotDrawn string

	WrittenStale        string
	NoWrittenReview     string
	BranchReadOnly      string
	QueuedComments      string
	CommentNoun         string
	GitHubThreadsLoaded string
	ReviewThreadNoun    string
	Conversations       string
	TheChange           string
	ComponentNoun       string
	Base                string
	Head                string
	BranchHeader        string
	StackHeader         string
	ByAuthor            string
	OnBase              string
	MemberHead          string
	Combined            string
	ShowingOnly         string
	PullHeader          string
	ReviewMoved         string
	QueueMoved          string
	CommentGoesOn       string
	NoCommentsQueued    string
	OnDate              string
	PullEnded           string
	PullLeftStack       string
	PostedComments      string
	ApprovedWith        string
	Approved            string
	ShownAtLine         string
	MayHaveMoved        string
	TheWholeFile        string
	WasLine             string
	MessageNoun         string
	WaitingForClaude    string
	NoAnswer            string

	BinaryFile            string
	RenamedOnly           string
	ModeOnly              string
	PartialRendered       string
	Deleted               string
	Renamed               string
	UnchangedLineNoun     string
	WholeBadge            string
	RenderedBadge         string
	LineNoun              string
	YouSaid               string
	ClaudeDidNotAnswer    string
	ClaudeSaid            string
	MoreOnGitHub          string
	MoreCommentNoun       string
	AConversation         string
	AThreadOn             string
	QueuedNumber          string
	QueuedNoteKeys        string
	WaitingForClaudeWords string
	ConversationNoteKeys  string
	ThreadNoteKeys        string

	SelectStartsOnLine string
	NoRenderedView     string
	NoWholeText        string
	NoSuchLine         string
	NoMoreOf           string

	NoReviewOpen          string
	NoBoxHasLabel         string
	NoTypesToDraw         string
	NoFilesInChange       string
	LinkOut               string
	LinkIn                string
	NothingLinksTo        string
	HasNoLinks            string
	WhatLinksTo           string
	LinksToNothing        string
	LinksTo               string
	MemberNoun            string
	OnlyTypeHasProperties string
	NoStoredProperties    string
	PropertyNoun          string
	Inferred              string
	NothingIsCalled       string
	MatchOfFor            string
	NoTypeInFile          string
	TypesIn               string
	ShowTypeHow           string
	NothingToDrawHere     string
	PackageNoun           string
	InPulls               string
	Members               string
	Labels                string

	CommentFromAnswer       string
	CheckingWhere           string
	EditsQueuedComment      string
	CommentsOnLinesOrAnswer string
	CommentsOnLines         string
	CommentOn               string
	FollowUpOn              string
	ConversationSoFar       string
	You                     string
	DidNotAnswer            string
	AsksAboutLines          string
	QuestionAbout           string
	ClaudeAnswersHere       string
	EditsQueuedAtCursor     string
	QueuedCommentOn         string
	SavingReplaces          string
	DropsQueuedAtCursor     string
	KeptQueued              string
	Dropping                string
	CouldNotMakeFile        string
	NothingBelowLine        string
	CouldNotWrite           string
	EditorStopped           string
	CouldNotReadWritten     string
	NothingWritten          string
	QueuedAsItWas           string
	Queueing                string
	Saving                  string
	ClaudeIsWorking         string
	ClaudeIsWorkingOn       string
	Stopping                string
	StoppedClaude           string
	NotWorking              string
	StopsClaude             string
	StopClaude              string
	HelpStop                string
	RefusedStep             string
	TookSteps               string
	TookStepsRefused        string
	StepNoun                string
	ClaudeSaysWhy           string
	ClaudeAnswered          string
	SavedQueued             string
	DroppedQueued           string
	QueuedAbout             string

	AddComments             string
	ApproveWithComments     string
	Approve                 string
	NoQueueToPost           string
	ReviewMovedNoSend       string
	PostsAtCursor           string
	PullWasState            string
	CheckingThenSending     string
	SentToGitHub            string
	SentHowToClose          string
	NothingPosted           string
	NothingPostedHowToClose string
	PostTo                  string
	YesAction               string
	Cancel                  string
	Choose                  string
	ShowWhatItSends         string

	FinderTitle      string
	NothingMatches   string
	OnlyStackFilters string
	NoPullInStack    string

	ReadingAlready          string
	CouldNotReadGitHubAgain string
	OneLoadAtATime          string
	CouldNotLoad            string
	LoadedOpenFromHome      string
	DraftNotSent            string
	LoadAndOpen             string
	Repositories            string
	Refresh                 string
	ReadAgain               string
	LoadingFromGitHub       string
	ReadingGitHub           string

	PullRequestsTitle    string
	CouldNotReadList     string
	NoAccountsSet        string
	NotRead              string
	NotReadFromGitHub    string
	NoneOpen             string
	CouldNotReadPulls    string
	ReadingPulls         string
	AccountsTitle        string
	NameAccountsIn       string
	ConfigSample         string
	EachAccount          string
	CouldNotReadAs       string
	LeftOutUntil         string
	NothingInvolves      string
	ReadsGitHubAgain     string
	PullBy               string
	WaitingOnYou         string
	LoadedInLazyreview   string
	NotLoadedYet         string
	YouOpenedIt          string
	YouCommented         string
	ADraft               string
	LoadingFromGitHubNow string
	OpensStack           string
	OpensReview          string
	LoadsStack           string
	LoadsIt              string
	StackBottomFirst     string
	NoDescription        string

	LoadedElsewhere string
	LoadedAgain     string
}

// EnglishTranslationSet is the strings in English.
func EnglishTranslationSet() *TranslationSet {
	return &TranslationSet{
		ServerFileFlag:       "the lazyreview server's own record, server.json, holding its address and token",
		UserConfigFlag:       "the terminal app's part of lazyreview's config, as JSON: the config file's path and the editor",
		Usage:                "usage: lazyreview-terminal --server-file <server.json> --user-config <json> [<review id>]",
		TerminalFailed:       "lazyreview terminal",
		UserConfigUnreadable: "could not read the config handed over by lazyreview",
		MoreThanOneValue:     "it holds more than one value",

		ServerNotRunning:        "the lazyreview server is not running; lazyreview start starts it",
		CouldNotRead:            "could not read {{.path}}",
		CouldNotListen:          "could not listen for changes to {{.review}}",
		ServerDidNotAnswer:      "the lazyreview server did not answer",
		ServerAnsweredListening: "the server answered {{.status}} to listening for changes",
		CouldNotReadNotice:      "could not read a change notice",
		StreamBroke:             "the stream of changes broke",
		StreamEnded:             "the server ended the stream of changes",
		CouldNotAsk:             "could not ask the server for {{.path}}",
		CouldNotWriteChange:     "could not write the change to {{.path}}",
		CouldNotSendChange:      "could not send the change to {{.path}}",
		CouldNotReadAnswer:      "could not read the server's answer to {{.path}}",
		ServerAnswered:          "the server answered {{.status}}",

		LinkHolds:        "holds",
		LinkConformsTo:   "conforms to",
		LinkInheritsFrom: "inherits from",
		LinkExtends:      "extends",
		LinkUses:         "uses",
		LinkTests:        "tests",
		GroupOf:          "a group of {{.count}}",
		InPlace:          "in {{.place}}",
		FileNoun:         "file",
		TypeNoun:         "type",
		Unchanged:        "unchanged",
		New:              "new",
		Changed:          "changed",
		Removed:          "removed",

		CouldNotReadReviewAgain: "could not read the review again",
		StoppedHearing:          "Stopped hearing changes made elsewhere: {{.error}}. {{.refresh}} loads the review again.",
		NoReviewOpenYet:         "No review is open yet: {{.open}} opens one",
		ReviewNotOpenYet:        "the review is not open yet; {{.refresh}} tries again",
		HowToQuit:               "{{.quit}} or {{.command}}q quits",
		NoPullRequestOnGitHub:   "this has no pull request on GitHub",
		CouldNotOpen:            "could not open {{.name}}",
		Opened:                  "opened {{.address}}",
		EveryPanel:              "Every panel",
		HomeTitle:               "Home",
		Marks:                   "Marks",
		KeysTitle:               "Keys",
		NotACommand:             "not a command: :{{.command}}",
		OpeningReviewAgain:      "Opening the review again…",
		LoadingReviewAgain:      "Loading the review again…",

		HelpFocusPanels:    "the Review, Files, Notes or Diagram panel, from a panel on the left",
		HelpFocusMain:      "the main panel, on the right",
		HelpEscape:         "closes the top popup; with none, back to the panel on the left that the main panel shows",
		HelpScreenMode:     "more of the screen for the panel with the keys, then all of it; {{.smaller}} goes back",
		HelpCommandLine:    "the command line: {{.command}}q quits, {{.command}}q! quits and drops drafts",
		HelpKeys:           "lists the keys of this panel",
		HelpRefresh:        "loads the review again: from GitHub, or a branch from its repository",
		HelpReviewView:     "the Review view",
		HelpHome:           "the home screen: the open pull requests of your accounts",
		HelpDiagram:        "the diagram",
		HelpFinder:         "finds a file, type or component by name",
		HelpGitHub:         "opens the pull request, or the thread at the cursor, on GitHub",
		HelpWeb:            "opens this place in the web app",
		HelpPullFilter:     "on a stack, moves the Review panel's cursor to the next or previous pull request: All, then each one",
		HelpJumps:          "back and forward through jumps",
		HelpClose:          "closes the top popup",
		HelpQuit:           "quits, as {{.command}}q does",
		HelpPullsMove:      "the next, previous, first or last pull request, or All on a stack; the other panels show only what the cursor is on",
		HelpPanelsRound:    "the panel above or below; {{.previousPanel}} goes round to Diagram",
		HelpOpenReviewView: "the Review view, in the main panel",
		HelpHighlight:      "on a stack, highlights this pull request; {{.move}} grow or shrink the highlight, and the other panels show only what it holds",
		HelpClearHighlight: "goes to All, to show the whole stack",
		HelpPostPull:       "post: the actions offered for this pull request",
		HelpFilesMove:      "the next, previous, first or last file; the main panel shows its diff",
		HelpPanels:         "the panel above or below",
		HelpOpenFile:       "on a file, its diff in the main panel; on a component or a folder, folds it",
		HelpFoldTree:       "folds or unfolds a component or a folder",
		HelpNotesMove:      "the next, previous, first or last note; the main panel shows it in its diff",
		HelpOpenNote:       "the note in its diff, in the main panel",
		HelpEditDrop:       "edit or drop the queued comment",
		HelpPostComment:    "post: the actions offered for the comment's pull request",
		HelpBoxesMove:      "the next, previous, first or last box, in reading order",
		HelpPanelsToReview: "the panel above or below; {{.nextPanel}} goes round to Review",
		HelpOpenDiagram:    "the diagram in the main panel, which moves to the nearest box and steps in",
		HelpStepIn:         "steps in; on a type, opens its diff; on a group, lists its members",
		HelpStepOut:        "steps out",
		HelpFollowBox:      "follows a link of the box",
		HelpReferences:     "lists the boxes that link to it",
		HelpFindBox:        "find a type or file, forward or back",
		HelpLabelsKeys:     "{{.labels}} then a label",
		HelpLabels:         "jumps to the box with that label",
		HelpBoxLinks:       "lists the box's links, in and out",
		HelpProperties:     "lists its properties",
		HelpBoxDiff:        "opens its diff",
		HelpAllLinks:       "draws every link faintly, or stops",
		HelpItemsMove:      "the next or previous item: a section, a queued comment, a conversation or a link",
		HelpSections:       "the next or previous section",
		HelpItemsEnds:      "the first or last item",
		HelpOpenItem:       "opens the item in its diff; a component opens its diff",
		HelpFoldSection:    "folds or unfolds a section",
		HelpPostAtCursor:   "post: the actions offered for the pull request at the cursor",
		HelpLinesMove:      "the next or previous line",
		HelpPages:          "half a page and a page, down and up",
		HelpLineEnds:       "the top, the bottom, or line n of this file",
		HelpHunks:          "the next or previous hunk",
		HelpFiles:          "the next or previous file",
		HelpNotes:          "the next or previous note: a queued comment, a conversation or a colleague's thread",
		HelpSearch:         "search the text, forward or back",
		HelpFold:           "toggle, open or close the fold at the cursor: a file, a run of unchanged lines or a note",
		HelpFoldAll:        "open every fold, close every fold",
		HelpToggleFile:     "on a file's header, toggles the file",
		HelpSelect:         "selects lines; {{.otherEnd}} goes to the other end; {{.stop}} stops",
		HelpComment:        "comments on the line or the selection; on Claude's answer, uses it as the comment",
		HelpAsk:            "asks Claude about the line or the selection; in a conversation, asks a follow-up",
		HelpEditAtCursor:   "edits the queued comment at the cursor",
		HelpDropAtCursor:   "drops the queued comment at the cursor, after {{.confirm}} confirms",
		HelpShowType:       "shows the type at the cursor on the diagram; on a file's header, lists its types",
		HelpBack:           "goes back to the diagram or the Review view",
		HelpSplit:          "split or unified",
		HelpWrap:           "wraps lines, or stops",
		HelpWhole:          "the whole file, or back to the diff",
		HelpRendered:       "a Markdown file rendered, or back to the diff",
		HelpBoxesNearest:   "focus the nearest box left, down, up or right",
		HelpBoxesOrder:     "the next or previous box in reading order",
		HelpBoxesEnds:      "the first or last box",
		HelpFollow:         "follows a link of the focused box",
		HelpLinks:          "lists the focused box's links, in and out",
		HelpHomeMove:       "the next or previous pull request, on into the next or previous repository",
		HelpHomeEnds:       "the first or last pull request",
		HelpRepositories:   "the repository above or below, going round at the ends",
		HelpFirstNineKeys:  "{{.first}} to {{.last}}",
		HelpFirstNine:      "the first nine repositories",
		HelpOpenPull:       "opens the pull request's review; one no review holds yet is loaded from GitHub first",
		HelpReadAgain:      "reads the open pull requests from GitHub again",
		HelpGitHubPull:     "opens the pull request on GitHub",
		HelpWebList:        "opens the web app's list of reviews",
		HelpReviewOpen:     "the Review view of the review open",
		HelpListMode:       "more of the screen for the list, then all of it; {{.smaller}} goes back",
		HelpCommandQuit:    "the command line: {{.command}}q quits",
		MarkWaiting:        "your review is waiting: asked of you, or of your team",
		MarkLoaded:         "loaded in lazyreview",
		MarkStack:          "a stack, bottom first, reviewed as one",
		Dim:                "dim",
		MarkDraft:          "a draft",
		Keys:               "Keys",
		ReviewView:         "Review view",
		Home:               "Home",
		Split:              "Split",
		Wrap:               "Wrap",
		WholeFile:          "Whole file",
		Rendered:           "Rendered",
		Diagram:            "Diagram",
		Find:               "Find",
		Links:              "Links",
		Properties:         "Properties",
		Diff:               "Diff",
		EveryLink:          "Every link",

		Visual:             "VISUAL",
		OfTotal:            "{{.at}} of {{.total}}",
		All:                "All",
		Loading:            "Loading…",
		TryAgainOrQuit:     "{{.refresh}} tries again; {{.command}}q quits.",
		NothingToDraw:      "Nothing to draw: this change has no parts the diagram knows.",
		MainTitle:          "Main",
		ReviewTitle:        "Review",
		FilesTitle:         "Files",
		NotesTitle:         "Notes",
		DiagramTitle:       "Diagram",
		DiffTitle:          "Diff",
		DropQuestion:       "Drop queued comment {{.comment}} on #{{.pull}}? Its text is gone once dropped.",
		DropTitle:          "Drop comment {{.comment}}",
		DropFooter:         "{{.confirm}} drop · any other key keeps it",
		Drop:               "Drop",
		AnyOtherKey:        "any other key",
		Keep:               "Keep",
		TypeKey:            "type",
		Narrow:             "Narrow",
		Move:               "Move",
		Go:                 "Go",
		Close:              "Close",
		TryAgain:           "Try again",
		Quit:               "Quit",
		ALabelKey:          "a label",
		JumpToBox:          "Jump to its box",
		Stop:               "Stop",
		Fold:               "Fold",
		PullRequest:        "Pull request",
		Panels:             "Panels",
		Open:               "Open",
		Edit:               "Edit",
		Post:               "Post",
		In:                 "In",
		Out:                "Out",
		Follow:             "Follow",
		GrowShrink:         "Grow, shrink",
		Review:             "Review",
		GitHub:             "GitHub",
		Highlight:          "Highlight",
		List:               "List",
		Comment:            "Comment",
		AskClaude:          "Ask Claude",
		OtherEnd:           "Other end",
		EditDrop:           "Edit, drop",
		Sections:           "Sections",
		SelectionInOneFile: "a selection stays in one file",
		Notes:              "Notes",
		Back:               "Back",
		CommentWithAnswer:  "Comment with this answer",
		FollowUp:           "Follow-up",
		OnGitHub:           "On GitHub",
		Hunks:              "Hunks",
		Select:             "Select",
		Search:             "Search",
		Added:              "added",
		FileHeader:         "file header",
		HunkHeader:         "hunk header",
		UnchangedLines:     "unchanged lines",
		QueuedComment:      "queued comment {{.comment}}",
		LinkNoun:           "link",
		ThreadNoun:         "thread",
		ConversationNoun:   "conversation",
		NotInDiff:          "{{.path}} is not in this change's diff",

		CommandLine:      "Command line",
		WebApp:           "Web app",
		Bigger:           "Bigger",
		Smaller:          "Smaller",
		NextPull:         "Next pull request",
		PreviousPull:     "Previous pull request",
		JumpBack:         "Jump back",
		JumpForward:      "Jump forward",
		FocusMain:        "Main panel",
		Down:             "Down",
		Up:               "Up",
		Top:              "Top",
		Bottom:           "Bottom",
		PreviousPanel:    "Previous panel",
		NextPanel:        "Next panel",
		Left:             "Left",
		Right:            "Right",
		NextBox:          "Next box",
		PreviousBox:      "Previous box",
		References:       "References",
		SearchBack:       "Search back",
		NextMatch:        "Next match",
		PreviousMatch:    "Previous match",
		NextSection:      "Next section",
		PreviousSection:  "Previous section",
		HalfPageDown:     "Half a page down",
		HalfPageUp:       "Half a page up",
		PageDown:         "Page down",
		PageUp:           "Page up",
		NextHunk:         "Next hunk",
		PreviousHunk:     "Previous hunk",
		NextFile:         "Next file",
		PreviousFile:     "Previous file",
		OpenFold:         "Open fold",
		CloseFold:        "Close fold",
		OpenFolds:        "Open every fold",
		CloseFolds:       "Close every fold",
		ToggleFile:       "Toggle file",
		ShowType:         "Show type",
		NextNote:         "Next note",
		PreviousNote:     "Previous note",
		RepositoryNumber: "Repository {{.number}}",
		Clear:            "Clear",
		ChooseNumber:     "Choose {{.number}}",
		QuitCommandKeys:  "{{.command}}q",
		HelpRefreshKeys:  "{{.refresh}} {{.command}}refresh",
		HelpLineEndsKeys: "{{.top}} {{.bottom}} {n}{{.bottom}}",

		Stale:            "stale",
		QueuedHeading:    "Queued",
		Claude:           "Claude",
		Waiting:          "waiting",
		OnGitHubHeading:  "On GitHub",
		Resolved:         "resolved",
		Outdated:         "outdated",
		NothingOnYet:     "Nothing on {{.pulls}} yet",
		NothingQueuedYet: "Nothing queued or asked yet",
		Components:       "Components",
		ListAnd:          "{{.list}} and {{.last}}",

		LineAt:          "line {{.line}}",
		RemovedLineAt:   "removed line {{.line}}",
		LinesAt:         "lines {{.from}}-{{.to}}",
		RemovedLinesAt:  "removed lines {{.from}}-{{.to}}",
		LineToLine:      "{{.from}} to {{.to}}",
		DiagramNotDrawn: "A diagram is here that a terminal cannot draw. {{.web}} opens this review in the web app.",

		WrittenStale:        "This written review read head {{.head}}; the pull request has moved on since, so it may not match the diff.",
		NoWrittenReview:     "No written review is attached. The diagram and the diffs hold the change.",
		BranchReadOnly:      "This branch has no pull request yet, so its review is read only.",
		QueuedComments:      "Queued comments",
		CommentNoun:         "comment",
		GitHubThreadsLoaded: "{{.threads}}, {{.open}} unresolved, as GitHub had them when this review was loaded.",
		ReviewThreadNoun:    "review thread",
		Conversations:       "Conversations",
		TheChange:           "The change",
		ComponentNoun:       "component",
		Base:                "Base {{.ref}}",
		Head:                "Head {{.sha}}",
		BranchHeader:        "Branch {{.branch}} in {{.repo}}, with no pull request yet",
		StackHeader:         "A stack of {{.count}} pull requests, each built on the one before, reviewed together",
		ByAuthor:            "by {{.author}}",
		OnBase:              "on {{.base}}",
		MemberHead:          "head {{.sha}}",
		Combined:            "Combined:",
		ShowingOnly:         "Showing {{.pulls}} only. {{.close}} in the Review panel shows the whole stack.",
		PullHeader:          "Pull request #{{.pull}} by {{.author}}",
		ReviewMoved:         "This review moved: its pull requests are reviewed together on {{.to}}. Its queued comments moved there, and nothing can be sent from here.",
		QueueMoved:          "This review moved to {{.to}}. Its queued comments moved there.",
		CommentGoesOn:       "A comment goes on the pull request that last changed its line, shown as → #number.",
		NoCommentsQueued:    "No comments queued.",
		OnDate:              "on {{.date}}",
		PullEnded:           "#{{.pull}} was {{.state}}{{.at}}. Nothing can be sent to it.",
		PullLeftStack:       "#{{.pull}} left this stack. Nothing can be sent to it from here.",
		PostedComments:      "Posted {{.comments}}",
		ApprovedWith:        "Approved with {{.comments}}",
		Approved:            "Approved",
		ShownAtLine:         "shown at line {{.line}}",
		MayHaveMoved:        "Its line may have moved: {{.reason}}. Nothing sends it until it is queued again or dropped.",
		TheWholeFile:        "the whole file",
		WasLine:             "was line {{.line}}",
		MessageNoun:         "message",
		WaitingForClaude:    "Claude is working…",
		NoAnswer:            "No answer",

		BinaryFile:            "A binary file. Its contents are not shown.",
		RenamedOnly:           "Renamed, with no change to its contents.",
		ModeOnly:              "No change to its contents, only to its mode.",
		PartialRendered:       "Only the changed parts show: the page kept no whole text of this file.",
		Deleted:               "deleted",
		Renamed:               "renamed",
		UnchangedLineNoun:     "unchanged line",
		WholeBadge:            "whole",
		RenderedBadge:         "rendered",
		LineNoun:              "line",
		YouSaid:               "You: {{.text}}",
		ClaudeDidNotAnswer:    "Claude did not answer",
		ClaudeSaid:            "Claude: {{.text}}",
		MoreOnGitHub:          "{{.comments}} on GitHub",
		MoreCommentNoun:       "more comment",
		AConversation:         "a conversation with Claude",
		AThreadOn:             "a thread on #{{.pull}}",
		QueuedNumber:          "Queued {{.comment}}",
		QueuedNoteKeys:        "{{.edit}} edit  {{.drop}} drop",
		WaitingForClaudeWords: "waiting for Claude",
		ConversationNoteKeys:  "{{.ask}} ask  {{.fold}}",
		ThreadNoteKeys:        "{{.github}} GitHub  {{.fold}}",

		SelectStartsOnLine: "{{.select}} starts a selection on a line",
		NoRenderedView:     "{{.rendered}} shows a Markdown file rendered, and this file has no rendered view",
		NoWholeText:        "this file has no whole text to show",
		NoSuchLine:         "{{.path}} shows no line {{.line}}",
		NoMoreOf:           "no more of {{.query}}",

		NoReviewOpen:          "no review is open; {{.open}} on one opens it",
		NoBoxHasLabel:         "no box has the label {{.label}}",
		NoTypesToDraw:         "{{.title}} has no types to draw; {{.boxDiff}} opens its diff",
		NoFilesInChange:       "{{.title}} has no files in this change",
		LinkOut:               "out",
		LinkIn:                "in",
		NothingLinksTo:        "nothing links to {{.title}}",
		HasNoLinks:            "{{.title}} has no links",
		WhatLinksTo:           "What links to {{.name}}",
		LinksToNothing:        "{{.title}} links to nothing",
		LinksTo:               "{{.name}} links to",
		MemberNoun:            "member",
		OnlyTypeHasProperties: "only a type has properties",
		NoStoredProperties:    "{{.name}} has no stored properties",
		PropertyNoun:          "property",
		Inferred:              "inferred",
		NothingIsCalled:       "no type or file is called {{.query}}",
		MatchOfFor:            "{{.at}} of {{.total}} for {{.query}}",
		NoTypeInFile:          "{{.file}} holds no type the diagram draws",
		TypesIn:               "Types in {{.file}}",
		ShowTypeHow:           "{{.showType}} shows the type at a line, or lists a file's types from its header",
		NothingToDrawHere:     "nothing to draw",
		PackageNoun:           "package",
		InPulls:               "in {{.pulls}}",
		Members:               "Members",
		Labels:                "Labels",

		CommentFromAnswer:       "A comment on {{.on}}, from Claude's answer.",
		CheckingWhere:           "Checking where it goes…",
		EditsQueuedComment:      "{{.edit}} edits a queued comment",
		CommentsOnLinesOrAnswer: "{{.comment}} comments on a line or a selection, or on Claude's answer uses it",
		CommentsOnLines:         "{{.comment}} comments on a line or a selection",
		CommentOn:               "A comment on {{.on}}.",
		FollowUpOn:              "A follow-up to Claude on {{.on}}.",
		ConversationSoFar:       "The conversation so far:",
		You:                     "You",
		DidNotAnswer:            "did not answer",
		AsksAboutLines:          "{{.ask}} asks Claude about a line or a selection, or in a conversation asks a follow-up",
		QuestionAbout:           "A question to Claude about {{.on}}.",
		ClaudeAnswersHere:       "Claude answers here, in the review. Nothing goes to GitHub.",
		EditsQueuedAtCursor:     "{{.edit}} edits the queued comment at the cursor",
		QueuedCommentOn:         "Queued comment {{.comment}} on {{.on}}, for #{{.pull}}.",
		SavingReplaces:          "Saving replaces it.",
		DropsQueuedAtCursor:     "{{.drop}} drops the queued comment at the cursor",
		KeptQueued:              "Kept queued comment {{.comment}}.",
		Dropping:                "Dropping…",
		CouldNotMakeFile:        "could not make a file to write in",
		NothingBelowLine:        "Nothing below the line above is sent. Saving with no text sends nothing.",
		CouldNotWrite:           "could not write {{.path}}",
		EditorStopped:           "the editor stopped",
		CouldNotReadWritten:     "could not read what was written",
		NothingWritten:          "Nothing was written, so nothing was sent.",
		QueuedAsItWas:           "Queued comment {{.comment}} is as it was.",
		Queueing:                "Queueing…",
		Saving:                  "Saving…",
		ClaudeIsWorking:         "Claude is working",
		ClaudeIsWorkingOn:       "Claude is working: {{.step}}",
		Stopping:                "Stopping Claude…",
		StoppedClaude:           "Stopped Claude.",
		NotWorking:              "Claude is not working on this conversation.",
		StopsClaude:             "{{.stop}} stops Claude on a conversation it is answering",
		StopClaude:              "Stop Claude",
		HelpStop:                "stops Claude answering the conversation at the cursor",
		RefusedStep:             "Refused: {{.step}} ({{.why}})",
		TookSteps:               "Took {{.steps}}.",
		TookStepsRefused:        "Took {{.steps}}, {{.refused}} refused.",
		StepNoun:                "step",
		ClaudeSaysWhy:           "Claude did not answer; the conversation says why.",
		ClaudeAnswered:          "Claude answered.",
		SavedQueued:             "Saved queued comment {{.comment}}.",
		DroppedQueued:           "Dropped queued comment {{.comment}}.",
		QueuedAbout:             "Queued: {{.about}}",

		AddComments:             "Add Comments",
		ApproveWithComments:     "Approve With Comments",
		Approve:                 "Approve",
		NoQueueToPost:           "only a pull request's review has a queue to post",
		ReviewMovedNoSend:       "this review moved, so nothing can be sent from here",
		PostsAtCursor:           "{{.post}} posts for the pull request at the cursor; move to it, or to one of its comments",
		PullWasState:            "#{{.pull}} was {{.state}}, so nothing can be sent to it",
		CheckingThenSending:     "Checking the pull request with GitHub, then sending…",
		SentToGitHub:            "Sent to GitHub",
		SentHowToClose:          "Sent; {{.close}} closes the server's answer.",
		NothingPosted:           "Nothing was posted",
		NothingPostedHowToClose: "Nothing was posted; {{.close}} closes why.",
		PostTo:                  "Post to {{.repo}}#{{.pull}}",
		YesAction:               "Yes, {{.action}}",
		Cancel:                  "Cancel",
		Choose:                  "Choose",
		ShowWhatItSends:         "Show what it sends",

		FinderTitle:      "Find a file, type or component",
		NothingMatches:   "nothing matches",
		OnlyStackFilters: "only a stack has pull requests to filter by",
		NoPullInStack:    "no pull request #{{.pull}} in this stack",

		ReadingAlready:          "Reading GitHub already",
		CouldNotReadGitHubAgain: "Could not read GitHub again",
		OneLoadAtATime:          "One load at a time: {{.name}} is loading",
		CouldNotLoad:            "Could not load {{.name}}",
		LoadedOpenFromHome:      "Loaded {{.name}}; {{.open}} on the home screen opens it",
		DraftNotSent:            "Your draft in {{.title}} is not sent yet: {{.review}} goes back to it",
		LoadAndOpen:             "Load and open",
		Repositories:            "Repositories",
		Refresh:                 "Refresh",
		ReadAgain:               "Read again",
		LoadingFromGitHub:       "Loading {{.name}} from GitHub…",
		ReadingGitHub:           "Reading GitHub…",

		PullRequestsTitle:    "Pull requests",
		CouldNotReadList:     "Could not read the list",
		NoAccountsSet:        "No accounts are set",
		NotRead:              "not read",
		NotReadFromGitHub:    "Not read from GitHub",
		NoneOpen:             "None open",
		CouldNotReadPulls:    "Could not read the open pull requests",
		ReadingPulls:         "Reading the open pull requests from GitHub…",
		AccountsTitle:        "Accounts",
		NameAccountsIn:       "No GitHub accounts are set, so there is nothing to list yet. Name them in {{.path}}:",
		ConfigSample:         "github:\n  accounts:\n    - login: your-login\n    - login: your-work-login\n      teams:\n        - name: your-org/your-team\n          since: 2026-10-01",
		EachAccount:          "Each account is read by its own token, so each must be signed in to gh. A review asked of a team counts as asked of you, for pull requests opened on or after its since date. {{.refresh}} reads the config again.",
		CouldNotReadAs:       "GitHub could not be read as {{.login}}:",
		LeftOutUntil:         "Its pull requests are left out until GitHub answers. {{.refresh}} tries again.",
		NothingInvolves:      "No open pull request involves {{.logins}}: none asks for a review from them or their teams, none is theirs, and none has their comments.",
		ReadsGitHubAgain:     "{{.refresh}} reads GitHub again.",
		PullBy:               "#{{.pull}} by {{.author}}",
		WaitingOnYou:         "Waiting on your review, asked of {{.who}}.",
		LoadedInLazyreview:   "Loaded in lazyreview.",
		NotLoadedYet:         "Not loaded in lazyreview yet.",
		YouOpenedIt:          "You opened it.",
		YouCommented:         "You have commented on it.",
		ADraft:               "A draft, not ready for review yet.",
		LoadingFromGitHubNow: "Loading from GitHub…",
		OpensStack:           "{{.open}} opens the stack's review.",
		OpensReview:          "{{.open}} opens its review.",
		LoadsStack:           "{{.open}} loads the stack, {{.pulls}}, as {{.account}}, then opens it.",
		LoadsIt:              "{{.open}} loads it as {{.account}}, then opens it.",
		StackBottomFirst:     "The stack, bottom first:",
		NoDescription:        "No description.",

		LoadedElsewhere: "This review was loaded again. {{.refresh}} loads it here; your drafts stay.",
		LoadedAgain:     "Loaded again.",
	}
}
