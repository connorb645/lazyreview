import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, writeFileSync, readFileSync, rmSync, mkdirSync, symlinkSync, lstatSync, statSync, chmodSync, readdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { saveListChange } from '../../src/config/list-save.mjs';

const KEY = 'github.hiddenRepositories';
const WRITTEN = '# Mine.\ngithub:\n  accounts:\n    # - login: old\n    - login: lee\n';

function folder() {
  const dir = mkdtempSync(join(tmpdir(), 'lazyreview-save-'));
  return { dir, done: () => { chmodSync(dir, 0o755); rmSync(dir, { recursive: true, force: true }); } };
}

test('a change is saved in place, the file keeping its mode, with no other file left beside it', () => {
  const f = folder();
  try {
    const file = join(f.dir, 'config.yml');
    writeFileSync(file, WRITTEN);
    chmodSync(file, 0o600);
    saveListChange(file, KEY, { add: ['octo/*'] });
    assert.equal(readFileSync(file, 'utf8'), `${WRITTEN}  hiddenRepositories:\n    - octo/*\n`);
    assert.equal(statSync(file).mode & 0o777, 0o600);
    assert.deepEqual(readdirSync(f.dir), ['config.yml']);
  } finally {
    f.done();
  }
});

test('a config that is a link keeps its link, and the file it points to changes', () => {
  const f = folder();
  try {
    mkdirSync(join(f.dir, 'dotfiles'));
    const target = join(f.dir, 'dotfiles', 'lazyreview.yml');
    const link = join(f.dir, 'config.yml');
    writeFileSync(target, WRITTEN);
    symlinkSync(target, link);
    saveListChange(link, KEY, { add: ['octo/widgets'] });
    assert.ok(lstatSync(link).isSymbolicLink());
    assert.match(readFileSync(target, 'utf8'), /hiddenRepositories:\n {4}- octo\/widgets\n$/);
  } finally {
    f.done();
  }
});

test('with no file, one is made, and its folder, holding only the list', () => {
  const f = folder();
  try {
    const file = join(f.dir, 'lazyreview', 'config.yml');
    saveListChange(file, KEY, { add: ['octo/*'] });
    assert.equal(readFileSync(file, 'utf8'), 'github:\n  hiddenRepositories:\n    - octo/*\n');
    saveListChange(join(f.dir, 'other', 'config.yml'), KEY, { remove: ['octo/*'] });
    assert.deepEqual(readdirSync(f.dir), ['lazyreview'], 'taking off what no file holds makes no file');
  } finally {
    f.done();
  }
});

test('a save that fails, or that the file refuses, leaves the old file whole and says why, naming it', () => {
  const f = folder();
  try {
    const file = join(f.dir, 'config.yml');
    writeFileSync(file, WRITTEN);
    chmodSync(f.dir, 0o555);
    assert.throws(() => saveListChange(file, KEY, { add: ['octo/*'] }), new RegExp(`^Error: ${file.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')} was not changed: .*permission denied.*; change it by hand, as docs/Config.md shows$`));
    chmodSync(f.dir, 0o755);
    assert.equal(readFileSync(file, 'utf8'), WRITTEN);
    assert.deepEqual(readdirSync(f.dir), ['config.yml']);
    writeFileSync(file, 'github: {accounts: []}\n');
    assert.throws(() => saveListChange(file, KEY, { add: ['octo/*'] }), /was not changed: github is written in a way lazyreview cannot change line by line/);
    assert.equal(readFileSync(file, 'utf8'), 'github: {accounts: []}\n');
  } finally {
    f.done();
  }
});
