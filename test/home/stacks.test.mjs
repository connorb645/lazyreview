import { test } from 'node:test';
import assert from 'node:assert/strict';
import { stacks } from '../../src/home/stacks.mjs';

const pull = (number, base, head, repo = 'octo/widgets') => ({ number, base: { ref: base }, head: { ref: head, repo: { full_name: repo } } });
const numbers = found => Object.fromEntries([...found].map(([n, group]) => [n, group.map(p => p.number)]));

test('each pull request of a chain is reviewed with the whole chain, bottom first', () => {
  const pulls = [pull(23, 'basket', 'checkout'), pull(21, 'main', 'search'), pull(22, 'search', 'basket'), pull(30, 'main', 'wording')];
  assert.deepEqual(numbers(stacks(pulls, 'octo/widgets')), { 21: [21, 22, 23], 22: [21, 22, 23], 23: [21, 22, 23], 30: [30] });
});

test('a chain that forks is no stack, and a branch from another repository chains nothing', () => {
  const forked = [pull(21, 'main', 'search'), pull(22, 'search', 'basket'), pull(24, 'search', 'filters')];
  assert.deepEqual(numbers(stacks(forked, 'octo/widgets')), { 21: [21], 22: [22], 24: [24] });
  const elsewhere = [pull(21, 'main', 'search', 'sam/widgets'), pull(22, 'search', 'basket')];
  assert.deepEqual(numbers(stacks(elsewhere, 'octo/widgets')), { 21: [21], 22: [22] });
});
