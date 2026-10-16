import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, rmSync, realpathSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { throwawayRepo } from '../support/throwaway-repo.mjs';
import { DataHome } from '../../src/store/data-home.mjs';
import { ReviewStore } from '../../src/store/review-store.mjs';
import { BranchLoader } from '../../src/loader/branch-loader.mjs';
import { reviewState } from '../../src/server/review-state.mjs';

// A repository with main and a branch, feature/search, one commit ahead of it,
// with main checked out, and a data home to load it into.
function bench() {
  const repo = throwawayRepo();
  repo.commit('main', { 'src/catalog.ts': 'export const items: string[] = [];\n' }, 'base');
  repo.commit('feature/search', { 'src/search.ts': 'export function find(text: string) {\n  return text;\n}\n' }, 'Add search');
  repo.checkout('main');
  const root = mkdtempSync(join(tmpdir(), 'review-desk-home-'));
  const store = new ReviewStore(new DataHome(root));
  let tick = 0;
  const now = () => new Date(Date.UTC(2026, 9, 16, 9, 0, tick++)).toISOString();
  return {
    repo, store, root,
    loader: new BranchLoader({ store, now }),
    done() {
      repo.done();
      rmSync(root, { recursive: true, force: true });
    },
  };
}

test('a branch loads from its repository into a read-only review, kept by repository and branch', async () => {
  const b = bench();
  try {
    const loaded = await b.loader.load({ repo: b.repo.work, base: 'main', head: 'feature/search' });
    assert.match(loaded.ref.id, /^local\/work-[0-9a-f]{8}\/feature~search$/);
    assert.deepEqual(loaded.counts, { files: 1, adds: 3, dels: 0, components: 1, types: 0 });
    const review = b.store.read(loaded.ref);
    assert.equal(review.kind, 'branch');
    assert.equal(review.title, 'feature/search', 'a branch is named for itself unless given a title');
    assert.equal(review.path, b.repo.work);
    assert.deepEqual(review.base, { ref: 'main', sha: b.repo.sha('main') });
    assert.deepEqual(review.head, { ref: 'feature/search', sha: b.repo.sha('feature/search') });
    assert.equal(review.start, b.repo.sha('main'));
    const page = b.store.readPage(loaded.ref).page;
    assert.deepEqual(Object.keys(page.diff), ['src/search.ts']);
    assert.equal(page.stack, null);
    assert.match(page.model.heading.context.map(p => p[0]).join(''), /^Branch feature\/search on main: head [0-9a-f]{8}, merge base [0-9a-f]{8}, 1 commit$/);
    assert.deepEqual(b.store.list().map(e => e.ref.id), [loaded.ref.id]);
    const state = reviewState(b.store, loaded.ref);
    assert.equal(state.queue, null, 'a branch review has no queue');
    assert.equal(state.members, undefined, 'and no pull requests');
  } finally {
    b.done();
  }
});

test('loading the same branch again, from any worktree of its repository, updates its review in place', async () => {
  const b = bench();
  const outside = realpathSync(mkdtempSync(join(tmpdir(), 'review-desk-worktree-')));
  const second = join(outside, 'second');
  try {
    const first = await b.loader.load({ repo: b.repo.work, base: 'main', head: 'feature/search', title: 'Search', context: true });
    b.repo.git('worktree', 'add', '-q', second, 'feature/search');
    b.repo.git('-C', second, 'commit', '-q', '--allow-empty', '-m', 'Nothing');
    const again = await b.loader.load({ repo: second, base: 'main', head: 'HEAD' });
    assert.equal(again.ref.id, first.ref.id);
    const review = b.store.read(again.ref);
    assert.equal(review.head.sha, b.repo.sha('feature/search'), 'the head is the branch as it now stands');
    assert.equal(review.path, second);
    assert.equal(review.title, 'Search', 'the title given once is kept');
    assert.equal(review.context, true, 'so is the choice of context');
    assert.equal(b.store.list().length, 1);

    b.repo.git('-C', second, 'commit', '-q', '--allow-empty', '-m', 'Again');
    const refreshed = await b.loader.refresh(first.ref);
    assert.equal(refreshed.review.head.sha, b.repo.sha('feature/search'));
    assert.equal(refreshed.review.loaded_at, '2026-10-16T09:00:02.000Z');
  } finally {
    b.repo.git('worktree', 'remove', '--force', second);
    rmSync(outside, { recursive: true, force: true });
    b.done();
  }
});

test('two repositories of the same name keep their branches apart', async () => {
  const one = bench();
  const two = bench();
  try {
    const a = await one.loader.load({ repo: one.repo.work, base: 'main', head: 'feature/search' });
    const b = await two.loader.load({ repo: two.repo.work, base: 'main', head: 'feature/search' });
    assert.notEqual(a.ref.id, b.ref.id);
    assert.equal(a.ref.repoKey.split('-')[0], b.ref.repoKey.split('-')[0]);
  } finally {
    one.done();
    two.done();
  }
});

test('a branch load names what was wrong, and writes nothing', async () => {
  const b = bench();
  const empty = mkdtempSync(join(tmpdir(), 'review-desk-empty-'));
  try {
    await assert.rejects(b.loader.load({ repo: b.repo.work, base: 'main', head: b.repo.sha('feature/search') }),
      /is not a branch of .*; give a branch, so loading it again updates the same review/);
    await assert.rejects(b.loader.load({ repo: b.repo.work, base: 'main', head: 'feature/nothing' }), /"feature\/nothing" is not a branch of /);
    await assert.rejects(b.loader.load({ repo: b.repo.work, base: 'nowhere', head: 'feature/search' }), /"nowhere" is not a commit in this repository/);
    await assert.rejects(b.loader.load({ repo: empty, base: 'main', head: 'main' }), /not a git repository/);
    await assert.rejects(b.loader.load({ repo: b.repo.work, base: 'main', head: 'main' }), /there is no difference between main and main/);
    assert.deepEqual(b.store.list(), []);
  } finally {
    rmSync(empty, { recursive: true, force: true });
    b.done();
  }
});
