# Review Desk

A web app, run on your own machine, for reviewing a pull request or a stack of
them, or a branch that has no pull request yet. One command loads a pull
request from its address; the app then shows a clickable class diagram of the
change, every diff, the comments you queue, and the buttons that send your
review to GitHub. Ask Claude about any line and the answer appears beside it.

Nothing is built per review: the command writes the data, and the app, built
once, draws it. An agent loading a review spends only the tokens of one command.

## What you need

- Node.js 22 or later
- git, and the GitHub command line `gh`, signed in
- Claude Code (`claude`), signed in, for questions about a line; the rest works without it

There is nothing to install: the app has no dependencies beyond the two
libraries and the one font in `vendor/`, and it loads nothing from the network.

## Start the app

    bin/review-desk start      # in the background
    bin/review-desk serve      # or in this terminal
    bin/review-desk stop

It listens on http://127.0.0.1:4747, on this machine only. `load` starts it
when it is not running, so starting it by hand is rarely needed.

## Load a pull request or a stack

    bin/review-desk load https://github.com/owner/repo/pull/12

A stack is every pull request of it, bottom first, each based on the head
branch of the one below:

    bin/review-desk load https://github.com/owner/repo/pull/12 \
      https://github.com/owner/repo/pull/13 https://github.com/owner/repo/pull/14

It prints what it loaded and where to open it:

    loaded #12, #13, #14: 24 files +966 -13, 8 components, 31 types
    open http://127.0.0.1:4747/review/github.com/owner/repo/12

Loading again refreshes the review in place, as does **Refresh** in the
app's bar, in every view. A stack's review lives with its lowest pull request that
already has one, and the comments queued on its members' own reviews move into
it. Options:

- `--account <login>` acts as that GitHub account, by its own token, without
  changing the account gh has active. The review keeps it.
- `--config <file>` names the components and packages to draw, as below. The
  review keeps it.
- `--context` also draws the unchanged code the change touches, set apart in
  a dashed component of its own, its code read only; `--no-context` stops it.
  The review keeps the choice.

`bin/review-desk list` prints every review with its address.

### For an agent

An agent asked to review a pull request runs one command and passes on the
address it prints:

    bin/review-desk load <pull request address>...

That is the whole job. Comments, questions and the review itself happen in the
app. A change with no pull request yet is one command too:

    bin/review-desk load --repo <folder> --base <ref> --head <branch>

An agent that has written a review of a pull request can attach it, and the
app then opens on it:

    bin/review-desk attach <pull request address> review.md

The written review is Markdown: a header naming what it read, then its
sections as `##` headings. A `file:line` in backticks, such as
`` `Basket.swift:12` ``, opens the diff at that line; `[Certain]`, `[Likely]`
and `[Guessing]` show as tags, and `#13` as that pull request's colour.

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

    bin/review-desk load --repo ~/code/widgets --base main --head feature/search

It prints what it loaded and where to open it:

    loaded feature/search against main: 6 files +210 -14, 2 components, 5 types
    open http://127.0.0.1:4747/review/local/widgets-1a2b3c4d/feature~search

The review is kept by repository and branch, so loading the same branch again,
from any worktree of the repository, refreshes it in place, as does
**Refresh** in the app. `--head` names a branch, or `HEAD` on one. It gets
the same diagram and diffs, and Ask Claude, but it is read only: it has no
queue and no buttons, since there is no pull request to post to. `--title
<text>` names it, where the branch's name is the default; `--config`,
`--context` and `--no-context` work as for a pull request. The review keeps
each of them.

## Using the app

- **Diagram.** The types the change adds and edits, in components and
  packages. Point at a type to see everything it links to; select it, or a
  component, to open its diff. On a stack, each pull request has a colour, and
  the strip under the bar narrows the diagram and the diffs to one of them.
- **Diffs.** Unified or split, wrapped lines, whole files, the unchanged lines
  between hunks, and a Rendered view of each Markdown file.
- **A line.** Select a line number to open a box on it. For a range, drag
  across line numbers, or shift-select another; on a touch screen, tap a
  second one. In the unified view a range may run from a removed line to an
  added one. **Ask Claude** asks about those lines in one short call that
  carries only the file's hunk and the code around it, and the answer stays
  under the line. **Queue comment** queues the text for the pull request. On a
  stack the comment goes to the pull request that last changed the line, at
  its own line number. As GitHub requires, a comment goes only on lines of the
  diff, and a range only inside one hunk; the box says when a choice breaks
  that, and still asks Claude.
- **A passage.** In the Rendered view of a Markdown file, select a passage and
  press the button beside it: the same box opens on the source lines the
  passage came from, quoting it.
- **Threads on GitHub.** The review threads already on the pull request,
  colleagues', bots' and your own, sit under their lines, marked resolved or
  outdated. A thread on no line shown sits at the top of its file. Each
  thread, and each conversation with Claude, folds to one line; resolved and
  outdated threads start folded. **Refresh** reads them again.
- **Review.** The pull request or the stack, any written review, the queued
  comments with Edit and Drop, every thread on GitHub, and the buttons: **Add Comments**, **Approve
  With Comments** and **Approve**. A press asks once more, then posts only if
  the queue still gives exactly the set of comments the button offered, the
  pull request is open, its head has not moved, every comment sits on a line
  of its diff, and GitHub answers as the account the review was loaded as. A
  press whose review GitHub already holds posts nothing twice. No button asks
  for changes, and no approval shows on your own pull request.

## Components and packages

Without a config, files are grouped by folder. A config names the parts:

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

Everything lives under `~/.review-desk`: a bare clone of each repository, and
one folder per review holding its record (`review.json`: the pull requests,
the queue and the reviews sent), the drawn page (`page.json`, with the review
threads GitHub had when it was loaded), any config and written review, and one
file per conversation. A branch's review lives under
`reviews/local/`, its record holding the branch, its base and the folder it
was read from; it is read from the repository itself, with no clone. Settings
come from the environment:

| Variable | Meaning | Default |
| --- | --- | --- |
| `REVIEW_DESK_HOME` | where reviews and clones are kept | `~/.review-desk` |
| `REVIEW_DESK_PORT` | the app's port, on this machine only | `4747` |
| `REVIEW_DESK_CLAUDE` | the Claude Code command | `claude` |
| `REVIEW_DESK_MODEL` | the model that answers questions | Claude Code's default |

The server answers only requests addressed to 127.0.0.1 or localhost, and
takes changes only as JSON from its own pages.

## Code

- `bin/review-desk` is the command; `src/cli` reads it.
- `src/loader` loads a review, of pull requests or of a branch; `src/local`
  names a branch's review; `src/build` reads the diff, the types and the
  stack, and lays out the diagram with ELK (`vendor/`).
- `src/queue` holds the queue's rules and the one way a review is posted;
  `src/review` reads a written review and places GitHub's review threads.
- `src/ask` asks Claude; `src/server` serves the app and its JSON.
- `app/` is the web app, plain JavaScript modules with no build step. Its
  look is one stylesheet, `app/styles.css`, with a dark and a light theme
  that follow the system setting.

Run the tests with `npm test`. They make their own throwaway repositories and
answer for GitHub and Claude Code with stand-ins, so they reach no network.
