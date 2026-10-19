import { test } from 'node:test';
import assert from 'node:assert/strict';
import { hiddenBy } from '../../src/home/hidden-by.mjs';

test('an entry hides the repository it names, and owner/* every repository of the owner', () => {
  const entries = ['octo/widgets', 'reviewer-home/*'];
  assert.deepEqual(hiddenBy(entries, 'octo/widgets'), ['octo/widgets']);
  assert.deepEqual(hiddenBy(entries, 'octo/gadgets'), []);
  assert.deepEqual(hiddenBy(entries, 'reviewer-home/notes'), ['reviewer-home/*']);
  assert.deepEqual(hiddenBy(entries, 'reviewer/notes'), [], 'an owner is matched whole, not by its start');
  assert.deepEqual(hiddenBy(['octo/widgets'], 'octo/widgets-two'), [], 'a name is matched whole');
  assert.deepEqual(hiddenBy([], 'octo/widgets'), []);
});

test('every entry that hides a repository is named, as the config writes it, whatever its case', () => {
  assert.deepEqual(hiddenBy(['Octo/*', 'octo/Widgets', 'octo/gadgets'], 'octo/widgets'), ['Octo/*', 'octo/Widgets']);
  assert.deepEqual(hiddenBy(['octo/*'], 'OCTO/Widgets'), ['octo/*']);
});
