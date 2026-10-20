import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { throwawayRepo } from '../support/throwaway-repo.mjs';
import { Git } from '../../src/commands/git/git.mjs';
import { readDiff } from '../../src/build/diff/diff-reader.mjs';
import { readStackSteps } from '../../src/build/stack/stack-reader.mjs';
import { mapStackLines } from '../../src/build/stack/line-map.mjs';

// Made-up stacks, each as the commits that make its repository and the
// answers Node's stack reader and line map give for it, which the Go stack
// package is tested against.
const RECORD = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'terminal', 'testdata', 'stacks.json');

const lines = (prefix, n) => Array.from({ length: n }, (_, i) => `${prefix} ${i + 1}`);
const text = rows => rows.join('\n') + '\n';
const cache = ['final class Cache {', ...lines('    // line', 18), '}'];
const withProperty = [...cache.slice(0, 3), '    let clock: Clock', ...cache.slice(3)];
const clock = ['protocol Clock {', ...lines('    // clock', 8), '}'];
const notes = lines('note', 12);

const CASES = [
  {
    name: 'three members in step, with a rename, an edit and a removed line',
    base: 'main',
    members: [[101, 'pr1'], [102, 'pr2'], [103, 'pr3']],
    script: [
      { commit: ['main', { 'src/Cache.swift': text(cache), 'src/Store.swift': 'protocol Store {}\n', 'notes.md': text(notes) }, 'base'] },
      { commit: ['pr1', { 'src/Cache.swift': text(withProperty), 'src/Clock.swift': text(clock), 'notes.md': text(notes.filter(l => l !== 'note 6')) }, 'pr1'] },
      { commit: ['pr2', {
        'src/Cache.swift': text(withProperty.map(l => (l === '    // line 15' ? '    // line fifteen' : l))),
        'src/Clock.swift': null,
        'src/Time/Clock.swift': text(clock.map(l => (l === '    // clock 7' ? '    // clock seven' : l))),
        'src/Timer.swift': text(['struct Timer: Clock {', '    let every = 1', '    let label = "a"', '}']),
      }, 'pr2'] },
      { commit: ['pr3', { 'src/Timer.swift': text(['struct Timer: Clock {', '    let every = 5', '    let label = "a"', '}']), 'README.md': '# Readme\n' }, 'pr3'] },
    ],
  },
  {
    name: 'out of step: the member below gains a commit the one above lacks',
    base: 'main',
    members: [[201, 'lower'], [202, 'upper']],
    script: [
      { commit: ['main', { 'web/cart.ts': text(lines('// cart', 20)), 'web/total.ts': text(lines('// total', 10)) }, 'base'] },
      { commit: ['lower', { 'web/cart.ts': text(lines('// cart', 20).map(l => (l === '// cart 4' ? '// cart four' : l))) }, 'lower'] },
      { commit: ['upper', { 'web/total.ts': text(lines('// total', 10).map(l => (l === '// total 2' ? '// total two' : l))) }, 'upper'] },
      { checkout: 'lower' },
      { commit: ['lower', { 'web/cart.ts': text(lines('// cart', 20).map(l => (l === '// cart 4' ? '// cart four' : l === '// cart 15' ? '// cart fifteen' : l))) }, 'lower again'] },
    ],
  },
  {
    name: 'out of step: the member below rewritten, the one above still on its old commit',
    base: 'main',
    members: [[301, 'lower'], [302, 'upper']],
    script: [
      { commit: ['main', { 'web/cart.ts': text(lines('// cart', 20)) }, 'base'] },
      { commit: ['lower', { 'web/cart.ts': text(lines('// cart', 20).map(l => (l === '// cart 4' ? '// cart four' : l))) }, 'lower'] },
      { commit: ['upper', { 'web/cart.ts': text(lines('// cart', 20).map(l => (l === '// cart 4' ? '// cart four' : l === '// cart 12' ? '// cart twelve' : l))) }, 'upper'] },
      { checkout: 'lower' },
      { commit: ['lower', { 'web/cart.ts': text(lines('// cart', 20).map(l => (l === '// cart 4' ? '// cart 4, again' : l))) }, 'lower amended'] },
      { git: ['reset', '-q', '--soft', 'HEAD~2'] },
      { git: ['commit', '-q', '-m', 'lower rewritten'] },
    ],
  },
  {
    name: 'out of step: the member below takes back a line it added and puts back one it removed',
    base: 'main',
    members: [[351, 'lower'], [352, 'upper']],
    script: [
      { commit: ['main', { 'web/list.ts': text(lines('// list', 30)) }, 'base'] },
      { commit: ['lower', { 'web/list.ts': text(lines('// list', 30).flatMap(l => (l === '// list 5' ? [l, '// list 5a'] : l === '// list 15' ? [] : [l]))) }, 'lower'] },
      { commit: ['upper', { 'web/list.ts': text(lines('// list', 30).flatMap(l => (l === '// list 5' ? [l, '// list 5a'] : l === '// list 15' ? [] : l === '// list 27' ? ['// list twenty-seven'] : [l]))) }, 'upper'] },
      { checkout: 'lower' },
      { commit: ['lower', { 'web/list.ts': text(lines('// list', 30)) }, 'lower takes it back'] },
    ],
  },
  {
    name: 'a pure rename in one member, then a change to the renamed file in the next',
    base: 'main',
    members: [[401, 'move'], [402, 'edit']],
    script: [
      { commit: ['main', { 'src/Old.swift': text(['struct Old {', ...lines('    // old', 12), '}']), 'docs/a.md': text(lines('words', 9)) }, 'base'] },
      { git: ['checkout', '-q', '-b', 'move'] },
      { git: ['mv', 'src/Old.swift', 'src/New.swift'] },
      { git: ['mv', 'docs/a.md', 'docs/b.md'] },
      { commit: ['move', {}, 'move'] },
      { commit: ['edit', { 'src/New.swift': text(['struct Old {', ...lines('    // old', 12).map(l => (l === '    // old 9' ? '    // old nine' : l)), '}']) }, 'edit'] },
    ],
  },
  {
    name: 'files deleted, one the member below changed first',
    base: 'main',
    members: [[501, 'first'], [502, 'second']],
    script: [
      { commit: ['main', { 'src/Gone.swift': text(['enum Gone {', ...lines('    // gone', 5), '}']), 'src/Temp.swift': text(lines('// temp', 8)) }, 'base'] },
      { commit: ['first', { 'src/Gone.swift': null, 'src/Temp.swift': text(lines('// temp', 8).map(l => (l === '// temp 3' ? '// temp three' : l))) }, 'first'] },
      { commit: ['second', { 'src/Temp.swift': null }, 'second'] },
    ],
  },
  {
    name: 'a binary file added in one member and changed in the next',
    base: 'main',
    members: [[601, 'add-icon'], [602, 'new-icon']],
    script: [
      { commit: ['main', { 'README.md': '# Readme\n' }, 'base'] },
      { commit: ['add-icon', { 'assets/icon.bin': '\u0000\u0001icon\u0000one\n' }, 'add icon'] },
      { commit: ['new-icon', { 'assets/icon.bin': '\u0000\u0002icon\u0000two\n', 'README.md': '# Readme\n\nNow with an icon.\n' }, 'new icon'] },
    ],
  },
  {
    name: 'lines removed between lines another member added, and lines shown by both',
    base: 'main',
    members: [[701, 'adds'], [702, 'removes']],
    script: [
      { commit: ['main', { 'notes.md': text(lines('note', 30)) }, 'base'] },
      { commit: ['adds', { 'notes.md': text(lines('note', 30).flatMap(l => (l === 'note 10' ? [l, 'note 10a'] : [l]))) }, 'adds'] },
      { commit: ['removes', { 'notes.md': text(lines('note', 30).flatMap(l => (l === 'note 10' ? [l, 'note 10a'] : l === 'note 11' || l === 'note 16' ? [] : [l]))) }, 'removes'] },
    ],
  },
  {
    name: 'a member whose head is not in the repository',
    base: 'main',
    members: [[801, 'only'], [802, '0123456789abcdef0123456789abcdef01234567']],
    script: [
      { commit: ['main', { 'notes.md': text(lines('note', 5)) }, 'base'] },
      { commit: ['only', { 'notes.md': text(lines('note', 6)) }, 'only'] },
    ],
  },
];

function play(repo, script) {
  for (const op of script) {
    if (op.commit) repo.commit(...op.commit);
    else if (op.checkout) repo.checkout(op.checkout);
    else repo.git(...op.git);
  }
}

function answers(repo, c) {
  const git = new Git(repo.work);
  const members = c.members.map(([pull, head]) => ({ pull, head: /^[0-9a-f]{40}$/.test(head) ? head : repo.sha(head) }));
  const known = members.filter(m => !/^0123/.test(m.head));
  const diff = readDiff(git, { base: c.base, head: known[known.length - 1].head, exact: false });
  try {
    const read = readStackSteps(git, { combinedBase: diff.mergeBase, baseCommit: diff.baseCommit, top: diff.headCommit, members });
    const mapped = mapStackLines(read.steps, diff.chunks, members.map(m => m.pull));
    return { members, top: diff.headCommit, heads: read.heads, starts: read.steps.map(s => s.start), warnings: read.warnings, counts: read.counts, lines: mapped.lines, files: mapped.files };
  } catch (error) {
    return { members, top: diff.headCommit, error: error.message };
  }
}

function recorded() {
  return CASES.map(c => {
    const repo = throwawayRepo();
    try {
      play(repo, c.script);
      return { name: c.name, base: c.base, script: c.script, ...answers(repo, c) };
    } finally {
      repo.done();
    }
  });
}

test('the stacks the Go stack reader and line map are tested against read as Node reads them now', () => {
  const now = JSON.parse(JSON.stringify(recorded()));
  if (process.env.LAZYREVIEW_RECORD) {
    writeFileSync(RECORD, JSON.stringify(now, null, 1) + '\n');
    return;
  }
  assert.deepEqual(JSON.parse(readFileSync(RECORD, 'utf8')), now, `${RECORD} is not what Node reads now; write it again with LAZYREVIEW_RECORD=1 npm test, then run the Go tests`);
});
