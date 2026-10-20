# Config

lazyreview reads its settings from one file, `config.yml`, in the same
place lazygit keeps its own. Every setting has a default, so the file only
needs what you change, and a missing file means every default.

This is the user config. It is not the components config, the JSON a
review is drawn with, which `lazyreview load --config` takes.

## Where the file is

| Platform | Default path | Also searched, in order |
| --- | --- | --- |
| macOS | `~/Library/Application Support/lazyreview/config.yml` | `~/Library/Preferences`, `/Library/Application Support`, `/Library/Preferences`, `~/.config` |
| Linux | `~/.config/lazyreview/config.yml` | `/etc/xdg` |
| Windows | `%LOCALAPPDATA%\lazyreview\config.yml` | `%PROGRAMDATA%`, `%APPDATA%` |

lazyreview reads the first `lazyreview/config.yml` it finds in the default
folder, then in the others. Three variables move it:

- `LAZYREVIEW_CONFIG_FILE` names the file itself, which must exist.
- `LAZYREVIEW_CONFIG_DIR` names the folder that holds `config.yml`.
- `XDG_CONFIG_HOME` takes the place of the default folder, and
  `XDG_CONFIG_DIRS` of the folders searched after it.

`lazyreview config path` prints where the file is, or would be, on its
first line, and whether it exists on the next. lazyreview makes the file
only when asked: by `config move`, or by hiding a repository from the home
screen with no file yet, which makes one holding only that list.

## Editor help

Put this line at the top of the file, and an editor that reads YAML
schemas checks every setting and completes its name as you type:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/connorb645/lazyreview/main/schema/config.json
```

## Environment variables

A variable overrides its setting while it is set, over the file and the
default. An empty variable counts as not set.

<!-- START VARIABLES: written from the config's schema; do not change it by hand -->
| Variable | Overrides |
| --- | --- |
| `LAZYREVIEW_HOME` | `dataHome` |
| `LAZYREVIEW_PORT` | `server.port` |
| `LAZYREVIEW_CLAUDE` | `claude.command` |
| `LAZYREVIEW_MODEL` | `claude.model` |
| `LAZYREVIEW_EDITOR` | `os.editor` |
<!-- END VARIABLES -->

`VISUAL` and `EDITOR` are not lazyreview's own, so they do not override
`os.editor`. They choose the editor when `os.editor` is empty.

## Default

`lazyreview config defaults` prints the same text.

<!-- START DEFAULT CONFIG: written from the config's schema; do not change it by hand -->
```yaml
# Where lazyreview keeps reviews, clones and the running app's files.
dataHome: ~/.lazyreview                # LAZYREVIEW_HOME

server:
  # The port the app listens on, on this machine only.
  port: 4747                           # LAZYREVIEW_PORT

claude:
  # The Claude Code command that answers a question about a line.
  command: claude                      # LAZYREVIEW_CLAUDE
  # The model that answers. Empty: Claude Code's own default.
  model: ''                            # LAZYREVIEW_MODEL

os:
  # The command that opens a comment, a question or an edit.
  # Empty: VISUAL, then EDITOR, then vi, as git chooses.
  editor: ''                           # LAZYREVIEW_EDITOR

github:
  # The GitHub accounts whose open pull requests the home screen lists.
  # Each is read by its own gh token. A review asked of a team counts as
  # asked of the account, for pull requests opened on or after `since`.
  accounts: []
  # accounts:
  #   - login: lee
  #   - login: lee-at-work
  #     teams:
  #       - name: example-org/mobile
  #         since: 2026-10-01

  # Repositories the home screen leaves out, for every account: owner/name,
  # or owner/* for every repository of the owner. A review of one still opens.
  hiddenRepositories: []
  # hiddenRepositories:
  #   - example-org/old-app
  #   - example-bots/*

gui:
  # How the terminal app's diff numbers its lines: absolute, each line its
  # own number; relative, the cursor's line its own number and every other
  # line its distance from it, as Vim's number and relativenumber do; or none.
  lineNumbers: absolute
  theme:
    # The terminal app's colours, for a dark terminal and a light one.
    # Each is #rrggbb or a terminal colour number from 0 to 255.
    dark:
      defaultFgColor: '#f1f0fb'
      mutedFgColor: '#a09dbf'
      faintFgColor: '#6d6a8a'
      accentColor: '#a796ff'
      activeBorderColor: '#a796ff'
      inactiveBorderColor: '#4a4766'
      addedColor: '#7ce2a7'
      removedColor: '#ff9fb6'
      dimAddedColor: '#4d9a74'
      dimRemovedColor: '#b0647a'
      warningColor: '#f2b26a'
      selectedLineBgColor: '#3a3270'
      inactiveViewSelectedLineBgColor: '#26233d'
      visualSelectionBgColor: '#45475a'
      pullRequestColors: ['#3987e5', '#d95926', '#199e70', '#c98500', '#d55181', '#008300', '#9085e9', '#e66767']
      componentColors: {blue: '#7fa8f8', teal: '#4cd3c2', purple: '#b893fb', amber: '#f2b26a', olive: '#b8cc6a', rose: '#f08cb6', indigo: '#a4a9ff', slate: '#a3afc8', grey: '#9d9bb8'}
    light:
      defaultFgColor: '#17152f'
      mutedFgColor: '#615e7f'
      faintFgColor: '#9b98b5'
      accentColor: '#5233e6'
      activeBorderColor: '#5233e6'
      inactiveBorderColor: '#c4c0dc'
      addedColor: '#146b3a'
      removedColor: '#b21f43'
      dimAddedColor: '#5c9a78'
      dimRemovedColor: '#c8768a'
      warningColor: '#9c5212'
      selectedLineBgColor: '#dcd4ff'
      inactiveViewSelectedLineBgColor: '#efebfb'
      visualSelectionBgColor: '#c9cbd6'
      pullRequestColors: ['#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#e87ba4', '#008300', '#6250d6', '#e34948']
      componentColors: {blue: '#2c66a8', teal: '#0b6e6e', purple: '#7448ad', amber: '#9c5212', olive: '#5f6e12', rose: '#a8325e', indigo: '#4a4fb0', slate: '#556170', grey: '#6f7987'}

keybinding:
  # Keys as lazygit writes them: a letter, or <space>, <enter>, <esc>,
  # <tab>, <up>, <down>, <left>, <right>, <c-x> for Ctrl-x. A sequence is
  # its keys in a row, such as gg, ]c or <space>v. A list gives an action
  # more than one key. <disabled> removes a key.
  universal:              # every screen
    command: ':'
    keys: '<space>?'
    review: '<space>v'
    github: 'gx'
    close: ['q', '<esc>']
    quit: ['ZZ', '<c-c>']
    refresh: 'R'
    bigger: '+'
    smaller: '_'
  home:                   # the home screen
    down: ['j', '<down>']
    up: ['k', '<up>']
    top: 'gg'
    bottom: 'G'
    halfDown: '<c-d>'
    halfUp: '<c-u>'
    pageDown: '<c-f>'
    pageUp: '<c-b>'
    open: '<enter>'
    previousPanel: 'h'
    nextPanel: 'l'
    jumpToRepository: ['1', '2', '3', '4', '5', '6', '7', '8', '9']
    hideRepository: 'i'
    showHidden: 'I'
  reviewScreen:           # every panel of a review
    nextPull: ']p'
    previousPull: '[p'
    jumpBack: '<c-o>'
    jumpForward: '<c-i>'
    diagram: '<space>m'
    finder: '<space>f'
    home: '<space>h'
    focusMain: '0'
  panels:                 # every panel on the left
    down: ['j', '<down>']
    up: ['k', '<up>']
    top: 'gg'
    bottom: 'G'
    halfDown: '<c-d>'
    halfUp: '<c-u>'
    pageDown: '<c-f>'
    pageUp: '<c-b>'
    open: '<enter>'
    previousPanel: 'h'
    nextPanel: 'l'
    jumpToBlock: ['1', '2', '3', '4']
  reviewPanel:
    post: 'P'
    highlight: 'v'
  filesPanel:
    toggleFold: 'za'
    select: 'v'
  notesPanel:
    edit: 'e'
    drop: 'dd'
    post: 'P'
    stop: 'x'
  diagramPanel:
    nextBox: 'w'
    previousBox: 'b'
    back: '-'
    follow: 'gd'
    references: 'gr'
    search: '/'
    searchBack: '?'
    nextMatch: 'n'
    previousMatch: 'N'
    labels: 'f'
    links: '<space>l'
    properties: '<space>p'
    boxDiff: '<space>d'
    allLinks: '<space>a'
  reviewView:             # the Review view in the main panel
    down: ['j', '<down>']
    up: ['k', '<up>']
    top: 'gg'
    bottom: 'G'
    halfDown: '<c-d>'
    halfUp: '<c-u>'
    pageDown: '<c-f>'
    pageUp: '<c-b>'
    nextSection: ']]'
    previousSection: '[['
    fold: 'za'
    open: '<enter>'
    edit: 'e'
    drop: 'dd'
    post: 'P'
    drawings: 'gX'
  diff:
    down: ['j', '<down>']
    up: ['k', '<up>']
    halfDown: '<c-d>'
    halfUp: '<c-u>'
    pageDown: '<c-f>'
    pageUp: '<c-b>'
    top: 'gg'
    bottom: 'G'
    scrollLeft: ['h', '<left>']
    scrollRight: ['l', '<right>']
    nextHunk: ']c'
    previousHunk: '[c'
    nextFile: ']f'
    previousFile: '[f'
    search: '/'
    searchBack: '?'
    nextMatch: 'n'
    previousMatch: 'N'
    toggleFold: 'za'
    openFold: 'zo'
    closeFold: 'zc'
    openFolds: 'zR'
    closeFolds: 'zM'
    open: '<enter>'
    back: '-'
    split: '<space>s'
    wrap: '<space>w'
    whole: '<space>o'
    rendered: '<space>r'
    showType: 'gd'
    nextNote: ']n'
    previousNote: '[n'
    select: 'V'
    otherEnd: 'o'
    comment: 'c'
    ask: 'a'
    edit: 'e'
    drop: 'dd'
    stop: 'x'
  diagram:                # the diagram in the main panel
    left: ['h', '<left>']
    down: ['j', '<down>']
    up: ['k', '<up>']
    right: ['l', '<right>']
    nextBox: 'w'
    previousBox: 'b'
    top: 'gg'
    bottom: 'G'
    open: '<enter>'
    back: '-'
    follow: 'gd'
    references: 'gr'
    search: '/'
    searchBack: '?'
    nextMatch: 'n'
    previousMatch: 'N'
    labels: 'f'
    links: '<space>l'
    properties: '<space>p'
    boxDiff: '<space>d'
    allLinks: '<space>a'
  menu:                   # every list that pops up
    down: ['j', '<down>']
    up: ['k', '<up>']
    top: 'gg'
    bottom: 'G'
    halfDown: '<c-d>'
    halfUp: '<c-u>'
    pageDown: '<c-f>'
    pageUp: '<c-b>'
    open: '<enter>'
    close: ['<esc>', 'q']
  finder:
    down: ['<c-n>', '<down>']
    up: ['<c-p>', '<up>']
    open: '<enter>'
    close: '<esc>'
    clear: '<c-u>'
  posting:
    down: ['j', '<down>']
    up: ['k', '<up>']
    choose: ['1', '2', '3']
    open: '<enter>'
    close: ['<esc>', 'n', 'q']
    post: 'y'
    back: ['n', '<esc>', 'q']
  confirmation:
    confirm: 'y'
```
<!-- END DEFAULT CONFIG -->

A setting in the file replaces its default. A group of settings, such as
`server`, keeps the defaults of the settings the file leaves out. A list
replaces the default list whole.

`lazyreview config check` prints every setting as it is read, and where its
value came from: the default, the file or a variable.

## GitHub accounts

The home screen of the terminal app lists the open pull requests that
involve each account:

```yaml
github:
  accounts:
    - login: lee
    - login: lee-at-work
      teams:
        - name: example-org/mobile
          since: 2026-10-01
```

Each account is read by its own token, so each must be signed in to `gh`.
A review asked of one of an account's teams counts as asked of the
account, for pull requests opened on or after the team's `since` date. A
team with no `since` counts from any date.

## Hidden repositories

`github.hiddenRepositories` names repositories the home screen leaves out,
for every account and team:

```yaml
github:
  hiddenRepositories:
    - example-org/old-app      # one repository
    - example-bots/*           # every repository of an owner
```

Case does not count, as it does not in GitHub's names. A hidden repository
has no panel on the home screen, and the bar counts the ones hidden. Nothing
else changes: a review of one still opens, and `lazyreview
load` still loads one.

The home screen's keys write the list for you: `hideRepository`, `i`, hides
the repository selected or every repository of its owner, and on a hidden
one takes its entry off; `showHidden`, `I`, shows the hidden ones, dim.
Either key can be changed or disabled under `keybinding.home`. A change
edits the file the config was read from, `LAZYREVIEW_CONFIG_FILE` or
`LAZYREVIEW_CONFIG_DIR` included, in place: every comment, blank line,
commented-out line, order and value stays as you wrote it, and the list
keeps its own style. It writes a new file beside the old one and renames it
over it, so a failure leaves the old file whole, and a config that is a
link keeps its link. A `github` section written in braces, or a list in
brackets over several lines, is refused, saying so; edit those by hand.

## Keys

`keybinding` holds every key of the terminal app, by place and action,
written as lazygit writes keys:

| Write | For |
| --- | --- |
| `j`, `G`, `?`, `]` | that character |
| `<space>`, `<enter>`, `<esc>`, `<tab>` | those keys |
| `<up>`, `<down>`, `<left>`, `<right>` | the arrow keys |
| `<c-o>` | Ctrl-o; a terminal sends Tab for `<c-i>` |
| `gg`, `]c`, `<space>v` | a sequence, its keys in a row |
| `['j', '<down>']` | a list: each key does the action |
| `<disabled>` | no key |

```yaml
keybinding:
  universal:
    quit: ['ZZ', '<c-q>']
  diff:
    nextHunk: ']h'
    previousHunk: '[h'
  home:
    jumpToRepository: ['1', '2', '3', '<disabled>', '<disabled>', '<disabled>', '<disabled>', '<disabled>', '<disabled>']
```

An action the file leaves out keeps its default keys. A list replaces the
action's default keys whole. `jumpToRepository`, `jumpToBlock` and
`posting.choose` are numbered: each item is the key of that place, so the
list keeps its length, and `<disabled>` leaves a place with no key.

Where each part works:

- `universal`: every screen, the home screen and a review.
- `home`: the home screen of open pull requests.
- `reviewScreen`: every panel of a review, on the left and the main one.
- `panels`: every panel on the left; `reviewPanel`, `filesPanel`,
  `notesPanel` and `diagramPanel` add each panel's own.
- `reviewView`, `diff` and `diagram`: the main panel, as it shows each.
- `menu`, `finder`, `posting` and `confirmation`: the popups, which take
  keys first.

A place's own keys win over `universal`'s. Typing stays as it is: in the
command line, the finder's search and the diagram's labels, letters,
Backspace, Enter and Esc are text. `Space ?` lists the keys of the place
you are in, as they are bound.

## Line numbers

`gui.lineNumbers` sets how the terminal app's diff numbers its lines, as
Vim's `number` and `relativenumber` do:

```yaml
gui:
  lineNumbers: relative
```

| Value | The diff shows |
| --- | --- |
| `absolute` | each line's own numbers, in the old file and the new one; the default |
| `relative` | the cursor's line its own numbers, and every other line how far it is from the cursor: the count `j` or `k` takes to reach it |
| `none` | no numbers, so the code has their room |

With relative numbers, every row the cursor stops on counts, a removed line
as much as an added one, and a hunk's header or a note too, though those
show no number. In the split view, each side shows the same distance, and a
side with no line shows none. A rendered Markdown file numbers its rows the
same way.

## Theme

`gui.theme` colours the terminal app. It has two sets, `dark` and `light`,
and the app takes the one that suits the terminal's background. A colour is
`'#rrggbb'`, in quotes, since a YAML comment starts with `#`, or a terminal
colour number from 0 to 255:

```yaml
gui:
  theme:
    dark:
      accentColor: '#ff8800'
      selectedLineBgColor: 237
      componentColors: {teal: '#00d7d7'}
```

A colour the file leaves out keeps its default, so this changes three
colours of the dark set and nothing else. `pullRequestColors` are the
colours of a stack's pull requests, in order: a list of 8, which replaces
the default list whole. `dimAddedColor` and `dimRemovedColor` colour the
added and removed lines a stack's filter leaves out. `componentColors` are
the colours a components config can name, one by one. The names follow
lazygit's where the colour does the same thing, such as `activeBorderColor`
and `selectedLineBgColor`. Bold and plain text stay as they are.

## A bad file

A command that reads the config stops before it does anything when the
file has a mistake, and says what is wrong, with the file's path:

- YAML that cannot be read gives the reason, with its line and column.
- A setting lazyreview does not have is refused, by name, so a mistyped
  one never does nothing silently.
- A setting lazyreview no longer has is refused the same way, and the
  message says what does its work now.
- A setting that does nothing now, such as `keybinding.universal.web`,
  which opened the web app, still reads, so the file keeps working.
  `config check` names it, and why it can go.
- A value of the wrong kind names the setting, the value, and what it
  must be.
- Every mistake in the file is named at once.

`lazyreview help`, `config path` and `config defaults` still work, so a bad
file can always be found and fixed. The running app reads the accounts
again each time the home screen lists pull requests, and a mistake shows
there in place of the list.

## Moving from settings.json

Earlier versions read the accounts from `settings.json` in the data home,
and the other settings from `LAZYREVIEW_` variables only. A one-time move
puts them in the config:

```
lazyreview config move --check     # reads, writes nothing, and shows what it would do
lazyreview config move             # writes config.yml, and renames settings.json
lazyreview config check            # every setting, and where it came from
```

The move writes only values that differ from their defaults. It refuses
when a config file exists, and never overwrites one. It renames
`settings.json` to `settings.json.moved`, keeping its bytes, and never
deletes it. If a step fails, it puts back what it did.

Until the move, `settings.json` is no longer read, and the home screen says
so in place of its list. Every other command keeps working. A variable
still set after the move still overrides the config, so taking it out of
your shell's setup is safe, and not needed.

To go back, remove `config.yml` and rename `settings.json.moved` to
`settings.json`, then run the version you came from.
