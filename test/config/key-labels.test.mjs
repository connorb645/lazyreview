import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readerName, labelProblem } from '../../src/config/key-labels.mjs';
import { configFrom, terminalPart } from '../../src/config/user-config.mjs';

const PATH = '/home/lee/.config/lazyreview/config.yml';
const read = text => configFrom(text, PATH, '/home/lee');

test('every form of key turns into the name the terminal app\'s key reader gives it', () => {
  const forms = {
    j: 'j', G: 'G', '?': '?', ':': ':', '+': '+', _: '_', ']': ']', 0: '0', é: 'é',
    '<space>': 'space', '<enter>': 'enter', '<esc>': 'esc', '<tab>': 'tab', '<up>': 'up', '<down>': 'down', '<left>': 'left', '<right>': 'right',
    '<c-o>': 'ctrl+o', '<c-c>': 'ctrl+c', '<c-i>': 'tab',
    gg: 'g g', ZZ: 'Z Z', ']c': '] c', '[[': '[ [', zR: 'z R', '<space>v': 'space v', '<space>?': 'space ?', 'g<space>': 'g space', '<c-w>j': 'ctrl+w j',
    '<disabled>': '',
  };
  for (const [label, name] of Object.entries(forms)) {
    assert.equal(labelProblem(label), null, label);
    assert.equal(readerName(label), name, label);
  }
});

test('a key that is not one is refused, saying why', () => {
  const refused = {
    '': 'a key is not empty',
    '<': 'a < starts a named key, such as <space>, and needs its >',
    '<space': 'a < starts a named key, such as <space>, and needs its >',
    '<foo>': '<foo> is not a key; a named key is <space>, <enter>, <esc>, <tab>, <up>, <down>, <left>, <right>, or <c-x> for Ctrl-x',
    '<c-1>': '<c-1> is not a key; a named key is <space>, <enter>, <esc>, <tab>, <up>, <down>, <left>, <right>, or <c-x> for Ctrl-x',
    '<C-o>': '<C-o> is not a key; a named key is <space>, <enter>, <esc>, <tab>, <up>, <down>, <left>, <right>, or <c-x> for Ctrl-x',
    'g<disabled>': '<disabled> stands alone, for no key',
  };
  for (const [label, why] of Object.entries(refused)) assert.equal(labelProblem(label), why, label);
  assert.equal(labelProblem(3), 'a key is written as text');
});

test('a file changes one action\'s keys and keeps the rest, and the terminal app is handed them by name', () => {
  const { values, sources } = read("keybinding:\n  universal:\n    quit: ['ZZ', '<c-q>']\n  diff:\n    nextHunk: ']h'\n    drop: '<disabled>'\n  home:\n    jumpToRepository: ['1', '2', '<disabled>', '4', '5', '6', '7', '8', '9']\n");
  assert.deepEqual(values.keybinding.universal.quit, ['ZZ', '<c-q>']);
  assert.equal(values.keybinding.universal.close[1], '<esc>', 'an action the file leaves out keeps its keys');
  assert.deepEqual([sources['keybinding.diff.nextHunk'], sources['keybinding.diff.previousHunk']], ['file', 'default']);
  const handed = terminalPart({ path: PATH, values }, {}).keybinding;
  assert.deepEqual(handed.universal.quit, ['Z Z', 'ctrl+q']);
  assert.deepEqual(handed.diff.nextHunk, ['] h']);
  assert.deepEqual(handed.diff.drop, [], 'a disabled action has no key');
  assert.deepEqual(handed.home.jumpToRepository, ['1', '2', '', '4', '5', '6', '7', '8', '9'], 'a numbered list keeps its places');
});

test('keys the config cannot take are refused, naming the action', () => {
  const one = (text, words) => assert.throws(() => read(text), new RegExp(`^Error: ${PATH.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&')} ${words.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&')}$`), text);
  one("keybinding:\n  diff:\n    nextHunk: '<hunk>'\n", 'gives keybinding.diff.nextHunk "<hunk>", which is not a key or a list of keys: <hunk> is not a key; a named key is <space>, <enter>, <esc>, <tab>, <up>, <down>, <left>, <right>, or <c-x> for Ctrl-x');
  one('keybinding:\n  diff:\n    nextHunk: []\n', 'gives keybinding.diff.nextHunk [], which is not a key or a list of keys: a key is not empty');
  one('keybinding:\n  diff:\n    nextHunk: 3\n', 'gives keybinding.diff.nextHunk 3, which is not a key or a list of keys: a key is written as text');
  one("keybinding:\n  home:\n    jumpToRepository: ['1', '2']\n", 'gives keybinding.home.jumpToRepository ["1","2"], which is not a list of 9 keys, one for each by its place');
  one("keybinding:\n  posting:\n    choose: '1'\n", 'gives keybinding.posting.choose "1", which is not a list of 3 keys, one for each by its place');
  one("keybinding:\n  diff:\n    nextHnk: ']h'\n", 'sets keybinding.diff.nextHnk, which is not a setting');
  one("keybindings:\n  diff:\n    nextHunk: ']h'\n", 'sets keybindings, which is not a setting');
});
