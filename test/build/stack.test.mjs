import { test } from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync, unlinkSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { readHunkLines, towardOld, towardNew } from '../../src/build/stack/hunk-lines.mjs';
import { lineEntry } from '../../src/build/stack/line-map.mjs';
import { buildPage } from '../../src/build/page-builder.mjs';
import { pullsSubject } from '../../src/build/page/heading.mjs';
import { readConfig } from '../../src/build/config/config.mjs';
import { Git } from '../../src/build/git/git.mjs';

const ELK = createRequire(import.meta.url)('../../vendor/elk-0.9.3.bundled.js');

const lines = (prefix, n) => Array.from({ length: n }, (_, i) => `${prefix} ${i + 1}`);
const text = rows => rows.join('\n') + '\n';

// A repo holding main and a stack of three branches, each on the one before:
//   pr1  adds a property to Cache, adds Clock, removes a line of notes.md
//   pr2  changes another line of Cache, moves Clock to Time/ and edits it, adds Timer
//   pr3  rewrites one line pr2 added to Timer, adds README.md
function stackRepo() {
  const repo = mkdtempSync(join(tmpdir(), 'lazyreview-stack-'));
  const git = (...args) => execFileSync('git', ['-C', repo, '-c', 'user.name=Test', '-c', 'user.email=test@example.com', ...args], { encoding: 'utf8' }).trim();
  const write = files => {
    for (const [path, body] of Object.entries(files)) {
      mkdirSync(join(repo, dirname(path)), { recursive: true });
      if (body === null) unlinkSync(join(repo, path));
      else writeFileSync(join(repo, path), body);
    }
  };
  const commit = message => { git('add', '-A'); git('commit', '-q', '-m', message); return git('rev-parse', 'HEAD'); };
  const cache = ['final class Cache {', ...lines('    // line', 18), '}'];
  git('init', '-q', '-b', 'main');
  write({
    'src/Cache.swift': text(cache),
    'src/Store.swift': 'protocol Store {}\n',
    'notes.md': text(lines('note', 12)),
  });
  commit('base');
  git('checkout', '-q', '-b', 'pr1');
  const withProperty = [...cache.slice(0, 3), '    let clock: Clock', ...cache.slice(3)];
  write({
    'src/Cache.swift': text(withProperty),
    'src/Clock.swift': text(['protocol Clock {', ...lines('    // clock', 8), '}']),
    'notes.md': text(lines('note', 12).filter(l => l !== 'note 6')),
  });
  const pr1 = commit('pr1');
  git('checkout', '-q', '-b', 'pr2');
  const cache2 = withProperty.map(l => (l === '    // line 15' ? '    // line fifteen' : l));
  write({
    'src/Cache.swift': text(cache2),
    'src/Clock.swift': null,
    'src/Time/Clock.swift': text(['protocol Clock {', ...lines('    // clock', 8).map(l => (l === '    // clock 7' ? '    // clock seven' : l)), '}']),
    'src/Timer.swift': text(['struct Timer: Clock {', '    let every = 1', '    let label = "a"', '}']),
  });
  const pr2 = commit('pr2');
  git('checkout', '-q', '-b', 'pr3');
  write({
    'src/Timer.swift': text(['struct Timer: Clock {', '    let every = 5', '    let label = "a"', '}']),
    'README.md': '# Readme\n',
  });
  const pr3 = commit('pr3');
  return { repo, git, write, commit, pr1, pr2, pr3 };
}

// Builds the page of a stack whose members, numbered from 101, have these heads.
async function build(repo, heads) {
  const git = new Git(repo);
  const built = await buildPage({
    git, base: git.resolve('main'), members: heads.map((head, i) => ({ pull: 101 + i, head })), baseBranch: 'main', title: 'Stack', subject: pullsSubject(heads.map((_, i) => 101 + i)),
    config: readConfig(undefined), context: false, rootTitle: 'r', elk: new ELK(),
  });
  const map = { head: built.page.model.heading.head, members: built.page.stack.members, lines: built.page.stack.lines };
  return { ...built, map };
}

// Every line the map gives a pull request must be a line that pull request's
// own diff changes, reading the same at its head (or its start) as on the page.
function checkAgainstOwnDiffs(r, map, starts) {
  const show = (commit, path) => execFileSync('git', ['-C', r.repo, 'show', `${commit}:${path}`], { encoding: 'utf8' }).split('\n');
  const own = {};
  map.members.forEach((m, i) => {
    own[m.pull] = {};
    const diff = execFileSync('git', ['-C', r.repo, 'diff', '--find-renames', starts[i], m.head], { encoding: 'utf8' });
    for (const chunk of diff.split(/^diff --git /m).slice(1)) {
      const path = /\n\+\+\+ b\/(.*)\n/.exec(chunk)?.[1] || /\n--- a\/(.*)\n/.exec(chunk)?.[1];
      own[m.pull][path] = readHunkLines(chunk);
    }
  });
  let checked = 0;
  for (const [path, sides] of Object.entries(map.lines)) {
    for (const [n, entry] of Object.entries(sides.RIGHT)) {
      if (!Array.isArray(entry)) continue;
      const [pull, line, at = path] = entry;
      const hunks = own[pull][at];
      assert.ok(hunks && hunks.some(h => h.added.has(line)), `${path}:${n} is not added by #${pull} at ${at}:${line}`);
      const head = map.members.find(m => m.pull === pull).head;
      assert.equal(show(head, at)[line - 1], show(map.head, path)[+n - 1], `${path}:${n} reads differently in #${pull}`);
      checked++;
    }
  }
  return checked;
}

test('a line number maps across a diff both ways, around hunks with an empty side', () => {
  const diff = '@@ -10,0 +11,3 @@\n+a\n+b\n+c\n@@ -20,2 +22,0 @@\n-x\n-y\n@@ -30,3 +31,3 @@\n one\n-two\n+TWO\n three\n';
  const hunks = readHunkLines(diff);
  assert.deepEqual(towardOld(hunks, 10), { kind: 'outside', line: 10 });
  assert.deepEqual(towardOld(hunks, 12), { kind: 'added' });
  assert.deepEqual(towardOld(hunks, 14), { kind: 'outside', line: 11 });
  assert.deepEqual(towardOld(hunks, 23), { kind: 'outside', line: 22 });
  assert.deepEqual(towardOld(hunks, 31), { kind: 'context', line: 30 });
  assert.deepEqual(towardOld(hunks, 32), { kind: 'added' });
  assert.deepEqual(towardOld(hunks, 40), { kind: 'outside', line: 39 });
  assert.deepEqual(towardNew(hunks, 11), { kind: 'outside', line: 14 });
  assert.deepEqual(towardNew(hunks, 20), { kind: 'removed' });
  assert.deepEqual(towardNew(hunks, 22), { kind: 'outside', line: 23 });
  assert.deepEqual(towardNew(hunks, 31), { kind: 'removed' });
  assert.deepEqual(towardNew(hunks, 32), { kind: 'context', line: 33 });
  assert.equal(hunks[2].removedAt.get(31), 32);
});

test('a stack page shows which pull request changed each type, file and line, and maps each line to its own diff', async () => {
  const r = stackRepo();
  try {
    const { page, notes, starts, map } = await build(r.repo, [r.pr1, r.pr2, r.pr3]);
    assert.equal(page.model.files.length, 5);
    assert.deepEqual(notes, [], 'a stack in step has no warning');
    const model = page.model;
    assert.deepEqual(page.stack.members.map(m => [m.pull, m.files, m.adds, m.dels]), [[101, 3, 11, 1], [102, 3, 6, 2], [103, 2, 2, 1]]);
    assert.deepEqual(page.stack.warnings, []);
    assert.deepEqual(starts, ['main', r.pr1, r.pr2].map(ref => r.git('rev-parse', ref)));
    assert.match(model.heading.context.map(p => p[0]).join(''), /^Stack #101 to #103 on main: head [0-9a-f]{8}, merge base [0-9a-f]{8}, 3 commits$/);

    const node = name => model.nodes.find(n => n.name === name);
    assert.deepEqual(node('Cache').pulls, [101, 102]);
    assert.deepEqual(node('Clock').pulls, [101, 102]);
    assert.deepEqual(node('Timer').pulls, [102, 103]);
    assert.equal(node('Store'), undefined);
    const file = path => model.files.find(f => f.path === path);
    assert.deepEqual(file('src/Cache.swift').pulls, [101, 102]);
    assert.deepEqual(file('src/Cache.swift').byPull, { 101: [1, 0], 102: [1, 1] });
    assert.deepEqual(file('notes.md').pulls, [101]);
    assert.deepEqual(file('README.md').pulls, [103]);

    const at = (path, line, side = 'RIGHT') => lineEntry(map.lines, path, line, side);
    assert.deepEqual(at('src/Cache.swift', 4), { pull: 101, path: 'src/Cache.swift', line: 4, side: 'RIGHT' });
    assert.deepEqual(at('src/Cache.swift', 17), { pull: 102, path: 'src/Cache.swift', line: 17, side: 'RIGHT' });
    assert.deepEqual(at('src/Cache.swift', 16, 'LEFT'), { pull: 102, path: 'src/Cache.swift', line: 17, side: 'LEFT' });
    assert.deepEqual(at('src/Time/Clock.swift', 2), { pull: 101, path: 'src/Clock.swift', line: 2, side: 'RIGHT' }, 'a line a later pull request moved goes to the one that wrote it, at its old path');
    assert.deepEqual(at('src/Time/Clock.swift', 8), { pull: 102, path: 'src/Time/Clock.swift', line: 8, side: 'RIGHT' });
    assert.deepEqual(at('src/Timer.swift', 2), { pull: 103, path: 'src/Timer.swift', line: 2, side: 'RIGHT' });
    assert.deepEqual(at('src/Timer.swift', 3), { pull: 102, path: 'src/Timer.swift', line: 3, side: 'RIGHT' });
    assert.deepEqual(at('notes.md', 6, 'LEFT'), { pull: 101, path: 'notes.md', line: 6, side: 'LEFT' });
    assert.deepEqual(at('src/Cache.swift', 2), { candidates: [{ pull: 101, path: 'src/Cache.swift', line: 2, side: 'RIGHT' }] });
    assert.deepEqual(at('src/Cache.swift', 14), { candidates: [{ pull: 102, path: 'src/Cache.swift', line: 14, side: 'RIGHT' }] });
    assert.deepEqual(at('src/Cache.swift', 9), { none: 'the line is not in the combined diff' });
    assert.equal(checkAgainstOwnDiffs(r, map, starts), 17);
  } finally {
    rmSync(r.repo, { recursive: true, force: true });
  }
});

test('a stack out of step says so, and still sends each line where that pull request\'s own diff holds it', async () => {
  const r = stackRepo();
  try {
    r.git('checkout', '-q', 'pr1');
    r.write({ 'notes.md': text(lines('note', 12).filter(l => l !== 'note 6').map(l => (l === 'note 11' ? 'note eleven' : l))) });
    const pr1b = r.commit('pr1, after pr2 was cut from it');
    const { notes, map, starts } = await build(r.repo, [pr1b, r.pr2, r.pr3]);
    assert.match(notes.join('\n'), /#102 is not built on #101's head: #101 has commits #102 does not hold/);
    assert.deepEqual(lineEntry(map.lines, 'src/Cache.swift', 4, 'RIGHT').pull, 101, 'pr1\'s head still holds its line, so it stays with pr1');
    assert.deepEqual(lineEntry(map.lines, 'notes.md', 6, 'LEFT').pull, 101);
    assert.deepEqual(starts, [r.git('rev-parse', 'main'), r.git('merge-base', pr1b, r.pr2), r.pr2]);
    assert.equal(checkAgainstOwnDiffs(r, map, starts), 17);
  } finally {
    rmSync(r.repo, { recursive: true, force: true });
  }
});

test('a removed line goes to its pull request at the path that pull request\'s own diff gives the file', async () => {
  const repo = mkdtempSync(join(tmpdir(), 'lazyreview-rename-'));
  const git = (...args) => execFileSync('git', ['-C', repo, '-c', 'user.name=Test', '-c', 'user.email=test@example.com', ...args], { encoding: 'utf8' }).trim();
  try {
    const rows = lines('row', 12);
    git('init', '-q', '-b', 'main');
    writeFileSync(join(repo, 'a.txt'), text(rows));
    git('add', '-A'); git('commit', '-q', '-m', 'base');
    git('checkout', '-q', '-b', 'one');
    writeFileSync(join(repo, 'a.txt'), text(rows.filter(r => r !== 'row 5')));
    git('add', '-A'); git('commit', '-q', '-m', 'one');
    const one = git('rev-parse', 'HEAD');
    git('checkout', '-q', '-b', 'two');
    git('mv', 'a.txt', 'b.txt');
    writeFileSync(join(repo, 'b.txt'), text(rows.filter(r => r !== 'row 5' && r !== 'row 9')));
    git('add', '-A'); git('commit', '-q', '-m', 'two');
    const two = git('rev-parse', 'HEAD');
    const { map } = await build(repo, [one, two]);
    assert.deepEqual(lineEntry(map.lines, 'b.txt', 5, 'LEFT'), { pull: 101, path: 'a.txt', line: 5, side: 'LEFT' });
    assert.deepEqual(lineEntry(map.lines, 'b.txt', 9, 'LEFT'), { pull: 102, path: 'b.txt', line: 8, side: 'LEFT' });
    assert.deepEqual(lineEntry(map.lines, 'b.txt', 7, 'LEFT'), { candidates: [
      { pull: 101, path: 'a.txt', line: 7, side: 'LEFT' }, { pull: 102, path: 'b.txt', line: 6, side: 'LEFT' }] });
  } finally {
    rmSync(repo, { recursive: true, force: true });
  }
});

test('a stack whose member head is not in the clone is refused, naming how to fetch it', async () => {
  const r = stackRepo();
  try {
    await assert.rejects(build(r.repo, [r.pr1, 'abc1234', r.pr3]), /#102's head abc1234 is not a commit in this repository; fetch it with git fetch origin pull\/102\/head/);
  } finally {
    rmSync(r.repo, { recursive: true, force: true });
  }
});
