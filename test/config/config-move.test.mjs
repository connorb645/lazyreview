import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, existsSync, readdirSync, rmSync, renameSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { planMove, checkLines, move } from '../../src/config/config-move.mjs';
import { readUserConfig } from '../../src/config/user-config.mjs';
import { checkedConfigText } from '../../src/config/config-text.mjs';
import { defaults } from '../../src/config/schema.mjs';

const SETTINGS = '{"accounts": [\n  {"login": "lee"},\n  {"login": "lee-at-work", "teams": [{"name": "example-org/mobile", "since": "2026-10-01"}, {"name": "example-org/web"}]}\n]}\n';

// A stand-in home folder, with settings.json in its data home when given.
function standIn(settings = SETTINGS) {
  const home = mkdtempSync(join(tmpdir(), 'lazyreview-move-'));
  const data = join(home, '.lazyreview');
  mkdirSync(data);
  if (settings !== null) writeFileSync(join(data, 'settings.json'), settings);
  const config = join(home, 'Library', 'Application Support', 'lazyreview', 'config.yml');
  return { home, data, settings: join(data, 'settings.json'), config, where: env => ({ env, platform: 'darwin', home }), done: () => rmSync(home, { recursive: true, force: true }) };
}

test('the check reads settings.json and the variables, shows what the move would write, and writes nothing', () => {
  const s = standIn();
  try {
    const lines = checkLines(planMove(s.where({ LAZYREVIEW_PORT: '4748', LAZYREVIEW_CLAUDE: 'claude' })));
    assert.deepEqual(lines, [
      `would write ${s.config}:`,
      '',
      '  # yaml-language-server: $schema=https://raw.githubusercontent.com/connorb645/lazyreview/main/schema/config.json',
      '  # lazyreview\'s config. docs/Config.md lists every setting and its default.',
      '  server:',
      '    port: 4748',
      '  github:',
      '    accounts:',
      '      - login: lee',
      '      - login: lee-at-work',
      '        teams:',
      '          - name: example-org/mobile',
      '            since: \'2026-10-01\'',
      '          - name: example-org/web',
      '',
      'server.port from LAZYREVIEW_PORT',
      `github.accounts from ${s.settings}`,
      `would rename ${s.settings} to settings.json.moved`,
      'nothing was written; lazyreview config move makes the move',
    ], 'a variable equal to its default is not written');
    assert.equal(readFileSync(s.settings, 'utf8'), SETTINGS);
    assert.ok(!existsSync(join(s.home, 'Library')));
  } finally {
    s.done();
  }
});

test('the move loses nothing: every account, team and variable reads the same after it, and settings.json is kept, byte for byte', () => {
  const s = standIn();
  try {
    const env = { LAZYREVIEW_HOME: s.data, LAZYREVIEW_PORT: '4748', LAZYREVIEW_MODEL: 'claude-sonnet', LAZYREVIEW_EDITOR: 'nvim' };
    const said = move(planMove(s.where(env)));
    assert.equal(said[0], `wrote ${s.config}:`);
    assert.deepEqual(said.slice(-6), [
      '',
      `renamed ${s.settings} to settings.json.moved`,
      'LAZYREVIEW_HOME is still set, and overrides dataHome while it is; taking it out of your shell\'s setup is safe, and not needed',
      'LAZYREVIEW_PORT is still set, and overrides server.port while it is; taking it out of your shell\'s setup is safe, and not needed',
      'LAZYREVIEW_MODEL is still set, and overrides claude.model while it is; taking it out of your shell\'s setup is safe, and not needed',
      'LAZYREVIEW_EDITOR is still set, and overrides os.editor while it is; taking it out of your shell\'s setup is safe, and not needed',
    ]);
    assert.equal(readFileSync(join(s.data, 'settings.json.moved'), 'utf8'), SETTINGS);
    assert.ok(!existsSync(s.settings));
    assert.deepEqual(readdirSync(join(s.home, 'Library', 'Application Support', 'lazyreview')), ['config.yml'], 'no temporary file is left');

    const withVariables = readUserConfig(s.where(env));
    const fromFile = readUserConfig(s.where({}));
    for (const config of [withVariables, fromFile]) {
      assert.deepEqual(config.values, {
        dataHome: s.data, server: { port: 4748 }, claude: { command: 'claude', model: 'claude-sonnet' }, os: { editor: 'nvim' },
        github: { accounts: [{ login: 'lee', teams: [] }, { login: 'lee-at-work', teams: [{ name: 'example-org/mobile', since: '2026-10-01' }, { name: 'example-org/web', since: null }] }] },
        gui: defaults().gui,
        keybinding: defaults().keybinding,
      });
    }
    const checked = checkedConfigText(fromFile);
    for (const value of ['lee', 'lee-at-work', 'example-org/mobile', '2026-10-01', 'example-org/web', '4748', 'claude-sonnet', 'nvim', s.data]) assert.ok(checked.includes(value), value);
    assert.match(checkedConfigText(withVariables), /port: 4748 +# LAZYREVIEW_PORT\n/);
    assert.match(checked, /port: 4748 +# the file\n/);
  } finally {
    s.done();
  }
});

test('the move never overwrites a config, nor a settings.json.moved, and with nothing to move it writes nothing', () => {
  const s = standIn();
  try {
    mkdirSync(join(s.home, 'Library', 'Application Support', 'lazyreview'), { recursive: true });
    writeFileSync(s.config, 'server:\n  port: 4750\n');
    const plan = planMove(s.where({}));
    assert.equal(checkLines(plan).at(-2), `${s.config} exists already, and the move never overwrites a config`);
    assert.throws(() => move(plan), new RegExp(`${s.config.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&')} exists already, and the move never overwrites a config$`));
    assert.equal(readFileSync(s.config, 'utf8'), 'server:\n  port: 4750\n');
    rmSync(s.config);
    writeFileSync(join(s.data, 'settings.json.moved'), 'kept');
    assert.throws(() => move(planMove(s.where({}))), /settings\.json\.moved exists already, so settings\.json cannot be renamed to it; nothing was written$/);
    assert.equal(readFileSync(s.settings, 'utf8'), SETTINGS);
  } finally {
    s.done();
  }
  const none = standIn(null);
  try {
    const plan = planMove(none.where({}));
    const said = [`there is nothing to move: there is no ${none.settings}, and no LAZYREVIEW_ variable is set`];
    assert.deepEqual(checkLines(plan), said);
    assert.deepEqual(move(plan), said);
    assert.ok(!existsSync(none.config));
  } finally {
    none.done();
  }
});

test('a move that fails partway puts back what it did, and a settings.json that is wrong says so, naming it', () => {
  const s = standIn();
  try {
    const plan = planMove(s.where({}));
    let renames = 0;
    const files = {
      exists: path => existsSync(path),
      read: path => readFileSync(path, 'utf8'),
      write: (path, text) => writeFileSync(path, text, { flag: 'wx' }),
      rename: (from, to) => {
        if (++renames === 2) throw new Error('the disk is full');
        renameSync(from, to);
      },
      remove: path => rmSync(path, { force: true }),
      folder: path => mkdirSync(path, { recursive: true }),
    };
    assert.throws(() => move(plan, { files }), /^Error: the move failed, and nothing changed: the disk is full$/);
    assert.equal(readFileSync(s.settings, 'utf8'), SETTINGS);
    assert.ok(!existsSync(join(s.data, 'settings.json.moved')));
    assert.deepEqual(readdirSync(join(s.home, 'Library', 'Application Support', 'lazyreview')), []);
  } finally {
    s.done();
  }
  const wrong = standIn('{"accounts": [{"login": "lee author:alex"}]}');
  try {
    assert.throws(() => planMove(wrong.where({})), new RegExp(`^Error: ${wrong.settings.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&')} gives account 1 no GitHub login$`));
    assert.throws(() => planMove(wrong.where({ LAZYREVIEW_PORT: 'x' })), /^Error: LAZYREVIEW_PORT must be a port number, not "x"$/);
  } finally {
    wrong.done();
  }
});
