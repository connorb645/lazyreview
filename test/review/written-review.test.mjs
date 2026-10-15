import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readWrittenReview, checkWrittenReview } from '../../src/review/written-review.mjs';

const review = { members: [{ pull: 12, head: '3f2a9c1d0e77' }, { pull: 13, head: '9d01c2aa45bb' }] };

test('a written review is read into what it read and its sections, in order', () => {
  const written = readWrittenReview('---\nstack: 12, 13\nhead: 9d01c2aa\ntitle: Search and basket\n---\n## What it does\nAdds search.\n\n```md\n## not a heading\n```\n## Risks\n### Smaller\nNone.\n## Risks\nAgain.\n');
  assert.deepEqual(written.pulls, [12, 13]);
  assert.equal(written.head, '9d01c2aa');
  assert.equal(written.title, 'Search and basket');
  assert.deepEqual(written.sections.map(s => [s.id, s.title]), [['what-it-does', 'What it does'], ['risks', 'Risks'], ['risks-2', 'Risks']]);
  assert.match(written.sections[0].markdown, /## not a heading/);
  assert.doesNotThrow(() => checkWrittenReview(written, review));
});

test('a written review with problems names every one, and must match the review it is attached to', () => {
  assert.throws(() => readWrittenReview('## Only\nx\n'), /must start with a header/);
  assert.throws(() => readWrittenReview('---\npull: 12\nstack: 12, 13\nurl: x\n---\nintro\n## Empty\n'),
    /problems:[\s\S]*unknown field "url"[\s\S]*needs "head"[\s\S]*not both[\s\S]*text before the first ## section heading/);
  assert.throws(() => readWrittenReview('---\npull: 12\nhead: abcdef1\n---\n## Empty\n\n'), /"## Empty" is empty/);
  const single = readWrittenReview('---\npull: 12\nhead: abcdef1\n---\n## Notes\nx\n');
  assert.throws(() => checkWrittenReview(single, review), /names #12, but the review shows #12, #13/);
  const old = readWrittenReview('---\nstack: 12, 13\nhead: 1111111\n---\n## Notes\nx\n');
  assert.throws(() => checkWrittenReview(old, review), /read 1111111, but the review shows head 9d01c2aa45bb/);
});
