import { test } from 'node:test';
import assert from 'node:assert/strict';
import { PullRef } from '../../src/github/pull-ref.mjs';

test('a pull request is read from its address, its review id or its short form', () => {
  for (const text of ['https://github.com/octo/widgets/pull/12', 'https://github.com/octo/widgets/pull/12/files#diff-1', 'github.com/octo/widgets/12', 'octo/widgets#12']) {
    const ref = PullRef.parse(text);
    assert.deepEqual([ref.host, ref.repo, ref.pull, ref.id, ref.url],
      ['github.com', 'octo/widgets', 12, 'github.com/octo/widgets/12', 'https://github.com/octo/widgets/pull/12'], text);
  }
  assert.equal(PullRef.parse('https://git.example.com/team/app/pull/3').id, 'git.example.com/team/app/3');
  assert.equal(PullRef.parse('octo/widgets#12').withPull(13).id, 'github.com/octo/widgets/13');
});

test('anything else is refused, including names that would climb out of a folder', () => {
  for (const text of ['', 'https://github.com/octo/widgets/issues/12', 'octo#12', 'https://github.com/../x/pull/1', 'github.com/octo/../1', 'octo/widgets#0']) {
    assert.throws(() => PullRef.parse(text), /pull request/, text);
  }
});
