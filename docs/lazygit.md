# lazyreview and lazygit

lazyreview takes lazygit's way of laying out a program: its folders, its
layers, its strings, its theme and its config. Each part takes it in its
own language. The terminal app is Go, as lazygit is; the command and the
server stay JavaScript.

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

- `terminal/main.go` reads the flags, and `internal/app` builds `Common`,
  the server's client and the gui, then runs Bubble Tea.
- `internal/common` holds `Common`: the strings, the config handed over and
  the theme. `internal/i18n/english.go` holds `TranslationSet` and
  `EnglishTranslationSet()`, every string the terminal app shows, with
  named placeholders filled by `utils.ResolvePlaceholderString`.
- `internal/config` reads the part of the config lazyreview hands over:
  the config file's path, the editor, the theme's colours and every key.
  `internal/theme` turns the colours for the terminal's background into
  styles.
- `internal/commands/server` is the server's client and the types of its
  answers; `internal/commands/oscommands` opens an address in the browser,
  and writes the page of a written review's drawings that it opens.
- `internal/gui` is the app: `views` draw framed rectangles and the canvas;
  `context` holds one context per view, of lazygit's kinds, in a
  `ContextTree` with a focus stack whose top takes keys first;
  `controllers` give each context its bindings, help and hints, with the
  steps several share in `controllers/helpers`, the only layer that calls
  the server; `presentation` builds rows, with the diagram in
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
- Reading the config again while the app runs. A terminal session is
  short, and the server reads the accounts again for each list.

## The command and the server

### What lazygit does

- Every program runs through `pkg/commands`, one part per area
  (lazygit:`pkg/commands/git.go:18-56`), through `oscommands`
  (lazygit:`pkg/commands/oscommands/cmd_obj.go:14-40`). The gui reaches git
  only through it (lazygit:`pkg/gui/controllers/basic_commits_controller.go:143`),
  and models are plain structs (lazygit:`pkg/commands/models/github.go:3-10`).
- `pkg/app` builds what the program uses once
  (lazygit:`pkg/app/app.go:96-142`).

### What lazyreview adopts, and where

- `src/commands` runs every other program: `git` and the bare clones,
  `gh`, Claude Code, the background server and the terminal app. Nothing
  else does, and `test/layers.test.mjs` checks it.
- `src/models` holds the refs that name a pull request, a branch or a
  review, as plain values.
- `src/app` reads the config and builds every service once, and runs the
  server; `src/cli` reads the arguments and runs one command.
- `src/i18n/english.mjs` holds every sentence of the command and the
  server, as one frozen object by area; a sentence with values is a
  function. A test fails when an error is made from written words anywhere
  else.
- `test/layers.test.mjs` checks the direction of every import: the commands
  use only Node, the models and the strings; the logic and the store never
  reach the server, the command line or the start.

### What it leaves out, and why

- Views, contexts and controllers. The command has no screen: what it
  presents is the server's answers and the command's lines, which keep
  their place and take their words from the strings.

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
  of folders. `src/config/config-file.mjs` finds it.
- `src/config/schema.mjs` holds every setting once: its default, its check,
  the variable that overrides it and the words that describe it. The
  reader, `docs/Config.md`, `schema/config.json` and `lazyreview config
  defaults` are all made from it. `npm run docs` writes the docs, and a
  test fails when they are out of date.
- `src/config/user-config.mjs` reads the file onto the defaults, with
  js-yaml's core schema (`vendor/js-yaml-4.3.2.min.js`), merging maps key by
  key and replacing lists whole, and checks every setting.
- `gui.theme` with lazygit's colour names where the role is the same, and
  `keybinding` with lazygit's way of writing keys.
- `lazyreview config path`, `defaults`, `check` and `move`, for
  lazygit's `--print-config-dir` and `--config`
  (lazygit:`pkg/app/entry_point.go:204-208`), and the one-time move.
- Node reads the config; the terminal app receives its part as one JSON
  argument, `--user-config`, and decodes it strictly. One reader, one set
  of defaults, one checker and one set of messages serve both parts.

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
- Go reading YAML. Node reads it already for the server and the command,
  and the terminal app is only ever started by Node.

## Choices made in the building

- Two modules that run programs had touched files on disk. The clones'
  lock moved into the loader, and the background server's record into
  `src/app`, so the commands run programs and nothing else.
- The menu and the finder share one popup, so opening one closes the
  other, as before. Popups sit on the focus stack in a fixed order.
- A place's keys are the universal ones, then its own, its own winning a
  key both bind. The first key of an action is the one the help shows, so
  the menu's and posting's `close` list `<esc>` first.
- A popup matches a key against its bindings, and gives any other key to
  its own handling, which says whether the key passes on, as the menu
  passes it on.
- A number in a colour is handed over as text; the list of pull request
  colours is 8 long, as the stack's colours are.
- The data home in the file is a full path or starts with `~/`.
- `config check` prints the config as read, in the default config's shape,
  each value with where it came from beside it.
