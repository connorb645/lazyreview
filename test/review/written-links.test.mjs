import { test } from 'node:test';
import assert from 'node:assert/strict';
import { writtenLinks } from '../../src/review/written-links.mjs';

test('a written review links each file:line in inline code to the one file whose path ends with it', () => {
  const markdown = 'See `Basket.swift:12`, `log.ts:3-4` and `Basket.swift:12` again, not `x.ts:1`, nor `nowhere.ts:2`.\n\n```\nBasket.swift:2\n```\n\n- in a list, `Sources/Basket/Basket.swift:5`';
  const paths = ['Sources/Basket/Basket.swift', 'src/log.ts', 'a/x.ts', 'b/x.ts'];
  assert.deepEqual(writtenLinks(markdown, paths), [
    { text: 'Basket.swift:12', path: 'Sources/Basket/Basket.swift', line: 12 },
    { text: 'log.ts:3-4', path: 'src/log.ts', line: 3 },
    { text: 'Sources/Basket/Basket.swift:5', path: 'Sources/Basket/Basket.swift', line: 5 },
  ], 'a name two paths end with, one no path ends with, and code in a code block are no links');
  assert.deepEqual(writtenLinks('', paths), []);
});
