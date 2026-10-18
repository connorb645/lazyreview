/**
 * Every setting of lazyreview's config, once: its default, its check, the
 * environment variable that overrides it, and the words that say what it
 * does. The reader, the documentation and the editors' schema are all made
 * from this tree, as lazygit makes its own from GetDefaultConfig.
 *
 * SCHEMA is a tree of sections, {about, settings, flow, note}, whose leaves
 * are settings, {check, default, env, about, example, choices}. `check`
 * names how the reader checks a value, and `choices` are the words a
 * setting checked as a choice takes; `about` and `example` are lines of the default
 * config's comments, `note` is the words beside a section's name, and a
 * section with `flow` is written on one line. defaults() returns the default config as plain values,
 * and leaves(section) every setting with its dotted key, such as
 * "server.port", in the tree's order. PATTERNS are what a login, a team's
 * name and a date must match, as regular expressions' sources. RETIRED
 * names, by dotted key, each setting lazyreview no longer has, with what
 * does its work now.
 */
import { english } from '../i18n/english.mjs';

const words = english.config.about;

export const PATTERNS = Object.freeze({
  colour: '^#[0-9a-fA-F]{6}$',
  login: '^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?$',
  team: '^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?/[A-Za-z0-9_.-]+$',
  date: '^\\d{4}-\\d{2}-\\d{2}$',
});

export const RETIRED = Object.freeze({
  'keybinding.diagramPanel.left': english.config.retiredWhy.diagramPanelMove('left'),
  'keybinding.diagramPanel.right': english.config.retiredWhy.diagramPanelMove('right'),
});

const setting = fields => Object.freeze({ kind: 'setting', env: null, example: [], ...fields });
const section = (settings, about = [], { flow = false, note = '' } = {}) => Object.freeze({ kind: 'section', about, flow, note, settings: Object.freeze(settings) });
const colour = value => setting({ check: 'colour', default: value, about: [] });

// A terminal's colours, one for each role, then a stack's pull requests' and the components'.
const colours = given => section({
  ...Object.fromEntries(Object.entries(given).filter(([name]) => name !== 'pullRequestColors' && name !== 'componentColors').map(([name, value]) => [name, colour(value)])),
  pullRequestColors: setting({ check: 'colours', default: given.pullRequestColors, about: [] }),
  componentColors: section(Object.fromEntries(Object.entries(given.componentColors).map(([name, value]) => [name, colour(value)])), [], { flow: true }),
});

// The keys of one place, by action: a label, or a list of them. A list of numbered places, such as the
// repositories 1 to 9, gives each its key by its place in the list.
const keys = (given, note = '') => section(Object.fromEntries(Object.entries(given).map(([action, value]) => [action,
  setting({ check: NUMBERED.includes(action) ? 'numberedKeys' : 'keys', default: value, about: [] })])), [], { note });
const NUMBERED = ['jumpToRepository', 'jumpToBlock', 'choose'];
const MOVE = { down: ['j', '<down>'], up: ['k', '<up>'], top: 'gg', bottom: 'G' };
const PAGES = { halfDown: '<c-d>', halfUp: '<c-u>', pageDown: '<c-f>', pageUp: '<c-b>' };
const BOXES = {
  nextBox: 'w', previousBox: 'b', top: 'gg', bottom: 'G', open: '<enter>', back: '-', follow: 'gd', references: 'gr', search: '/', searchBack: '?',
  nextMatch: 'n', previousMatch: 'N', labels: 'f', links: '<space>l', properties: '<space>p', boxDiff: '<space>d', allLinks: '<space>a',
};
const notes = english.config.about.keySections;

const KEYBINDING = section({
  universal: keys({ command: ':', keys: '<space>?', review: '<space>v', github: 'gx', web: 'gX', close: ['q', '<esc>'], quit: ['ZZ', '<c-c>'], refresh: 'R', bigger: '+', smaller: '_' }, notes.universal),
  home: keys({ ...MOVE, ...PAGES, open: '<enter>', previousPanel: 'h', nextPanel: 'l', jumpToRepository: ['1', '2', '3', '4', '5', '6', '7', '8', '9'] }, notes.home),
  reviewScreen: keys({ nextPull: ']p', previousPull: '[p', jumpBack: '<c-o>', jumpForward: '<c-i>', diagram: '<space>m', finder: '<space>f', home: '<space>h', focusMain: '0' }, notes.reviewScreen),
  panels: keys({ ...MOVE, ...PAGES, open: '<enter>', previousPanel: 'h', nextPanel: 'l', jumpToBlock: ['1', '2', '3', '4'] }, notes.panels),
  reviewPanel: keys({ post: 'P', highlight: 'v' }),
  filesPanel: keys({ toggleFold: 'za' }),
  notesPanel: keys({ edit: 'e', drop: 'dd', post: 'P' }),
  diagramPanel: keys(Object.fromEntries(Object.entries(BOXES).filter(([action]) => !['top', 'bottom', 'open'].includes(action)))),
  reviewView: keys({ ...MOVE, ...PAGES, nextSection: ']]', previousSection: '[[', fold: 'za', open: '<enter>', edit: 'e', drop: 'dd', post: 'P' }, notes.reviewView),
  diff: keys({
    down: ['j', '<down>'], up: ['k', '<up>'], halfDown: '<c-d>', halfUp: '<c-u>', pageDown: '<c-f>', pageUp: '<c-b>', top: 'gg', bottom: 'G',
    nextHunk: ']c', previousHunk: '[c', nextFile: ']f', previousFile: '[f', search: '/', searchBack: '?', nextMatch: 'n', previousMatch: 'N',
    toggleFold: 'za', openFold: 'zo', closeFold: 'zc', openFolds: 'zR', closeFolds: 'zM', open: '<enter>', back: '-', split: '<space>s', wrap: '<space>w',
    whole: '<space>o', rendered: '<space>r', showType: 'gd', nextNote: ']n', previousNote: '[n', select: 'V', otherEnd: 'o', comment: 'c', ask: 'a', edit: 'e', drop: 'dd',
  }),
  diagram: keys({ left: ['h', '<left>'], down: ['j', '<down>'], up: ['k', '<up>'], right: ['l', '<right>'], ...BOXES }, notes.diagram),
  menu: keys({ ...MOVE, ...PAGES, open: '<enter>', close: ['<esc>', 'q'] }, notes.menu),
  finder: keys({ down: ['<c-n>', '<down>'], up: ['<c-p>', '<up>'], open: '<enter>', close: '<esc>', clear: '<c-u>' }),
  posting: keys({ down: ['j', '<down>'], up: ['k', '<up>'], choose: ['1', '2', '3'], open: '<enter>', close: ['<esc>', 'n', 'q'], post: 'y', back: ['n', '<esc>', 'q'] }),
  confirmation: keys({ confirm: 'y' }),
}, english.config.about.keys);

const DARK = {
  defaultFgColor: '#f1f0fb', mutedFgColor: '#a09dbf', faintFgColor: '#6d6a8a', accentColor: '#a796ff', activeBorderColor: '#a796ff', inactiveBorderColor: '#4a4766',
  addedColor: '#7ce2a7', removedColor: '#ff9fb6', warningColor: '#f2b26a', selectedLineBgColor: '#3a3270', inactiveViewSelectedLineBgColor: '#26233d', visualSelectionBgColor: '#45475a',
  pullRequestColors: ['#3987e5', '#d95926', '#199e70', '#c98500', '#d55181', '#008300', '#9085e9', '#e66767'],
  componentColors: { blue: '#7fa8f8', teal: '#4cd3c2', purple: '#b893fb', amber: '#f2b26a', olive: '#b8cc6a', rose: '#f08cb6', indigo: '#a4a9ff', slate: '#a3afc8', grey: '#9d9bb8' },
};

const LIGHT = {
  defaultFgColor: '#17152f', mutedFgColor: '#615e7f', faintFgColor: '#9b98b5', accentColor: '#5233e6', activeBorderColor: '#5233e6', inactiveBorderColor: '#c4c0dc',
  addedColor: '#146b3a', removedColor: '#b21f43', warningColor: '#9c5212', selectedLineBgColor: '#dcd4ff', inactiveViewSelectedLineBgColor: '#efebfb', visualSelectionBgColor: '#c9cbd6',
  pullRequestColors: ['#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#e87ba4', '#008300', '#6250d6', '#e34948'],
  componentColors: { blue: '#2c66a8', teal: '#0b6e6e', purple: '#7448ad', amber: '#9c5212', olive: '#5f6e12', rose: '#a8325e', indigo: '#4a4fb0', slate: '#556170', grey: '#6f7987' },
};

export const SCHEMA = section({
  dataHome: setting({ check: 'folder', default: '~/.lazyreview', env: 'LAZYREVIEW_HOME', about: words.dataHome }),
  server: section({
    port: setting({ check: 'port', default: 4747, env: 'LAZYREVIEW_PORT', about: words.port }),
  }),
  claude: section({
    command: setting({ check: 'command', default: 'claude', env: 'LAZYREVIEW_CLAUDE', about: words.claudeCommand }),
    model: setting({ check: 'text', default: '', env: 'LAZYREVIEW_MODEL', about: words.claudeModel }),
  }),
  os: section({
    editor: setting({ check: 'text', default: '', env: 'LAZYREVIEW_EDITOR', about: words.editor }),
  }),
  github: section({
    accounts: setting({ check: 'accounts', default: [], about: words.accounts, example: words.accountsExample }),
  }),
  gui: section({
    lineNumbers: setting({ check: 'choice', choices: ['absolute', 'relative', 'none'], default: 'absolute', about: words.lineNumbers }),
    theme: section({ dark: colours(DARK), light: colours(LIGHT) }, words.theme),
  }),
  keybinding: KEYBINDING,
});

export function defaults(node = SCHEMA) {
  if (node.kind === 'setting') return structuredClone(node.default);
  return Object.fromEntries(Object.entries(node.settings).map(([name, inner]) => [name, defaults(inner)]));
}

export function leaves(node = SCHEMA, prefix = '') {
  if (node.kind === 'setting') return [{ key: prefix, setting: node }];
  return Object.entries(node.settings).flatMap(([name, inner]) => leaves(inner, prefix ? `${prefix}.${name}` : name));
}
