import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readTerminalLink } from '../../src/cli/terminal-link.mjs';

const reviewOf = text => readTerminalLink(text).review?.id;

test('the address load and list print names its review, on any port, whatever follows it', () => {
  for (const address of [
    'http://127.0.0.1:4747/review/github.com/octo/widgets/30',
    'http://127.0.0.1:48231/review/github.com/octo/widgets/30#review',
    'http://localhost:4747/review/github.com/octo/widgets/30/?file=a.swift',
    'https://127.0.0.1/review/github.com/octo/widgets/30',
  ]) {
    assert.equal(reviewOf(address), 'github.com/octo/widgets/30', address);
  }
  assert.equal(reviewOf('http://127.0.0.1:4747/review/local/widgets-1a2b3c4d/feature~search'), 'local/widgets-1a2b3c4d/feature~search');
  assert.equal(readTerminalLink('http://127.0.0.1:4747/review/local/widgets-1a2b3c4d/fix%2525').review.branch, 'fix%25', 'an address is decoded once, as the app decodes it');
});

test('a pull request link names the pull request, and a review id its review', () => {
  assert.equal(readTerminalLink('https://github.com/octo/widgets/pull/30/files').pull.id, 'github.com/octo/widgets/30');
  assert.equal(readTerminalLink('https://github.com/octo/widgets/pull/30').review, undefined);
  assert.equal(reviewOf('github.com/octo/widgets/30'), 'github.com/octo/widgets/30');
  assert.equal(reviewOf('local/widgets-1a2b3c4d/feature~search'), 'local/widgets-1a2b3c4d/feature~search');
});

test('text that names no review or pull request is refused in one line naming all of it', () => {
  for (const text of [
    'widgets',
    'http://127.0.0.1:4747/',
    'http://127.0.0.1:4747/review/garbage',
    'http://127.0.0.1:4747/review/github.com/octo/widgets/%E0',
    'http://127.0.0.1:4747/api/reviews/github.com/octo/widgets/30',
    'https://example.com/octo/widgets',
  ]) {
    assert.throws(() => readTerminalLink(text), { message: `"${text}" is not the address of a review or of a pull request; lazyreview list shows every review` }, text);
  }
});
