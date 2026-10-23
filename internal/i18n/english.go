// Package i18n holds every string the terminal app shows, in English.
package i18n

// TranslationSet is every string the terminal app shows. A string with
// values names them as placeholders, such as {{.path}}, which
// utils.ResolvePlaceholderString fills.
type TranslationSet struct {
	DataHomeFlag     string
	ClaudeFlag       string
	ModelFlag        string
	Usage            string
	MoreThanOneValue string
	NoDataHome       string

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
	NoDrawings              string
	CouldNotWriteDrawings   string
	DrawingsOf              string
	DrawingsOpened          string
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
	HelpDrawings       string
	HelpPullFilter     string
	HelpJumps          string
	HelpCount          string
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
	HelpSelectFiles    string
	HelpStopSelection  string
	HelpFoldTree       string
	HelpNotesMove      string
	HelpOpenNote       string
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
	HelpHalfPages      string
	HelpPages          string
	HelpScrollCode     string
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
	HelpBoxesAlong     string
	HelpBoxesDown      string
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
	HelpReviewOpen     string
	HelpListMode       string
	HelpCommandQuit    string
	HelpHide           string
	HelpShowHidden     string
	MarkWaiting        string
	MarkLoaded         string
	MarkStack          string
	Dim                string
	MarkDraft          string
	HiddenTag          string
	MarkHidden         string
	MarkHiddenShownBy  string
	Keys               string
	ReviewView         string
	Home               string
	Split              string
	Wrap               string
	ScrollLeft         string
	ScrollRight        string
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
	DeleteQuestion     string
	DeleteTitle        string
	DeleteFooter       string
	DeleteStopsClaude  string
	DeleteGone         string
	Delete             string
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
	Branch             string
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
	Drawings         string
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
	HideRepository   string
	ShowHidden       string
	Clear            string
	ChooseNumber     string
	QuitCommandKeys  string
	HelpRefreshKeys  string
	HelpLineEndsKeys string
	CountKeys        string

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
	DrawingNotShown string

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
	BranchStackHeader   string
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
	DropsAtCursor           string
	KeptQueued              string
	KeptConversation        string
	Dropping                string
	Deleting                string
	StoppingThenDeleting    string
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
	DeletedConversation     string
	QueuedOn                string
	QueuedAnswerOn          string

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
	NoBranchInStack  string

	ReadingAlready          string
	CouldNotReadGitHubAgain string
	OneLoadAtATime          string
	LoadedOpenFromHome      string
	DraftNotSent            string
	LoadAndOpen             string
	Repositories            string
	Refresh                 string
	ReadAgain               string
	LoadingFromGitHub       string
	ReadingGitHub           string
	HideTitle               string
	HideThis                string
	HideOwner               string
	HiddenByTitle           string
	ShowOwnerAgain          string
	HidRepo                 string
	HidOwner                string
	ShowsAgain              string
	OwnerShowsAgain         string
	NoneHidden              string
	CouldNotChangeHidden    string
	OneHidden               string
	ManyHidden              string
	ShowingOneHidden        string
	ShowingManyHidden       string
	ShowsIt                 string
	ShowsThem               string
	HidesIt                 string
	HidesThem               string

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
	EveryOneHidden       string
	AllHidden            string
	ShowKeyShowsThem     string
	HiddenFromHome       string
	HideKeyShowsIt       string
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

	LoadedElsewhere   string
	LoadedAgain       string
	LoadedAgainNoting string
	LoadedNoting      string
	NoteNoun          string

	NotAPull                  string
	NotAPullAddress           string
	NotABranch                string
	NotABranchID              string
	NotAReviewID              string
	GaveUpWaiting             string
	HeldElsewhere             string
	NoBuild                   string
	LoadedAgainSince          string
	NotWritten                string
	Unreadable                string
	NoFile                    string
	NotAFileOfReview          string
	NoSuchReview              string
	ReviewNone                string
	NotInCombinedDiff         string
	StackPlace                string
	StackPlaceOf              string
	ChangedIn                 string
	LastChanged               string
	BranchChangedIn           string
	BranchLastChanged         string
	BranchNotTraced           string
	BranchUnchanged           string
	NoPullTakes               string
	NoneShows                 string
	OneShows                  string
	SeveralShow               string
	And                       string
	WrittenNoHeader           string
	WrittenHeaderLine         string
	WrittenUnknownField       string
	WrittenNeedsHead          string
	WrittenPullOrStack        string
	WrittenPullNumber         string
	WrittenStackNumbers       string
	WrittenTextBefore         string
	WrittenNoSection          string
	WrittenEmptySection       string
	WrittenProblems           string
	WrittenOtherPulls         string
	WrittenOtherHead          string
	ProblemNoun               string
	LineWord                  string
	RemovedLineWord           string
	OneLine                   string
	LineToLine2               string
	LinesRange                string
	BranchTakesNone           string
	ReviewMovedHere           string
	NotInChange               string
	NotADiffLine              string
	LeavesHunk                string
	Backwards                 string
	NotThisReview             string
	OtherHeads                string
	LastChangedBy             string
	NotShownBy                string
	NotAllTo                  string
	NotAllToOne               string
	ChooseAmong               string
	LeftStack                 string
	PullEndedHere             string
	GoesOn                    string
	GoesOnOf                  string
	PressComment              string
	PressApproveWithComments  string
	PressApprove              string
	PressApproveKeeps         string
	ThisCommentGoes           string
	TheseCommentsGo           string
	Stays                     string
	Stay                      string
	QueuedCommentNoun         string
	EmptyText                 string
	NotOnDiff                 string
	StartsOff                 string
	AcrossHunks               string
	StartsAfter               string
	RemovedWord               string
	QueueReviewMoved          string
	NotInStackAnyMore         string
	NotInReview               string
	NoVersion                 string
	ChangedSince              string
	NoComment                 string
	NotQueued                 string
	NotChanged                string
	NotDropped                string
	NothingWasPosted          string
	HeadMoved                 string
	CommentAt                 string
	CommentAtOf               string
	Placed                    string
	NotAnAction               string
	NotOffered                string
	OtherSet                  string
	OtherAccount              string
	AlreadyDone               string
	OwnPull                   string
	ReadBack                  string
	PostedComment             string
	PostedApproveWithComments string
	PostedApprove             string
	PullIsEnded               string
	HeadMovedFrom             string
	CommentNamed              string
	CommentProblem            string
	CommentText               string
	CommentStale              string
	GitHubNoToken             string
	GitHubTooSlow             string
	GitHubUnreachable         string
	GitHubRefused             string
	GitHubUnreadable          string

	AskSystem            string
	AskPull              string
	AskPullDescription   string
	AskStack             string
	AskStackMember       string
	AskStackOwner        string
	AskBranch            string
	AskBranchTitle       string
	AskBranchStack       string
	AskCodebase          string
	AskThisChange        string
	AskChanges           string
	AskChangedFile       string
	AskChangesMore       string
	AskFile              string
	AskFileAt            string
	AskAdded             string
	AskDeleted           string
	AskRenamed           string
	AskChanged           string
	AskUnchanged         string
	AskAbout             string
	AskAboutChangedBy    string
	AskThere             string
	AskDiff              string
	AskHead              string
	AskHeadOf            string
	AskHeadWindow        string
	AskConversation      string
	AskYouAnswered       string
	AskTheyAsked         string
	AskQuestion          string
	AskFollowUp          string
	StepRead             string
	StepSearched         string
	StepSearchedIn       string
	StepListed           string
	StepRan              string
	StepEdited           string
	StepWrote            string
	StepFetched          string
	StepSearchedWeb      string
	StepHelper           string
	StepSkill            string
	StepPlanned          string
	StepMcp              string
	StepUsed             string
	AskFirst             string
	AskNeedsLine         string
	NotAsked             string
	NoConversation       string
	NoCodebase           string
	StillWorking         string
	NotDeleted           string
	DidNotStop           string
	AnswerEnded          string
	ClaudeTooSlow        string
	ClaudeStopped        string
	ClaudeNotApproved    string
	ClaudeSandboxed      string
	ClaudeNoAnswer       string
	ClaudeExitCode       string
	ClaudeNotFound       string
	ClaudeCouldNotAnswer string
	ClaudeTooManyTurns   string
	ClaudeNothing        string
	GuardMarker          string
	GuardOutside         string
	GuardPush            string
	GuardPost            string
	GuardPublish         string
	GuardElsewhere       string
	GuardGh              string
	GuardMcp             string
	GuardTool            string
	GuardFailed          string
	GitFailed            string
	CommitMissing        string

	ConfigAboutDataHome             string
	ConfigAboutClaudeCommand        string
	ConfigAboutClaudeModel          string
	ConfigAboutEditor               string
	ConfigAboutAccounts             string
	ConfigAboutHiddenRepositories   string
	ConfigAboutLineNumbers          string
	ConfigAboutTheme                string
	ConfigAboutKeys                 string
	ConfigKeysUniversal             string
	ConfigKeysHome                  string
	ConfigKeysReviewScreen          string
	ConfigKeysPanels                string
	ConfigKeysReviewView            string
	ConfigKeysDiagram               string
	ConfigKeysMenu                  string
	ConfigAccountsExample           string
	ConfigHiddenRepositoriesExample string
	ConfigFileMissing               string
	ConfigUnreadable                string
	ConfigYAMLError                 string
	ConfigDuplicatedKey             string
	ConfigProblem                   string
	ConfigProblems                  string
	ConfigListedProblem             string
	ConfigNotAMap                   string
	ConfigUnknown                   string
	ConfigRetired                   string
	ConfigRetiredDiagramPanelMove   string
	ConfigUnusedWeb                 string
	ConfigNotASection               string
	ConfigNotAFolder                string
	ConfigNotACommand               string
	ConfigNotText                   string
	ConfigNotAChoice                string
	ConfigNotAccounts               string
	ConfigNotRepositories           string
	ConfigNotARepository            string
	ConfigNotAnEntry                string
	ConfigNotAColour                string
	ConfigNotColours                string
	ConfigNotKeys                   string
	ConfigNotNumberedKeys           string
	ConfigKeyNotText                string
	ConfigKeyEmpty                  string
	ConfigKeyUnclosed               string
	ConfigKeyDisabledAlone          string
	ConfigKeyUnknown                string
	ConfigNoLogin                   string
	ConfigAccountKey                string
	ConfigLoginTwice                string
	ConfigTeamsNotList              string
	ConfigTeamNotNamed              string
	ConfigTeamKey                   string
	ConfigSinceNotDate              string
	ConfigNotMoved                  string
	ConfigSchemaTitle               string
	ConfigSchemaDescription         string
	ConfigDocsGenerated             string
	ConfigDocsNoMarkers             string
	ConfigDocsVariable              string
	ConfigDocsOverrides             string
	ConfigCannotChange              string
	ConfigEditFailed                string

	ConfigUnusedLine         string
	ConfigSourceDefault      string
	ConfigSourceFile         string
	ConfigMovedHeader        string
	ConfigExists             string
	ConfigMissing            string
	ConfigChecked            string
	ConfigUnset              string
	ConfigMoveWouldWrite     string
	ConfigMoveFrom           string
	ConfigMoveWouldRename    string
	ConfigMoveRefused        string
	ConfigMoveNothingWritten string
	ConfigMoveNothing        string
	ConfigMoveMovedExists    string
	ConfigMoveNeedsAccounts  string
	ConfigMoveFailed         string
	ConfigMoveFailedStuck    string
	ConfigMoveWrote          string
	ConfigMoveRenamed        string
	ConfigMoveStillSet       string

	NotACommit            string
	LoaderGiveLinks       string
	LoaderOtherRepository string
	LoaderNamedTwice      string
	LoaderNotAStack       string
	LoaderNotOnBelow      string
	LoaderAlsoReviews     string
	LoaderThreadsUnread   string
	LoaderStackTitle      string
	LoaderMovedLoadThat   string
	LoaderNotABranch      string
	LoaderGiveHead        string
	LoaderBranchTwice     string
	LoaderBranchesTitle   string
	LoaderNoDifference    string
	LoaderLeftOut         string
	LoaderLargeFile       string
	StackHeadMissing      string
	StackNotBuiltOn       string
	StackBranchNotBuiltOn string
	StackNotAdded         string
	StackNotRemoved       string
	StackBranchNotAdded   string
	StackBranchNotRemoved string

	CommitNoun                string
	ChangeOn                  string
	ChangeHead                string
	ChangeMergeBase           string
	ChangeCommits             string
	SubjectPull               string
	SubjectStack              string
	SubjectTo                 string
	SubjectBranch             string
	DiagramRepositoryRoot     string
	DiagramUnchangedCode      string
	DiagramOtherFiles         string
	DiagramContextDescription string
	DiagramTopLevel           string
	DiagramGroup              string
	DiagramConforming         string
	DiagramInheriting         string
	DiagramEachConforms       string
	DiagramEachInherits       string
	DiagramShows              string
	DiagramTests              string
	DiagramFile               string
	DiagramExtension          string
	DiagramExtensionOf        string
	DiagramExtensionName      string
	DiagramSummaryLabel       string
	DiagramChangedFiles       string
	DiagramRemovedLines       string
	DiagramUnchangedFiles     string
	DiagramOpenDiff           string

	ComponentsProblems       string
	ComponentsRoot           string
	ComponentsContextTypes   string
	ComponentsNamed          string
	ComponentsNumbered       string
	ComponentsComponent      string
	ComponentsPackage        string
	ComponentsArea           string
	ComponentsGroup          string
	ComponentsPackageOf      string
	ComponentsLink           string
	ComponentsSummary        string
	ComponentsNotAnObject    string
	ComponentsUnknownKey     string
	ComponentsNeeds          string
	ComponentsNotText        string
	ComponentsBadID          string
	ComponentsIDTwice        string
	ComponentsNotAList       string
	ComponentsNotStrings     string
	ComponentsBadColor       string
	ComponentsBadContext     string
	ComponentsNeedsPath      string
	ComponentsNeedsPaths     string
	ComponentsNeedsComponent string
	ComponentsInTwoAreas     string
	ComponentsBadKind        string
	ComponentsBadMax         string
	ComponentsAreaUnknown    string
	ComponentsSummaryUnknown string

	GitHubThreadsRefused string
	GitHubNoThreads      string
	QueueStillOpen       string
	QueueLeftTheStack    string
	QueueRenumbered      string
	QueueStaleHead       string
	QueueStaleStart      string
	QueueMarkedStale     string

	CliFailed             string
	CliUnknownCommand     string
	CliUnknownOption      string
	CliNeedsValue         string
	CliNeedsArguments     string
	CliTakesArguments     string
	CliNoArguments        string
	ArgumentNoun          string
	CliNeedsRepo          string
	CliRepoNeeds          string
	CliLoadNeedsSomething string
	CliNotBoth            string
	CliAccountForPulls    string
	CliLoaded             string
	CliOpen               string
	CliNote               string
	CliNotLoaded          string
	CliReviewMoved        string
	CliAttached           string
	CliNoReviews          string
	CliListedMoved        string
	CliListed             string
	CliBranchName         string
	CliNoServer           string
	CliAnswersStopped     string
	AnswerNoun            string
	CliOldServerStopped   string
	CliNothingRunning     string
	CliLoading            string
	CliNoReview           string
	CliNotALink           string
	CliConfigUnreadable   string
	CliConfigNeeds        string
	CliConfigNeedsNot     string
	CliCheckIsForMove     string

	LoaderNotFetched        string
	LoaderPullHead          string
	LoaderBaseAt            string
	NothingLoadedHowToClose string
	LoadAgainTitle          string
	LoadTitle               string

	ConfigUnusedServer string
}

// EnglishTranslationSet is the strings in English.
func EnglishTranslationSet() *TranslationSet {
	return &TranslationSet{
		DataHomeFlag:     "the data home that holds the review",
		ClaudeFlag:       "the Claude Code command to ask",
		ModelFlag:        "the model Claude answers with, over the user's default",
		Usage:            "usage: lazyreview <command> [options]\n\n  load <pull link>...    load a pull request into the app, or a stack of\n                         them, bottom first, and print where to open it;\n                         loading one again refreshes it\n      --account <login>  act as this GitHub account, not gh's active one\n      --config <file>    JSON naming the components and packages to draw\n      --context          also draw the unchanged code the change touches\n      --no-context       stop drawing it, for a review loaded with it\n  load --repo <folder> --base <ref> --head <branch>...\n                         load a branch with no pull request, read only, as\n                         its diff from base would show, or a stack of them,\n                         a --head for each, bottom first; loading the same\n                         branches again refreshes the review\n      --title <text>     name the review; the branch's name, or the stack's\n                         bottom and top, otherwise\n      --config, --context and --no-context work as above\n  attach <pull link> <file>\n                         attach a written review, in Markdown, to the review\n                         holding that pull request\n  list                   list every review, with its address\n  terminal [<address>]   review in this terminal, with Vim keys: the review\n                         at the address load and list print, as\n                         lazyreview://<review id>, or the one holding a\n                         pull request link, which loads it first when no\n                         review holds it yet; with no address, the open\n                         pull requests of the accounts in the config\n  config path            where the config file is, or would be, and whether\n                         it exists\n  config defaults        the default config, saying what each setting does\n  config check           every setting's value, and where it came from: the\n                         default, the config file or a variable\n  config move [--check]  move settings.json and the LAZYREVIEW_ variables\n                         into the config file, once; --check shows what it\n                         would do, and writes nothing\n  start                  say that lazyreview runs no server now\n  stop                   stop every answer Claude is still giving, and a\n                         server an older lazyreview left running\n  help                   show this",
		MoreThanOneValue: "it holds more than one value",
		NoDataHome:       "the config names no data home by its full path",

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
		NoDrawings:              "this review has no drawings: only a written review holds them",
		CouldNotWriteDrawings:   "could not write the page of drawings",
		DrawingsOf:              "Drawings of {{.title}}",
		DrawingsOpened:          "opened the written review's drawings in the browser",
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
		HelpDrawings:       "opens the written review's drawings in the browser",
		HelpPullFilter:     "on a stack, moves the Review panel's cursor to the next or previous pull request: All, then each one",
		HelpJumps:          "back and forward through jumps",
		HelpCount:          "a count before a move, as 5j or 12G; on the left, a digit that jumps to a panel starts none",
		HelpClose:          "closes the top popup",
		HelpQuit:           "quits, as {{.command}}q does",
		HelpPullsMove:      "the next, previous, first or last pull request, or All on a stack; the other panels show only what the cursor is on",
		HelpPanelsRound:    "the panel above or below; {{.previousPanel}} goes round to Diagram",
		HelpOpenReviewView: "the Review view, in the main panel",
		HelpHighlight:      "on a stack, highlights this pull request; {{.move}} grow or shrink the highlight, and the other panels show only what it holds",
		HelpClearHighlight: "goes to All, to show the whole stack",
		HelpPostPull:       "post: the actions offered for this pull request",
		HelpFilesMove:      "the next, previous, first or last row; the main panel shows the diff of its file, or of every file under its folder or component",
		HelpPanels:         "the panel above or below",
		HelpOpenFile:       "on a file, gives the main panel the keys; on a component or a folder, folds it",
		HelpSelectFiles:    "selects rows, from this one; {{.move}} grow or shrink the selection, {{.select}} or {{.stop}} stops it, and the main panel shows every file it holds",
		HelpStopSelection:  "stops the selection, so the main panel shows the row the cursor is on",
		HelpFoldTree:       "folds or unfolds a component or a folder",
		HelpNotesMove:      "the next, previous, first or last note; the main panel shows it in its diff",
		HelpOpenNote:       "the note in its diff, in the main panel",
		HelpPostComment:    "post: the actions offered for the comment's pull request",
		HelpBoxesMove:      "the next, previous, first or last box, in reading order",
		HelpPanelsToReview: "the panel above or below; {{.nextPanel}} goes round to Review",
		HelpOpenDiagram:    "the diagram in the main panel, which moves through its rows of boxes and steps in",
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
		HelpItemsEnds:      "the first or last item, or item n",
		HelpOpenItem:       "opens the item in its diff; a component opens its diff",
		HelpFoldSection:    "folds or unfolds a section",
		HelpPostAtCursor:   "post: the actions offered for the pull request at the cursor",
		HelpLinesMove:      "the next or previous line",
		HelpHalfPages:      "half a page down or up",
		HelpPages:          "a page down or up",
		HelpScrollCode:     "scrolls the code left or right, half its width at a time; the gutter and the headers stay put",
		HelpLineEnds:       "the first or last line of this file, or line n of it",
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
		HelpDropAtCursor:   "drops the queued comment, or deletes the conversation with Claude, at the cursor, after {{.confirm}} confirms",
		HelpShowType:       "shows the type at the cursor on the diagram; on a file's header, lists its types",
		HelpBack:           "goes back to the diagram or the Review view",
		HelpSplit:          "split or unified",
		HelpWrap:           "wraps lines, or stops",
		HelpWhole:          "the whole file, or back to the diff",
		HelpRendered:       "a Markdown file rendered, or back to the diff",
		HelpBoxesAlong:     "the box left or right in the row",
		HelpBoxesDown:      "the row below or above, at the same place from the left, or a shorter row's last box; the next row with that place goes back to it",
		HelpBoxesOrder:     "the next or previous box in reading order",
		HelpBoxesEnds:      "the first or last box",
		HelpFollow:         "follows a link of the focused box",
		HelpLinks:          "lists the focused box's links, in and out",
		HelpHomeMove:       "the next or previous pull request of the repository",
		HelpHomeEnds:       "the first or last pull request of the repository",
		HelpRepositories:   "the repository above or below, going round at the ends",
		HelpFirstNineKeys:  "{{.first}} to {{.last}}",
		HelpFirstNine:      "the first nine repositories",
		HelpOpenPull:       "opens the pull request's review; one no review holds yet is loaded from GitHub first",
		HelpReadAgain:      "reads the open pull requests from GitHub again",
		HelpGitHubPull:     "opens the pull request on GitHub",
		HelpReviewOpen:     "the Review view of the review open",
		HelpListMode:       "more of the screen for the list, then all of it; {{.smaller}} goes back",
		HelpCommandQuit:    "the command line: {{.command}}q quits",
		HelpHide:           "hides the repository from the home screen, or every repository of its owner; on a hidden one, shows it again",
		HelpShowHidden:     "shows the hidden repositories, dim, after the others, or hides them again",
		MarkWaiting:        "your review is waiting: asked of you, or of your team",
		MarkLoaded:         "loaded in lazyreview",
		MarkStack:          "a stack, bottom first, reviewed as one",
		Dim:                "dim",
		MarkDraft:          "a draft",
		HiddenTag:          "hidden",
		MarkHidden:         "before a repository's name: the config hides it",
		MarkHiddenShownBy:  "before a repository's name: the config hides it, and {{.show}} shows it",
		Keys:               "Keys",
		ReviewView:         "Review view",
		Home:               "Home",
		Split:              "Split",
		Wrap:               "Wrap",
		ScrollLeft:         "Scroll left",
		ScrollRight:        "Scroll right",
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
		DeleteQuestion:     "Delete the conversation with Claude on {{.on}}?",
		DeleteTitle:        "Delete conversation",
		DeleteFooter:       "{{.confirm}} delete · any other key keeps it",
		DeleteStopsClaude:  "Claude is still answering; deleting stops it first.",
		DeleteGone:         "It is gone for good once deleted.",
		Delete:             "Delete",
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
		Branch:             "Branch",
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
		Drawings:         "Drawings",
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
		HideRepository:   "Hide repository",
		ShowHidden:       "Show hidden",
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
		CountKeys:        "{n}",

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
		DrawingNotShown: "A drawing is here that a terminal cannot show. {{.drawings}} opens the written review's drawings in the browser.",

		WrittenStale:        "This written review read head {{.head}}; the pull request has moved on since, so it may not match the diff.",
		NoWrittenReview:     "No written review is attached. The diagram and the diffs hold the change.",
		BranchReadOnly:      "This review is read only: nothing can be posted from a branch.",
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
		BranchStackHeader:   "A stack of {{.count}} branches in {{.repo}}, each built on the one before, with no pull request yet",
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
		DropsAtCursor:           "{{.drop}} drops the queued comment, or deletes the conversation with Claude, at the cursor",
		KeptQueued:              "Kept queued comment {{.comment}}.",
		KeptConversation:        "Kept the conversation on {{.on}}.",
		Dropping:                "Dropping…",
		Deleting:                "Deleting…",
		StoppingThenDeleting:    "Stopping Claude, then deleting…",
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
		DeletedConversation:     "Deleted the conversation on {{.on}}.",
		QueuedOn:                "Queued a comment on {{.on}}.",
		QueuedAnswerOn:          "Queued Claude's answer on {{.on}}.",

		AddComments:             "Add Comments",
		ApproveWithComments:     "Approve With Comments",
		Approve:                 "Approve",
		NoQueueToPost:           "only a pull request's review has a queue to post",
		ReviewMovedNoSend:       "this review moved, so nothing can be sent from here",
		PostsAtCursor:           "{{.post}} posts for the pull request at the cursor; move to it, or to one of its comments",
		PullWasState:            "#{{.pull}} was {{.state}}, so nothing can be sent to it",
		CheckingThenSending:     "Checking the pull request with GitHub, then sending…",
		SentToGitHub:            "Sent to GitHub",
		SentHowToClose:          "Sent; {{.close}} closes GitHub's answer.",
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
		NoBranchInStack:  "no branch {{.branch}} in this stack",

		ReadingAlready:          "Reading GitHub already",
		CouldNotReadGitHubAgain: "Could not read GitHub again",
		OneLoadAtATime:          "One load at a time: {{.name}} is loading",
		LoadedOpenFromHome:      "Loaded {{.name}}; {{.open}} on the home screen opens it",
		DraftNotSent:            "Your draft in {{.title}} is not sent yet: {{.review}} goes back to it",
		LoadAndOpen:             "Load and open",
		Repositories:            "Repositories",
		Refresh:                 "Refresh",
		ReadAgain:               "Read again",
		LoadingFromGitHub:       "Loading {{.name}} from GitHub…",
		ReadingGitHub:           "Reading GitHub…",
		HideTitle:               "Hide from the home screen",
		HideThis:                "Hide this repository",
		HideOwner:               "Hide every repository of {{.owner}}",
		HiddenByTitle:           "Hidden by {{.entry}}",
		ShowOwnerAgain:          "Take it off the list: every repository of {{.owner}} shows again",
		HidRepo:                 "{{.repo}} is hidden",
		HidOwner:                "Every repository of {{.owner}} is hidden",
		ShowsAgain:              "{{.repo}} shows again",
		OwnerShowsAgain:         "Every repository of {{.owner}} shows again",
		NoneHidden:              "No repository here is hidden",
		CouldNotChangeHidden:    "Could not change the hidden repositories",
		OneHidden:               "1 repository hidden",
		ManyHidden:              "{{.count}} repositories hidden",
		ShowingOneHidden:        "Showing 1 hidden",
		ShowingManyHidden:       "Showing {{.count}} hidden",
		ShowsIt:                 "{{.key}} shows it",
		ShowsThem:               "{{.key}} shows them",
		HidesIt:                 "{{.key}} hides it",
		HidesThem:               "{{.key}} hides them",

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
		EveryOneHidden:       "Every repository is hidden",
		AllHidden:            "The config hides every repository with open pull requests that involve {{.logins}}.",
		ShowKeyShowsThem:     "{{.show}} shows them.",
		HiddenFromHome:       "Hidden from the home screen by {{.entries}} in the config.",
		HideKeyShowsIt:       "{{.hide}} shows it again.",
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

		LoadedElsewhere:   "This review was loaded again. {{.refresh}} loads it here; your drafts stay.",
		LoadedAgain:       "Loaded again.",
		LoadedAgainNoting: "Loaded again, with {{.notes}}, shown in the Review view.",
		LoadedNoting:      "Loaded, with {{.notes}}, shown in the Review view.",
		NoteNoun:          "note",

		NotAPull:                  "not a pull request: {{.host}}/{{.repo}}#{{.pull}}",
		NotAPullAddress:           "\"{{.text}}\" is not a pull request's address, such as https://github.com/owner/repo/pull/12",
		NotABranch:                "not a branch review: {{.repoKey}} {{.branch}}",
		NotABranchID:              "\"{{.text}}\" is not a branch review's id, such as local/widgets-1a2b3c4d/feature~search",
		NotAReviewID:              "\"{{.id}}\" is not a review id, such as github.com/owner/repo/12",
		GaveUpWaiting:             "gave up waiting for {{.path}}",
		HeldElsewhere:             "another process holds {{.path}}",
		NoBuild:                   "the change does not name the page build its window was drawn from",
		LoadedAgainSince:          "the review was loaded again after this window opened, so its lines may have moved; reload it, then try again",
		NotWritten:                "could not write the review, so every file is as it was: {{.why}}",
		Unreadable:                "could not read {{.path}}: {{.why}}",
		NoFile:                    "no file: {{.why}}",
		NotAFileOfReview:          "{{.path}} is not a file of this review",
		NoSuchReview:              "no such review",
		ReviewNone:                "there is no review {{.id}}",
		NotInCombinedDiff:         "the line is not in the combined diff",
		StackPlace:                "#{{.pull}}, line {{.line}}",
		StackPlaceOf:              "#{{.pull}}, line {{.line}} of {{.path}}",
		ChangedIn:                 "Changed in #{{.pull}}. A comment here goes on {{.place}}.",
		LastChanged:               "Unchanged here; #{{.pull}} last changed it. A comment here goes on {{.place}}.",
		BranchChangedIn:           "Changed in {{.branch}}.",
		BranchLastChanged:         "Unchanged here; {{.branch}} last changed it.",
		BranchNotTraced:           "Not traced to a branch: {{.why}}.",
		BranchUnchanged:           "Unchanged.",
		NoPullTakes:               "No pull request in the stack can take a comment here: {{.why}}.",
		NoneShows:                 "Unchanged, and no pull request in the stack shows this line, so it takes no comment.",
		OneShows:                  "Unchanged. A comment here goes on {{.place}}.",
		SeveralShow:               "Unchanged, and shown by {{.places}}. A comment here names which one it is for.",
		And:                       "and",
		WrittenNoHeader:           "the written review must start with a header between --- lines",
		WrittenHeaderLine:         "the header line \"{{.line}}\" is not \"name: value\"",
		WrittenUnknownField:       "the header has an unknown field \"{{.field}}\"",
		WrittenNeedsHead:          "the header needs \"head\", the head commit the review read",
		WrittenPullOrStack:        "the header needs \"pull\", or \"stack\" for a stack, and not both",
		WrittenPullNumber:         "the header's \"pull\" must be the pull request number",
		WrittenStackNumbers:       "the header's \"stack\" must list two or more pull request numbers, bottom first",
		WrittenTextBefore:         "text before the first ## section heading",
		WrittenNoSection:          "the written review needs at least one ## section",
		WrittenEmptySection:       "\"## {{.title}}\" is empty; a section with nothing to say says so in one line",
		WrittenProblems:           "the written review has {{.problems}}",
		WrittenOtherPulls:         "the written review names {{.written}}, but the review shows {{.shown}}",
		WrittenOtherHead:          "the written review read {{.read}}, but the review shows head {{.head}}; review the head the app shows",
		ProblemNoun:               "problem",
		LineWord:                  "line",
		RemovedLineWord:           "removed line",
		OneLine:                   "{{.side}} {{.line}}",
		LineToLine2:               "{{.startSide}} {{.start}} to {{.side}} {{.line}}",
		LinesRange:                "{{.side}}s {{.start}}-{{.line}}",
		BranchTakesNone:           "A branch with no pull request takes no comments. Ask Claude about these lines instead.",
		ReviewMovedHere:           "This review moved, so nothing can be queued here.",
		NotInChange:               "This file is not in the change, so a comment cannot go on it. Ask Claude about it instead.",
		NotADiffLine:              "GitHub takes comments only on lines of the diff, and this one is not. Ask Claude about it instead.",
		LeavesHunk:                "GitHub takes a range only inside one hunk of the diff, and this one leaves it. Pick lines inside one hunk, or ask Claude about these.",
		Backwards:                 "A range runs from its first line to its last, and this one does not.",
		NotThisReview:             "This review is #{{.own}}'s, not #{{.given}}'s.",
		OtherHeads:                "This page shows other heads of {{.pulls}} than the review holds. Load the stack again, then queue.",
		LastChangedBy:             "#{{.last}} last changed this line, not #{{.given}}.",
		NotShownBy:                "#{{.pull}}'s diff does not show this line.",
		NotAllTo:                  "These lines do not all go to #{{.pull}}. Comment on fewer lines, or choose another pull request.",
		NotAllToOne:               "These lines do not all go to one pull request of the stack. Comment on fewer lines, or one comment per pull request.",
		ChooseAmong:               "Unchanged, and shown by {{.pulls}}. Choose the pull request it goes on.",
		LeftStack:                 "#{{.pull}} is not in this stack any more.",
		PullEndedHere:             "#{{.pull}} is {{.state}}, so nothing can be queued for it.",
		GoesOn:                    "Goes on #{{.pull}}, {{.lines}}.",
		GoesOnOf:                  "Goes on #{{.pull}}, {{.lines}} of {{.path}}.",
		PressComment:              "{{.these}} on {{.where}} under your name, exactly as shown.",
		PressApproveWithComments:  "You approve {{.where}}, and {{.these}} with the approval under your name, exactly as shown.",
		PressApprove:              "You approve {{.where}} with no comments.",
		PressApproveKeeps:         " The queued {{.comments}} {{.stay}} queued, unposted.",
		ThisCommentGoes:           "this comment goes",
		TheseCommentsGo:           "these {{.comments}} go",
		Stays:                     "stays",
		Stay:                      "stay",
		QueuedCommentNoun:         "queued comment",
		EmptyText:                 "the text is empty",
		NotOnDiff:                 "is not on a line of the diff",
		StartsOff:                 "starts on {{.removed}}line {{.line}}, which is not on a line of the diff",
		AcrossHunks:               "runs from {{.removed}}line {{.line}} across more than one hunk of the diff, and GitHub takes a range only inside one hunk",
		StartsAfter:               "starts on {{.removed}}line {{.line}}, which does not come before its last line in the diff",
		RemovedWord:               "removed ",
		QueueReviewMoved:          "this review moved to {{.to}}; nothing changed here",
		NotInStackAnyMore:         "#{{.pull}} is not in this stack any more",
		NotInReview:               "#{{.pull}} is not part of this review",
		NoVersion:                 "the change does not name the version of comment {{.id}} it was made on",
		ChangedSince:              "comment {{.id}} was changed after this window showed it, and now reads \"{{.body}}\"",
		NoComment:                 "no queued comment {{.id}}",
		NotQueued:                 "not queued: {{.why}}",
		NotChanged:                "not changed: {{.why}}",
		NotDropped:                "not dropped: {{.why}}",
		NothingWasPosted:          "nothing was posted: {{.why}}",
		HeadMoved:                 "#{{.pull}} moved from {{.from}} to {{.to}}; load it again, then queue",
		CommentAt:                 "{{.path}}:{{.lines}}",
		CommentAtOf:               "{{.path}}:{{.lines}} of #{{.pull}}",
		Placed:                    "{{.at}} {{.problem}}",
		NotAnAction:               "\"{{.action}}\" is not an action; nothing was posted",
		NotOffered:                "\"{{.action}}\" is not offered for #{{.pull}}; nothing was posted",
		OtherSet:                  "the page showed set {{.shown}}, but \"{{.action}}\" now gives set {{.now}}; nothing was posted",
		OtherAccount:              "GitHub answers as {{.now}} now, but this review was loaded as {{.loaded}}; nothing was posted",
		AlreadyDone:               "already done as {{.url}}; posted nothing new",
		OwnPull:                   "this is your own pull request, which GitHub does not let you approve",
		ReadBack:                  "posted {{.url}}, but GitHub reads back state {{.state}} with {{.back}} of {{.comments}}: check it",
		PostedComment:             "posted {{.comments}} as one review: {{.url}}",
		PostedApproveWithComments: "approved with {{.comments}}: {{.url}}",
		PostedApprove:             "approved, with no comments: {{.url}}",
		PullIsEnded:               "#{{.pull}} is {{.state}}",
		HeadMovedFrom:             "the head moved from {{.from}} to {{.to}}",
		CommentNamed:              "comment {{.id}} ({{.path}}:{{.line}})",
		CommentProblem:            "{{.comment}} {{.problem}}",
		CommentText:               "{{.comment}}: {{.problem}}",
		CommentStale:              "{{.comment}} is stale: {{.why}}; queue it again or drop it",
		GitHubNoToken:             "gh has no token for {{.account}} on {{.host}}: {{.why}}",
		GitHubTooSlow:             "GitHub did not answer {{.method}} {{.path}} in time; nothing changed",
		GitHubUnreachable:         "could not reach {{.host}}: {{.why}}",
		GitHubRefused:             "GitHub refused {{.method}} {{.path}}: {{.why}}",
		GitHubUnreadable:          "could not read GitHub's answer to {{.method}} {{.path}}: {{.why}}",

		AskSystem:            "You help someone review a change to code, from lazyreview. Your working folder is a throwaway copy of the codebase as it stands after the change: a git repository, with a branch for the change and one for its base. Read, search, run and change anything in it, such as tests and builds, to find what the answer needs, such as callers, tests and definitions. Work in the copy itself: nothing you change there leaves it, and it is thrown away after, so there is nothing to clean up. Anything that would change something outside the copy, or leave this machine as a change, such as a push, a comment on GitHub, a message, or a request that writes to a web service, is refused, as nobody is there to approve it. Do not try to get round a refusal; say what you could not do. The message holds the part of the diff around the lines they ask about, the code around those lines, and their question. Answer in plain words and short sentences, and only what they asked. Say so when what you found is not enough to be sure. When they ask you to write a review comment, reply with the comment text alone, as they would write it: short, plain, one point. Never use the em dash; use a plain dash.",
		AskPull:              "Pull request #{{.pull}}: {{.title}}",
		AskPullDescription:   "\n\nIts description:\n{{.description}}",
		AskStack:             "A stack of pull requests, bottom first, reviewed together:\n{{.members}}",
		AskStackMember:       "#{{.pull}}: {{.title}}",
		AskStackOwner:        "\n\n{{.owner}}",
		AskBranch:            "Branch {{.branch}}{{.title}}. It has no pull request yet, and its diff is against {{.base}}.",
		AskBranchTitle:       ": {{.title}}",
		AskBranchStack:       "A stack of branches, bottom first, reviewed together. None has a pull request yet, and the stack's diff is against {{.base}}:\n{{.branches}}",
		AskCodebase:          "The codebase as it stands after {{.after}} is in {{.folder}}, your working folder, with the branch {{.branch}} checked out and the branch {{.base}} at the base of the change.",
		AskThisChange:        "this change",
		AskChanges:           "The files this change touches:\n{{.files}}",
		AskChangedFile:       "{{.path}} ({{.state}})",
		AskChangesMore:       "\nand {{.more}} more",
		AskFile:              "File: {{.path}} ({{.state}})",
		AskFileAt:            ", at {{.at}}",
		AskAdded:             "new",
		AskDeleted:           "deleted",
		AskRenamed:           "renamed from {{.from}}",
		AskChanged:           "changed",
		AskUnchanged:         "unchanged by this change",
		AskAbout:             "They ask about {{.lines}}.",
		AskAboutChangedBy:    "They ask about {{.lines}}, which {{.member}} changed.",
		AskThere:             "In the codebase, after {{.member}}, they are {{.lines}}.",
		AskDiff:              "The diff of this file around those lines:\n```diff\n{{.hunk}}\n```",
		AskHead:              "The file at the head{{.of}}{{.window}}:\n```\n{{.text}}\n```",
		AskHeadOf:            " of {{.member}}",
		AskHeadWindow:        ", lines {{.from}}-{{.to}} of {{.count}}",
		AskConversation:      "The conversation so far:\n{{.messages}}",
		AskYouAnswered:       "You answered: {{.text}}",
		AskTheyAsked:         "They asked: {{.text}}",
		AskQuestion:          "Their question:\n{{.text}}",
		AskFollowUp:          "Their follow-up question, on the same lines:\n{{.text}}",
		StepRead:             "Read {{.path}}",
		StepSearched:         "Searched for {{.pattern}}",
		StepSearchedIn:       "Searched for {{.pattern}} in {{.path}}",
		StepListed:           "Looked for files named {{.pattern}}",
		StepRan:              "Ran {{.command}}",
		StepEdited:           "Edited {{.path}}",
		StepWrote:            "Wrote {{.path}}",
		StepFetched:          "Read {{.address}}",
		StepSearchedWeb:      "Searched the web for {{.query}}",
		StepHelper:           "Asked a helper to {{.what}}",
		StepSkill:            "Used the skill {{.name}}",
		StepPlanned:          "Planned its steps",
		StepMcp:              "Used {{.tool}} on {{.server}}",
		StepUsed:             "Used {{.tool}}",
		AskFirst:             "ask something first",
		AskNeedsLine:         "a conversation needs a file, a side and a line",
		NotAsked:             "not asked: {{.why}}",
		NoConversation:       "there is no conversation {{.id}} on {{.review}}",
		NoCodebase:           "could not make a copy of the code for Claude: {{.why}}",
		StillWorking:         "Claude is still working on this conversation; wait for its answer, or stop it",
		NotDeleted:           "not deleted: {{.why}}",
		DidNotStop:           "Claude is still answering it and did not stop",
		AnswerEnded:          "the answer stopped before Claude gave one: {{.why}}",
		ClaudeTooSlow:        "Claude did not answer within {{.minutes}} minutes",
		ClaudeStopped:        "you stopped it",
		ClaudeNotApproved:    "it needed approval, and nobody is there to give it",
		ClaudeSandboxed:      "the sandbox stopped it: shell commands change only the copy, and reach only the hosts allowed",
		ClaudeNoAnswer:       "Claude Code did not answer: {{.why}}",
		ClaudeExitCode:       "it stopped with exit code {{.code}}",
		ClaudeNotFound:       "{{.command}} was not found",
		ClaudeCouldNotAnswer: "Claude could not answer: {{.why}}",
		ClaudeTooManyTurns:   "Claude worked for {{.turns}} turns without answering; ask about less at once",
		ClaudeNothing:        "no answer",
		GuardMarker:          "lazyreview refused this",
		GuardOutside:         "lazyreview refused this: the file is outside the copy of the code, and only the copy may change",
		GuardPush:            "lazyreview refused this: it would push, and nobody is there to approve it",
		GuardPost:            "lazyreview refused this: it would send a change to a web service, and nobody is there to approve it",
		GuardPublish:         "lazyreview refused this: it would publish a package, and nobody is there to approve it",
		GuardElsewhere:       "lazyreview refused this: it would reach a program or a machine outside the copy, and nobody is there to approve it",
		GuardGh:              "lazyreview refused this: gh may only read, one command at a time",
		GuardMcp:             "lazyreview refused this: {{.tool}} on {{.server}} may change something, and only tools that read are allowed",
		GuardTool:            "lazyreview refused this: {{.tool}} is not allowed while nobody is there to approve it",
		GuardFailed:          "lazyreview refused this, as its guard could not read the call: {{.why}}",
		GitFailed:            "git {{.command}} failed: {{.why}}",
		CommitMissing:        "commit {{.commit}} is no longer in {{.repo}}; load the review again",

		ConfigAboutDataHome:             "Where lazyreview keeps reviews, clones and the copies of the code Claude answers in.",
		ConfigAboutClaudeCommand:        "The Claude Code command that answers a question about a line.",
		ConfigAboutClaudeModel:          "The model that answers. Empty: Claude Code's own default.",
		ConfigAboutEditor:               "The command that opens a comment, a question or an edit.\nEmpty: VISUAL, then EDITOR, then vi, as git chooses.",
		ConfigAboutAccounts:             "The GitHub accounts whose open pull requests the home screen lists.\nEach is read by its own gh token. A review asked of a team counts as\nasked of the account, for pull requests opened on or after `since`.",
		ConfigAboutHiddenRepositories:   "Repositories the home screen leaves out, for every account: owner/name,\nor owner/* for every repository of the owner. A review of one still opens.",
		ConfigAboutLineNumbers:          "How the terminal app's diff numbers its lines: absolute, each line its\nown number; relative, the cursor's line its own number and every other\nline its distance from it, as Vim's number and relativenumber do; or none.",
		ConfigAboutTheme:                "The terminal app's colours, for a dark terminal and a light one.\nEach is #rrggbb or a terminal colour number from 0 to 255.",
		ConfigAboutKeys:                 "Keys as lazygit writes them: a letter, or <space>, <enter>, <esc>,\n<tab>, <up>, <down>, <left>, <right>, <c-x> for Ctrl-x. A sequence is\nits keys in a row, such as gg, ]c or <space>v. A list gives an action\nmore than one key. <disabled> removes a key.",
		ConfigKeysUniversal:             "every screen",
		ConfigKeysHome:                  "the home screen",
		ConfigKeysReviewScreen:          "every panel of a review",
		ConfigKeysPanels:                "every panel on the left",
		ConfigKeysReviewView:            "the Review view in the main panel",
		ConfigKeysDiagram:               "the diagram in the main panel",
		ConfigKeysMenu:                  "every list that pops up",
		ConfigAccountsExample:           "accounts:\n  - login: lee\n  - login: lee-at-work\n    teams:\n      - name: example-org/mobile\n        since: 2026-10-01",
		ConfigHiddenRepositoriesExample: "hiddenRepositories:\n  - example-org/old-app\n  - example-bots/*",
		ConfigFileMissing:               "LAZYREVIEW_CONFIG_FILE names {{.path}}, which does not exist",
		ConfigUnreadable:                "{{.path}} could not be read: {{.why}}",
		ConfigYAMLError:                 "{{.reason}} ({{.line}}:{{.column}})",
		ConfigDuplicatedKey:             "duplicated mapping key",
		ConfigProblem:                   "{{.path}} {{.problem}}",
		ConfigProblems:                  "{{.path}} has {{.count}} problems:\n{{.problems}}",
		ConfigListedProblem:             "  - {{.problem}}",
		ConfigNotAMap:                   "holds no settings, written as lazyreview config defaults shows them",
		ConfigUnknown:                   "sets {{.key}}, which is not a setting",
		ConfigRetired:                   "sets {{.key}}, which is no longer a setting, so remove it: {{.why}}",
		ConfigRetiredDiagramPanelMove:   "the Diagram panel switches panels as every panel on the left does, and keybinding.diagram.{{.side}} moves to the box on the {{.side}} in the main panel's diagram",
		ConfigUnusedWeb:                 "lazyreview has no web app; keybinding.reviewView.drawings opens a written review's drawings in the browser",
		ConfigNotASection:               "gives {{.key}} {{.value}}, which is not a group of settings",
		ConfigNotAFolder:                "gives {{.key}} {{.value}}, which is not a full path, or one starting with ~/",
		ConfigNotACommand:               "gives {{.key}} {{.value}}, which is not a command",
		ConfigNotText:                   "gives {{.key}} {{.value}}, which is not text",
		ConfigNotAChoice:                "gives {{.key}} {{.value}}, which is not {{.choices}} or {{.last}}",
		ConfigNotAccounts:               "gives {{.key}} {{.value}}, which is not a list of accounts",
		ConfigNotRepositories:           "gives {{.key}} {{.value}}, which is not a list of repositories",
		ConfigNotARepository:            "gives {{.key}} {{.value}}, which is not a repository as owner/name, or owner/* for every repository of the owner",
		ConfigNotAnEntry:                "{{.value}} is not a repository as owner/name, or owner/* for every repository of the owner",
		ConfigNotAColour:                "gives {{.key}} {{.value}}, which is not a colour: #rrggbb or a terminal colour number from 0 to 255",
		ConfigNotColours:                "gives {{.key}} {{.value}}, which is not a list of {{.count}} colours",
		ConfigNotKeys:                   "gives {{.key}} {{.value}}, which is not a key or a list of keys: {{.why}}",
		ConfigNotNumberedKeys:           "gives {{.key}} {{.value}}, which is not a list of {{.count}} keys, one for each by its place",
		ConfigKeyNotText:                "a key is written as text",
		ConfigKeyEmpty:                  "a key is not empty",
		ConfigKeyUnclosed:               "a < starts a named key, such as <space>, and needs its >",
		ConfigKeyDisabledAlone:          "<disabled> stands alone, for no key",
		ConfigKeyUnknown:                "{{.token}} is not a key; a named key is <space>, <enter>, <esc>, <tab>, <up>, <down>, <left>, <right>, or <c-x> for Ctrl-x",
		ConfigNoLogin:                   "gives account {{.number}} no GitHub login",
		ConfigAccountKey:                "gives {{.login}} \"{{.key}}\", which an account does not have",
		ConfigLoginTwice:                "names {{.login}} twice",
		ConfigTeamsNotList:              "gives {{.login}} \"teams\" that are not a list",
		ConfigTeamNotNamed:              "gives {{.login}} a team that is not named as organisation/team",
		ConfigTeamKey:                   "gives {{.team}} \"{{.key}}\", which a team does not have",
		ConfigSinceNotDate:              "gives {{.team}} a \"since\" that is not a date such as 2026-10-01",
		ConfigNotMoved:                  "{{.settings}} is no longer read; lazyreview config move moves its accounts into {{.config}}",
		ConfigSchemaTitle:               "lazyreview config",
		ConfigSchemaDescription:         "lazyreview's config file, config.yml. docs/Config.md lists every setting and its default.",
		ConfigDocsGenerated:             "written from the config's schema; do not change it by hand",
		ConfigDocsNoMarkers:             "docs/Config.md has no START {{.name}} and END {{.name}} markers to write between",
		ConfigDocsVariable:              "Variable",
		ConfigDocsOverrides:             "Overrides",
		ConfigCannotChange:              "{{.key}} is written in a way lazyreview cannot change line by line",
		ConfigEditFailed:                "{{.path}} was not changed: {{.why}}; change it by hand, as docs/Config.md shows",

		ConfigUnusedLine:         "# {{.key}} does nothing now, so it can go: {{.why}}",
		ConfigSourceDefault:      "default",
		ConfigSourceFile:         "the file",
		ConfigMovedHeader:        "lazyreview's config. docs/Config.md lists every setting and its default.",
		ConfigExists:             "it exists",
		ConfigMissing:            "there is no file there yet, so every setting has its default",
		ConfigChecked:            "the config at {{.path}}, on top of the defaults:",
		ConfigUnset:              "no config at {{.path}}, so every setting has its default:",
		ConfigMoveWouldWrite:     "would write {{.path}}:",
		ConfigMoveFrom:           "{{.key}} from {{.source}}",
		ConfigMoveWouldRename:    "would rename {{.path}} to {{.name}}",
		ConfigMoveRefused:        "{{.path}} exists already, and the move never overwrites a config",
		ConfigMoveNothingWritten: "nothing was written; lazyreview config move makes the move",
		ConfigMoveNothing:        "there is nothing to move: there is no {{.settings}}, and no LAZYREVIEW_ variable is set",
		ConfigMoveMovedExists:    "{{.path}} exists already, so settings.json cannot be renamed to it; nothing was written",
		ConfigMoveNeedsAccounts:  "needs \"accounts\", a list of {\"login\": \"...\"}",
		ConfigMoveFailed:         "the move failed, and nothing changed: {{.why}}",
		ConfigMoveFailedStuck:    "the move failed: {{.why}}; and this could not be put back: {{.stuck}}",
		ConfigMoveWrote:          "wrote {{.path}}:",
		ConfigMoveRenamed:        "renamed {{.path}} to {{.name}}",
		ConfigMoveStillSet:       "{{.name}} is still set, and overrides {{.key}} while it is; taking it out of your shell's setup is safe, and not needed",

		NotACommit:            "\"{{.ref}}\" is not a commit in this repository",
		LoaderGiveLinks:       "give the pull request to load, or every pull request of a stack, bottom first",
		LoaderOtherRepository: "a stack is in one repository, but {{.other}} is not {{.first}}",
		LoaderNamedTwice:      "a pull request is named twice",
		LoaderNotAStack:       "these are not a stack, bottom first: {{.why}}",
		LoaderNotOnBelow:      "#{{.above}} is based on {{.base}}, not on #{{.below}}'s branch {{.head}}",
		LoaderAlsoReviews:     "{{.id}} also reviews {{.pulls}}, which this load does not name; load the whole stack, bottom first",
		LoaderThreadsUnread:   "#{{.pull}}'s review threads on GitHub are not shown, as they could not be read: {{.why}}",
		LoaderStackTitle:      "Stack #{{.first}} to #{{.last}}",
		LoaderMovedLoadThat:   "this review moved to {{.to}}; load that one",
		LoaderNotABranch:      "\"{{.head}}\" is not a branch of {{.path}}; give a branch, so loading it again updates the same review",
		LoaderGiveHead:        "give the branch to load, or every branch of a stack, bottom first",
		LoaderBranchTwice:     "{{.branch}} is named twice; name each branch of a stack once, bottom first",
		LoaderBranchesTitle:   "Stack {{.first}} to {{.last}}",
		LoaderNoDifference:    "there is no difference between {{.base}} and {{.head}}",
		LoaderLeftOut:         "whole text left out for {{.files}}, which show their diff only: {{.paths}}",
		LoaderLargeFile:       "large file",
		StackHeadMissing:      "#{{.pull}}'s head {{.head}} is not a commit in this repository; fetch it with git fetch origin pull/{{.pull}}/head",
		StackNotBuiltOn:       "#{{.pull}} is not built on #{{.below}}'s head: #{{.below}} has commits #{{.pull}} does not hold. Until #{{.pull}} is rebased, its own diff also shows the older #{{.below}} it holds; a comment on a line #{{.below}}'s head still holds goes to #{{.below}}.",
		StackBranchNotBuiltOn: "{{.branch}} is not built on {{.below}}'s head: {{.below}} has commits {{.branch}} does not hold. Until {{.branch}} is rebased, its own diff also shows the older {{.below}} it holds.",
		StackNotAdded:         "no pull request's own diff adds this line as it reads here",
		StackNotRemoved:       "no pull request's own diff removes this line as it reads here",
		StackBranchNotAdded:   "no branch's own diff adds this line as it reads here",
		StackBranchNotRemoved: "no branch's own diff removes this line as it reads here",

		CommitNoun:                "commit",
		ChangeOn:                  " on ",
		ChangeHead:                ": head ",
		ChangeMergeBase:           ", merge base ",
		ChangeCommits:             ", {{.commits}}",
		SubjectPull:               "Pull request ",
		SubjectStack:              "Stack ",
		SubjectTo:                 " to ",
		SubjectBranch:             "Branch ",
		DiagramRepositoryRoot:     "Repository root",
		DiagramUnchangedCode:      "Unchanged code",
		DiagramOtherFiles:         "Other files",
		DiagramContextDescription: "Code the change uses or plugs into, unchanged, shown for context.",
		DiagramTopLevel:           "{{.name}}, top level",
		DiagramGroup:              "{{.count}} {{.title}}",
		DiagramConforming:         "types conforming to {{.name}}",
		DiagramInheriting:         "types inheriting from {{.name}}",
		DiagramEachConforms:       "each conforms to",
		DiagramEachInherits:       "each inherits from",
		DiagramShows:              "shows",
		DiagramTests:              "tests",
		DiagramFile:               "«file»",
		DiagramExtension:          "«extension»",
		DiagramExtensionOf:        "«extension» of {{.names}}",
		DiagramExtensionName:      "{{.name}} extension",
		DiagramSummaryLabel:       "{{.label}} · ",
		DiagramChangedFiles:       "{{.count}} files · +{{.adds}}",
		DiagramRemovedLines:       " −{{.dels}}",
		DiagramUnchangedFiles:     "{{.count}} files · unchanged",
		DiagramOpenDiff:           "Open diff →",

		ComponentsProblems:       "the config has {{.count}}:\n{{.problems}}",
		ComponentsRoot:           "the config",
		ComponentsContextTypes:   "contextTypes",
		ComponentsNamed:          "{{.kind}} \"{{.id}}\"",
		ComponentsNumbered:       "{{.kind}} {{.number}}",
		ComponentsComponent:      "component",
		ComponentsPackage:        "package",
		ComponentsArea:           "area",
		ComponentsGroup:          "group",
		ComponentsPackageOf:      "{{.component}}, {{.package}}",
		ComponentsLink:           "link {{.number}}",
		ComponentsSummary:        "summary {{.number}}",
		ComponentsNotAnObject:    "{{.label}} is not an object",
		ComponentsUnknownKey:     "{{.label}} has an unknown key \"{{.key}}\"",
		ComponentsNeeds:          "{{.label}} needs \"{{.key}}\"",
		ComponentsNotText:        "{{.label}}: \"{{.key}}\" must be a non-empty string",
		ComponentsBadID:          "{{.label}}: id must be lower case letters, digits and dashes",
		ComponentsIDTwice:        "{{.label}}: id \"{{.id}}\" is used twice",
		ComponentsNotAList:       "{{.label}}: \"{{.key}}\" must be a list",
		ComponentsNotStrings:     "{{.label}}: \"{{.key}}\" must hold non-empty strings",
		ComponentsBadColor:       "{{.label}}: color must be one of {{.colors}}",
		ComponentsBadContext:     "{{.label}}: \"context\" must be true or false",
		ComponentsNeedsPath:      "{{.label}} needs at least one path",
		ComponentsNeedsPaths:     "{{.label}} needs paths or packages",
		ComponentsNeedsComponent: "{{.label}} needs at least one component",
		ComponentsInTwoAreas:     "{{.label}}: component \"{{.component}}\" is already in another area",
		ComponentsBadKind:        "{{.label}}: kind must be use or has",
		ComponentsBadMax:         "contextTypes: max must be a whole number, 0 or more",
		ComponentsAreaUnknown:    "area \"{{.area}}\" names component \"{{.component}}\", which the config does not declare",
		ComponentsSummaryUnknown: "a summary names component \"{{.component}}\", which the config does not declare",

		GitHubThreadsRefused: "GitHub refused the review threads of {{.repo}}#{{.number}}: {{.why}}",
		GitHubNoThreads:      "GitHub gave no review threads for {{.repo}}#{{.number}}",
		QueueStillOpen:       "#{{.pull}} is still open with {{.count}}, so it cannot leave the stack; post or drop them first",
		QueueLeftTheStack:    "#{{.pull}} left the stack ({{.state}})",
		QueueRenumbered:      "#{{.pull}}'s comment {{.id}} is now {{.next}}",
		QueueStaleHead:       "queued at {{.from}}, and #{{.pull}} is now at {{.to}}",
		QueueStaleStart:      "queued on a removed line, and #{{.pull}}'s diff now starts from {{.now}}, not {{.before}}",
		QueueMarkedStale:     "#{{.pull}}: {{.count}} marked stale ({{.why}}); queue each again or drop it",

		CliFailed:             "lazyreview: {{.why}}",
		CliUnknownCommand:     "unknown command \"{{.command}}\"",
		CliUnknownOption:      "unknown option {{.option}} for {{.command}}",
		CliNeedsValue:         "{{.option}} needs a value",
		CliNeedsArguments:     "{{.command}} needs {{.count}} arguments",
		CliTakesArguments:     "{{.command}} takes {{.arguments}}",
		CliNoArguments:        "no arguments",
		ArgumentNoun:          "argument",
		CliNeedsRepo:          "{{.option}} needs --repo",
		CliRepoNeeds:          "--repo needs {{.option}}",
		CliLoadNeedsSomething: "load needs a pull request, or --repo with --base and --head",
		CliNotBoth:            "load takes pull requests or --repo, not both",
		CliAccountForPulls:    "--account is for pull requests; a branch loads from its own repository",
		CliLoaded:             "loaded {{.name}}: {{.files}} +{{.adds}} -{{.dels}}, {{.components}}, {{.types}}",
		CliOpen:               "open {{.address}}",
		CliNote:               "note: {{.note}}",
		CliNotLoaded:          "{{.link}} is not loaded; load it first",
		CliReviewMoved:        "that review moved to {{.to}}",
		CliAttached:           "attached to {{.id}}; open {{.address}}",
		CliNoReviews:          "no reviews yet; load one with lazyreview load <pull link>",
		CliListedMoved:        "{{.id}}  moved to {{.to}}",
		CliListed:             "{{.address}}  {{.name}}  {{.title}}",
		CliBranchName:         "{{.head}} against {{.base}}",
		CliNoServer:           "lazyreview runs no server now; lazyreview terminal opens the app",
		CliAnswersStopped:     "stopped {{.answers}} Claude was still giving",
		AnswerNoun:            "answer",
		CliOldServerStopped:   "stopped the server an older lazyreview left running",
		CliNothingRunning:     "nothing was running",
		CliLoading:            "loading {{.url}}…",
		CliNoReview:           "there is no review {{.text}}; lazyreview list shows every review",
		CliNotALink:           "\"{{.text}}\" is not the address of a review or of a pull request; lazyreview list shows every review",
		CliConfigUnreadable:   "could not read the config {{.path}}: {{.why}}",
		CliConfigNeeds:        "config needs one of path, defaults, check or move",
		CliConfigNeedsNot:     "config needs one of path, defaults, check or move, not \"{{.given}}\"",
		CliCheckIsForMove:     "--check is for config move",

		LoaderNotFetched:        "could not fetch {{.what}} from {{.repo}}: {{.why}}",
		LoaderPullHead:          "#{{.pull}}'s head {{.head}}",
		LoaderBaseAt:            "the base {{.base}} at {{.sha}}",
		NothingLoadedHowToClose: "Nothing was loaded; {{.close}} closes why.",
		LoadAgainTitle:          "Load the review again",
		LoadTitle:               "Load {{.name}}",

		ConfigUnusedServer: "lazyreview runs no server now; lazyreview terminal opens the app",
	}
}
