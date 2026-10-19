import { test } from 'node:test';
import assert from 'node:assert/strict';
import { execFile } from 'node:child_process';
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, existsSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { defaultConfigText } from '../../src/config/config-text.mjs';

const BIN = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'bin', 'lazyreview');

function run(args, env) {
  return new Promise(resolve => {
    execFile(BIN, args, { env, timeout: 60_000 }, (error, stdout, stderr) => resolve({ code: error ? error.code : 0, stdout, stderr }));
  });
}

// A stand-in home folder, with its data home and config folder, and the environment that names them and nothing else of lazyreview's.
function standIn() {
  const home = mkdtempSync(join(tmpdir(), 'lazyreview-home-'));
  const config = join(home, 'config');
  mkdirSync(config);
  return { home, config, file: join(config, 'config.yml'), env: { PATH: process.env.PATH, HOME: home, LAZYREVIEW_CONFIG_DIR: config }, done: () => rmSync(home, { recursive: true, force: true }) };
}

test('config path says where the file is, or would be, and config defaults prints the default config', async () => {
  const s = standIn();
  try {
    assert.deepEqual(await run(['config', 'path'], s.env), { code: 0, stdout: `${s.file}\nthere is no file there yet, so every setting has its default\n`, stderr: '' });
    writeFileSync(s.file, 'server:\n  port: 4748\n');
    assert.equal((await run(['config', 'path'], s.env)).stdout, `${s.file}\nit exists\n`);
    assert.deepEqual(await run(['config', 'defaults'], s.env), { code: 0, stdout: defaultConfigText(), stderr: '' });
  } finally {
    s.done();
  }
});

test('config check prints every setting as read, and where it came from', async () => {
  const s = standIn();
  try {
    writeFileSync(s.file, 'server:\n  port: 4748\ngithub:\n  accounts:\n    - login: lee\nkeybinding:\n  universal:\n    web: gW\n');
    const checked = await run(['config', 'check'], { ...s.env, LAZYREVIEW_MODEL: 'claude-sonnet' });
    assert.equal(checked.code, 0, checked.stderr);
    const [head, theme] = checked.stdout.replace(/\n\n# keybinding\.universal\.web .*\n$/, '\n').split('\ngui:\n');
    assert.ok(theme.trimEnd().split('\n').every(line => !line || /^ *\w+:( +# [A-Za-z ]+)?$|# default$/.test(line)), 'every colour and key is its default');
    assert.equal(head, [
      `the config at ${s.file}, on top of the defaults:`,
      '',
      `dataHome: ${join(s.home, '.lazyreview')}`.padEnd(38) + ' # default',
      '',
      'server:',
      '  port: 4748                           # the file',
      '',
      'claude:',
      '  command: claude                      # default',
      '  model: claude-sonnet                 # LAZYREVIEW_MODEL',
      '',
      'os:',
      '  editor: \'\'                           # default',
      '',
      'github:',
      '  accounts:                            # the file',
      '    - login: lee',
      '      teams: []',
      '  hiddenRepositories: []               # default',
      '',
    ].join('\n'));
    assert.match(theme, /^ {4}dark:\n {6}defaultFgColor: '#f1f0fb' +# default\n/m);
    assert.match(checked.stdout, /\n\n# keybinding\.universal\.web does nothing now, so it can go: lazyreview has no web app; keybinding\.reviewView\.drawings opens a written review's drawings in the browser\n$/);
  } finally {
    s.done();
  }
});

test('a config with a mistake stops a command, naming the file and what is wrong, and help, config path and config defaults still work', async () => {
  const s = standIn();
  try {
    writeFileSync(s.file, 'server:\n  port: abc\n');
    assert.deepEqual(await run(['list'], s.env), { code: 1, stdout: '', stderr: `lazyreview: ${s.file} gives server.port "abc", which is not a port number from 1 to 65535\n` });
    assert.equal((await run(['config', 'check'], s.env)).code, 1);
    for (const args of [['help'], ['config', 'path'], ['config', 'defaults']]) assert.equal((await run(args, s.env)).code, 0, args.join(' '));
  } finally {
    s.done();
  }
});

test('config move moves settings.json and the variables into the config, after a check that writes nothing', async () => {
  const s = standIn();
  try {
    const data = join(s.home, '.lazyreview');
    mkdirSync(data);
    writeFileSync(join(data, 'settings.json'), '{"accounts": [{"login": "lee"}]}');
    const env = { ...s.env, LAZYREVIEW_PORT: '4748' };
    const checked = await run(['config', 'move', '--check'], env);
    assert.equal(checked.code, 0, checked.stderr);
    assert.match(checked.stdout, /^would write .*config\.yml:\n[\s\S]*\nnothing was written; lazyreview config move makes the move\n$/);
    assert.ok(!existsSync(s.file));
    const moved = await run(['config', 'move'], env);
    assert.equal(moved.code, 0, moved.stderr);
    assert.equal(readFileSync(s.file, 'utf8'), '# yaml-language-server: $schema=https://raw.githubusercontent.com/connorb645/lazyreview/main/schema/config.json\n# lazyreview\'s config. docs/Config.md lists every setting and its default.\nserver:\n  port: 4748\ngithub:\n  accounts:\n    - login: lee\n');
    assert.ok(existsSync(join(data, 'settings.json.moved')));
    assert.match((await run(['config', 'move'], env)).stderr, /exists already, and the move never overwrites a config\n$/);
  } finally {
    s.done();
  }
});

test('the config command names what was wrong in it', async () => {
  const s = standIn();
  try {
    assert.match((await run(['config'], s.env)).stderr, /^lazyreview: config needs one of path, defaults, check or move\n\nusage:/);
    assert.match((await run(['config', 'frob'], s.env)).stderr, /^lazyreview: config needs one of path, defaults, check or move, not "frob"\n\nusage:/);
    assert.match((await run(['config', 'path', '--check'], s.env)).stderr, /^lazyreview: --check is for config move\n\nusage:/);
    assert.match((await run(['config', 'path', 'more'], s.env)).stderr, /^lazyreview: config takes 1 argument\n\nusage:/);
  } finally {
    s.done();
  }
});
