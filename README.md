# Review Desk

A web app, run on your own machine, for reviewing a pull request or a stack of
them. One command loads a pull request from its address; the app then shows a
clickable class diagram of the change, every diff, the comments you queue, and
the buttons that send your review to GitHub. Ask Claude about any line and the
answer appears beside it.

Nothing is built per review: the command writes the data, and the app, built
once, draws it. An agent loading a review spends only the tokens of one command.

## What you need

- Node.js 22 or later
- git, and the GitHub command line `gh`, signed in
- Claude Code (`claude`), signed in, for questions about a line; the rest works without it

There is nothing to install: the app has no dependencies beyond the two
libraries in `vendor/`.

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

Loading again refreshes the review in place, as does **Load again from
GitHub** in the app. A stack's review lives with its lowest pull request that
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
app. An agent that has written a review of the change can attach it, and the
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

## Using the app

- **Diagram.** The types the change adds and edits, in components and
  packages. Point at a type to see everything it links to; select it, or a
  component, to open its diff. On a stack, each pull request has a colour, and
  the strip under the bar narrows the diagram and the diffs to one of them.
- **Diffs.** Unified or split, wrapped lines, whole files, the unchanged lines
  between hunks, and a Rendered view of each Markdown file.
- **A line.** Select a line number to open a box on it; shift-select another
  for a range. **Ask Claude** asks about those lines in one short call that
  carries only the file's hunk and the code around it, and the answer stays
  under the line. **Queue comment** queues the text for the pull request. On a
  stack the comment goes to the pull request that last changed the line, at
  its own line number.
- **Review.** The pull request or the stack, any written review, the queued
  comments with Edit and Drop, and the buttons: **Add Comments**, **Approve
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
the queue and the reviews sent), the drawn page (`page.json`), any config and
written review, and one file per conversation. Settings come from the
environment:

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
- `src/loader` loads a review; `src/build` reads the diff, the types and the
  stack, and lays out the diagram with ELK (`vendor/`).
- `src/queue` holds the queue's rules and the one way a review is posted.
- `src/ask` asks Claude; `src/server` serves the app and its JSON.
- `app/` is the web app, plain JavaScript modules with no build step.

Run the tests with `npm test`. They make their own throwaway repositories and
answer for GitHub and Claude Code with stand-ins, so they reach no network.
