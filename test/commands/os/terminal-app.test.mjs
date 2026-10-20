import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, utimesSync, chmodSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { TerminalApp } from '../../../src/commands/os/terminal-app.mjs';

// A terminal folder whose program records how it was run, built by a stand-in for go.
function folder() {
  const root = mkdtempSync(join(tmpdir(), 'lazyreview-terminal-'));
  mkdirSync(join(root, 'internal'), { recursive: true });
  mkdirSync(join(root, 'vendor', 'x'), { recursive: true });
  writeFileSync(join(root, 'go.mod'), 'module x\n');
  writeFileSync(join(root, 'main.go'), 'package main\n');
  writeFileSync(join(root, 'internal', 'view.go'), 'package internal\n');
  writeFileSync(join(root, 'internal', 'view_test.go'), 'package internal\n');
  writeFileSync(join(root, 'vendor', 'x', 'lib.go'), 'package x\n');
  const builds = [];
  const run = async (command, args, { cwd }) => {
    if (command !== 'go') {
      const [, ...given] = args;
      if (given.includes('--fail')) return { code: 1, stdout: '', stderr: 'there is no difference between main and same\n' };
      return { code: 0, stdout: JSON.stringify({ id: 'github.com/octo/widgets/30', given }) + '\n', stderr: '' };
    }
    builds.push({ command, args, cwd });
    const program = args[args.indexOf('-o') + 1];
    mkdirSync(join(root, 'bin'), { recursive: true });
    writeFileSync(program, `#!/bin/sh\necho "$@" > "${join(root, 'ran.txt')}"\n`);
    chmodSync(program, 0o755);
    return { code: 0, stderr: '' };
  };
  return { root, builds, run, done: () => rmSync(root, { recursive: true, force: true }) };
}

const later = path => { const t = Date.now() / 1000 + 60; utimesSync(path, t, t); };

test('the terminal app is built the first time it runs, and again only when its own code changes, saying so first', async () => {
  const f = folder();
  const said = [];
  const say = line => said.push(line);
  try {
    const app = new TerminalApp({ folder: f.root, run: f.run });
    assert.equal(app.stale(), true, 'with no program, it is built');
    await app.launch({ review: 'github.com/octo/widgets/21', env: process.env, say });
    assert.deepEqual(f.builds.map(b => [b.command, ...b.args]), [['go', 'build', '-mod=vendor', '-o', app.program, '.']]);
    assert.deepEqual(said, ['building the terminal app…']);
    assert.equal(readFileSync(join(f.root, 'ran.txt'), 'utf8'), 'github.com/octo/widgets/21\n');
    assert.equal(app.stale(), false);
    await app.launch({ review: 'github.com/octo/widgets/21', env: process.env, say });
    assert.deepEqual([f.builds.length, said.length], [1, 1], 'a program up to date runs without a word');
    later(join(f.root, 'internal', 'view_test.go'));
    later(join(f.root, 'vendor', 'x', 'lib.go'));
    assert.equal(app.stale(), false, 'a test and a library\'s own file do not go into the program');
    later(join(f.root, 'internal', 'view.go'));
    assert.equal(app.stale(), true);
    await app.launch({ review: 'github.com/octo/widgets/30', env: process.env, say });
    assert.equal(f.builds.length, 2);
    assert.equal(said.length, 2);
    assert.equal(readFileSync(join(f.root, 'ran.txt'), 'utf8'), 'github.com/octo/widgets/30\n');
    await app.launch({ review: null, env: process.env, say });
    assert.equal(readFileSync(join(f.root, 'ran.txt'), 'utf8'), '\n', 'with no review, it opens on the home screen');
  } finally {
    f.done();
  }
});

test('the terminal app says how to get Go when it cannot be built', async () => {
  const f = folder();
  try {
    const missing = new TerminalApp({ folder: f.root, run: async () => ({ code: 'ENOENT', stderr: '' }) });
    await assert.rejects(missing.launch({ review: 'github.com/octo/widgets/30', env: process.env }), /built with Go 1\.24 or later, and go is not on the PATH/);
    const broken = new TerminalApp({ folder: f.root, run: async () => ({ code: 1, stderr: 'main.go:1: syntax error\n' }) });
    await assert.rejects(broken.launch({ review: 'github.com/octo/widgets/30', env: process.env }), /could not build the terminal app: main\.go:1: syntax error/);
  } finally {
    f.done();
  }
});

test('the terminal app loads a review, built first when its code changed, and says why it could not', async () => {
  const f = folder();
  const said = [];
  try {
    const app = new TerminalApp({ folder: f.root, run: f.run });
    const loaded = await app.load(['--context', 'https://github.com/octo/widgets/pull/30'], { env: process.env, say: line => said.push(line) });
    assert.deepEqual(loaded, { id: 'github.com/octo/widgets/30', given: ['--context', 'https://github.com/octo/widgets/pull/30'] });
    assert.deepEqual([f.builds.length, said], [1, ['building the terminal app…']]);
    await assert.rejects(app.load(['--fail'], { env: process.env }), { message: 'there is no difference between main and same' });
    assert.equal(f.builds.length, 1, 'a program up to date loads without building');
  } finally {
    f.done();
  }
});
