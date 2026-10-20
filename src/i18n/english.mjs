/**
 * Every sentence the lazyreview command shows, in English: its lines and
 * its usage, every error, the config's words, and the written review's
 * refusals. The terminal app's own words are Go's. It imports nothing.
 *
 * `english` is frozen and grouped by area. Fixed text is a string; text
 * with values is a function that holds the sentence's shape, plurals
 * included. A function that refuses something takes why, as words or as a
 * list of them.
 */
const plural = (count, one, many = `${one}s`) => `${count} ${count === 1 ? one : many}`;
const pulls = (numbers, joiner = ', ') => numbers.map(n => `#${n}`).join(joiner);
const listed = (count, problems) => `${count}:\n${problems.map(p => `  - ${p}`).join('\n')}`;
const shown = value => JSON.stringify(value);

const USAGE = `usage: lazyreview <command> [options]

  load <pull link>...    load a pull request into the app, or a stack of
                         them, bottom first, and print where to open it;
                         loading one again refreshes it
      --account <login>  act as this GitHub account, not gh's active one
      --config <file>    JSON naming the components and packages to draw
      --context          also draw the unchanged code the change touches
      --no-context       stop drawing it, for a review loaded with it
  load --repo <folder> --base <ref> --head <branch>
                         load a branch with no pull request, read only, as
                         its diff from base would show; loading the same
                         branch again refreshes it
      --title <text>     name the review; the branch's name otherwise
      --config, --context and --no-context work as above
  attach <pull link> <file>
                         attach a written review, in Markdown, to the review
                         holding that pull request
  list                   list every review, with its address
  terminal [<address>]   review in this terminal, with Vim keys: the review
                         at the address load and list print, as
                         lazyreview://<review id>, or the one holding a
                         pull request link, which loads it first when no
                         review holds it yet; with no address, the open
                         pull requests of the accounts in the config
  config path            where the config file is, or would be, and whether
                         it exists
  config defaults        the default config, saying what each setting does
  config check           every setting's value, and where it came from: the
                         default, the config file or a variable
  config move [--check]  move settings.json and the LAZYREVIEW_ variables
                         into the config file, once; --check shows what it
                         would do, and writes nothing
  start                  say that lazyreview runs no server now
  stop                   stop every answer Claude is still giving, and a
                         server an older lazyreview left running
  help                   show this`;


export const english = frozen({
  cli: {
    usage: USAGE,
    withUsage: message => `${message}\n\n${USAGE}`,
    unknownCommand: command => `unknown command "${command}"`,
    notACommand: command => `unknown command ${command}`,
    unknownOption: (option, command) => `unknown option ${option} for ${command}`,
    needsValue: option => `${option} needs a value`,
    needsArguments: (command, count) => `${command} needs ${count} arguments`,
    takesArguments: (command, count) => `${command} takes ${count || 'no'} argument${count === 1 ? '' : 's'}`,
    needsRepo: option => `${option} needs --repo`,
    repoNeeds: option => `--repo needs ${option}`,
    loadNeedsSomething: 'load needs a pull request, or --repo with --base and --head',
    notBoth: 'load takes pull requests or --repo, not both',
    accountForPulls: '--account is for pull requests; a branch loads from its own repository',
    failed: why => `lazyreview: ${why}`,
    loaded: (name, counts) => `loaded ${name}: ${plural(counts.files, 'file')} +${counts.adds} -${counts.dels}, ${plural(counts.components, 'component')}, ${plural(counts.types, 'type')}`,
    open: address => `open ${address}`,
    note: note => `note: ${note}`,
    notLoaded: link => `${link} is not loaded; load it first`,
    reviewMoved: to => `that review moved to ${to}`,
    attached: (id, address) => `attached to ${id}; open ${address}`,
    noReviews: 'no reviews yet; load one with lazyreview load <pull link>',
    listedMoved: (id, to) => `${id}  moved to ${to}`,
    listed: (address, name, title) => `${address}  ${name}  ${title}`,
    noServer: 'lazyreview runs no server now; lazyreview terminal opens the app',
    answersStopped: count => `stopped ${plural(count, 'answer')} Claude was still giving`,
    oldServerStopped: 'stopped the server an older lazyreview left running',
    nothingRunning: 'nothing was running',
    loading: url => `loading ${url}…`,
    noReview: text => `there is no review ${text}; lazyreview list shows every review`,
    notALink: text => `"${text}" is not the address of a review or of a pull request; lazyreview list shows every review`,
    configUnreadable: (path, why) => `could not read the config ${path}: ${why}`,
    configNeeds: (given, commands) => `config needs one of ${commands.slice(0, -1).join(', ')} or ${commands.at(-1)}${given ? `, not "${given}"` : ''}`,
    checkIsForMove: '--check is for config move',
  },

  config: {
    about: {
      dataHome: ['Where lazyreview keeps reviews, clones and the copies of the code Claude answers in.'],
      claudeCommand: ['The Claude Code command that answers a question about a line.'],
      claudeModel: ['The model that answers. Empty: Claude Code\'s own default.'],
      editor: ['The command that opens a comment, a question or an edit.', 'Empty: VISUAL, then EDITOR, then vi, as git chooses.'],
      accounts: [
        'The GitHub accounts whose open pull requests the home screen lists.',
        'Each is read by its own gh token. A review asked of a team counts as',
        'asked of the account, for pull requests opened on or after `since`.',
      ],
      hiddenRepositories: [
        'Repositories the home screen leaves out, for every account: owner/name,',
        'or owner/* for every repository of the owner. A review of one still opens.',
      ],
      lineNumbers: [
        'How the terminal app\'s diff numbers its lines: absolute, each line its',
        'own number; relative, the cursor\'s line its own number and every other',
        'line its distance from it, as Vim\'s number and relativenumber do; or none.',
      ],
      theme: ['The terminal app\'s colours, for a dark terminal and a light one.', 'Each is #rrggbb or a terminal colour number from 0 to 255.'],
      keys: [
        'Keys as lazygit writes them: a letter, or <space>, <enter>, <esc>,',
        '<tab>, <up>, <down>, <left>, <right>, <c-x> for Ctrl-x. A sequence is',
        'its keys in a row, such as gg, ]c or <space>v. A list gives an action',
        'more than one key. <disabled> removes a key.',
      ],
      keySections: {
        universal: 'every screen',
        home: 'the home screen',
        reviewScreen: 'every panel of a review',
        panels: 'every panel on the left',
        reviewView: 'the Review view in the main panel',
        diagram: 'the diagram in the main panel',
        menu: 'every list that pops up',
      },
      accountsExample: ['accounts:', '  - login: lee', '  - login: lee-at-work', '    teams:', '      - name: example-org/mobile', '        since: 2026-10-01'],
      hiddenRepositoriesExample: ['hiddenRepositories:', '  - example-org/old-app', '  - example-bots/*'],
    },
    fileMissing: path => `LAZYREVIEW_CONFIG_FILE names ${path}, which does not exist`,
    unreadable: (path, why) => `${path} could not be read: ${why}`,
    yamlError: (reason, line, column) => `${reason} (${line}:${column})`,
    problem: (path, problem) => `${path} ${problem}`,
    problems: (path, problems) => `${path} has ${listed(plural(problems.length, 'problem'), problems)}`,
    notAMap: 'holds no settings, written as lazyreview config defaults shows them',
    unknown: key => `sets ${key}, which is not a setting`,
    retired: (key, why) => `sets ${key}, which is no longer a setting, so remove it: ${why}`,
    retiredWhy: {
      diagramPanelMove: side => `the Diagram panel switches panels as every panel on the left does, and keybinding.diagram.${side} moves to the box on the ${side} in the main panel's diagram`,
    },
    unused: (key, why) => `# ${key} does nothing now, so it can go: ${why}`,
    unusedWhy: {
      web: 'lazyreview has no web app; keybinding.reviewView.drawings opens a written review\'s drawings in the browser',
      server: 'lazyreview runs no server now; lazyreview terminal opens the app',
    },
    notASection: (key, value) => `gives ${key} ${shown(value)}, which is not a group of settings`,
    notAFolder: (key, value) => `gives ${key} ${shown(value)}, which is not a full path, or one starting with ~/`,
    notACommand: (key, value) => `gives ${key} ${shown(value)}, which is not a command`,
    notText: (key, value) => `gives ${key} ${shown(value)}, which is not text`,
    notAChoice: (key, value, choices) => `gives ${key} ${shown(value)}, which is not ${choices.slice(0, -1).join(', ')} or ${choices.at(-1)}`,
    notAccounts: (key, value) => `gives ${key} ${shown(value)}, which is not a list of accounts`,
    notRepositories: (key, value) => `gives ${key} ${shown(value)}, which is not a list of repositories`,
    notARepository: (key, value) => `gives ${key} ${shown(value)}, which is not a repository as owner/name, or owner/* for every repository of the owner`,
    notAnEntry: value => `${shown(value)} is not a repository as owner/name, or owner/* for every repository of the owner`,
    notAColour: (key, value) => `gives ${key} ${shown(value)}, which is not a colour: #rrggbb or a terminal colour number from 0 to 255`,
    notColours: (key, value, count) => `gives ${key} ${shown(value)}, which is not a list of ${count} colours`,
    notKeys: (key, value, why) => `gives ${key} ${shown(value)}, which is not a key or a list of keys: ${why}`,
    notNumberedKeys: (key, value, count) => `gives ${key} ${shown(value)}, which is not a list of ${count} keys, one for each by its place`,
    keys: {
      notText: 'a key is written as text',
      empty: 'a key is not empty',
      unclosed: 'a < starts a named key, such as <space>, and needs its >',
      disabledAlone: '<disabled> stands alone, for no key',
      unknownKey: token => `${token} is not a key; a named key is <space>, <enter>, <esc>, <tab>, <up>, <down>, <left>, <right>, or <c-x> for Ctrl-x`,
    },
    noLogin: number => `gives account ${number} no GitHub login`,
    accountKey: (login, key) => `gives ${login} "${key}", which an account does not have`,
    loginTwice: login => `names ${login} twice`,
    teamsNotList: login => `gives ${login} "teams" that are not a list`,
    teamNotNamed: login => `gives ${login} a team that is not named as organisation/team`,
    teamKey: (team, key) => `gives ${team} "${key}", which a team does not have`,
    sinceNotDate: team => `gives ${team} a "since" that is not a date such as 2026-10-01`,
    notMoved: (settings, config) => `${settings} is no longer read; lazyreview config move moves its accounts into ${config}`,
    source: source => (source === 'default' ? 'default' : source === 'file' ? 'the file' : source),
    movedHeader: 'lazyreview\'s config. docs/Config.md lists every setting and its default.',
    exists: 'it exists',
    missing: 'there is no file there yet, so every setting has its default',
    checked: path => `the config at ${path}, on top of the defaults:`,
    unset: path => `no config at ${path}, so every setting has its default:`,
    docs: {
      schemaTitle: 'lazyreview config',
      schemaDescription: 'lazyreview\'s config file, config.yml. docs/Config.md lists every setting and its default.',
      generated: 'written from the config\'s schema; do not change it by hand',
      noMarkers: name => `docs/Config.md has no START ${name} and END ${name} markers to write between`,
      variable: 'Variable',
      overrides: 'Overrides',
    },
    edit: {
      cannotChange: key => `${key} is written in a way lazyreview cannot change line by line`,
      failed: (path, why) => `${path} was not changed: ${why}; change it by hand, as docs/Config.md shows`,
    },
    move: {
      wouldWrite: path => `would write ${path}:`,
      from: (key, source) => `${key} from ${source}`,
      wouldRename: (path, name) => `would rename ${path} to ${name}`,
      refused: path => `${path} exists already, and the move never overwrites a config`,
      nothingWritten: 'nothing was written; lazyreview config move makes the move',
      nothing: settings => `there is nothing to move: there is no ${settings}, and no LAZYREVIEW_ variable is set`,
      movedExists: path => `${path} exists already, so settings.json cannot be renamed to it; nothing was written`,
      needsAccounts: 'needs "accounts", a list of {"login": "..."}',
      failed: (why, stuck) => (stuck.length
        ? `the move failed: ${why}; and this could not be put back: ${stuck.join('; ')}`
        : `the move failed, and nothing changed: ${why}`),
      wrote: path => `wrote ${path}:`,
      renamed: (path, name) => `renamed ${path} to ${name}`,
      stillSet: (name, key) => `${name} is still set, and overrides ${key} while it is; taking it out of your shell's setup is safe, and not needed`,
    },
  },

  terminalApp: {
    building: 'building the terminal app…',
    needsGo: 'the terminal app is built with Go 1.24 or later, and go is not on the PATH; install it from https://go.dev/dl',
    buildFailed: why => `could not build the terminal app: ${why}`,
    stopped: code => `the terminal app stopped with exit code ${code}`,
  },

  refs: {
    notAPull: (host, repo, pull) => `not a pull request: ${host}/${repo}#${pull}`,
    notAPullAddress: text => `"${text}" is not a pull request's address, such as https://github.com/owner/repo/pull/12`,
    notABranch: (repoKey, branch) => `not a branch review: ${repoKey} ${branch}`,
    notABranchId: text => `"${text}" is not a branch review's id, such as local/widgets-1a2b3c4d/feature~search`,
    notAReviewId: id => `"${id}" is not a review id, such as github.com/owner/repo/12`,
  },

  store: {
    gaveUp: path => `gave up waiting for ${path}`,
    heldElsewhere: path => `another process holds ${path}`,
    notWritten: why => `could not write the review, so every file is as it was: ${why}`,
    unreadable: (path, why) => `could not read ${path}: ${why}`,
  },

  review: {
    pullsName: numbers => pulls(numbers),
    branchName: (head, base) => `${head} against ${base}`,
  },

  written: {
    noHeader: 'the written review must start with a header between --- lines',
    headerLine: line => `the header line "${line}" is not "name: value"`,
    unknownField: field => `the header has an unknown field "${field}"`,
    needsHead: 'the header needs "head", the head commit the review read',
    pullOrStack: 'the header needs "pull", or "stack" for a stack, and not both',
    pullNumber: 'the header\'s "pull" must be the pull request number',
    stackNumbers: 'the header\'s "stack" must list two or more pull request numbers, bottom first',
    textBefore: 'text before the first ## section heading',
    noSection: 'the written review needs at least one ## section',
    emptySection: title => `"## ${title}" is empty; a section with nothing to say says so in one line`,
    problems: problems => `the written review has ${listed(plural(problems.length, 'problem'), problems)}`,
    otherPulls: (written, shown) => `the written review names ${pulls(written)}, but the review shows ${pulls(shown)}`,
    otherHead: (read, head) => `the written review read ${read}, but the review shows head ${head}; review the head the app shows`,
  },

});

function frozen(value) {
  if (value && typeof value === 'object') {
    for (const inner of Object.values(value)) frozen(inner);
    Object.freeze(value);
  }
  return value;
}
