import { test } from 'node:test';
import assert from 'node:assert/strict';
import { execFile } from 'node:child_process';
import { createServer } from 'node:net';
import { writeFileSync, readFileSync, existsSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { workbench } from '../support/workbench.mjs';
import { fakeClaude } from '../support/fake-claude.mjs';

const BIN = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'bin', 'review-desk');

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

    const stack = await run(['load', w.link(21), w.link(22), '--no-context'], env);
    assert.equal(stack.code, 0, stack.stderr);
    assert.match(stack.stdout, /^loaded #21, #22: 4 files \+\d+ -\d+, \d+ components?, \d+ types?\nopen http:\/\/127\.0\.0\.1:\d+\/review\/github\.com\/octo\/widgets\/21\n$/);

    const listed = await run(['list'], env);
    assert.equal(listed.stdout.trim().split('\n').length, 2);
    assert.match(listed.stdout, /\/review\/github\.com\/octo\/widgets\/21 {2}#21, #22 {2}Stack #21 to #22/);

    const file = join(w.home.root, 'written.md');
    writeFileSync(file, `---\nstack: 21, 22\nhead: ${w.repo.sha('feature/basket')}\n---\n## What it does\nAdds search, then a basket.\n`);
    const attached = await run(['attach', w.link(22), file], env);
    assert.equal(attached.code, 0, attached.stderr);
    assert.match(attached.stdout, /^attached to github\.com\/octo\/widgets\/21; open http:/);
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

test('the command explains itself, and names what was wrong', async () => {
  const help = await run(['help'], process.env);
  assert.match(help.stdout, /^usage: review-desk <command> \[options\]/);
  const wrong = await run(['load'], process.env);
  assert.equal(wrong.code, 1);
  assert.match(wrong.stderr, /^review-desk: load needs a pull request\n\nusage:/);
  assert.match((await run(['frobnicate'], process.env)).stderr, /unknown command "frobnicate"/);
  assert.match((await run(['load', 'x', '--colour'], process.env)).stderr, /unknown option --colour for load/);
});
