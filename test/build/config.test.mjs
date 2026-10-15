import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readConfig } from '../../src/build/config/config.mjs';

test('no config is the empty config', () => {
  assert.deepEqual(readConfig(undefined), {
    title: '', summary: '', areas: [], components: [], links: [], summaries: [], groups: [],
    contextTypes: { include: [], exclude: [], max: 40 },
  });
});

test('a full config is read with its defaults filled in', () => {
  const config = readConfig({
    title: 'Search',
    areas: [{ id: 'sdk', title: 'SDK', components: ['engine', 'context'] }],
    components: [{ id: 'engine', title: 'Engine', color: 'teal', packages: [{ id: 'core', title: 'Core', paths: ['Sources/'] }] }],
    links: [{ from: 'A', to: 'B', label: 'runs' }, { from: 'C', to: 'D', label: 'wraps', kind: 'has' }],
    groups: [{ id: 'payloads', of: 'Payload', title: 'payloads' }],
    contextTypes: { include: ['Monitor'], max: 10 },
  });
  assert.equal(config.components[0].packages[0].description, '');
  assert.equal(config.links[0].kind, 'use');
  assert.equal(config.links[1].kind, 'has');
  assert.deepEqual(config.contextTypes, { include: ['Monitor'], exclude: [], max: 10 });
  assert.deepEqual(config.areas[0].components, ['engine', 'context']);
});

test('every problem in a config is reported at once', () => {
  assert.throws(() => readConfig({
    titel: 'typo',
    components: [
      { id: 'Bad Id', title: 'One', paths: ['a/'] },
      { id: 'two', title: 'Two' },
      { id: 'three', title: 'Three', color: 'pink', packages: [{ id: 'p', title: 'P', paths: [] }] },
    ],
    areas: [{ id: 'x', title: 'X', components: ['missing'] }],
  }), error => {
    for (const bit of ['unknown key "titel"', 'id must be lower case', 'component "two" needs paths or packages',
      'color must be one of', 'needs at least one path', 'names component "missing"']) {
      assert.ok(error.message.includes(bit), `expected "${bit}" in:\n${error.message}`);
    }
    return true;
  });
});

const REVIEW = `---
pull: 42
title: Make it faster
url: https://github.com/o/r/pull/42
author: someone
base: main
head: abc1234
---
## For the user
Pages load in half the time.
## What this changes and why
It caches the index.
## Watch out for
Nothing needs a decision.
## What I think is wrong
Nothing at medium priority or above.
## The flow
\`\`\`
## not a heading, inside a fence
\`\`\`
## Unverified
Nothing.
`;
