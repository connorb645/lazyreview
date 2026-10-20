# The lazyreview guide

Everything lazyreview does, and how to use it. The [README](../README.md)
says what it needs and how to start.

## Start the app

    bin/lazyreview start      # in the background
    bin/lazyreview serve      # or in this terminal
    bin/lazyreview stop

It listens on http://127.0.0.1:4747, on this machine only. `load` starts it
when it is not running, so starting it by hand is rarely needed.

## Load a pull request or a stack

    bin/lazyreview load https://github.com/owner/repo/pull/12

A stack is every pull request of it, bottom first, each based on the head
branch of the one below:

    bin/lazyreview load https://github.com/owner/repo/pull/12 \
      https://github.com/owner/repo/pull/13 https://github.com/owner/repo/pull/14

It prints what it loaded and where to open it:

    loaded #12, #13, #14: 24 files +966 -13, 8 components, 31 types
    open http://127.0.0.1:4747/review/github.com/owner/repo/12

Loading again refreshes the review in place, as does `R` in the terminal
app. A stack's review lives with its lowest pull request that
already has one, and the comments queued on its members' own reviews move into
it. Options:

- `--account <login>` acts as that GitHub account, by its own token, without
  changing the account gh has active. The review keeps it.
- `--config <file>` names the components and packages to draw, as below. The
  review keeps it.
- `--context` also draws the unchanged code the change touches, set apart in
  a dashed component of its own, its code read only; `--no-context` stops it.
  The review keeps the choice.

`bin/lazyreview list` prints every review with its address.

### For an agent

An agent asked to review a pull request runs one command and passes on the
address it prints:

    bin/lazyreview load <pull request address>...

That is the whole job. Comments, questions and the review itself happen in the
app. A change with no pull request yet is one command too:

    bin/lazyreview load --repo <folder> --base <ref> --head <branch>

An agent that has written a review of a pull request can attach it, and the
app then opens on it:

    bin/lazyreview attach <pull request address> review.md

The written review is Markdown: a header naming what it read, then its
sections as `##` headings. A `file:line` in backticks, such as
`` `Basket.swift:12` ``, opens the diff at that line; `[Certain]`, `[Likely]`
and `[Guessing]` show as tags, and `#13` as that pull request's colour. A
drawing written as inline SVG, which a terminal cannot show, opens in the
browser with `gX` from the Review view.

    ---
    pull: 12            # or, for a stack: stack: 12, 13, 14
    head: 3f2a9c1d0e    # the head it read; the top's, for a stack
    title: Optional name
    ---
    ## What this changes and why
    ...

## Load a branch with no pull request

A branch about to be pushed, or any branch, loads straight from its
repository on this machine, as its pull request would show it: from where it
leaves the base to its head.

    bin/lazyreview load --repo ~/code/widgets --base main --head feature/search

It prints what it loaded and where to open it:

    loaded feature/search against main: 6 files +210 -14, 2 components, 5 types
    open http://127.0.0.1:4747/review/local/widgets-1a2b3c4d/feature~search

The review is kept by repository and branch, so loading the same branch again,
from any worktree of the repository, refreshes it in place, as does `R` in
the terminal app. `--head` names a branch, or `HEAD` on one. It gets
the same diagram and diffs, and Ask Claude, but it is read only: it has no
queue and no buttons, since there is no pull request to post to. `--title
<text>` names it, where the branch's name is the default; `--config`,
`--context` and `--no-context` work as for a pull request. The review keeps
each of them.

## Review in the terminal app

Every review opens in the terminal app, with Vim keys.

    bin/lazyreview terminal
    bin/lazyreview terminal http://127.0.0.1:4747/review/github.com/owner/repo/12
    bin/lazyreview terminal https://github.com/owner/repo/pull/12

With no link, it opens on the home screen of your open pull requests, below.
With one, it opens that review straight away: the address `load` printed, or
a pull request link. A pull request that no review holds yet is loaded first,
as `load` loads it, so one command opens it. A review id, such as
`github.com/owner/repo/12`, works too. It starts the server when it is not
running, as `load` does, and says whether it started it or found it running.
The app is a Go program in `terminal/`, built on this machine the first time
it runs and again whenever its code changes; building it needs Go 1.24 or
later, and its libraries are vendored, so the build reaches no network.

### The home screen

The home screen lists the open pull requests that involve your GitHub
accounts, read live from GitHub: those that ask for your review, or your
team's, those you opened, and those you have commented on or reviewed. Each
repository has a panel of its own on the left, numbered as Lazygit numbers
its panels. Each row shows a pull request's number, title and author, `●`
when your review is waiting and `◆` when it is loaded in lazyreview; a draft
is dim. A stack's pull requests sit together, bottom first, joined by a
bracket, and any of them opens the stack's review. The main panel shows the
pull request selected: why it involves you, its stack, its description, and
what `Enter` does. When GitHub cannot be read as an account, a panel says so
and why, and the others are still listed.

| Key | Does |
| --- | --- |
| `j` `k` | the next or previous pull request of the repository |
| `gg` `G` | the first or last pull request of the repository |
| `Ctrl-d` `Ctrl-u` `Ctrl-f` `Ctrl-b` | half a page or a page down or up the repository |
| `h` `l` | the repository above or below, going round from the last to the first and back |
| `1` to `9` | the first nine repositories |
| `Enter` | opens the pull request's review; one no review holds yet is loaded from GitHub first, as the account that found it |
| `i` | hides the repository, or every repository of its owner, from a menu; on a hidden one, shows it again |
| `I` | shows the hidden repositories, dim, after the others, or hides them again |
| `R` `:refresh` | reads GitHub again |
| `gx` | opens the pull request on GitHub |
| `Space v` | back to the review open |
| `Space h` | from any review, the home screen |

The accounts are named in the config, `config.yml`, which the home screen
reads each time it reads GitHub. `lazyreview config path` says where it is;
on macOS that is `~/Library/Application Support/lazyreview/config.yml`:

```yaml
github:
  accounts:
    - login: lee
    - login: lee-at-work
      teams:
        - name: example-org/mobile
          since: 2026-10-01
```

Each account is read by its own token, as `gh auth token --user <login>`
gives it, so each must be signed in to gh; the account gh has active is never
changed. A review asked of one of an account's teams counts as asked of you,
for pull requests opened on or after the team's `since` date when it has one.
With no accounts in the config, the home screen says how to name them.
[The config's own page](Config.md) lists every setting.

A repository you do not review can be hidden. `i` on it opens a menu: hide
this repository, or every repository of its owner. The choice goes in the
config's `github.hiddenRepositories`, as `owner/name` or `owner/*`, for every
account, and the file keeps every other line as you wrote it. A hidden
repository has no panel, and the bar counts the ones hidden, so no pull
request leaves without a word. `I` shows them, dim, after the others, each
title starting with `hidden`; `i` on one shows it again, and when an
`owner/*` entry hides it, the menu says so before taking that entry off. A
review of a hidden repository still opens, and `load` still loads one.

### In a review

The screen is laid out as in Lazygit. Four panels run down the left: 1
Review, the pull requests; 2 Files, the changed files by component and
folder; 3 Notes, the queued comments, conversations with Claude and threads on
GitHub; and 4 Diagram, the boxes of the level shown. The main panel, 0, on the
right shows what is selected on the left: the Review view, the diff of the
files selected, a note in its file's diff, or the diagram. When that runs past the main panel, a thumb
in its right border shows how much of it fits and where you are, as in
Lazygit. `Enter` gives the main panel the keys and `Esc` gives them back. The
bar at the bottom lists the keys that matter where you are, and `Space ?`
lists them all. A count goes before a move, such as `5j`, `3]c` or `12G`. In
a panel on the left, `1` to `4` jump to the panels, so a count there starts
with `5` to `9`, such as `7j` or `52j`; `jumpToBlock` in the config can free
them. In `Space ?` and every list that pops up, `j` `k` `gg` `G` and
`Ctrl-d` `Ctrl-u` `Ctrl-f` `Ctrl-b` move the list, after a count too; any
other key closes it and goes on to the panel under it.

On a stack, the Review panel lists All, then each pull request, and its cursor
chooses what the other panels show. On All they show the whole stack. On a
pull request, Files, Notes, Diagram and the main panel show only what it
changed, at once. Each count of files, lines and links counts only that: an
arrow between components or packages counts the links whose two ends the
filter shows. Unchanged code is context for every pull
request, so the filter never leaves it out. To read a run of pull requests
together, `v` starts a highlight on the pull request selected, and `j` and `k`
grow or shrink it, as Vim's visual mode highlights lines. What the cursor or
the highlight chose stays as you move to other panels, and when the review
loads again. Back in the Review panel, `j` and `k` choose again, and `Esc`
goes back to All. `]p` and `[p` move the cursor from any panel. A run of every
pull request is the whole stack. What the filter leaves out is dim: a box of
the diagram has a dotted outline, a link a dotted line, and a line of the diff
a dot in place of its bar. With no colour, a row of a panel on the left that
it leaves out starts with `·`. A comment still goes to the pull request that
last changed its line.

Every panel:

| Key | Does |
| --- | --- |
| `1` `2` `3` `4` | from a panel on the left: the Review, Files, Notes or Diagram panel |
| `0` | the main panel |
| `Esc` | closes a popup, or gives the keys back to the panel on the left |
| `+` `_` | gives the panel with the keys more of the screen, then all of it; `_` goes back |
| `Space v` | the Review view |
| `Space h` | the home screen |
| `Space m` | the diagram |
| `Space f` | finds a file, type or component as you type; `Ctrl-n` `Ctrl-p` move, `Enter` goes there |
| `R` `:refresh` | loads the review again: from GitHub, or a branch from its repository |
| `]p` `[p` | moves the Review panel's cursor to the next or previous pull request of a stack, or All |
| `Ctrl-o` `Ctrl-i` | back and forward through jumps |
| `gx` | opens the pull request, or the thread at the cursor, on GitHub |
| `q` | closes a popup |
| `:q` `ZZ` | quit |

The panels on the left:

| Key | Does |
| --- | --- |
| `j` `k` `gg` `G` | move; the main panel follows, and in Review on a stack, every other panel shows only what the cursor is on |
| `Ctrl-d` `Ctrl-u` `Ctrl-f` `Ctrl-b` `{n}G` | half a page or a page down or up, or row n |
| `h` `l` | the panel above or below, as Lazygit moves between its panels; `h` in Review goes round to Diagram, and `l` in Diagram to Review |
| `Enter` | gives the main panel the keys; in Files, on a component or a folder, folds it |
| `v` | in Review, on a stack, highlights the pull request selected; in Files, selects the row selected, and the diff shows every file the selection holds; `j` `k` grow or shrink either, and in Files `v` stops it |
| `Esc` | in Review, goes to All, to show the whole stack; in Files, stops the selection |
| `P` | in Review and Notes, posts for the pull request selected |
| `e` `dd` | in Notes, edits or drops the queued comment |
| `za` | in Files, folds a component or a folder |
| `-` and the diagram's other keys | in Diagram, as in the diagram below; `j` `k` go through the boxes in reading order |

The Review view, in the main panel:

| Key | Does |
| --- | --- |
| `j` `k` `gg` `G` | move |
| `Ctrl-d` `Ctrl-u` `Ctrl-f` `Ctrl-b` `{n}G` | half a page or a page down or up, or item n |
| `Enter` | opens the item at the cursor in its diff |
| `]]` `[[` | the next or previous section |
| `za` | folds or unfolds a section |
| `e` `dd` | edits or drops the queued comment at the cursor |
| `P` | posts: the actions offered for the pull request at the cursor, then exactly what one sends, then `y` |
| `gX` | opens the written review's drawings in the browser, from a page lazyreview writes that runs no script |

Posting offers **Add Comments**, **Approve With Comments** and **Approve**.
A press asks once more, then posts only if the queue still gives exactly the
set of comments the action offered, the pull request is open, its head has
not moved, every comment sits on a line of its diff, and GitHub answers as
the account the review was loaded as. A press whose review GitHub already
holds posts nothing twice. No action asks for changes, and no approval shows
on your own pull request. As GitHub requires, a comment goes only on lines of
the diff, and a range only inside one hunk.

The diagram, in the main panel, shows the components, then the packages of
one, then the types of one package, as rows of boxes:

| Key | Does |
| --- | --- |
| `h` `l` | the box left or right in the row, stopping at its ends |
| `j` `k` | the row below or above, at the same place from the left, or a shorter row's last box; the next row with that place goes back to it |
| `w` `b` `gg` `G` `{n}G` | the next, previous, first or last box, or box n in reading order |
| `Enter` `-` | steps in or out; on a type, `Enter` opens its diff |
| `gd` `gr` | follows a link of the box, or lists what links to it |
| `/` `?` `n` `N` | finds a type or file at any level |
| `f` | labels every box; typing a label jumps there |
| `Space l` `Space p` | lists the box's links, or its properties |
| `Space d` | opens the box's diff |
| `Space a` | draws every link faintly, or stops |

The diff, in the main panel, shows the files selected in Files, in the
tree's order: a file alone, every file under a folder or a component, or
every file a `v` selection holds. A jump to a file, from the finder, Notes or
the Review view, shows that file alone, and a box of the diagram its
component. Its lines are numbered as `gui.lineNumbers` in the config says:
each its own number, or how far each is from the cursor, as Vim's
`relativenumber` does, or none.

| Key | Does |
| --- | --- |
| `j` `k` `Ctrl-d` `Ctrl-u` `Ctrl-f` `Ctrl-b` | move by a line, half a page or a page |
| `h` `l` | scroll the code left or right, half its width at a time, until the longest line shown ends at the edge; the gutter and the headers stay put, and `…` marks text hidden on either side. With wrap on, or in a rendered file, they do nothing. The sideways wheel, or Shift and the wheel, scrolls too |
| `gg` `G` `{n}G` `:{n}` | the first or last line of this file, or line n of it |
| `]c` `[c` `]f` `[f` | the next or previous hunk, or file shown; `]f` and `[f` stay at the last and first |
| `]n` `[n` | the next or previous note: a queued comment, a conversation or a colleague's thread |
| `/` `?` `n` `N` | search the text |
| `za` `zo` `zc` `zR` `zM` | fold a file, a run of unchanged lines or a note |
| `V` | selects lines; `o` goes to the other end, `Esc` stops |
| `c` | comments on the line or the selection; on Claude's answer, uses it |
| `a` | asks Claude about the line or the selection; in a conversation, asks a follow-up |
| `e` `dd` | edits or drops the queued comment at the cursor; `dd` asks first |
| `gd` | shows the type at the cursor on the diagram |
| `Space s` `Space w` `Space o` | split or unified, wrap, the whole file |
| `Space r` | a Markdown file rendered, block by block, or back to its diff; `V`, `c` and `a` work on whole blocks |
| `-` | back to where it was opened from |

Changes made elsewhere, in another terminal or from the command line, show
within a second. When the review is loaded again, the app loads it
in place and keeps your view, folds and filter. With a draft open it says so
instead, and `R` loads it when you are ready.

A comment, a question or an edit is written in your own editor: the
config's `os.editor`, or with none the first of `VISUAL` and `EDITOR` that
is set, or `vi`, as git chooses. `LAZYREVIEW_EDITOR` overrides the config. Write above the cut line and save; what is below it says where the
comment goes and is never sent. Saving with no text sends nothing. When a
comment is refused, the bar at the bottom says why and `c` or `e` opens
your text again.

Ask Claude works as your own Claude Code: your settings, `CLAUDE.md`,
skills, MCP servers and model. Each conversation has a throwaway copy of the
code at the commit its lines belong to, in the data home, with no remote,
and keeps one Claude Code session, so a follow-up goes on from what Claude
found. A guard refuses anything that would change something outside the
copy, or leave the machine as a change, such as a push, a comment on GitHub
or a request that writes; reading stays allowed. While Claude works, the bar
says so and the conversation shows its latest steps; `x` stops it. An answer
goes on when the app quits, and shows when the review opens again. Each
answer keeps the steps Claude took, and a refused step says why.

The review threads already on GitHub, colleagues', bots' and your own, sit
under their lines, marked resolved or outdated. A thread on no line shown
sits at the top of its file. Each thread and each conversation starts
folded to one line that says who wrote, how many comments there are, and
whether it is resolved, outdated or waiting for Claude; `za` opens it. `R`
reads them again.

The server writes a token into `server.json` at each start, which only its
owner can read, and the terminal app sends it with each change.

## Components and packages

Without a config, files are grouped by folder, each group named by its
folder as it is on disk. A config names the parts:

```json
{
  "title": "Search and basket",
  "areas": [{ "id": "store", "title": "Store", "components": ["catalog", "basket"] }],
  "components": [
    { "id": "catalog", "title": "Catalog", "description": "Lists and finds items.",
      "packages": [{ "id": "search", "title": "Search", "paths": ["Sources/Catalog/Search/"] }] },
    { "id": "basket", "title": "Basket", "paths": ["Sources/Basket/"] }
  ],
  "links": [{ "from": "Basket", "to": "Search", "label": "finds items with" }],
  "summaries": [{ "from": "basket", "to": "catalog", "label": "reads" }]
}
```

The most specific path wins. `links` name the relationships that explain the
change, `summaries` put words on the arrow between two components, `groups`
draw many look-alike types as one box, and `contextTypes` include or exclude
unchanged types by name. The whole shape is at the top of
`src/build/config/config.mjs`; a config with mistakes is refused with every
mistake named.

## Where things are kept

Everything lives under `~/.lazyreview`: a bare clone of each repository, and
one folder per review holding its record (`review.json`: the pull requests,
the queue and the reviews sent), the drawn page (`page.json`), the review
threads GitHub had when it was loaded (`github.json`), any config and written
review, and one file per conversation. A branch's review lives under
`reviews/local/`, its record holding the branch, its base and the folder it
was read from; it is read from the repository itself, with no clone.

Every setting is in the config, `config.yml`: the data home itself, the
app's port, the Claude Code command and model, the editor and the accounts
the home screen lists. [Config](Config.md)
lists every setting, its default and the variable that overrides it, such
as `LAZYREVIEW_HOME` for the data home and `LAZYREVIEW_PORT` for the port.
`lazyreview config check` prints them as read. An earlier version's
`settings.json` is no longer read; `lazyreview config move` moves it in.

The server answers only requests addressed to 127.0.0.1 or localhost, and
takes changes only as JSON from a client carrying its token, as the terminal
app does.

## Code

The code takes lazygit's layers, each part in its own language: the
commands that run other programs sit apart from the logic that uses them,
and the logic apart from what the user sees. [lazyreview and lazygit](lazygit.md)
maps each part to lazygit's own.

- `bin/lazyreview` is the command. `src/cli` reads its arguments and runs
  one command; `src/app` reads the config and builds every service once.
- `src/commands` runs every other program: git and the clones, `gh`, Claude
  Code, the background server and the terminal app. `src/models` names a
  pull request, a branch or a review.
- `src/loader` loads a review, of pull requests or of a branch; `src/build`
  reads the diff, the types and the stack, and lays out the diagram with ELK
  (`vendor/`). `src/queue` holds the queue's rules and the one way a review
  is posted; `src/review` reads a written review and places GitHub's review
  threads; `src/ask` asks Claude; `src/home` lists the open pull requests.
  The terminal app now queues, posts and asks Claude itself; the server's
  queue and conversations stay as the reference its answers are checked
  against until the server goes.
- `src/store` keeps the files on disk, and `src/config` reads the config.
- `src/server` serves the JSON the terminal app reads and changes reviews
  through. `src/i18n/english.mjs` holds every sentence of the command and
  the server.
- `terminal/` is the terminal app, in Go with Bubble Tea, Lip Gloss, Bubbles
  and Glamour, laid out as lazygit's own: `internal/gui` with its views,
  contexts, controllers and presentation, `internal/i18n/english.go` for its
  words and `internal/theme` for its colours. It reads and changes reviews in
  the data home itself, through `internal/store`, `internal/review`,
  `internal/queue` and `internal/reviews`, posts through `gh`, and asks
  Claude through `internal/ask`.
  `lazyreview terminal` builds it and runs it.

Run the tests with `npm test`, and the terminal app's with `npm run
test:terminal`. They make their own throwaway repositories and answer for
GitHub and Claude Code with stand-ins, so they reach no network. The terminal
app's tests read the server's answers from `terminal/testdata/record.json`,
which a Node test checks against the server; when the server changes what it
answers, `LAZYREVIEW_RECORD=1 npm test` writes it again, with the data home
and GitHub's answers it was made from. A Go test checks that the Go code
gives each answer it owns, from the same stand-in data home.
