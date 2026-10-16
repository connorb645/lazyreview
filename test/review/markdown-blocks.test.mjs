import { test } from 'node:test';
import assert from 'node:assert/strict';
import { markdownBlocks } from '../../src/review/markdown-blocks.mjs';

const lines = text => text.replace(/\n$/, '').split('\n');

// A changed file whose whole text the page holds, its diff one hunk that removes every old line and adds every new one.
function changed(before, after) {
  const b = lines(before), a = lines(after);
  const rows = [...b.map(text => ({ kind: 'del', text })), ...a.map(text => ({ kind: 'add', text }))];
  return markdownBlocks({ status: 'M' }, { hunks: [{ old_start: 1, old_count: b.length, new_start: 1, new_count: a.length, rows }], whole: a }).blocks;
}
const find = (blocks, text) => blocks.find(b => b.markdown.includes(text));
const at = b => [b.state, b.side, b.start_line, b.line, b.text_line, b.left && [b.left.start_line, b.left.line]];

test('the Rendered view marks each added or removed block, and a changed list item by item', () => {
  const blocks = changed('# Guide\n\nOne.\n\n- a\n- b\n', '# Guide\n\nOne.\n\nTwo.\n\n- a\n- c\n');
  assert.deepEqual(blocks.map(b => [b.state, b.markdown]), [['same', '# Guide'], ['same', 'One.'], ['add', 'Two.'], ['same', '- a'], ['del', '- b'], ['add', '- c']]);
  const [a, b, c] = blocks.slice(3);
  assert.ok(a.list !== null && a.list === b.list && b.list === c.list, 'the items of one edited list stay one list');
  const grown = changed('Intro.\n\n- a\n- b\n', 'Intro, longer.\n\n- a\n- b\n- c\n\n1. one\n');
  assert.deepEqual(grown.map(b => [b.state, b.markdown]), [['del', 'Intro.'], ['add', 'Intro, longer.'], ['same', '- a'], ['same', '- b'], ['add', '- c'], ['add', '1. one']],
    'a list is paired with the one list of its kind, though a list of another kind is added beside it');
});

test('each block and list item names the source lines it was read from', () => {
  const blocks = changed('# Guide\n\nOne\nline.\n\n- a\n- b\n\n```\ncode\n```\n', '# Guide\n\nOne\nline.\n\nTwo.\n\n- a\n- c\n\n```\ncode\n```\n');
  assert.deepEqual(at(find(blocks, 'Guide')), ['same', 'RIGHT', 1, 1, 1, [1, 1]]);
  assert.deepEqual(at(find(blocks, 'One\nline')), ['same', 'RIGHT', 3, 4, 3, [3, 4]], 'a paragraph keeps its line breaks, so a sentence in it maps to its own line');
  assert.deepEqual(at(find(blocks, 'Two')), ['add', 'RIGHT', 6, 6, 6, null]);
  assert.deepEqual(at(find(blocks, '- a')), ['same', 'RIGHT', 8, 8, 8, [6, 6]]);
  assert.deepEqual(at(find(blocks, '- b')), ['del', 'LEFT', 7, 7, 7, null], 'a removed block names its lines before the change');
  assert.deepEqual(at(find(blocks, 'code')), ['same', 'RIGHT', 11, 13, 12, [9, 11]], 'a fenced block\'s text starts after its fence');
  assert.deepEqual(at(find(changed('[x]: https://example.com\n\nOld.\n', '[x]: https://example.com\n\nNew.\n'), 'New')), ['add', 'RIGHT', 3, 3, 3, null], 'a link definition, which draws nothing, still counts its lines');
});

test('an added file is read whole, an ordered list keeps each item\'s number, and a file without its whole text shows its changed parts', () => {
  const added = markdownBlocks({ status: 'A' }, { hunks: [{ rows: lines('First.\n\n1. x\n1. y\n').map(text => ({ kind: 'add', text })) }], whole: null });
  assert.deepEqual(added.blocks.map(b => [b.state, b.markdown, b.start_line]), [['add', 'First.', 1], ['add', '1. x', 3], ['add', '2. y', 4]]);
  const parts = markdownBlocks({ status: 'M' }, { hunks: [
    { old_start: 40, new_start: 42, rows: [{ kind: 'del', text: 'Old.' }, { kind: 'add', text: 'New.' }] },
    { old_start: 90, new_start: 92, rows: [{ kind: 'add', text: 'More.' }] },
  ], whole: null });
  assert.equal(parts.partial, true);
  assert.deepEqual(parts.blocks.map(b => [b.state, b.side, b.start_line, b.part]), [['del', 'LEFT', 40, 0], ['add', 'RIGHT', 42, 0], ['add', 'RIGHT', 92, 1]]);
});
