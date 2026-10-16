import { test } from 'node:test';
import assert from 'node:assert/strict';
import { execFile } from 'node:child_process';
import { createServer } from 'node:net';
import { setTimeout as sleep } from 'node:timers/promises';
import { writeFileSync, readFileSync, existsSync, mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { workbench } from '../support/workbench.mjs';
import { notices } from '../support/notices.mjs';
import { fakeClaude } from '../support/fake-claude.mjs';
import { fakeGh } from '../support/fake-gh.mjs';

const BIN = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'bin', 'review-desk');
const SLOW_DISK = pathToFileURL(join(dirname(fileURLToPath(import.meta.url)), '..', 'support', 'slow-disk.mjs')).href;

function run(args, env) {
  return new Promise(resolve => {
    execFile(BIN, args, { env, timeout: 120_000 }, (error, stdout, stderr) => resolve({ code: error ? error.code : 0, stdout, stderr }));
  });
}

function freePort() {
  return new Promise(resolve => {
    const server = createServer().listen(0, '127.0.0.1', () => {
      const { port } = server.address();
      server.close(() => resolve(port));
    });
  });
}

test('one command loads a pull request, starts the app, and prints where to open it', async () => {
  const w = workbench();
  const fake = fakeClaude();
  const port = await freePort();
  const env = { ...w.env, REVIEW_DESK_PORT: String(port), REVIEW_DESK_CLAUDE: fake.command };
  try {
    const loaded = await run(['load', w.link(30)], env);
    assert.equal(loaded.code, 0, loaded.stderr);
    assert.equal(loaded.stdout, `loaded #30: 1 file +1 -1, 1 component, 0 types\nopen http://127.0.0.1:${port}/review/github.com/octo/widgets/30\n`);
    const page = await fetch(`http://127.0.0.1:${port}/review/github.com/octo/widgets/30`);
    assert.equal(page.status, 200);
    const record = JSON.parse(readFileSync(w.home.serverFile, 'utf8'));
    assert.equal(record.address, `http://127.0.0.1:${port}`);

    const stack = await run(['load', w.link(21), w.link(22), '--context'], env);
    assert.equal(stack.code, 0, stack.stderr);
    assert.match(stack.stdout, /^loaded #21, #22: 4 files \+\d+ -\d+, \d+ components?, \d+ types?\nopen http:\/\/127\.0\.0\.1:\d+\/review\/github\.com\/octo\/widgets\/21\n$/);

    const listed = await run(['list'], env);
    assert.equal(listed.stdout.trim().split('\n').length, 2);
    assert.match(listed.stdout, /\/review\/github\.com\/octo\/widgets\/21 {2}#21, #22 {2}Stack #21 to #22/);

    const heard = notices(`http://127.0.0.1:${port}`, 'github.com/octo/widgets/21');
    const before = await heard.next();
    const file = join(w.home.root, 'written.md');
    writeFileSync(file, `---\nstack: 21, 22\nhead: ${w.repo.sha('feature/basket')}\n---\n## What it does\nAdds search in \`Search.swift:4\`, then a basket.\n`);
    const attached = await run(['attach', w.link(22), file], env);
    assert.equal(attached.code, 0, attached.stderr);
    assert.match(attached.stdout, /^attached to github\.com\/octo\/widgets\/21; open http:/);
    const state = await (await fetch(`http://127.0.0.1:${port}/api/reviews/github.com/octo/widgets/21`)).json();
    assert.deepEqual(state.written.sections[0].links, [{ text: 'Search.swift:4', path: 'Sources/Catalog/Search.swift', line: 4 }], 'the server resolves the written review\'s links');
    assert.deepEqual(await heard.next(1000), { state: state.version }, 'an app open on the review hears of an attach from another process');
    assert.notEqual(state.version, before.state);
    heard.close();
    writeFileSync(file, '---\npull: 30\nhead: 1111111\n---\n## Notes\nx\n');
    const stale = await run(['attach', w.link(30), file], env);
    assert.equal(stale.code, 1);
    assert.match(stale.stderr, /review-desk: the written review read 1111111, but the review shows head/);

    assert.equal((await run(['stop'], env)).stdout, 'stopped\n');
    assert.ok(!existsSync(w.home.serverFile));
    assert.equal((await run(['stop'], env)).stdout, 'not running\n');
  } finally {
    await run(['stop'], env);
    fake.done();
    w.done();
  }
});

test('one command loads a branch with no pull request, and prints the address that opens it', async () => {
  const w = workbench();
  const port = await freePort();
  const env = { ...w.env, REVIEW_DESK_PORT: String(port) };
  try {
    const loaded = await run(['load', '--repo', w.repo.work, '--base', 'main', '--head', 'fix/wording', '--title', 'Wording'], env);
    assert.equal(loaded.code, 0, loaded.stderr);
    const [summary, open, ...rest] = loaded.stdout.split('\n');
    assert.equal(summary, 'loaded fix/wording against main: 1 file +1 -1, 1 component, 0 types');
    assert.match(open, new RegExp(`^open http://127\\.0\\.0\\.1:${port}/review/local/work-[0-9a-f]{8}/fix~wording$`));
    assert.deepEqual(rest, ['']);
    const address = open.slice('open '.length);
    const page = await fetch(address);
    assert.equal(page.status, 200);
    const state = await fetch(address.replace('/review/', '/api/reviews/'));
    assert.equal((await state.json()).title, 'Wording');

    const again = await run(['load', '--repo', w.repo.work, '--base', 'main', '--head', 'fix/wording'], env);
    assert.equal(again.stdout.split('\n')[1], open, 'the same branch loads into the same review');
    const listed = await run(['list'], env);
    assert.equal(listed.stdout, `${address}  fix/wording against main  Wording\n`);
  } finally {
    await run(['stop'], env);
    w.done();
  }
});

test('the first load starts the app even when the disk is slow to fill the app\'s record', async () => {
  const w = workbench();
  const port = await freePort();
  const env = { ...w.env, REVIEW_DESK_PORT: String(port), NODE_OPTIONS: `--import=${SLOW_DISK}` };
  try {
    const loaded = await run(['load', w.link(30)], env);
    assert.equal(loaded.code, 0, loaded.stderr);
    assert.equal(loaded.stdout, `loaded #30: 1 file +1 -1, 1 component, 0 types\nopen http://127.0.0.1:${port}/review/github.com/octo/widgets/30\n`);
  } finally {
    for (let i = 0; i < 20 && (await run(['stop'], env)).code !== 0; i++) await sleep(100);
    w.done();
  }
});

test('a file that is not JSON fails the command, naming the file', async () => {
  const gh = fakeGh();
  const root = mkdtempSync(join(tmpdir(), 'review-desk-home-'));
  const env = { ...gh.env, REVIEW_DESK_HOME: root };
  try {
    writeFileSync(join(root, 'server.json'), '');
    const started = await run(['start'], env);
    assert.equal(started.code, 1);
    assert.equal(started.stderr, `review-desk: could not read ${join(root, 'server.json')}: Unexpected end of JSON input\n`);

    const config = join(root, 'components.json');
    writeFileSync(config, '{"components": [');
    const loaded = await run(['load', 'https://github.com/octo/widgets/pull/1', '--config', config], env);
    assert.equal(loaded.code, 1);
    assert.equal(loaded.stderr, `review-desk: could not read the config ${config}: Unexpected end of JSON input\n`);
    assert.deepEqual(gh.calls(), []);
  } finally {
    rmSync(root, { recursive: true, force: true });
    gh.done();
  }
});

test('the command explains itself, and names what was wrong', async () => {
  const help = await run(['help'], process.env);
  assert.match(help.stdout, /^usage: review-desk <command> \[options\]/);
  const wrong = await run(['load'], process.env);
  assert.equal(wrong.code, 1);
  assert.match(wrong.stderr, /^review-desk: load needs a pull request, or --repo with --base and --head\n\nusage:/);
  assert.match((await run(['load', '--repo', '.', '--base', 'main'], process.env)).stderr, /^review-desk: --repo needs --head\n/);
  assert.match((await run(['load', '--head', 'feature'], process.env)).stderr, /^review-desk: --head needs --repo\n/);
  assert.match((await run(['load', 'octo/widgets#1', '--repo', '.', '--base', 'a', '--head', 'b'], process.env)).stderr, /^review-desk: load takes pull requests or --repo, not both\n/);
  assert.match((await run(['load', '--repo', '.', '--base', 'a', '--head', 'b', '--account', 'me'], process.env)).stderr, /^review-desk: --account is for pull requests/);
  assert.match((await run(['frobnicate'], process.env)).stderr, /unknown command "frobnicate"/);
  assert.match((await run(['load', 'x', '--colour'], process.env)).stderr, /unknown option --colour for load/);
});
