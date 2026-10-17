import { test } from 'node:test';
import assert from 'node:assert/strict';
import { BranchRef, repositoryName } from '../../src/models/branch-ref.mjs';

test('a branch review id names the repository and the branch, and reads back', () => {
  const ref = BranchRef.of('/work/widgets/.git', 'feature/search');
  assert.match(ref.id, /^local\/widgets-[0-9a-f]{8}\/feature~search$/);
  const back = BranchRef.parse(ref.id);
  assert.deepEqual([back.repoKey, back.branch], [ref.repoKey, 'feature/search']);
  assert.notEqual(BranchRef.of('/elsewhere/widgets/.git', 'feature/search').id, ref.id, 'a repository of the same name elsewhere is another repository');
  assert.equal(BranchRef.of('/work/widgets/.git', 'feature/search').id, ref.id);
});

test('a repository is named for its folder, or a bare one for itself, as a name an address can hold', () => {
  assert.equal(repositoryName('/work/widgets/.git'), 'widgets');
  assert.equal(repositoryName('/remotes/widgets.git'), 'widgets');
  assert.equal(repositoryName('/work/.dot files/.git'), 'dot-files');
});

test('only a branch name and a key a review folder can safely hold are taken', () => {
  for (const id of ['local/widgets-1a2b3c4d/..', 'local/../feature', 'local/widgets-1a2b3c4d/.hidden', 'local/widgets/feature',
    'local/widgets-1a2b3c4d/a~~b', 'local/widgets-1a2b3c4d/a b', 'github.com/octo/widgets/1', 'local/widgets-1a2b3c4d/x.lock']) {
    assert.throws(() => BranchRef.parse(id), /not a branch review/, id);
  }
  assert.equal(BranchRef.parse('local/widgets-1a2b3c4d/fix#2').branch, 'fix#2');
});
