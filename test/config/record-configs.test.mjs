import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { configFrom, terminalPart } from '../../src/config/user-config.mjs';
import { parseYAML } from '../../src/config/yaml.mjs';
import { listEdited } from '../../src/config/list-edit.mjs';

// What the config reader makes of each text, or its refusal, and how YAML's plain scalars read, which the Go reader is
// tested against.
const RECORD = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'terminal', 'testdata', 'configs.json');
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

test('the config readings the Go reader is tested against are the ones Node gives now', () => {
  const now = recorded();
  if (process.env.LAZYREVIEW_RECORD) {
    writeFileSync(RECORD, JSON.stringify(now, null, 1) + '\n');
    return;
  }
  assert.deepEqual(JSON.parse(readFileSync(RECORD, 'utf8')), now, `${RECORD} is not what Node reads now; write it again with LAZYREVIEW_RECORD=1 npm test, then run the Go tests`);
});
