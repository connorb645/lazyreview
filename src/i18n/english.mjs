/**
 * Every sentence lazyreview shows, in English: the command's lines and its
 * usage, every error, the server's answers, the queue's and the loader's
 * words, the words a review's page draws, the message that asks Claude
 * about a line, and in `web` the web app's words. The server serves this
 * module to the browser at /i18n/english.mjs, so it imports nothing.
 *
 * `english` is frozen and grouped by area. Fixed text is a string; text
 * with values is a function that holds the sentence's shape, plurals
 * included. A function that refuses something takes why, as words or as a
 * list of them. The web's words are HTML text, apart from those it puts in
 * the page as plain text, and a function's values arrive ready for the
 * page.
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
                         requests of the accounts in settings.json
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
    stopped: 'stopped',
    notRunning: 'not running',
    loading: url => `loading ${url}…`,
    noReview: text => `there is no review ${text}; lazyreview list shows every review`,
    configUnreadable: (path, why) => `could not read the config ${path}: ${why}`,
  },

  app: {
    runningAlready: 'the app is running already; lazyreview stop stops it',
    running: address => `lazyreview is running at ${address}`,
    didNotStart: log => `the server did not start; see ${log}`,
  },

  settings: {
    badPort: value => `LAZYREVIEW_PORT must be a port number, not "${value}"`,
    unreadable: (file, why) => `${file} could not be read: ${why}`,
    needsAccounts: file => `${file} needs "accounts", a list of {"login": "..."}`,
    noLogin: (file, number) => `${file} gives account ${number} no GitHub login`,
    loginTwice: (file, login) => `${file} names ${login} twice`,
    teamsNotList: (file, login) => `${file} gives ${login} "teams" that are not a list`,
    teamNotNamed: (file, login) => `${file} gives ${login} a team that is not named as organisation/team`,
    sinceNotDate: (file, team) => `${file} gives ${team} a "since" that is not a date such as 2026-10-01`,
  },

  git: {
    failed: (command, why) => `git ${command} failed: ${why}`,
    notACommit: ref => `"${ref}" is not a commit in this repository`,
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
    tooSlow: seconds => `Claude did not answer within ${seconds} seconds`,
    noAnswer: why => `Claude Code did not answer: ${why}`,
    exitCode: code => `it stopped with exit code ${code}`,
    notFound: command => `${command} was not found`,
    couldNotAnswer: why => `Claude could not answer: ${why}`,
    nothing: 'no answer',
  },

  terminalApp: {
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
    notAllowed: 'not allowed',
    noSuchPage: 'no such page',
    changesFrom: 'changes come only from the app\'s own pages, or a client with its token',
    sendJSON: 'send JSON',
    nothingAt: (method, path) => `nothing at ${method} ${path}`,
    noSuchReview: 'no such review',
    noFile: refusal('no file'),
    noWords: marker => `index.html names ${marker}, which the strings do not hold`,
    notAFile: path => `${path} is not a file of this review`,
    textMissing: 'a text field is missing',
    giveLinks: 'give the pull requests to load as links, bottom first',
    notANumber: value => `"${value}" is not a line or pull request number`,
    sides: 'the side is RIGHT or LEFT',
    tooLarge: 'the request is too large',
    notJSON: 'the request is not JSON',
    stoppedWatching: (id, why) => `lazyreview: stopped watching ${id}: ${why}`,
    unreadableAfterChange: (id, why) => `lazyreview: could not read ${id} after a change: ${why}`,
  },

  ask: {
    system: [
      'You help someone review a change to code.',
      'You see one file of it: the part of the diff around the lines they ask about, the code around those lines, and their question.',
      'Answer in plain words and short sentences, and only what they asked.',
      'Say so when the code you were shown is not enough to be sure.',
      'When they ask you to write a review comment, reply with the comment text alone, as they would write it: short, plain, one point.',
      'Never use the em dash; use a plain dash.',
    ].join(' '),
    pull: (pull, title, description) => `Pull request #${pull}: ${title}${description ? `\n\nIts description:\n${description}` : ''}`,
    stack: (members, owner) => `A stack of pull requests, bottom first, reviewed together:\n${members.map(m => `#${m.pull}: ${m.title}`).join('\n')}${owner ? `\n\n${owner}` : ''}`,
    branch: (branch, title, base) => `Branch ${branch}${title !== branch ? `: ${title}` : ''}. It has no pull request yet, and its diff is against ${base}.`,
    file: (path, state) => `File: ${path} (${state})`,
    added: 'new',
    deleted: 'deleted',
    renamed: from => `renamed from ${from}`,
    changed: 'changed',
    unchanged: 'unchanged by this change',
    about: (lines, pull) => `They ask about ${lines}${pull ? `, which #${pull} changed` : ''}.`,
    diff: hunk => `The diff of this file around those lines:\n\`\`\`diff\n${hunk}\n\`\`\``,
    head: (window, text) => `The file at the head${window ? `, lines ${window.from}-${window.to} of ${window.count}` : ''}:\n\`\`\`\n${text}\n\`\`\``,
    conversation: messages => `The conversation so far:\n${messages.map(m => `${m.from === 'claude' ? 'You answered' : 'They asked'}: ${m.text}`).join('\n\n')}`,
    question: text => `Their question:\n${text}`,
    askFirst: 'ask something first',
    needsLine: 'a conversation needs a file, a side and a line',
    notAsked: refusal('not asked'),
    noConversation: (id, review) => `there is no conversation ${id} on ${review}`,
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

  web: {
    page: {
      everyReview: 'Every review',
      views: 'Views',
      review: 'Review',
      diagram: 'Diagram',
      findType: 'Find a type',
      findTypeOrFile: 'Find a type or file',
      allLinksTitle: 'Draw every dependency, faintly',
      allLinks: 'All links',
      pullRequest: 'Pull request',
      loading: 'Loading&hellip;',
      reviews: 'Reviews',
      reviewsLead: 'The pull requests, stacks and branches loaded here, newest first.',
      loadingAReview: 'Loading a review',
      askAnAgent: 'Ask an agent to load a pull request, or run it yourself:',
      aStack: 'A stack is every pull request, bottom first:',
      aBranch: 'A branch with no pull request yet loads from its repository, read only:',
      components: 'Components',
      notation: 'Notation',
      hint: 'Drag or scroll to move. Pinch, <kbd>&#8984;</kbd>/<kbd>Ctrl</kbd> and scroll, or the buttons, to zoom. Point at a type to see everything it links to. Select a type or a component to open its diff.',
      diagramLabel: 'Class diagram of the change, grouped by component',
      zoomOut: 'Zoom out',
      fit: 'Fit',
      zoomIn: 'Zoom in',
      filesLabel: 'Files in this component',
      loadedAgain: 'This review was loaded again. Reload to see it; what you are writing stays here until then.',
      reload: 'Reload',
    },

    app: {
      reviewFailed: 'This review could not open',
      listFailed: 'The list of reviews could not load',
    },

    server: {
      notJSON: status => `the server sent something that is not JSON (${status})`,
      answered: status => `the server answered ${status}`,
    },

    counts: {
      files: count => counted(count, 'file'),
      lines: count => counted(count, 'line'),
      unchangedLines: count => counted(count, 'unchanged line'),
      comments: count => counted(count, 'comment'),
      moreComments: count => counted(count, 'more comment'),
      queuedComments: count => counted(count, 'queued comment'),
      messages: count => counted(count, 'message'),
      questions: count => counted(count, 'question'),
      others: count => counted(count, 'other'),
      reviewThreads: count => counted(count, 'review thread'),
      githubThreads: count => counted(count, 'GitHub thread'),
      conversations: count => counted(count, 'conversation'),
    },

    markdown: {
      image: (text, address) => `[image: ${text}${address ? ` - ${address}` : ''}]`,
    },

    home: {
      ended: (pull, state) => `#${pull} ${state}`,
      loaded: at => `loaded ${at}`,
      queued: count => `${count} queued`,
      none: 'No reviews yet.',
    },

    bar: {
      all: 'All',
      allPulls: (count, base, top) => `All ${count} pull requests, combined, from <b>${base}</b> to the head of <b>#${top}</b>`,
      ofCombined: totals => `${totals} of the combined diff`,
      backToReview: '&larr; Review',
      backToDiagram: '&larr; Diagram',
    },

    refresh: {
      refresh: 'Refresh',
      refreshing: 'Refreshing',
      branch: 'Load the branch again',
      pulls: 'Load again from GitHub',
      title: (what, at) => `${what}. Loaded ${at}.`,
      failed: why => `Not refreshed: ${why}`,
    },

    live: {
      unread: why => `A change made elsewhere could not be read: ${why}`,
    },

    review: {
      pull: number => `Pull request #${number}`,
      by: (pull, author) => `${pull} by <b>${author}</b>`,
      base: base => `Base <b>${base}</b>`,
      head: head => `Head <b>${head}</b>`,
      stack: count => `A stack of ${count} pull requests, each built on the one before, reviewed together`,
      byAuthor: author => `by <b>${author}</b>`,
      onBase: base => `on <b>${base}</b>`,
      atHead: head => `head <b>${head}</b>`,
      combined: 'Combined: ',
      branch: (branch, repo) => `Branch <b>${branch}</b> in <b>${repo}</b>, with no pull request yet`,
      moved: (pulls, review) => `This review moved: ${pulls} are reviewed together on ${review}. Its queued comments moved there, and nothing can be sent from here.`,
      range: (first, last) => `${first} to ${last}`,
      itsPull: 'its pull request',
      stacksReview: 'the stack’s review',
      branchReadOnly: 'This branch has no pull request yet, so its review is read only. The diagram and the diffs hold the change; select a line number in a diff to ask Claude about it.',
      noWritten: 'No written review is attached. The diagram and the diffs hold the change; select a line number in a diff to ask Claude about it or to queue a comment.',
      staleWritten: head => `This written review read head ${head}; the pull request has moved on since, so it may not match the diff.`,
      conversations: 'Conversations',
      waiting: 'Waiting for Claude…',
      onGitHub: 'On GitHub',
      githubThreads: (threads, open) => `${threads}, ${open} unresolved, as GitHub had them when this review was loaded.`,
      resolved: 'resolved',
      outdated: 'outdated',
      queuedComments: 'Queued comments',
      openDiagram: 'Open the diagram &rarr;',
      showOnDiagram: 'Show on the diagram',
      showPull: (pull, title) => `Show #${pull} on the diagram${title ? `: ${title}` : ''}`,
      theChange: 'The change',
      onThisPage: 'On this page',
      thisReview: 'This review',
      loaded: (at, account) => `Loaded ${at}${account ? ` as <b>${account}</b>` : ''}`,
      openAt: (path, line) => `Open ${path} at line ${line}`,
    },

    queue: {
      title: 'Queued comments',
      labels: { 'comment': 'Add Comments', 'approve-with-comments': 'Approve With Comments', 'approve': 'Approve' },
      yes: { 'comment': 'Yes, add comments', 'approve-with-comments': 'Yes, approve with comments', 'approve': 'Yes, approve' },
      cancel: 'Cancel',
      save: 'Save',
      edit: 'Edit',
      drop: 'Drop',
      note: 'Add Comments sends a review that only comments. Approve With Comments approves and sends the comments with it. Approve approves alone. None of them asks for changes.',
      stackNote: 'A comment goes on the pull request that last changed its line, shown as → #number. Each pull request has its own buttons and gets its own review.',
      empty: 'No comments queued. Select a line number in the diff, write the comment, and press Queue comment. Ask Claude there to draft one.',
      noneFor: pull => `No comments queued for #${pull}.`,
      staleHere: 'Nothing queued here can be sent until each comment marked stale is queued again or dropped.',
      goesOn: target => `Each comment goes on ${target} exactly as shown.`,
      ended: (target, state, at, stacked) => `${target} was ${state}${at ? ` on ${at}` : ''}. Nothing can be sent ${stacked ? 'to it' : 'from here'}.`,
      leftStack: pull => `#${pull} left this stack. Nothing can be sent to it from here.`,
      shownHere: (line, path) => `line ${line}${path ? ` of ${path}` : ''} on this page`,
      stale: why => `Its line may have moved: ${why}. No button sends it until it is queued again or dropped.`,
      posted: (action, count, at) => `${action === 'approve' ? 'Approved' : `${action === 'approve-with-comments' ? 'Approved with' : 'Posted'} ${count} comment${count === 1 ? '' : 's'}`}, ${at}: `,
      theReview: 'the review',
      movedTo: 'This review moved to ',
      movedThere: '. Its queued comments moved there.',
      result: message => capital(message),
      sending: 'Checking the pull request with GitHub, then sending…',
      saveAgain: why => `${why}. Save again to replace it with your text.`,
      droppedElsewhere: (id, text) => `Comment ${id} was dropped elsewhere while you edited it. Your text was: ${text}`,
    },

    notes: {
      askClaude: 'Ask Claude',
      queueComment: 'Queue comment',
      you: 'You',
      claude: 'Claude',
      noAnswer: why => `No answer: ${why}`,
      useAsComment: 'Use as comment',
      waiting: 'waiting for an answer',
      thinking: 'Claude is reading the file and the diff&hellip;',
      reading: 'Claude is reading the file and the diff…',
      followUpOrComment: 'Ask a follow-up, or write a comment for the pull request',
      followUp: 'Ask a follow-up',
      queued: id => `Queued ${id}`,
      stale: 'stale',
      edit: 'Edit',
      drop: 'Drop',
      bot: 'bot',
      youTag: 'you',
      wholeFile: 'the whole file',
      wasOnLine: line => `was on line ${line}`,
      andOthers: others => ` and ${others}`,
      github: 'GitHub',
      resolved: 'Resolved',
      outdated: 'Outdated',
      moreOnGitHub: more => `${more} on GitHub. `,
      resolvedBy: who => `Resolved by ${who}. `,
      openOnGitHub: 'Open on GitHub',
      checking: 'Checking where a comment here goes…',
      checkingTitle: 'Checking where a comment here goes.',
      whereUnchecked: why => `Where a comment here goes could not be checked: ${why}`,
      commentOn: 'Comment on ',
      askClaudeAbout: 'Ask Claude about ',
      option: (pull, line) => `#${pull}, line ${line}`,
      passageHint: 'Select another passage to change the lines.',
      tapHint: 'Tap another line number for a range.',
      dragHint: 'Drag across line numbers, or shift-select another, for a range.',
      askOrComment: 'Ask Claude about these lines, or write a comment for the pull request',
      askAbout: 'Ask Claude about these lines',
      close: 'Close',
      cancel: 'Cancel',
      onNoLine: 'On no line shown here',
      checkingLine: 'Checking the line with GitHub…',
    },

    diagram: {
      kinds: { has: 'holds', realize: 'conforms to', inherit: 'inherits from', extend: 'extends', use: 'uses', test: 'tests' },
      unchangedCount: files => `${files} · unchanged`,
      readContext: head => `Unchanged. Select to read its code at ${head}`,
      readWhole: 'Unchanged, in a changed file. Select to read the file whole',
      removedOpen: component => `Removed by this change. Select to open the ${component} diff`,
      open: component => `Select to open the ${component} diff`,
      nodeLabel: (name, file, pulls, go) => `${name}, ${file}. ${pulls.length ? `Changed in ${pulls.map(p => `#${p}`).join(', ')}. ` : ''}${go}`,
      changed: 'changed',
      unchanged: 'unchanged',
      removed: 'removed',
      readCodeOf: component => `Read the code of ${component}`,
      openDiffFor: component => `Open the diff for ${component}`,
      readCode: 'Read code →',
      storedProperties: count => (count === 1 ? '1 stored property' : count ? `${grouped(count)} stored properties` : 'no stored properties'),
      inferred: 'inferred',
      inferredTitle: 'Read from what its initializer constructs',
      line: line => `line ${line}`,
      holds: name => `holds ${name}`,
      legend: {
        holds: 'Holds one as a stored property',
        conforms: 'Conforms to a protocol or interface',
        inherits: 'Inherits from a type',
        uses: 'Uses it (named links only, until you point)',
        added: 'New in this change',
        changed: 'Existing, changed',
        unchanged: 'Unchanged, shown for context',
        removed: 'Removed by this change',
        pull: 'Changed in that pull request of the stack',
      },
    },

    diffs: {
      showUnchanged: count => `show ${counted(count, 'unchanged line')}`,
      unchangedAt: head => `Unchanged at ${head}. Read only: select a line number to ask about it.`,
      binary: 'A binary file. Its contents are not shown.',
      renamedOnly: 'Renamed, with no change to its contents.',
      modeOnly: 'No change to its contents, only to its mode.',
      passageHint: 'Select a passage to comment on it or ask Claude about it.',
      passageHintAsk: 'Select a passage to ask Claude about it.',
      showOnDiagram: name => `Show ${name} on the diagram`,
      badges: { unchanged: 'unchanged', added: 'new', deleted: 'deleted', renamed: 'renamed', changed: 'changed' },
      from: path => `from ${path}`,
      wholeFile: 'Whole file',
      rendered: 'Rendered',
      pullLines: (pull, adds, dels) => `#${pull}&rsquo;s lines; the file changes +${adds} &minus;${dels} in all`,
      contextTotals: (files, head) => `${files}, unchanged at ${head}, read only`,
      otherComponents: 'Other components',
      noFileChanged: (component, pull) => `No file in ${component} changed in #${pull}.`,
      showEveryPull: 'Show every pull request',
      notLoaded: why => `The diff could not load: ${why}`,
      fileNotLoaded: why => `This file could not load: ${why}`,
      collapseAll: 'Collapse all',
      expandAll: 'Expand all',
      splitView: 'Split view',
      unifiedView: 'Unified view',
      wrapLines: 'Wrap lines',
      unwrapLines: 'Unwrap lines',
    },

    rendered: {
      added: 'Added',
      removed: 'Removed',
      addedKey: 'added',
      removedKey: 'removed',
      newFile: 'New file.',
      deletedFile: 'Deleted file. It read:',
      changedParts: 'Only the changed parts: the app holds the diff of this file, not its whole text. Marked:',
      marked: 'Marked:',
      readsTheSame: 'The change does not alter how the file reads.',
    },
  },
});

function frozen(value) {
  if (value && typeof value === 'object') {
    for (const inner of Object.values(value)) frozen(inner);
    Object.freeze(value);
  }
  return value;
}
