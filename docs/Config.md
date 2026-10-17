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
first line, and whether it exists on the next. lazyreview never makes the
file itself.

## Editor help

Put this line at the top of the file, and an editor that reads YAML
schemas checks every setting and completes its name as you type:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/connorb645/lazyreview/main/schema/config.json
```

## Environment variables

A variable overrides its setting while it is set, over the file and the
default. An empty variable counts as not set.

<!-- START VARIABLES: written by npm run docs from src/config/schema.mjs; do not change it by hand -->
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

<!-- START DEFAULT CONFIG: written by npm run docs from src/config/schema.mjs; do not change it by hand -->
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

## A bad file

A command that reads the config stops before it does anything when the
file has a mistake, and says what is wrong, with the file's path:

- YAML that cannot be read gives the reason, with its line and column.
- A setting lazyreview does not have is refused, by name, so a mistyped
  one never does nothing silently.
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
