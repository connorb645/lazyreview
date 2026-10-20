/**
 * Every sentence lazyreview shows, in English: the command's lines and its
 * usage, every error, the server's answers, the queue's and the loader's
 * words, the words a review's page draws, and the message that asks Claude
 * about a line. It imports nothing.
 *
 * `english` is frozen and grouped by area. Fixed text is a string; text
 * with values is a function that holds the sentence's shape, plurals
 * included. A function that refuses something takes why, as words or as a
 * list of them.
 */
const plural = (count, one, many = `${one}s`) => `${count} ${count === 1 ? one : many}`;
const pulls = (numbers, joiner = ', ') => numbers.map(n => `#${n}`).join(joiner);
const reasons = why => [why].flat().join('; ');
const refusal = what => why => `${what}: ${reasons(why)}`;
const listed = (count, problems) => `${count}:\n${problems.map(p => `  - ${p}`).join('\n')}`;
const named = (kind, id, number) => `${kind} ${id === null ? number : `"${id}"`}`;
const capital = words => `${words.charAt(0).toUpperCase()}${words.slice(1)}`;
const grouped = count => Number(count).toLocaleString('en-GB');
const counted = (count, one) => `${grouped(count)} ${count === 1 ? one : `${one}s`}`;
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
  list                   list every review, with where to open it
  terminal [<link>]      review in this terminal, with Vim keys: the review
                         at the address load printed, or the one holding a
                         pull request link, which loads it first when no
                         review holds it yet; with no link, the open pull
                         requests of the accounts in the config
  config path            where the config file is, or would be, and whether
                         it exists
  config defaults        the default config, saying what each setting does
  config check           every setting's value, and where it came from: the
                         default, the config file or a variable
  config move [--check]  move settings.json and the LAZYREVIEW_ variables
                         into the config file, once; --check shows what it
                         would do, and writes nothing
  serve                  run the app in this terminal
  start                  run the app in the background
  stop                   stop the app running in the background
  help                   show this`;

const these = count => (count === 1 ? 'this comment goes' : `these ${plural(count, 'comment')} go`);

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
    attached: (id, address) => `attached to ${id}${address ? `; open ${address}` : ''}`,
    noReviews: 'no reviews yet; load one with lazyreview load <pull link>',
    listedMoved: (id, to) => `${id}  moved to ${to}`,
    listed: (address, name, title) => `${address}  ${name}  ${title}`,
    running: address => `running at ${address}`,
    started: address => `started the app at ${address}`,
    runningAlready: address => `the app is running already at ${address}`,
    stopped: 'stopped',
    notRunning: 'not running',
    loading: url => `loading ${url}…`,
    noReview: text => `there is no review ${text}; lazyreview list shows every review`,
    notALink: text => `"${text}" is not the address of a review or of a pull request; lazyreview list shows every review`,
    configUnreadable: (path, why) => `could not read the config ${path}: ${why}`,
    configNeeds: (given, commands) => `config needs one of ${commands.slice(0, -1).join(', ')} or ${commands.at(-1)}${given ? `, not "${given}"` : ''}`,
    checkIsForMove: '--check is for config move',
  },

  app: {
    runningAlready: 'the app is running already; lazyreview stop stops it',
    running: address => `lazyreview is running at ${address}`,
    didNotStart: log => `the server did not start; see ${log}`,
  },

  config: {
    about: {
      dataHome: ['Where lazyreview keeps reviews, clones and the running app\'s files.'],
      port: ['The port the app listens on, on this machine only.'],
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
    },
    notASection: (key, value) => `gives ${key} ${shown(value)}, which is not a group of settings`,
    notAFolder: (key, value) => `gives ${key} ${shown(value)}, which is not a full path, or one starting with ~/`,
    notAPort: (key, value) => `gives ${key} ${shown(value)}, which is not a port number from 1 to 65535`,
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
    badPort: value => `LAZYREVIEW_PORT must be a port number, not "${value}"`,
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

  git: {
    failed: (command, why) => `git ${command} failed: ${why}`,
    notACommit: ref => `"${ref}" is not a commit in this repository`,
    missing: (commit, repo) => `commit ${commit.slice(0, 7)} is no longer in ${repo}; load the review again`,
  },

  github: {
    noToken: (account, host, why) => `gh has no token for ${account} on ${host}: ${why}`,
    tooSlow: (method, path) => `GitHub did not answer ${method} ${path} in time; nothing changed`,
    unreachable: (host, why) => `could not reach ${host}: ${why}`,
    refused: (method, path, why) => `GitHub refused ${method} ${path}: ${why}`,
    unreadable: (method, path, why) => `could not read GitHub's answer to ${method} ${path}: ${why}`,
    threadsRefused: (repo, number, why) => `GitHub refused the review threads of ${repo}#${number}: ${reasons(why)}`,
    noThreads: (repo, number) => `GitHub gave no review threads for ${repo}#${number}`,
  },

  claude: {
    tooSlow: minutes => `Claude did not answer within ${minutes} minutes`,
    stopped: 'you stopped it',
    notApproved: 'it needed approval, and nobody is there to give it',
    sandboxed: 'the sandbox stopped it: shell commands change only the copy, and reach only the hosts allowed',
    noAnswer: why => `Claude Code did not answer: ${why}`,
    exitCode: code => `it stopped with exit code ${code}`,
    notFound: command => `${command} was not found`,
    couldNotAnswer: why => `Claude could not answer: ${why}`,
    tooManyTurns: turns => `Claude worked for ${turns} turns without answering; ask about less at once`,
    nothing: 'no answer',
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
    noBuild: 'the change does not name the page build its window was drawn from',
    loadedAgain: 'the review was loaded again after this window opened, so its lines may have moved; reload it, then try again',
    notWritten: why => `could not write the review, so every file is as it was: ${why}`,
    unreadable: (path, why) => `could not read ${path}: ${why}`,
  },

  server: {
    onlyHere: 'this server answers only on 127.0.0.1 and localhost',
    changesFrom: 'changes come only from a client with the server\'s token',
    sendJSON: 'send JSON',
    nothingAt: (method, path) => `nothing at ${method} ${path}`,
    noSuchReview: 'no such review',
    noFile: refusal('no file'),
    notAFile: path => `${path} is not a file of this review`,
    textMissing: 'a text field is missing',
    giveLinks: 'give the pull requests to load as links, bottom first',
    notANumber: value => `"${value}" is not a line or pull request number`,
    sides: 'the side is RIGHT or LEFT',
    giveRepositories: 'give the repositories shown as a list of their names',
    giveHidden: 'give a repository to hide, or the entries to take out of the hidden list',
    tooLarge: 'the request is too large',
    notJSON: 'the request is not JSON',
    stoppedWatching: (id, why) => `lazyreview: stopped watching ${id}: ${why}`,
    unreadableAfterChange: (id, why) => `lazyreview: could not read ${id} after a change: ${why}`,
  },

  ask: {
    system: [
      'You help someone review a change to code, from lazyreview.',
      'Your working folder is a throwaway copy of the codebase as it stands after the change: a git repository, with a branch for the change and one for its base.',
      'Read, search, run and change anything in it, such as tests and builds, to find what the answer needs, such as callers, tests and definitions. Work in the copy itself: nothing you change there leaves it, and it is thrown away after, so there is nothing to clean up.',
      'Anything that would change something outside the copy, or leave this machine as a change, such as a push, a comment on GitHub, a message, or a request that writes to a web service, is refused, as nobody is there to approve it. Do not try to get round a refusal; say what you could not do.',
      'The message holds the part of the diff around the lines they ask about, the code around those lines, and their question.',
      'Answer in plain words and short sentences, and only what they asked.',
      'Say so when what you found is not enough to be sure.',
      'When they ask you to write a review comment, reply with the comment text alone, as they would write it: short, plain, one point.',
      'Never use the em dash; use a plain dash.',
    ].join(' '),
    pull: (pull, title, description) => `Pull request #${pull}: ${title}${description ? `\n\nIts description:\n${description}` : ''}`,
    stack: (members, owner) => `A stack of pull requests, bottom first, reviewed together:\n${members.map(m => `#${m.pull}: ${m.title}`).join('\n')}${owner ? `\n\n${owner}` : ''}`,
    branch: (branch, title, base) => `Branch ${branch}${title !== branch ? `: ${title}` : ''}. It has no pull request yet, and its diff is against ${base}.`,
    codebase: (folder, pull, branch, base) => `The codebase as it stands after ${pull ? `#${pull}` : 'this change'} is in ${folder}, your working folder, with the branch ${branch} checked out and the branch ${base} at the base of the change.`,
    changes: (files, more) => `The files this change touches:\n${files.map(f => `${f.path} (${f.state})`).join('\n')}${more ? `\nand ${more} more` : ''}`,
    file: (path, state, at) => `File: ${path} (${state})${at ? `, at ${at}` : ''}`,
    added: 'new',
    deleted: 'deleted',
    renamed: from => `renamed from ${from}`,
    changed: 'changed',
    unchanged: 'unchanged by this change',
    about: (lines, pull) => `They ask about ${lines}${pull ? `, which #${pull} changed` : ''}.`,
    there: (pull, lines) => `In the codebase, after #${pull}, they are ${lines}.`,
    diff: hunk => `The diff of this file around those lines:\n\`\`\`diff\n${hunk}\n\`\`\``,
    head: (window, text, top) => `The file at the head${top ? ` of #${top}` : ''}${window ? `, lines ${window.from}-${window.to} of ${window.count}` : ''}:\n\`\`\`\n${text}\n\`\`\``,
    conversation: messages => `The conversation so far:\n${messages.map(m => `${m.from === 'claude' ? 'You answered' : 'They asked'}: ${m.text}`).join('\n\n')}`,
    question: text => `Their question:\n${text}`,
    followUp: text => `Their follow-up question, on the same lines:\n${text}`,
    steps: {
      read: path => `Read ${path}`,
      searched: (pattern, path) => `Searched for ${pattern}${path ? ` in ${path}` : ''}`,
      listed: pattern => `Looked for files named ${pattern}`,
      ran: command => `Ran ${command}`,
      edited: path => `Edited ${path}`,
      wrote: path => `Wrote ${path}`,
      fetched: address => `Read ${address}`,
      searchedWeb: query => `Searched the web for ${query}`,
      helper: what => `Asked a helper to ${what}`,
      skill: name => `Used the skill ${name}`,
      planned: 'Planned its steps',
      mcp: (tool, server) => `Used ${tool} on ${server}`,
      used: tool => `Used ${tool}`,
    },
    askFirst: 'ask something first',
    needsLine: 'a conversation needs a file, a side and a line',
    notAsked: refusal('not asked'),
    noConversation: (id, review) => `there is no conversation ${id} on ${review}`,
    noCodebase: why => `could not make a copy of the code for Claude: ${why}`,
    stillWorking: 'Claude is still working on this conversation; wait for its answer, or stop it',
  },

  guard: {
    marker: 'lazyreview refused this',
    outside: 'lazyreview refused this: the file is outside the copy of the code, and only the copy may change',
    shell: kind => `lazyreview refused this: ${({ push: 'it would push', post: 'it would send a change to a web service', publish: 'it would publish a package', elsewhere: 'it would reach a program or a machine outside the copy' })[kind]}, and nobody is there to approve it`,
    gh: 'lazyreview refused this: gh may only read, one command at a time',
    mcp: (server, tool) => `lazyreview refused this: ${tool} on ${server} may change something, and only tools that read are allowed`,
    tool: tool => `lazyreview refused this: ${tool} is not allowed while nobody is there to approve it`,
    failed: why => `lazyreview refused this, as its guard could not read the call: ${why}`,
  },

  review: {
    lines: n => {
      const side = s => (s === 'LEFT' ? 'removed line' : 'line');
      const startSide = n.start_side || n.side;
      if (!n.start_line) return `${side(n.side)} ${n.line}`;
      if (startSide !== n.side) return `${side(startSide)} ${n.start_line} to ${side(n.side)} ${n.line}`;
      return `${side(n.side)}s ${n.start_line}-${n.line}`;
    },
    none: id => `there is no review ${id}`,
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

  queue: {
    branchTakesNone: 'A branch with no pull request takes no comments. Ask Claude about these lines instead.',
    moved: 'This review moved, so nothing can be queued here.',
    notInChange: 'This file is not in the change, so a comment cannot go on it. Ask Claude about it instead.',
    notADiffLine: 'GitHub takes comments only on lines of the diff, and this one is not. Ask Claude about it instead.',
    leavesHunk: 'GitHub takes a range only inside one hunk of the diff, and this one leaves it. Pick lines inside one hunk, or ask Claude about these.',
    backwards: 'A range runs from its first line to its last, and this one does not.',
    notThisReview: (own, given) => `This review is #${own}'s, not #${given}'s.`,
    otherHeads: numbers => `This page shows other heads of ${pulls(numbers)} than the review holds. Load the stack again, then queue.`,
    lastChangedBy: (last, given) => `#${last} last changed this line, not #${given}.`,
    notShownBy: pull => `#${pull}'s diff does not show this line.`,
    notAllTo: pull => `These lines do not all go to #${pull}. Comment on fewer lines, or choose another pull request.`,
    notAllToOne: 'These lines do not all go to one pull request of the stack. Comment on fewer lines, or one comment per pull request.',
    choose: numbers => `Unchanged, and shown by ${pulls(numbers, ' and ')}. Choose the pull request it goes on.`,
    leftStack: pull => `#${pull} is not in this stack any more.`,
    ended: (pull, state) => `#${pull} is ${state}, so nothing can be queued for it.`,
    goes: (pull, lines, path) => `Goes on #${pull}, ${lines}${path ? ` of ${path}` : ''}.`,
    sentence: words => `${capital(words)}.`,
    press: {
      'comment': (where, count) => `${capital(these(count))} on ${where} under your name, exactly as shown.`,
      'approve-with-comments': (where, count) => `You approve ${where}, and ${these(count)} with the approval under your name, exactly as shown.`,
      'approve': (where, count) => `You approve ${where} with no comments.${count ? ` The queued ${plural(count, 'comment')} ${count === 1 ? 'stays' : 'stay'} queued, unposted.` : ''}`,
    },
    emptyText: 'the text is empty',
    notOnDiff: 'is not on a line of the diff',
    startsOff: (side, line) => `starts on ${side === 'LEFT' ? 'removed ' : ''}line ${line}, which is not on a line of the diff`,
    acrossHunks: (side, line) => `runs from ${side === 'LEFT' ? 'removed ' : ''}line ${line} across more than one hunk of the diff, and GitHub takes a range only inside one hunk`,
    startsAfter: (side, line) => `starts on ${side === 'LEFT' ? 'removed ' : ''}line ${line}, which does not come before its last line in the diff`,
    reviewMoved: to => `this review moved to ${to}; nothing changed here`,
    notInStack: pull => `#${pull} is not in this stack any more`,
    notInReview: pull => `#${pull} is not part of this review`,
    noVersion: id => `the change does not name the version of comment ${id} it was made on`,
    changedSince: (id, body) => `comment ${id} was changed after this window showed it, and now reads "${body}"`,
    noComment: id => `no queued comment ${id}`,
    notQueued: refusal('not queued'),
    notChanged: refusal('not changed'),
    notDropped: refusal('not dropped'),
    nothingPosted: refusal('nothing was posted'),
    headMoved: (pull, from, to) => `#${pull} moved from ${from} to ${to}; load it again, then queue`,
    at: (path, start, line, pull) => `${path}:${start ? `${start}-` : ''}${line}${pull ? ` of #${pull}` : ''}`,
    placed: (at, problem) => `${at} ${problem}`,
    notAnAction: action => `"${action}" is not an action; nothing was posted`,
    notOffered: (action, pull) => `"${action}" is not offered for #${pull}; nothing was posted`,
    otherSet: (shown, action, now) => `the page showed set ${shown}, but "${action}" now gives set ${now}; nothing was posted`,
    otherAccount: (now, loaded) => `GitHub answers as ${now} now, but this review was loaded as ${loaded}; nothing was posted`,
    alreadyDone: url => `already done as ${url}; posted nothing new`,
    ownPull: 'this is your own pull request, which GitHub does not let you approve',
    readBack: (url, state, back, count) => `posted ${url}, but GitHub reads back state ${state} with ${back} of ${plural(count, 'comment')}: check it`,
    posted: {
      'comment': (count, url) => `posted ${plural(count, 'comment')} as one review: ${url}`,
      'approve-with-comments': (count, url) => `approved with ${plural(count, 'comment')}: ${url}`,
      'approve': (count, url) => `approved, with no comments: ${url}`,
    },
    pullEnded: (pull, state) => `#${pull} is ${state}`,
    headMovedFrom: (from, to) => `the head moved from ${from} to ${to}`,
    comment: (id, path, line) => `comment ${id} (${path}:${line})`,
    commentProblem: (comment, problem) => `${comment} ${problem}`,
    commentText: (comment, problem) => `${comment}: ${problem}`,
    commentStale: (comment, why) => `${comment} is stale: ${why}; queue it again or drop it`,
    stillOpen: (pull, count) => `#${pull} is still open with ${plural(count, 'queued comment')}, so it cannot leave the stack; post or drop them first`,
    leftTheStack: (pull, state) => `#${pull} left the stack (${state})`,
    renumbered: (pull, id, next) => `#${pull}'s comment ${id} is now ${next}`,
    staleHead: (from, pull, to) => `queued at ${from}, and #${pull} is now at ${to}`,
    staleStart: (pull, now, before) => `queued on a removed line, and #${pull}'s diff now starts from ${now}, not ${before}`,
    markedStale: (pull, count, why) => `#${pull}: ${plural(count, 'queued comment')} marked stale (${why}); queue each again or drop it`,
  },

  loader: {
    giveLinks: 'give the pull request to load, or every pull request of a stack, bottom first',
    otherRepository: (other, first) => `a stack is in one repository, but ${other} is not ${first}`,
    namedTwice: 'a pull request is named twice',
    notAStack: problems => `these are not a stack, bottom first: ${reasons(problems)}`,
    notOnBelow: (above, base, below, head) => `#${above} is based on ${base}, not on #${below}'s branch ${head}`,
    alsoReviews: (id, others) => `${id} also reviews ${pulls(others)}, which this load does not name; load the whole stack, bottom first`,
    threadsUnread: (pull, why) => `#${pull}'s review threads on GitHub are not shown, as they could not be read: ${why}`,
    stackTitle: (first, last) => `Stack #${first} to #${last}`,
    movedLoadThat: to => `this review moved to ${to}; load that one`,
    notABranch: (head, path) => `"${head}" is not a branch of ${path}; give a branch, so loading it again updates the same review`,
    noDifference: (base, head) => `there is no difference between ${base} and ${head}`,
    leftOut: paths => `whole text left out for ${plural(paths.length, 'large file')}, which show their diff only: ${paths.join(', ')}`,
  },

  components: {
    problems: problems => `the config has ${listed(plural(problems.length, 'problem'), problems)}`,
    root: 'the config',
    contextTypes: 'contextTypes',
    component: (id, number) => named('component', id, number),
    package: (component, id, number) => `${component}, ${named('package', id, number)}`,
    area: (id, number) => named('area', id, number),
    group: (id, number) => named('group', id, number),
    link: number => `link ${number}`,
    summary: number => `summary ${number}`,
    notAnObject: label => `${label} is not an object`,
    unknownKey: (label, key) => `${label} has an unknown key "${key}"`,
    needs: (label, key) => `${label} needs "${key}"`,
    notText: (label, key) => `${label}: "${key}" must be a non-empty string`,
    badId: label => `${label}: id must be lower case letters, digits and dashes`,
    idTwice: (label, id) => `${label}: id "${id}" is used twice`,
    notAList: (label, key) => `${label}: "${key}" must be a list`,
    notStrings: (label, key) => `${label}: "${key}" must hold non-empty strings`,
    badColor: (label, colors) => `${label}: color must be one of ${colors.join(', ')}`,
    badContext: label => `${label}: "context" must be true or false`,
    needsPath: label => `${label} needs at least one path`,
    needsPaths: label => `${label} needs paths or packages`,
    needsComponent: label => `${label} needs at least one component`,
    inTwoAreas: (label, component) => `${label}: component "${component}" is already in another area`,
    badKind: label => `${label}: kind must be use or has`,
    badMax: 'contextTypes: max must be a whole number, 0 or more',
    areaUnknown: (area, component) => `area "${area}" names component "${component}", which the config does not declare`,
    summaryUnknown: component => `a summary names component "${component}", which the config does not declare`,
  },

  diagram: {
    change: ({ subject, base, head, mergeBase, commits }) => [...subject, ...(base ? [[' on ', false], [base, true]] : []), [': head ', false], [head, true],
      [', merge base ', false], [mergeBase, true], [`, ${plural(commits, 'commit')}`, false]],
    pullSubject: pull => [['Pull request ', false], [`#${pull}`, true]],
    stackSubject: (first, last) => [['Stack ', false], [`#${first}`, true], [' to ', false], [`#${last}`, true]],
    branchSubject: branch => [['Branch ', false], [branch, true]],
    repositoryRoot: 'Repository root',
    unchangedCode: 'Unchanged code',
    otherFiles: 'Other files',
    contextDescription: 'Code the change uses or plugs into, unchanged, shown for context.',
    topLevel: name => `${name}, top level`,
    group: (count, title) => `${count} ${title}`,
    conforming: name => `types conforming to ${name}`,
    inheriting: name => `types inheriting from ${name}`,
    eachConforms: 'each conforms to',
    eachInherits: 'each inherits from',
    shows: 'shows',
    tests: 'tests',
    file: '«file»',
    extensionOf: names => `«extension»${names.length ? ` of ${names.join(', ')}` : ''}`,
    extension: name => `${name} extension`,
    links: count => plural(count, 'link'),
    summaryLine: (arrow, label, count) => `${arrow} ${label ? `${label} · ` : ''}${plural(count, 'link')}`,
    unchangedFiles: count => `${count} files · unchanged`,
    changedFiles: (count, adds, dels) => `${count} files · +${adds.toLocaleString('en-GB')}${dels ? ` −${dels.toLocaleString('en-GB')}` : ''}`,
    openDiff: 'Open diff →',
  },

  stack: {
    notAdded: 'no pull request\'s own diff adds this line as it reads here',
    notRemoved: 'no pull request\'s own diff removes this line as it reads here',
    notInDiff: 'the line is not in the combined diff',
    headMissing: (pull, head) => `#${pull}'s head ${head} is not a commit in this repository; fetch it with git fetch origin pull/${pull}/head`,
    notBuiltOn: (pull, below) => `#${pull} is not built on #${below}'s head: #${below} has commits #${pull} does not hold. ` +
      `Until #${pull} is rebased, its own diff also shows the older #${below} it holds; a comment on a line #${below}'s head still holds goes to #${below}.`,
    place: (pull, line, path) => `#${pull}, line ${line}${path ? ` of ${path}` : ''}`,
    changedIn: (pull, place) => `Changed in #${pull}. A comment here goes on ${place}.`,
    lastChanged: (pull, place) => `Unchanged here; #${pull} last changed it. A comment here goes on ${place}.`,
    noPull: why => `No pull request in the stack can take a comment here: ${why}.`,
    noneShows: 'Unchanged, and no pull request in the stack shows this line, so it takes no comment.',
    oneShows: place => `Unchanged. A comment here goes on ${place}.`,
    severalShow: places => `Unchanged, and shown by ${places.join(' and ')}. A comment here names which one it is for.`,
  },

});

function frozen(value) {
  if (value && typeof value === 'object') {
    for (const inner of Object.values(value)) frozen(inner);
    Object.freeze(value);
  }
  return value;
}
