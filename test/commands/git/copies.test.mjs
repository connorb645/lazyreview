import { test } from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, readdirSync, readFileSync, statSync, writeFileSync, existsSync, utimesSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { Copies } from '../../../src/commands/git/copies.mjs';
import { throwawayRepo } from '../../support/throwaway-repo.mjs';
import { folderState } from '../../support/folder-state.mjs';

const git = (repo, ...args) => execFileSync('git', ['-C', repo, ...args], { encoding: 'utf8' }).trim();

function cloned(repo, home) {
  const bare = join(home, 'clones', 'widgets.git');
  execFileSync('git', ['clone', '--quiet', '--bare', repo.work, bare]);
  return bare;
}

test('a copy is a git repository at the commit, on the branch given, with no remote, borrowing the clone\'s objects, and leaves the clone as it was', async () => {
  const repo = throwawayRepo();
  const home = mkdtempSync(join(tmpdir(), 'lazyreview-copies-'));
  try {
    const base = repo.commit('main', { 'src/money.ts': 'export const a = 1;\n', 'test/split.test.ts': 'splitPence(1000, 3);\n' }, 'base');
    const head = repo.commit('feature/split', { 'src/money.ts': 'export const a = 2;\n' }, 'split');
    repo.checkout('main');
    const source = cloned(repo, home);
    const before = folderState(source);
    const folder = join(home, 'copies', 'github.com', 'octo', 'widgets', '7', '1');
    const made = await new Copies().prepare(folder, { source, commit: head, branches: { main: base, 'pull/7': head }, branch: 'pull/7' });
    assert.equal(made, folder);
    assert.equal(readFileSync(join(folder, 'src/money.ts'), 'utf8'), 'export const a = 2;\n');
    assert.equal(readFileSync(join(folder, 'test/split.test.ts'), 'utf8'), 'splitPence(1000, 3);\n');
    assert.equal(git(folder, 'branch', '--show-current'), 'pull/7');
    assert.equal(git(folder, 'rev-parse', 'main'), base);
    assert.equal(git(folder, 'remote'), '', 'a push has nowhere to go');
    assert.match(readFileSync(join(folder, '.git/objects/info/alternates'), 'utf8'), /widgets\.git\/objects/);
    assert.equal(git(folder, 'status', '--porcelain'), '');
    assert.deepEqual(folderState(source), before, 'the clone it came from is as it was');
    assert.deepEqual(readdirSync(join(home, 'copies', 'github.com', 'octo', 'widgets', '7')), ['1'], 'no half-made copy is left beside it');
  } finally {
    repo.done();
    rmSync(home, { recursive: true, force: true });
  }
});

test('a copy is made once, then only marked used, keeping what was changed in it; a commit the clone lacks says so', async () => {
  const repo = throwawayRepo();
  const home = mkdtempSync(join(tmpdir(), 'lazyreview-copies-'));
  try {
    const commit = repo.commit('main', { 'a.txt': 'a\n' }, 'base');
    const source = cloned(repo, home);
    let now = Date.UTC(2026, 9, 15, 12);
    const copies = new Copies({ now: () => now });
    const folder = join(home, 'copies', '1');
    await copies.prepare(folder, { source, commit, branches: { main: commit }, branch: 'main' });
    writeFileSync(join(folder, 'a.txt'), 'changed by Claude\n');
    now += 60_000;
    await copies.prepare(folder, { source, commit, branches: { main: commit }, branch: 'main' });
    assert.equal(readFileSync(join(folder, 'a.txt'), 'utf8'), 'changed by Claude\n');
    assert.equal(statSync(folder).mtimeMs, now);

    const missing = '0'.repeat(40);
    await assert.rejects(copies.prepare(join(home, 'copies', '2'), { source, commit: missing, branches: { main: missing }, branch: 'main' }), { message: `commit 0000000 is no longer in ${source}; load the review again` });
    assert.equal(existsSync(join(home, 'copies', '2')), false);
  } finally {
    repo.done();
    rmSync(home, { recursive: true, force: true });
  }
});

test('pruning removes each copy not used for the time given, never the one kept, and the folders they leave empty', () => {
  const home = mkdtempSync(join(tmpdir(), 'lazyreview-copies-'));
  try {
    const now = Date.UTC(2026, 9, 15, 12);
    const day = 24 * 60 * 60_000;
    const make = (path, age) => {
      mkdirSync(join(home, path, '.git'), { recursive: true });
      utimesSync(join(home, path), new Date(now - age), new Date(now - age));
    };
    make('github.com/octo/widgets/7/1', 2 * day);
    make('github.com/octo/widgets/7/2', 60_000);
    make('local/work-1a2b3c4d/feature~x/1', 3 * day);
    make('github.com/octo/widgets/9/1', 3 * day);
    new Copies({ now: () => now }).prune(home, { unused: day, keep: join(home, 'github.com/octo/widgets/9/1') });
    assert.deepEqual(readdirSync(join(home, 'github.com/octo/widgets/7')), ['2']);
    assert.deepEqual(readdirSync(join(home, 'github.com/octo/widgets/9')), ['1']);
    assert.equal(existsSync(join(home, 'local')), false, 'a folder left empty goes too');
  } finally {
    rmSync(home, { recursive: true, force: true });
  }
});

test('a copy of a repository on this machine fetches only the commits asked for into lazyreview\'s own clone, and leaves the repository as it was', async () => {
  const { Clones } = await import('../../../src/commands/git/clones.mjs');
  const repo = throwawayRepo();
  const home = mkdtempSync(join(tmpdir(), 'lazyreview-copies-'));
  try {
    const base = repo.commit('main', { 'a.txt': 'a\n' }, 'base');
    const head = repo.commit('feature/x', { 'a.txt': 'b\n' }, 'x');
    const before = folderState(repo.work);
    const clone = join(home, 'clones', 'local', 'work-1a2b3c4d.git');
    assert.equal(await new Clones().copyFrom(clone, { from: repo.work, commits: [head, base] }), clone);
    assert.equal(git(clone, 'rev-parse', `refs/commits/${head}`), head);
    assert.equal(git(clone, 'rev-parse', `refs/commits/${base}`), base, 'each commit is kept under a ref, so it stays');
    assert.deepEqual(folderState(repo.work), before);
  } finally {
    repo.done();
    rmSync(home, { recursive: true, force: true });
  }
});
