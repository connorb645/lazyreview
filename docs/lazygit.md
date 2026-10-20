# lazyreview and lazygit

lazyreview takes lazygit's way of laying out a program: its folders, its
layers, its strings, its theme and its config. It is one Go program, as
lazygit is.

A citation starting `lazygit:` names a file and line of lazygit at
`v0.63.0`. Every other path is in this repository.

## The terminal app

### What lazygit does

- `main.go` reads the flags and hands over to `pkg/app`
  (lazygit:`main.go:15-23`). Below `pkg/` is one package per concern: `app`,
  `commands`, `config`, `common`, `gui`, `i18n`, `theme` and `utils`
  (lazygit:`docs/dev/Codebase_Guide.md:5-41`).
- The gui is views, contexts, controllers and helpers. A view is a named
  rectangle (lazygit:`pkg/gui/views.go:25-77`). A context is a view and its
  state, of a kind: side, main, popup or global
  (lazygit:`pkg/gui/types/context.go:12-35`). A stack of contexts says which
  has the keys (lazygit:`pkg/gui/context.go:17-23`). A controller gives a
  context its bindings, each with keys from the config, a handler and a
  description (lazygit:`pkg/gui/controllers/branches_controller.go:45-56`).
- Controllers may use helpers, contexts and views; helpers contexts and
  views; contexts only views (lazygit:`docs/dev/Codebase_Guide.md:72`).
- Presentation is pure functions from models to rows of styled text
  (lazygit:`pkg/gui/presentation/branches.go:29-44`).
- Every string is a field of `TranslationSet`, with the English ones in
  `EnglishTranslationSet()` (lazygit:`pkg/i18n/english.go:13`,
  lazygit:`pkg/i18n/english.go:1157`). A string with values names them as
  placeholders (lazygit:`AGENTS.md:258-280`).
- The theme turns the config's colours into styles
  (lazygit:`pkg/theme/theme.go:51-76`).
- The keys list and the bar show each key as a label made from the key
  bound (lazygit:`pkg/config/keynames.go:50`,
  lazygit:`pkg/gui/keybindings.go:48-61`).

### What lazyreview adopts, and where

- `main.go` hands the command line to `internal/app`, which runs
  one command; for `terminal` it builds `Common`, the reviews and the gui,
  then runs Bubble Tea.
- `internal/common` holds `Common`: the strings, the terminal app's part
  of the config and the theme. `internal/i18n/english.go` holds `TranslationSet` and
  `EnglishTranslationSet()`, every string the terminal app shows, with
  named placeholders filled by `utils.ResolvePlaceholderString`.
- `internal/config` reads `config.yml` itself, as the config section
  below says, and gives the terminal app its part: the config file's path,
  the data home, the editor, Claude Code, the theme's colours and every
  key. `internal/theme` turns the colours for the terminal's background
  into styles.
- `internal/models` holds plain values, as lazygit's
  `pkg/commands/models` does: the refs that name a pull request, a branch
  or a review, what a review's files keep, and what GitHub answers.
- `internal/store` reads and writes a review's folder in the data home.
  Each file is replaced whole, and a change to several files is all or
  none, under a lock folder that every process respects.
- `internal/review` builds what a page shows from what the store read: a
  file's hunks and whole text, the code a comment quotes, a Markdown
  file's blocks and a written review. `internal/queue` holds the queue's
  rules as pure functions, and the service that queues, changes, drops and
  posts comments.
- `internal/ask` asks Claude about lines: the guard on each tool call, the
  question, and the conversations, which it keeps, stops and deletes. An
  answer runs in a process of its own, the same program run as
  `lazyreview answer`, so it goes on when the app quits; Claude Code asks
  `lazyreview guard` about each tool call.
- `internal/reviews` is how the gui reads and changes a review. It answers
  from the data home, loads and refreshes reviews, and hears a change by
  watching the review's files.
- `internal/loader` loads a review: pull requests from GitHub, fetched into
  the repository's bare clone, or a branch straight from a repository on
  this machine. It builds the page through `internal/build`: `diff` reads
  the change, `languages` the types each file declares, `components` the
  config a change is drawn with, `placement` the component and package of
  each file, `model` the diagram's types and links, `unchanged` the
  unchanged code around the change, `stack` where a comment on each line
  of a stack goes, `layout` where each card sits, and `page` puts them
  together. `lazyreview load` runs the same loader.
- `internal/commands/github` runs `gh`: the account it answers as, a pull
  request, its files, reviews and review threads, sending a review, and
  the home screen's searches. `internal/commands/claude` runs Claude Code
  once per question, and `internal/commands/git` reads a repository,
  fetches into the bare clones and makes the throwaway copies of the code
  Claude works in. `internal/commands/oscommands` opens an address in the
  browser, writes the page of a written review's drawings that it opens,
  runs an answer in a process of its own, and stops a server an older
  lazyreview left running.
- `internal/gui` is the app: `views` draw framed rectangles and the canvas;
  `context` holds one context per view, of lazygit's kinds, in a
  `ContextTree` with a focus stack whose top takes keys first;
  `controllers` give each context its bindings, help and hints, with the
  steps several share in `controllers/helpers`, the only layer that reads
  or changes a review; `presentation` builds rows, with the diagram in
  `presentation/diagram`; `types` holds the values every layer shares.
- `internal/gui/layers_test.go` reads every package's imports and fails on
  one the rule forbids.
- `Space ?`, the bar's hints, the list after `Space` and the notices that
  name a key are made from the bindings, so a key changed in the config
  changes the words. `internal/keys` reads Vim-style key sequences and
  labels them.

### What it leaves out, and why

- gocui. Bubble Tea stays: every snapshot is the string `View()` returns,
  and `Update` is a pure function of the state and a message. A handler
  returns the next state, where lazygit's controllers change state through
  pointers.
- A theme in package variables. The theme is a value carried in `Common`,
  as no state is shared and changed. Each colour is one colour; bold stays
  in code, and lazygit's lists of a colour and its attributes are left out.
- Other languages and their translation. Nothing translates lazyreview, so
  a switch would have no user; a second language would be a second set
  with the same fields.
- Reading the config again while the app runs, but for the home screen's
  accounts and hidden repositories, which `internal/home` reads at each
  list and each change. A terminal session is short.

## The command

### What lazygit does

- Every program runs through `pkg/commands`, one part per area
  (lazygit:`pkg/commands/git.go:18-56`), through `oscommands`
  (lazygit:`pkg/commands/oscommands/cmd_obj.go:14-40`). The gui reaches git
  only through it (lazygit:`pkg/gui/controllers/basic_commits_controller.go:143`),
  and models are plain structs (lazygit:`pkg/commands/models/github.go:3-10`).
- `pkg/app` builds what the program uses once
  (lazygit:`pkg/app/app.go:96-142`).

### What lazyreview adopts, and where

- `bin/lazyreview` builds the program when its code changed, then runs it
  in its own process, so a signal to the command reaches the program.
- `internal/cli` reads the command line: the command, its options and
  arguments, the address `terminal` opens, and a review's address. It is
  pure, and its refusals end with the usage.
- `internal/app`'s `CommandLine` runs one command, building what it uses
  once: the store, a load, the stop of every answer, or the terminal app.
  Its sentences are in `internal/i18n/english.go` with the app's.
- Every other program runs through `internal/commands`, as the app's do,
  and `internal/gui/layers_test.go` checks the direction of every import.

### What it leaves out, and why

- A library for flags. A few commands, each with a few options, read in
  one small function, which keeps the usage and its refusals in lazyreview's
  own words.
- Views, contexts and controllers for the command. It has no screen: what
  it presents is its lines, which take their words from the strings.

## The config

### What lazygit does

- `config.yml` is found in `CONFIG_DIR`, then in the platform's config
  folders by the XDG base folders' order
  (lazygit:`pkg/config/app_config.go:626-644`,
  lazygit:`vendor/github.com/adrg/xdg/paths_darwin.go:16-31`), and
  `LG_CONFIG_FILE` names files instead
  (lazygit:`pkg/config/app_config.go:87-93`).
- Each file is read onto the defaults, from `GetDefaultConfig`
  (lazygit:`pkg/config/user_config.go:843`), then checked
  (lazygit:`pkg/config/app_config.go:154-219`). A bad file stops the start
  with its path and why (lazygit:`pkg/app/entry_point.go:139-142`). YAML
  merges maps and replaces lists.
- `gui.theme` holds a colour per role
  (lazygit:`pkg/config/user_config.go:231-257`), and `keybinding` the keys
  of each place (lazygit:`pkg/config/user_config.go:443`).
- `docs/Config.md` and `schema/config.json` are written from the config
  struct (lazygit:`pkg/jsonschema/generate.go:1-80`), and a check fails when
  they are out of date (lazygit:`.github/workflows/ci.yml:152-156`).

### What lazyreview adopts, and where

- `config.yml` in lazygit's place: on macOS
  `~/Library/Application Support/lazyreview/config.yml`, by the same order
  of folders. `internal/config/find_config_file.go` finds it.
- `internal/config/schema.go` holds every setting once: its
  default, its check, the variable that overrides it and the words that
  describe it. The reader, `docs/Config.md` and `schema/config.json` are
  made from it; `go generate ./internal/config` writes the docs, as
  lazygit writes its own, and a test fails when they are out of date.
- `internal/config` reads the file onto the defaults with YAML's
  core schema, through goccy/go-yaml, merging maps key by key and replacing
  lists whole, and checks every setting. A test holds it to the values and
  words recorded in `testdata/configs.json`. `config check` and
  `config move` write YAML as `testdata/config-commands.json`
  records it.
- `gui.theme` with lazygit's colour names where the role is the same, and
  `keybinding` with lazygit's way of writing keys.
- `lazyreview config path`, `defaults`, `check` and `move`, for
  lazygit's `--print-config-dir` and `--config`
  (lazygit:`pkg/app/entry_point.go:204-208`), and the one-time move.

### What it leaves out, and why

- `CONFIG_DIR` and `LG_CONFIG_FILE` as bare names. lazyreview's are
  `LAZYREVIEW_CONFIG_DIR` and `LAZYREVIEW_CONFIG_FILE`, so another
  program's variable cannot move its config.
- A list of config files. Nothing needs layered files yet.
- An empty file made at each start. A command writes only what it was
  asked to; `lazyreview config path` shows where the file goes.
- Unknown keys passing silently. A mistyped setting would do nothing, so
  lazyreview refuses it, naming it, with every other mistake at once.
- Changing old keys in place, with no copy. lazyreview's one change of
  format is a command run by hand, with a check that writes nothing first,
  and it renames the old file rather than deleting it.
- lazygit's older `jesseduffield/` folder, which lazyreview never had.

## Choices made in the building

- A module that runs programs must not touch files on disk: the clones'
  lock lives in the loader, and an older server's record is read in
  `internal/app`, so the commands run programs and nothing else.
- The menu and the finder share one popup, so opening one closes the
  other, as before. Popups sit on the focus stack in a fixed order.
- A place's keys are the universal ones, then its own, its own winning a
  key both bind. The first key of an action is the one the help shows, so
  the menu's and posting's `close` list `<esc>` first.
- A popup matches a key against its bindings, and gives any other key to
  its own handling, which says whether the key passes on, as the menu
  passes it on.
- A number in a colour is read as text; the list of pull request
  colours is 8 long, as the stack's colours are.
- The data home in the file is a full path or starts with `~/`.
- `config check` prints the config as read, in the default config's shape,
  each value with where it came from beside it.
