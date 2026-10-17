import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { readUserConfig, configFrom, terminalPart } from '../../src/config/user-config.mjs';
import { defaults } from '../../src/config/schema.mjs';

const PATH = '/home/lee/.config/lazyreview/config.yml';
const read = text => configFrom(text, PATH, '/home/lee');

function folder(text) {
  const dir = mkdtempSync(join(tmpdir(), 'lazyreview-config-'));
  if (text !== undefined) writeFileSync(join(dir, 'config.yml'), text);
  return { env: { LAZYREVIEW_CONFIG_DIR: dir }, file: join(dir, 'config.yml'), done: () => rmSync(dir, { recursive: true, force: true }) };
}

test('with no config every setting has its default, the data home in the home folder', () => {
  const f = folder();
  try {
    const config = readUserConfig({ env: f.env, home: '/home/lee' });
    assert.equal(config.path, f.file);
    assert.equal(config.exists, false);
    assert.deepEqual(config.values, { ...defaults(), dataHome: '/home/lee/.lazyreview' });
    assert.ok(Object.values(config.sources).every(source => source === 'default'));
  } finally {
    f.done();
  }
});

test('the file is read on top of the defaults, group by group, and a list replaces the default whole', () => {
  const { values, sources } = read('server:\n  port: 4748\nclaude:\n  model: claude-sonnet\ngithub:\n  accounts:\n    - login: lee\n    - login: lee-at-work\n      teams:\n        - name: example-org/mobile\n          since: 2026-10-01\n        - name: example-org/web\n');
  assert.equal(values.server.port, 4748);
  assert.deepEqual(values.claude, { command: 'claude', model: 'claude-sonnet' });
  assert.deepEqual(values.github.accounts, [
    { login: 'lee', teams: [] },
    { login: 'lee-at-work', teams: [{ name: 'example-org/mobile', since: '2026-10-01' }, { name: 'example-org/web', since: null }] },
  ], 'a date stays text');
  assert.deepEqual([sources['server.port'], sources['claude.command'], sources['claude.model'], sources['github.accounts']], ['file', 'default', 'file', 'file']);
  assert.equal(read('dataHome: ~/reviews\n').values.dataHome, '/home/lee/reviews');
  assert.deepEqual(read('').values, read('# only a comment\n').values, 'an empty file means every default');
});

test('the theme merges colour by colour, so a file can change one colour or one component\'s and keep the rest', () => {
  const { values, sources } = read("gui:\n  theme:\n    dark:\n      accentColor: '#ff0000'\n      selectedLineBgColor: 62\n      componentColors: {teal: '#00ffff'}\n    light:\n      pullRequestColors: [1, 2, 3, 4, 5, 6, 7, '#000000']\n");
  const fallback = defaults().gui.theme;
  assert.deepEqual(values.gui.theme.dark, { ...fallback.dark, accentColor: '#ff0000', selectedLineBgColor: '62', componentColors: { ...fallback.dark.componentColors, teal: '#00ffff' } }, 'a terminal colour number is handed over as text');
  assert.deepEqual(values.gui.theme.light, { ...fallback.light, pullRequestColors: ['1', '2', '3', '4', '5', '6', '7', '#000000'] });
  assert.deepEqual([sources['gui.theme.dark.accentColor'], sources['gui.theme.dark.componentColors.teal'], sources['gui.theme.dark.componentColors.blue']], ['file', 'file', 'default']);
});

test('each variable overrides its setting, over the file, and says so', () => {
  const f = folder('server:\n  port: 4748\nos:\n  editor: nano\n');
  try {
    const env = { ...f.env, LAZYREVIEW_HOME: '/srv/lr', LAZYREVIEW_PORT: '4749', LAZYREVIEW_CLAUDE: '/opt/claude', LAZYREVIEW_MODEL: 'claude-opus', LAZYREVIEW_EDITOR: 'nvim' };
    const config = readUserConfig({ env, home: '/home/lee' });
    assert.deepEqual(config.values, { ...defaults(), dataHome: '/srv/lr', server: { port: 4749 }, claude: { command: '/opt/claude', model: 'claude-opus' }, os: { editor: 'nvim' }, github: { accounts: [] } });
    assert.deepEqual(Object.entries(config.sources).filter(([key]) => !key.startsWith('gui.')), [['dataHome', 'LAZYREVIEW_HOME'], ['server.port', 'LAZYREVIEW_PORT'], ['claude.command', 'LAZYREVIEW_CLAUDE'], ['claude.model', 'LAZYREVIEW_MODEL'], ['os.editor', 'LAZYREVIEW_EDITOR'], ['github.accounts', 'default']]);
    assert.equal(readUserConfig({ env: { ...f.env, LAZYREVIEW_PORT: '' }, home: '/home/lee' }).values.server.port, 4748, 'an empty variable is not set');
    assert.throws(() => readUserConfig({ env: { ...f.env, LAZYREVIEW_PORT: 'x' }, home: '/home/lee' }), /^Error: LAZYREVIEW_PORT must be a port number, not "x"$/);
  } finally {
    f.done();
  }
});

test('the terminal app is handed the file\'s path and the editor: the variable, the setting, then VISUAL, EDITOR and vi', () => {
  const config = editor => ({ path: PATH, values: { ...defaults(), os: { editor } } });
  assert.deepEqual(terminalPart(config(''), {}), { path: PATH, editor: 'vi', theme: defaults().gui.theme });
  assert.equal(terminalPart(config(''), { EDITOR: 'nano' }).editor, 'nano');
  assert.equal(terminalPart(config(''), { EDITOR: 'nano', VISUAL: 'code -w' }).editor, 'code -w');
  assert.equal(terminalPart(config('hx'), { EDITOR: 'nano', VISUAL: 'code -w' }).editor, 'hx');
});

test('a file that is not YAML says why, and where', () => {
  assert.throws(() => read('server: [4747\n'), /^Error: \/home\/lee\/\.config\/lazyreview\/config\.yml could not be read: unexpected end of the stream within a flow collection \(2:1\)$/);
  assert.throws(() => read('server:\n  port: 1\n  port: 2\n'), /could not be read: duplicated mapping key \(3:3\)$/);
  assert.throws(() => read('just words\n'), /config\.yml holds no settings, written as lazyreview config defaults shows them$/);
});

test('every mistake in the file is named at once, each with what it must be', () => {
  const one = (text, words) => assert.throws(() => read(text), new RegExp(`^Error: ${PATH.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&')} ${words}$`), text);
  one('keybindings:\n  quit: q\n', 'sets keybindings, which is not a setting');
  one('server:\n  host: example\n', 'sets server.host, which is not a setting');
  one('server: 4747\n', 'gives server 4747, which is not a group of settings');
  one('server:\n  port: abc\n', 'gives server.port "abc", which is not a port number from 1 to 65535');
  one('server:\n  port: 70000\n', 'gives server.port 70000, which is not a port number from 1 to 65535');
  one('dataHome: reviews\n', 'gives dataHome "reviews", which is not a full path, or one starting with ~/');
  one('claude:\n  command: \'\'\n', 'gives claude.command "", which is not a command');
  one('os:\n  editor: 3\n', 'gives os.editor 3, which is not text');
  one('github:\n  accounts: lee\n', 'gives github.accounts "lee", which is not a list of accounts');
  one('github:\n  accounts:\n    - login: lee author:alex\n', 'gives account 1 no GitHub login');
  one('github:\n  accounts:\n    - login: lee\n    - login: lee\n', 'names lee twice');
  one('github:\n  accounts:\n    - login: lee\n      email: lee@example.com\n', 'gives lee "email", which an account does not have');
  one('github:\n  accounts:\n    - login: lee\n      teams: mobile\n', 'gives lee "teams" that are not a list');
  one('github:\n  accounts:\n    - login: lee\n      teams:\n        - name: mobile\n', 'gives lee a team that is not named as organisation/team');
  one('github:\n  accounts:\n    - login: lee\n      teams:\n        - name: example-org/mobile\n          since: last week\n', 'gives example-org/mobile a "since" that is not a date such as 2026-10-01');
  one('github:\n  accounts:\n    - login: lee\n      teams:\n        - name: example-org/mobile\n          until: 2026-12-01\n', 'gives example-org/mobile "until", which a team does not have');
  one('gui:\n  theme:\n    dark:\n      accentColor: violet\n', 'gives gui.theme.dark.accentColor "violet", which is not a colour: #rrggbb or a terminal colour number from 0 to 255');
  one('gui:\n  theme:\n    light:\n      addedColor: 256\n', 'gives gui.theme.light.addedColor 256, which is not a colour: #rrggbb or a terminal colour number from 0 to 255');
  one("gui:\n  theme:\n    dark:\n      pullRequestColors: ['#ffffff']\n", 'gives gui.theme.dark.pullRequestColors \\["#ffffff"\\], which is not a list of 8 colours');
  one('gui:\n  theme:\n    dark:\n      componentColors: {cyan: "#00ffff"}\n', 'sets gui.theme.dark.componentColors.cyan, which is not a setting');
  assert.throws(() => read('server:\n  port: abc\nkeybindings: {}\n'), new RegExp(`^Error: ${PATH.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&')} has 2 problems:\n  - gives server.port "abc", which is not a port number from 1 to 65535\n  - sets keybindings, which is not a setting$`));
});
