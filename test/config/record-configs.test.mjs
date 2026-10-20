import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync, mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { configFrom, terminalPart, readUserConfig } from '../../src/config/user-config.mjs';
import { parseYAML, dumpYAML } from '../../src/config/yaml.mjs';
import { listEdited } from '../../src/config/list-edit.mjs';
import { checkedConfigText, movedConfigText } from '../../src/config/config-text.mjs';
import { planMove, checkLines, move } from '../../src/config/config-move.mjs';

// What the config reader makes of each text, or its refusal, and how YAML's plain scalars read, which the Go reader is
// tested against.
const RECORD = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'terminal', 'testdata', 'configs.json');
// What the config commands print and write: each config as checked, YAML as js-yaml writes it, and each move.
const COMMANDS = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'terminal', 'testdata', 'config-commands.json');
const PATH = '/home/lee/.config/lazyreview/config.yml';
const HOME = '/home/lee';

const TEXTS = [
  '', '# only a comment\n', 'just words\n', 'server: [4747\n', 'server:\n  port: 1\n  port: 2\n',
  'server:\n  port: 4748\nclaude:\n  model: claude-sonnet\ngithub:\n  accounts:\n    - login: lee\n    - login: lee-at-work\n      teams:\n        - name: example-org/mobile\n          since: 2026-10-01\n        - name: example-org/web\n',
  'dataHome: ~/reviews\n', 'dataHome: ~\n', 'dataHome: /srv/reviews\n', 'dataHome: reviews\n',
  "gui:\n  theme:\n    dark:\n      accentColor: '#ff0000'\n      selectedLineBgColor: 62\n      componentColors: {teal: '#00ffff'}\n    light:\n      pullRequestColors: [1, 2, 3, 4, 5, 6, 7, '#000000']\n",
  'gui:\n  lineNumbers: relative\n', 'gui:\n  lineNumbers: none\n', 'gui:\n  lineNumbers: hybrid\n', 'gui:\n  lineNumbers: ~\n',
  'keybindings:\n  quit: q\n', 'server:\n  host: example\n', 'server: 4747\n', 'server:\n  port: abc\n', 'server:\n  port: 70000\n',
  "claude:\n  command: ''\n", "claude:\n  command: '   '\n", 'os:\n  editor: 3\n', 'os:\n  editor: "code -w"\n',
  'github:\n  accounts: lee\n', 'github:\n  accounts: {login: lee, teams: []}\n', 'github:\n  accounts:\n    - login: lee author:alex\n', 'github:\n  accounts:\n    - lee\n',
  'github:\n  accounts:\n    - login: lee\n    - login: lee\n', 'github:\n  accounts:\n    - login: lee\n      email: lee@example.com\n',
  'github:\n  accounts:\n    - login: lee\n      teams: mobile\n', 'github:\n  accounts:\n    - login: lee\n      teams: ~\n',
  'github:\n  accounts:\n    - login: lee\n      teams:\n        - name: mobile\n',
  'github:\n  accounts:\n    - login: lee\n      teams:\n        - name: example-org/mobile\n          since: last week\n',
  'github:\n  accounts:\n    - login: lee\n      teams:\n        - name: example-org/mobile\n          since:\n',
  'github:\n  accounts:\n    - login: lee\n      teams:\n        - name: example-org/mobile\n          until: 2026-12-01\n',
  'github:\n  accounts:\n    - email: x\n      login: 7\n    - login: lee\n      teams: [x, {name: a/b, since: 2026-10-01, x: 1}]\n',
  "gui:\n  theme:\n    dark:\n      accentColor: violet\n", 'gui:\n  theme:\n    light:\n      addedColor: 256\n', 'gui:\n  theme:\n    light:\n      addedColor: 0o17\n',
  'github:\n  hiddenRepositories: octo/*\n', 'github:\n  hiddenRepositories: [octo]\n', 'github:\n  hiddenRepositories: [octo/widgets, "*/x", 3]\n',
  'github:\n  hiddenRepositories:\n    - octo/widgets\n    - Reviewer-Home/*\n    - octo/web.site_2\n', 'github:\n  hiddenRepositories:\n  # - octo/widgets\n',
  "gui:\n  theme:\n    dark:\n      pullRequestColors: ['#ffffff']\n", 'gui:\n  theme:\n    dark:\n      componentColors: {cyan: "#00ffff"}\n',
  'server:\n  port: abc\nkeybindings: {}\n', "keybinding:\n  diagramPanel:\n    left: 'H'\n", "keybinding:\n  diagramPanel:\n    left: 'h'\n    right: 'l'\n",
  "keybinding:\n  diagram:\n    left: 'H'\n", "keybinding:\n  universal:\n    web: 'gW'\n",
  "keybinding:\n  universal:\n    quit: ['<c-q>', '<disabled>']\n    keys: '<c-i>'\n", "keybinding:\n  home:\n    jumpToRepository: ['<disabled>', '2', '3', '4', '5', '6', '7', '8', '9']\n",
  'keybinding:\n  universal:\n    quit: []\n', "keybinding:\n  universal:\n    quit: '<foo>'\n", "keybinding:\n  universal:\n    quit: 'a<'\n",
  "keybinding:\n  universal:\n    quit: ['<disabled>x']\n", "keybinding:\n  universal:\n    quit: [['q']]\n", 'keybinding:\n  universal:\n    quit: 3\n',
  "keybinding:\n  home:\n    jumpToRepository: ['1']\n", 'keybinding: 3\n', 'keybinding:\n  diff: [a]\n',
  'server:\n  port: 08080\n', 'server:\n  port: 0x1F90\n', 'server:\n  port: 1e3\n', 'server:\n  port: 1.5\n', 'server:\n  port: -1\n', 'server:\n  port: .inf\n',
  'server:\n  port: !!str 4747\n', 'server:\n  port: "a\\"b<>&"\n', 'x: &k 3\n', 'server: &s\n  port: 4748\nclaude: *s\n', '<<: {server: {port: 1}}\n',
  '1: x\n', 'true: x\n', 'null: x\n', '~: x\n', '- one\n- two\n', 'server:\n  port: |\n    4747\n',
];

const SCALARS = ['2026-10-01', 'yes', 'no', 'on', 'off', '~', 'null', 'Null', 'NULL', '', 'true', 'True', 'TRUE', 'tRue', 'false', '08080', '017', '0o17',
  '0x1F', '0X1F', '0b101', '1_000', '1.5', '.5', '5.', '1e3', '1E3', '1.5e-3', '-1e3', '+1', '-0', '+0', '.inf', '-.inf', '+.inf', '.Inf', '.NAN', '.nan',
  '12345678901234567890', '1:20', '0', '-7', '+7', '0xG', '0o8', '1__0', '_1', '0.', '-.5', '+.5', '1.2.3', 'a b', '0x_1F', '-0x1F', '+0x1F', '-0o7', '-0b1',
  '1e', 'e3', '.', '+', '1e400', '0.0000001', '123456789012345678901234', '00', '-00', '0o', '0x', '0b'];

const lines = (...given) => `${given.join('\n')}\n`;
const WRITTEN = lines('# yaml-language-server: $schema=https://example.com/config.json', '# My config.', '', 'github:', '  accounts:', '    - login: reviewer',
  '    # - login: old-account', '    - login: reviewer-home', '', 'gui:', '  lineNumbers: relative  # easier to jump');
const KEY = 'github.hiddenRepositories';
const EDITS = [
  [WRITTEN, { add: ['octo/widgets'] }], [WRITTEN, { remove: ['octo/*'] }],
  [lines('github:', '  accounts:', '    - login: reviewer', '  hiddenRepositories:', '    - octo/widgets', '', 'gui: {}'), { add: ['octo/*'] }],
  [lines('github:', '  hiddenRepositories:', '    - octo/widgets', '    - octo/*', '', 'gui: {}'), { add: ['octo/widgets'] }],
  [lines('github:', '  hiddenRepositories:', '    - octo/widgets', '    - octo/*', '', 'gui: {}'), { remove: ['octo/widgets'] }],
  [lines('github:', '  hiddenRepositories:', '    - octo/*', '', 'gui: {}'), { remove: ['octo/*'] }],
  [lines('github:', '  hiddenRepositories:', '  - a/b   # noisy', '  # - c/d', '  - e/*', 'gui: {}'), { add: ['x/y'], remove: ['a/b'] }],
  [lines('github:', '  hiddenRepositories:', '    # - old/one'), { add: ['x/y'] }],
  [lines('github:', "  hiddenRepositories: [a/b, 'c/*']  # mine"), { add: ['x/y'] }],
  [lines('github:', "  hiddenRepositories: [a/b, 'c/*']  # mine"), { remove: ['c/*'] }],
  [lines('github:', '  hiddenRepositories: []'), { add: ['x/y'] }], [lines('github:', '    accounts: []'), { add: ['x/y'] }],
  ['', { add: ['octo/*'] }], [lines('server:', '  port: 4800'), { add: ['octo/*'] }], ['server:\n  port: 4800', { add: ['octo/*'] }],
  [lines('github:'), { add: ['octo/*'] }], ['a: 1\r\n', { add: ['x/y'] }],
  [lines('github: {accounts: []}'), { add: ['x/y'] }], [lines('github:', '  hiddenRepositories: [', '    a/b]'), { add: ['x/y'] }],
  [lines('github:', '  hiddenRepositories: octo/*'), { add: ['x/y'] }], [lines('github:', '  hiddenRepositories:', '    - a/b', '      c/d'), { add: ['x/y'] }],
  [lines('github:', '  hiddenRepositories:', '    - "q/r"', "    - 'true/x'", '    - 7/x'), { add: ['true/y', 'a b/c'], remove: ['q/r'] }],
  [lines('# top', 'github:', '  hiddenRepositories:', '    - a/b', '# between', 'gui:', '  lineNumbers: none'), { add: ['c/*'] }],
  [lines('github:', '  accounts:', '    - login: x', 'server:', '  port: 4800'), { add: ['c/*'] }],
  [lines('github:', '  hiddenRepositories: ["a/b", c/d, \'e/*\']'), { remove: ['c/d'], add: ['f/g'] }],
];

function recorded() {
  const configs = TEXTS.map(text => {
    try {
      const read = configFrom(text, PATH, HOME);
      return { text, values: read.values, sources: read.sources, unused: read.unused, part: terminalPart({ path: PATH, values: read.values }, {}) };
    } catch (error) {
      return { text, error: error.message };
    }
  });
  const scalars = SCALARS.map(text => {
    const value = parseYAML(`a: ${text}`).a;
    return { text, type: value === null ? 'null' : typeof value, shown: JSON.stringify(value), string: String(value) };
  });
  const edits = EDITS.map(([text, change]) => {
    try {
      return { text, change, out: listEdited(text, KEY, change) };
    } catch (error) {
      return { text, change, error: error.message };
    }
  });
  return { configs, scalars, edits };
}

const CHECKED_WITH_VARIABLES = [
  { text: '', env: { LAZYREVIEW_MODEL: 'claude-sonnet' } },
  { text: 'github:\n  accounts:\n    - login: lee\n', env: { LAZYREVIEW_HOME: '/srv/reviews', LAZYREVIEW_CLAUDE: '/opt/claude bin/claude', LAZYREVIEW_EDITOR: 'code --wait' } },
  { text: 'claude:\n  model: claude-opus\nos:\n  editor: nvim\n', env: { LAZYREVIEW_MODEL: "it's #1: fast" } },
];

const ASCII = Array.from({ length: 95 }, (_, i) => String.fromCharCode(32 + i));
const STRINGS = [
  ...ASCII.flatMap(c => [c, `${c}a`, `a${c}b`, `a${c}`, `a${c} b`, `a ${c}b`, `${c}${c}`]),
  ...SCALARS, 'y', 'Y', 'n', 'N', 'Yes', 'YES', 'On', 'ON', 'No', 'NO', 'Off', 'OFF', 'False', 'FALSE', '<<', '=', '1:20:30', '190:20:30.5', '-1:20',
  '+1_0:2', '1:2:', '2001-12-14t21:59:43.10-05:00', '2001-12-14 21:59:43.10 -5', '2026-1-1 1:20:30', '2026-10-01T12:00:00Z', '2026-1-1', '2026-10-1',
  '20261001', '', ' ', '  ', ' a', 'a ', '\ta', 'a\tb', 'é', 'naïve café', '日本語', '😀', 'a😀b', ' ', 'a b', '\u0085', ' ', 'a b',
  '﻿', 'a﻿b', '\u007f', '\u0001', 'a\u001bb', '\u0000', '', '�', '￿', 'a\nb', 'a\n', 'a\n\n', '\n', '\n\n', '\na', ' a\nb',
  '\n a', 'a\r\nb', 'a\rb', 'line one\nline two\n', '  lead\n', 'a\n b\n', 'a\n\nb', 'a😀\nb', 'a\n\u0001', '- a', '? a', ': a', 'a: b', 'a:b', 'a #b',
  'a# b', '#a', 'http://x.y/z?a=b#c', '~/reviews', '/srv/my reviews', 'code --wait', 'claude-sonnet', 'example-org/mobile', "it's", '"q"', 'a\\b',
  'long '.repeat(40).trim(),
];
const VALUES = [
  {}, [], [[]], [{}], { a: {} }, { a: [] }, { a: [[]] }, [[1, 2], [3]], [[{ a: 1 }]], { a: [{ b: [{ c: 'd' }] }] },
  [{ a: 1, b: [1, 2] }, { c: { d: [{ e: null }] } }],
  { a: null, b: true, c: false, d: 1, e: -1, f: 1.5, g: 0.1, h: 1e21, i: 1.5e-7, j: 1e-7, k: 123456789012345678901234, l: 2.5e300, m: 0 },
  { 'a b': 1, '': 2, true: 3, 1: 4, 'a:b': 5, '-x': 6, '0x1F': 8, null: 9, '~': 10, 'multi\nline': 11, "it's": 12 },
  { dataHome: '/srv/reviews', claude: { command: 'claude', model: 'claude-sonnet' }, os: { editor: 'code --wait' },
    github: { accounts: [{ login: 'lee' }, { login: 'lee-at-work', teams: [{ name: 'example-org/mobile', since: '2026-10-01' }, { name: 'example-org/web' }] }] } },
  [{ login: 'lee', teams: [] }, { login: 'lee-at-work', teams: [{ name: 'example-org/mobile', since: '2026-10-01' }, { name: 'example-org/web', since: null }] }],
  { a: 'multi\nline' }, ['multi\nline', '  lead\nx\n'], { a: { b: '  lead\nx' } }, [['a\nb']], 'top\nlevel', null, true, 7, -0.5,
];

const SETTINGS = '{"accounts": [\n  {"login": "lee"},\n  {"login": "lee-at-work", "teams": [{"name": "example-org/mobile", "since": "2026-10-01"}, {"name": "example-org/web"}]}\n]}\n';
const DATA = `${HOME}/.lazyreview`;
const IN_DATA = name => `${DATA}/${name}`;
const MAC_CONFIG = `${HOME}/Library/Application Support/lazyreview/config.yml`;
const MOVES = [
  { platform: 'linux', env: {}, files: {} },
  { platform: 'darwin', env: { LAZYREVIEW_MODEL: 'claude-sonnet', LAZYREVIEW_CLAUDE: 'claude' }, files: { [IN_DATA('settings.json')]: SETTINGS } },
  { platform: 'linux', env: { LAZYREVIEW_HOME: '/srv/reviews', LAZYREVIEW_EDITOR: 'code --wait', LAZYREVIEW_MODEL: 'claude: fast' }, files: { '/srv/reviews/settings.json': SETTINGS } },
  { platform: 'linux', env: { LAZYREVIEW_CLAUDE: '/opt/claude bin/claude', LAZYREVIEW_EDITOR: "it's #1" }, files: {} },
  { platform: 'linux', env: { LAZYREVIEW_HOME: '/srv/r' }, files: {} },
  { platform: 'linux', env: { LAZYREVIEW_CLAUDE: 'claude' }, files: {} },
  { platform: 'darwin', env: {}, files: { [IN_DATA('settings.json')]: SETTINGS, [MAC_CONFIG]: 'server:\n  port: 4750\n' } },
  { platform: 'linux', env: {}, files: { [IN_DATA('settings.json')]: SETTINGS, [IN_DATA('settings.json.moved')]: 'kept' } },
  { platform: 'linux', env: {}, files: { [IN_DATA('settings.json')]: '{"accounts": []}' } },
  { platform: 'linux', env: {}, files: { [IN_DATA('settings.json')]: '{"accounts": [{"login": "lee", "teams": []}], "other": 1}' } },
  { platform: 'linux', env: {}, files: { [IN_DATA('settings.json')]: '{"accounts": [{"login": "b", "teams": [{"name": "a/b"}]}, {"teams": [], "login": "a"}]}' } },
  { platform: 'linux', env: {}, files: { [IN_DATA('settings.json')]: 'not json' } },
  { platform: 'linux', env: {}, files: { [IN_DATA('settings.json')]: '' } },
  ...['null', '[]', '{"accounts": {}}', '{}', '3', '"x"'].map(text => ({ platform: 'linux', env: {}, files: { [IN_DATA('settings.json')]: text } })),
  { platform: 'linux', env: {}, files: { [IN_DATA('settings.json')]: '{"accounts": [{"login": "lee author:alex"}]}' } },
  { platform: 'linux', env: {}, files: { [IN_DATA('settings.json')]: '{"accounts": [{"email": "x", "login": 7}, {"login": "lee", "teams": [{"name": "a/b", "since": 3, "x": 1}, "c/d"]}, {"login": "lee"}]}' } },
  { platform: 'linux', env: { XDG_CONFIG_HOME: `${HOME}/xdg` }, files: { [IN_DATA('settings.json')]: SETTINGS } },
  { platform: 'linux', env: { LAZYREVIEW_CONFIG_DIR: '/etc/lr', LAZYREVIEW_MODEL: 'm' }, files: {} },
  { platform: 'linux', env: { LAZYREVIEW_CONFIG_FILE: '/etc/lr/mine.yml', LAZYREVIEW_MODEL: 'm' }, files: {} },
  { platform: 'linux', env: { LAZYREVIEW_CONFIG_FILE: '/etc/lr/mine.yml', LAZYREVIEW_MODEL: 'm' }, files: { '/etc/lr/mine.yml': 'a: 1\n' } },
  ...[{ write: [1] }, { folder: [1] }, { rename: [1] }, { rename: [2] }, { rename: [2], remove: [1] }, { rename: [2, 3] }, { rename: [2, 3], remove: [1] }]
    .map(failures => ({ platform: 'linux', env: { LAZYREVIEW_MODEL: 'm' }, files: { [IN_DATA('settings.json')]: SETTINGS }, failures })),
  { platform: 'linux', env: { LAZYREVIEW_MODEL: 'm' }, files: {}, failures: { rename: [1], remove: [1] } },
];

// Files in memory, each step of a move failing on the calls `failures` names, in the words the Go stand-in uses too.
function memoryFiles(initial, failures = {}) {
  const files = new Map(Object.entries(initial));
  const calls = {};
  const step = name => {
    calls[name] = (calls[name] || 0) + 1;
    if ((failures[name] || []).includes(calls[name])) throw new Error(`the disk refused the ${name}`);
  };
  return {
    files,
    exists: path => files.has(path),
    read: path => files.get(path),
    write: (path, text) => { step('write'); if (files.has(path)) throw new Error(`${path} exists`); files.set(path, text); },
    rename: (from, to) => { step('rename'); files.set(to, files.get(from)); files.delete(from); },
    remove: path => { step('remove'); files.delete(path); },
    folder: () => step('folder'),
  };
}

const attempt = work => {
  try {
    return { out: work() };
  } catch (error) {
    return { error: error.message };
  }
};

function commands() {
  const checked = TEXTS.map(text => attempt(() => checkedConfigText(configFrom(text, PATH, HOME))))
    .map((got, i) => ({ text: TEXTS[i], checked: got.out })).filter(c => c.checked !== undefined);
  const folder = mkdtempSync(join(tmpdir(), 'lazyreview-record-'));
  try {
    for (const c of CHECKED_WITH_VARIABLES) {
      writeFileSync(join(folder, 'config.yml'), c.text);
      checked.push({ ...c, checked: checkedConfigText(readUserConfig({ env: { ...c.env, LAZYREVIEW_CONFIG_DIR: folder }, platform: 'linux', home: HOME })) });
    }
  } finally {
    rmSync(folder, { recursive: true, force: true });
  }
  const long = 'x'.repeat(1100);
  const dumps = [
    ...STRINGS.flatMap(s => [s, { k: s }, [s], { [s]: 1 }]),
    ...STRINGS.filter(s => !ASCII.some(c => s.includes(c)) || /[\s:#'"]/.test(s)).flatMap(s => [{ a: { b: s } }, [[s]], [{ a: s, b: s }]]),
    long, { [long]: 1 }, ...VALUES,
  ].map(value => [value, dumpYAML(value)]);
  const moved = [{}, { claude: { model: 'm' } }, { dataHome: '/srv/my reviews', github: { accounts: [{ login: 'lee' }] } }].map(values => ({ values, text: movedConfigText(values) }));
  const pid = `.${process.pid}.moving`;
  const moves = MOVES.map(c => {
    const files = memoryFiles(c.files, c.failures);
    const where = { env: c.env, platform: c.platform, home: HOME, files };
    const planned = attempt(() => planMove(where));
    if (planned.error) return { ...c, error: planned.error };
    const plan = planned.out;
    const done = attempt(() => move(plan, { files }));
    const after = Object.fromEntries([...files.files].sort(([a], [b]) => (a < b ? -1 : 1)).map(([path, text]) => [path.replace(pid, '.<pid>.moving'), text]));
    return { ...c, plan: { ...plan }, check: checkLines(plan), moved: done.out, moveError: done.error, after };
  });
  return { checked, dumps, moved, moves };
}

test('the config readings the Go reader is tested against are the ones Node gives now', () => {
  const now = recorded();
  const said = commands();
  if (process.env.LAZYREVIEW_RECORD) {
    writeFileSync(RECORD, JSON.stringify(now, null, 1) + '\n');
    writeFileSync(COMMANDS, JSON.stringify(said, null, 1) + '\n');
    return;
  }
  assert.deepEqual(JSON.parse(readFileSync(RECORD, 'utf8')), now, `${RECORD} is not what Node reads now; write it again with LAZYREVIEW_RECORD=1 npm test, then run the Go tests`);
  assert.deepEqual(JSON.parse(readFileSync(COMMANDS, 'utf8')), JSON.parse(JSON.stringify(said)), `${COMMANDS} is not what Node prints now; write it again with LAZYREVIEW_RECORD=1 npm test, then run the Go tests`);
});
