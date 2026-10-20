import { test } from 'node:test';
import assert from 'node:assert/strict';
import { execFile, spawn } from 'node:child_process';
import { writeFileSync, readFileSync, existsSync, mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { workbench } from '../support/workbench.mjs';
import { fakeGh } from '../support/fake-gh.mjs';
import { run as command } from '../../src/cli/main.mjs';
import { TerminalApp } from '../../src/commands/os/terminal-app.mjs';
import { PullRef } from '../../src/models/pull-ref.mjs';

const BIN = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'bin', 'lazyreview');

// Runs the command; whether the terminal app's program must be built first
// depends on this machine, not on the command, so a line saying it is
// building is left out.
function run(args, env) {
  return new Promise(resolve => {
    execFile(BIN, args, { env, timeout: 600_000 }, (error, stdout, stderr) => resolve({ code: error ? error.code : 0, stdout: stdout.replace(/^building the terminal app…\n/, ''), stderr }));
  });
}

function alive(pid) {
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}

test('one command loads a pull request and prints its address, with no server', async () => {
  const w = workbench();
  const env = w.env;
  try {
    const loaded = await run(['load', w.link(30)], env);
    assert.equal(loaded.code, 0, loaded.stderr);
    assert.equal(loaded.stdout, 'loaded #30: 1 file +1 -1, 1 component, 0 types\nopen lazyreview://github.com/octo/widgets/30\n');
    assert.ok(!existsSync(w.home.serverFile), 'nothing runs in the background');

    const stack = await run(['load', w.link(21), w.link(22), '--context'], env);
    assert.equal(stack.code, 0, stack.stderr);
    assert.match(stack.stdout, /^loaded #21, #22: 4 files \+\d+ -\d+, \d+ components?, \d+ types?\nopen lazyreview:\/\/github\.com\/octo\/widgets\/21\n$/);

    assert.match((await run(['terminal', 'github.com/octo/widgets/99'], env)).stderr, /^lazyreview: there is no review github\.com\/octo\/widgets\/99; lazyreview list shows every review\n/);
    assert.match((await run(['terminal', 'lazyreview://github.com/octo/widgets/99'], env)).stderr, /^lazyreview: there is no review github\.com\/octo\/widgets\/99;/);
    const missing = await run(['terminal', 'https://github.com/octo/widgets/pull/99'], env);
    assert.equal(missing.stdout, 'loading https://github.com/octo/widgets/pull/99…\n', 'a pull request no review holds is loaded first');
    assert.match(missing.stderr, /^lazyreview: .*pulls\/99/);
    assert.equal((await run(['terminal', 'widgets'], env)).stderr, 'lazyreview: "widgets" is not the address of a review or of a pull request; lazyreview list shows every review\n');
    const listed = await run(['list'], env);
    assert.equal(listed.stdout.trim().split('\n').length, 2);
    assert.match(listed.stdout, /^lazyreview:\/\/github\.com\/octo\/widgets\/21 {2}#21, #22 {2}Stack #21 to #22$/m);

    const file = join(w.home.root, 'written.md');
    const written = `---\nstack: 21, 22\nhead: ${w.repo.sha('feature/basket')}\n---\n## What it does\nAdds search in \`Search.swift:4\`, then a basket.\n`;
    writeFileSync(file, written);
    const attached = await run(['attach', w.link(22), file], env);
    assert.equal(attached.code, 0, attached.stderr);
    assert.equal(attached.stdout, 'attached to github.com/octo/widgets/21; open lazyreview://github.com/octo/widgets/21\n');
    assert.equal(readFileSync(w.home.review(new PullRef('github.com', 'octo/widgets', 21)).written, 'utf8'), written);
    writeFileSync(file, '---\npull: 30\nhead: 1111111\n---\n## Notes\nx\n');
    const stale = await run(['attach', w.link(30), file], env);
    assert.equal(stale.code, 1);
    assert.match(stale.stderr, /lazyreview: the written review read 1111111, but the review shows head/);
  } finally {
    w.done();
  }
});

test('the terminal app opens on the review an address names, loading a pull request no review holds yet, and with none on the home screen', async () => {
  const w = workbench();
  const env = w.env;
  const said = [];
  const opened = [];
  const program = new TerminalApp();
  const terminal = { launch: async ({ review }) => { opened.push(review); }, load: (args, options) => program.load(args, options) };
  const open = link => command(['terminal', link], { env, say: line => said.push(line), terminal });
  try {
    await assert.rejects(open('http://127.0.0.1:4747/'), /is not the address of a review or of a pull request/);
    await open(w.link(30));
    assert.deepEqual(said, ['loading https://github.com/octo/widgets/pull/30…']);
    await open('lazyreview://github.com/octo/widgets/30');
    await open('http://127.0.0.1:4747/review/github.com/octo/widgets/30#review');
    await open(w.link(30));
    assert.equal(said.length, 1, 'a review already loaded opens as it is');
    assert.deepEqual(opened, Array(4).fill('github.com/octo/widgets/30'), 'an address an older lazyreview printed still opens');
    await command(['terminal'], { env, say: line => said.push(line), terminal });
    assert.equal(opened.at(-1), null, 'with no address, the app opens on the home screen');
  } finally {
    w.done();
  }
});

test('one command loads a branch with no pull request, and prints the address that opens it', async () => {
  const w = workbench();
  const env = w.env;
  try {
    const loaded = await run(['load', '--repo', w.repo.work, '--base', 'main', '--head', 'fix/wording', '--title', 'Wording'], env);
    assert.equal(loaded.code, 0, loaded.stderr);
    const [summary, open, ...rest] = loaded.stdout.split('\n');
    assert.equal(summary, 'loaded fix/wording against main: 1 file +1 -1, 1 component, 0 types');
    assert.match(open, /^open lazyreview:\/\/local\/work-[0-9a-f]{8}\/fix~wording$/);
    assert.deepEqual(rest, ['']);
    const address = open.slice('open '.length);
    const record = join(w.home.reviews, ...address.slice('lazyreview://'.length).split('/'), 'review.json');
    assert.equal(JSON.parse(readFileSync(record, 'utf8')).title, 'Wording');

    const again = await run(['load', '--repo', w.repo.work, '--base', 'main', '--head', 'fix/wording'], env);
    assert.equal(again.stdout.split('\n')[1], open, 'the same branch loads into the same review');
    const listed = await run(['list'], env);
    assert.equal(listed.stdout, `${address}  fix/wording against main  Wording\n`);
  } finally {
    w.done();
  }
});

test('start says no server runs, and stop ends a server an older lazyreview left running, by the process id it recorded', async () => {
  const root = mkdtempSync(join(tmpdir(), 'lazyreview-home-'));
  const env = { PATH: process.env.PATH, HOME: process.env.HOME, LAZYREVIEW_HOME: root, LAZYREVIEW_CONFIG_DIR: root };
  const old = spawn(process.execPath, ['-e', 'setInterval(() => {}, 1000)'], { detached: true, stdio: 'ignore' });
  try {
    assert.equal((await run(['start'], env)).stdout, 'lazyreview runs no server now; lazyreview terminal opens the app\n');
    assert.equal((await run(['stop'], env)).stdout, 'nothing was running\n');
    writeFileSync(join(root, 'server.json'), JSON.stringify({ pid: old.pid, address: 'http://127.0.0.1:4747', token: 'x' }));
    const stopped = await run(['stop'], env);
    assert.equal(stopped.code, 0, stopped.stderr);
    assert.equal(stopped.stdout, 'stopped the server an older lazyreview left running\n');
    assert.ok(!alive(old.pid), 'the old server is gone');
    assert.ok(!existsSync(join(root, 'server.json')), 'and so is its record');
    assert.equal((await run(['stop'], env)).stdout, 'nothing was running\n');
  } finally {
    if (alive(old.pid)) process.kill(old.pid);
    rmSync(root, { recursive: true, force: true });
  }
});

test('a file that is not JSON fails the command, naming the file', async () => {
  const gh = fakeGh();
  const root = mkdtempSync(join(tmpdir(), 'lazyreview-home-'));
  const env = { ...gh.env, LAZYREVIEW_HOME: root };
  try {
    writeFileSync(join(root, 'server.json'), '');
    const stopped = await run(['stop'], env);
    assert.equal(stopped.code, 1);
    assert.equal(stopped.stderr, `lazyreview: could not read ${join(root, 'server.json')}: Unexpected end of JSON input\n`);

    const config = join(root, 'components.json');
    writeFileSync(config, '{"components": [');
    const loaded = await run(['load', 'https://github.com/octo/widgets/pull/1', '--config', config], env);
    assert.equal(loaded.code, 1);
    assert.equal(loaded.stderr, `lazyreview: could not read the config ${config}: Unexpected end of JSON input\n`);
    assert.deepEqual(gh.calls(), []);
  } finally {
    rmSync(root, { recursive: true, force: true });
    gh.done();
  }
});

test('the command explains itself, and names what was wrong', async () => {
  const help = await run(['help'], process.env);
  assert.match(help.stdout, /^usage: lazyreview <command> \[options\]/);
  const wrong = await run(['load'], process.env);
  assert.equal(wrong.code, 1);
  assert.match(wrong.stderr, /^lazyreview: load needs a pull request, or --repo with --base and --head\n\nusage:/);
  assert.match((await run(['load', '--repo', '.', '--base', 'main'], process.env)).stderr, /^lazyreview: --repo needs --head\n/);
  assert.match((await run(['load', '--head', 'feature'], process.env)).stderr, /^lazyreview: --head needs --repo\n/);
  assert.match((await run(['load', 'octo/widgets#1', '--repo', '.', '--base', 'a', '--head', 'b'], process.env)).stderr, /^lazyreview: load takes pull requests or --repo, not both\n/);
  assert.match((await run(['load', '--repo', '.', '--base', 'a', '--head', 'b', '--account', 'me'], process.env)).stderr, /^lazyreview: --account is for pull requests/);
  assert.match((await run(['terminal', 'octo/widgets#1', 'octo/widgets#2'], process.env)).stderr, /^lazyreview: terminal takes 1 argument\n\nusage:/);
  assert.match((await run(['frobnicate'], process.env)).stderr, /unknown command "frobnicate"/);
  assert.match((await run(['serve'], process.env)).stderr, /unknown command "serve"/, 'there is no server to run');
  assert.match((await run(['load', 'x', '--colour'], process.env)).stderr, /unknown option --colour for load/);
});
