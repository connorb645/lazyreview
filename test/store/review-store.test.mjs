import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, rmSync, readFileSync, writeFileSync, mkdirSync, existsSync, chmodSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { DataHome } from '../../src/store/data-home.mjs';
import { ReviewStore } from '../../src/store/review-store.mjs';
import { withLocks } from '../../src/store/file-lock.mjs';
import { PullRef } from '../../src/models/pull-ref.mjs';

const record = (store, ref, value) => [store.home.review(ref).record, JSON.stringify(value, null, 2) + '\n'];

function scratch() {
  const root = mkdtempSync(join(tmpdir(), 'lazyreview-store-'));
  return { root, store: new ReviewStore(new DataHome(root)), done: () => rmSync(root, { recursive: true, force: true }) };
}

test('a change writes every file or none', async () => {
  const s = scratch();
  try {
    const a = new PullRef('github.com', 'octo/widgets', 1);
    await s.store.change([a], async () => ({ files: [record(s.store, a, { id: a.id, loaded_at: 'x', members: [] })], result: null }));
    const before = readFileSync(s.store.home.review(a).record, 'utf8');
    const blocked = join(s.root, 'blocked');
    mkdirSync(blocked);
    chmodSync(blocked, 0o500);
    await assert.rejects(s.store.change([a], async () => ({
      files: [record(s.store, a, { id: a.id, changed: true }), [join(blocked, 'deeper', 'file.json'), '{}']],
      result: null,
    })), /could not write the review, so every file is as it was/);
    chmodSync(blocked, 0o700);
    assert.equal(readFileSync(s.store.home.review(a).record, 'utf8'), before);
  } finally {
    s.done();
  }
});

test('a review file that is not JSON is named in the error', () => {
  const s = scratch();
  try {
    const a = new PullRef('github.com', 'octo/widgets', 1);
    const folder = s.store.home.review(a);
    mkdirSync(folder.path, { recursive: true });
    writeFileSync(folder.record, '{"id":');
    assert.throws(() => s.store.read(a), { message: `could not read ${folder.record}: Unexpected end of JSON input` });
  } finally {
    s.done();
  }
});

test('the store lists every review newest first, and finds the one holding a pull request', async () => {
  const s = scratch();
  try {
    const a = new PullRef('github.com', 'octo/widgets', 1);
    const b = new PullRef('github.com', 'octo/widgets', 2);
    const member = pull => ({ pull, comments: [], posted: [] });
    await s.store.change([a, b], async () => ({
      files: [
        record(s.store, a, { id: a.id, loaded_at: '2026-10-15T10:00:00Z', moved: null, members: [member(1), member(2)], earlier: [] }),
        record(s.store, b, { id: b.id, loaded_at: '2026-10-15T09:00:00Z', moved: { to: a.id }, members: [member(2)], earlier: [] }),
      ],
      result: null,
    }));
    assert.deepEqual(s.store.list().map(e => e.ref.id), [a.id, b.id]);
    assert.equal(s.store.find(b).ref.id, a.id);
    assert.equal(s.store.find(new PullRef('github.com', 'octo/widgets', 3)), null);
  } finally {
    s.done();
  }
});

test('locks let one holder at a time in, and a lock whose holder died is taken over', async () => {
  const s = scratch();
  try {
    const lock = join(s.root, 'one.lock');
    const order = [];
    await Promise.all([
      withLocks([lock], async () => { order.push('a in'); await new Promise(r => setTimeout(r, 30)); order.push('a out'); }),
      withLocks([lock], async () => { order.push('b in'); order.push('b out'); }),
    ]);
    assert.deepEqual(order, ['a in', 'a out', 'b in', 'b out']);
    mkdirSync(lock);
    writeFileSync(join(lock, 'pid'), '999999');
    assert.equal(await withLocks([lock], async () => 'taken over'), 'taken over');
    assert.ok(!existsSync(lock));
  } finally {
    s.done();
  }
});
