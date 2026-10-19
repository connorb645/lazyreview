import { test } from 'node:test';
import assert from 'node:assert/strict';
import { listEdited } from '../../src/config/list-edit.mjs';

const KEY = 'github.hiddenRepositories';
const lines = (...given) => `${given.join('\n')}\n`;

// A config a person wrote: a comment at the top, a commented-out account, a value with a comment beside it, blank lines.
const WRITTEN = lines(
  '# yaml-language-server: $schema=https://example.com/config.json',
  '# My config.',
  '',
  'github:',
  '  accounts:',
  '    - login: reviewer',
  '    # - login: old-account',
  '    - login: reviewer-home',
  '',
  'gui:',
  '  lineNumbers: relative  # easier to jump',
);

test('an entry goes at the end of its section, a list of its own, and every other line stays as it was', () => {
  const once = listEdited(WRITTEN, KEY, { add: ['octo/widgets'] });
  assert.equal(once, lines(
    '# yaml-language-server: $schema=https://example.com/config.json',
    '# My config.',
    '',
    'github:',
    '  accounts:',
    '    - login: reviewer',
    '    # - login: old-account',
    '    - login: reviewer-home',
    '  hiddenRepositories:',
    '    - octo/widgets',
    '',
    'gui:',
    '  lineNumbers: relative  # easier to jump',
  ));
  const twice = listEdited(once, KEY, { add: ['octo/*'] });
  assert.equal(twice, once.replace('    - octo/widgets\n', '    - octo/widgets\n    - octo/*\n'), 'the next entry goes after the last');
  assert.equal(listEdited(twice, KEY, { add: ['octo/widgets'] }), twice, 'an entry the list holds already is not added twice');
});

test('taking an entry off removes its line alone, and the last one leaves the list empty', () => {
  const both = listEdited(listEdited(WRITTEN, KEY, { add: ['octo/widgets'] }), KEY, { add: ['octo/*'] });
  const one = listEdited(both, KEY, { remove: ['octo/widgets'] });
  assert.equal(one, both.replace('    - octo/widgets\n', ''));
  const none = listEdited(one, KEY, { remove: ['octo/*'] });
  assert.equal(none, one.replace('    - octo/*\n', ''));
  assert.match(none, /^ {2}hiddenRepositories:\n\ngui:/m, 'the list stays, empty, which reads as none');
  assert.equal(listEdited(WRITTEN, KEY, { remove: ['octo/*'] }), WRITTEN, 'taking off what is not there changes nothing');
});

test('a list already written keeps its own style: its indent, its comments and commented-out entries, or its brackets', () => {
  assert.equal(
    listEdited(lines('github:', '  hiddenRepositories:', '  - a/b   # noisy', '  # - c/d', '  - e/*', 'gui: {}'), KEY, { add: ['x/y'], remove: ['a/b'] }),
    lines('github:', '  hiddenRepositories:', '  # - c/d', '  - e/*', '  - x/y', 'gui: {}'),
  );
  assert.equal(
    listEdited(lines('github:', '  hiddenRepositories:', '    # - old/one'), KEY, { add: ['x/y'] }),
    lines('github:', '  hiddenRepositories:', '    - x/y', '    # - old/one'),
  );
  assert.equal(listEdited(lines('github:', "  hiddenRepositories: [a/b, 'c/*']  # mine"), KEY, { add: ['x/y'] }), lines('github:', "  hiddenRepositories: [a/b, 'c/*', x/y]  # mine"));
  assert.equal(listEdited(lines('github:', "  hiddenRepositories: [a/b, 'c/*']  # mine"), KEY, { remove: ['c/*'] }), lines('github:', '  hiddenRepositories: [a/b]  # mine'));
  assert.equal(listEdited(lines('github:', '  hiddenRepositories: []'), KEY, { add: ['x/y'] }), lines('github:', '  hiddenRepositories: [x/y]'));
  assert.equal(listEdited(lines('github:', '    accounts: []'), KEY, { add: ['x/y'] }), lines('github:', '    accounts: []', '    hiddenRepositories:', '        - x/y'), 'a file indented by four is written by four');
});

test('with no section, or no file, the list is written at the end, holding only the entry', () => {
  assert.equal(listEdited('', KEY, { add: ['octo/*'] }), lines('github:', '  hiddenRepositories:', '    - octo/*'));
  assert.equal(listEdited(lines('server:', '  port: 4800'), KEY, { add: ['octo/*'] }), lines('server:', '  port: 4800', '', 'github:', '  hiddenRepositories:', '    - octo/*'));
  assert.equal(listEdited('server:\n  port: 4800', KEY, { add: ['octo/*'] }), lines('server:', '  port: 4800', '', 'github:', '  hiddenRepositories:', '    - octo/*'), 'a last line with no line break gets one');
  assert.equal(listEdited(lines('github:'), KEY, { add: ['octo/*'] }), lines('github:', '  hiddenRepositories:', '    - octo/*'));
  assert.equal(listEdited('a: 1\r\n', KEY, { add: ['x/y'] }), 'a: 1\r\n\r\ngithub:\r\n  hiddenRepositories:\r\n    - x/y\r\n', 'Windows line breaks stay');
});

test('a list or a section written so that lines cannot change it safely is refused, as is text that is not YAML', () => {
  assert.throws(() => listEdited(lines('github: {accounts: []}'), KEY, { add: ['x/y'] }), /^Error: github is written in a way lazyreview cannot change line by line$/);
  assert.throws(() => listEdited(lines('github:', '  hiddenRepositories: [', '    a/b]'), KEY, { add: ['x/y'] }), /^Error: github\.hiddenRepositories is written in a way lazyreview cannot change line by line$/);
  assert.throws(() => listEdited(lines('github:', '  hiddenRepositories: octo/*'), KEY, { add: ['x/y'] }), /cannot change line by line/);
  assert.throws(() => listEdited(lines('github:', '  hiddenRepositories:', '    - a/b', '      c/d'), KEY, { add: ['x/y'] }), /cannot change line by line/, 'an item over two lines is refused');
  assert.throws(() => listEdited('github: [', KEY, { add: ['x/y'] }), /\(\d+:\d+\)$/);
});
