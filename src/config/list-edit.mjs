/**
 * Changes one list setting in config.yml's text in place, as a person would
 * edit it, so every other line stays as it was: comments, blank lines,
 * commented-out settings, the order and every value.
 *
 * listEdited(text, key, {add, remove}) returns the text with each item of
 * `add` the list lacks put at its end, and each item equal to one of
 * `remove` taken out. `key` names a section's list, such as
 * "github.hiddenRepositories". A list the text lacks is written at the end of
 * its section, and a section it lacks at the end of the text, so empty text
 * gives a file holding only the list. Items keep the list's own style: one a
 * line, or all on one line in brackets. It throws an error saying why when
 * the text is not YAML, or when the list or its section is written in a way
 * it cannot change line by line, such as brackets over several lines; and it
 * reads what it wrote, throwing unless that holds the list asked for and
 * every other value as it was.
 */
import { isDeepStrictEqual } from 'node:util';
import { english } from '../i18n/english.mjs';
import { parseYAML } from './yaml.mjs';

const words = english.config.edit;
const PLAIN = /^[A-Za-z0-9][A-Za-z0-9._/*-]*$/;
const SPECIAL = /^(true|false|null|~|[0-9][0-9._]*)$/i;

export function listEdited(text, key, { add = [], remove = [] } = {}) {
  const [section, name] = key.split('.');
  const cannot = () => new Error(words.cannotChange(key));
  const before = parseYAML(text) ?? {};
  const was = listIn(before, section, name, cannot);
  const wanted = [...was.filter(item => !remove.includes(item)), ...[...new Set(add)].filter(item => !was.includes(item))];
  if (isDeepStrictEqual(was, wanted)) return text;
  const lines = text.split('\n');
  const ending = text.includes('\r\n') ? '\r' : '';
  const step = indentUnit(lines);
  const header = lines.findIndex(line => new RegExp(`^${section}\\s*:(\\s|$)`).test(bare(line)));
  if (header < 0) {
    appendSection(lines, [`${section}:`, `${' '.repeat(step)}${name}:`, ...wanted.map(item => `${' '.repeat(step * 2)}- ${plain(item)}`)], ending);
  } else {
    if (valueAfterColon(bare(lines[header]))) throw new Error(words.cannotChange(section));
    const end = sectionEnd(lines, header);
    const first = lines.slice(header + 1, end).find(isContent);
    const child = first === undefined ? step : indentOf(first);
    const at = lines.findIndex((line, i) => i > header && i < end && isContent(line) && indentOf(line) === child && new RegExp(`^\\s*${name}\\s*:(\\s|$)`).test(bare(line)));
    if (at < 0) {
      const last = lines.findLastIndex((line, i) => i > header && i < end && !isBlank(line) && indentOf(line) > 0);
      const after = last < 0 ? header : last;
      lines.splice(after + 1, 0, ...[`${' '.repeat(child)}${name}:`, ...wanted.map(item => `${' '.repeat(child + step)}- ${plain(item)}`)].map(line => line + ending));
    } else {
      const value = valueAfterColon(bare(lines[at]));
      if (value.startsWith('[')) lines[at] = flowEdited(lines[at], wanted, remove, cannot);
      else if (value) throw cannot();
      else blockEdited(lines, at, child, step, wanted, remove, ending, cannot);
    }
  }
  const out = lines.join('\n');
  let after;
  try {
    after = parseYAML(out) ?? {};
  } catch {
    throw cannot();
  }
  if (!isDeepStrictEqual(listIn(after, section, name, cannot), wanted) || !isDeepStrictEqual(without(before, section, name), without(after, section, name))) throw cannot();
  return out;
}

function listIn(value, section, name, cannot) {
  if (!isMap(value)) throw cannot();
  const held = value[section];
  if (held === undefined || held === null) return [];
  if (!isMap(held)) throw cannot();
  const list = held[name];
  if (list === undefined || list === null) return [];
  if (!Array.isArray(list)) throw cannot();
  return list;
}

function without(value, section, name) {
  const copy = structuredClone(value);
  if (isMap(copy[section])) delete copy[section][name];
  if (copy[section] === null || (isMap(copy[section]) && !Object.keys(copy[section]).length)) delete copy[section];
  return copy;
}

function sectionEnd(lines, header) {
  const next = lines.findIndex((line, i) => i > header && isContent(line) && indentOf(line) === 0);
  return next < 0 ? lines.length : next;
}

function blockEdited(lines, at, child, step, wanted, remove, ending, cannot) {
  let end = lines.findIndex((line, i) => i > at && isContent(line) && (indentOf(line) < child || (indentOf(line) === child && !isItem(line))));
  if (end < 0) end = lines.length;
  const items = [];
  for (let i = at + 1; i < end; i++) {
    if (!isContent(lines[i])) continue;
    if (!isItem(lines[i]) || (items.length && indentOf(lines[i]) !== indentOf(lines[items[0]]))) throw cannot();
    items.push(i);
  }
  const indent = items.length ? indentOf(lines[items[0]]) : child + step;
  const value = i => parseYAML(bare(lines[i]).trimStart().slice(1));
  const removed = new Set(items.filter(i => remove.includes(value(i))));
  const kept = items.filter(i => !removed.has(i));
  const held = new Set(kept.map(value));
  const added = wanted.filter(item => !held.has(item)).map(item => `${' '.repeat(indent)}- ${plain(item)}${ending}`);
  const last = kept.length ? kept.at(-1) : at;
  const out = [];
  lines.forEach((line, i) => {
    if (!removed.has(i)) out.push(line);
    if (i === last) out.push(...added);
  });
  lines.splice(0, lines.length, ...out);
}

function flowEdited(line, wanted, remove, cannot) {
  const text = bare(line);
  const open = text.indexOf('[', text.indexOf(':'));
  const close = closingBracket(text, open);
  if (close < 0 || valueAfterColon(`:${text.slice(close + 1)}`)) throw cannot();
  const raw = splitItems(text.slice(open + 1, close)).filter(item => !remove.includes(parseYAML(item)));
  const held = new Set(raw.map(item => parseYAML(item)));
  const items = [...raw, ...wanted.filter(item => !held.has(item)).map(plain)];
  return `${text.slice(0, open)}[${items.join(', ')}]${text.slice(close + 1)}${line.endsWith('\r') ? '\r' : ''}`;
}

function closingBracket(text, open) {
  let quote = null;
  for (let i = open + 1; i < text.length; i++) {
    const ch = text[i];
    if (quote === '"' && ch === '\\') i++;
    else if (quote && ch === quote) quote = null;
    else if (!quote && (ch === '"' || ch === '\'')) quote = ch;
    else if (!quote && (ch === '[' || ch === '{')) return -1;
    else if (!quote && ch === ']') return i;
  }
  return -1;
}

function splitItems(inside) {
  const items = [];
  let quote = null;
  let from = 0;
  for (let i = 0; i <= inside.length; i++) {
    const ch = inside[i];
    if (quote === '"' && ch === '\\') i++;
    else if (quote && ch === quote) quote = null;
    else if (!quote && (ch === '"' || ch === '\'')) quote = ch;
    else if (!quote && (ch === ',' || i === inside.length)) {
      items.push(inside.slice(from, i).trim());
      from = i + 1;
    }
  }
  return items.filter(Boolean);
}

function appendSection(lines, block, ending) {
  if (lines.at(-1) !== '') {
    lines[lines.length - 1] += ending;
    lines.push('');
  }
  const spaced = lines.length === 1 || isBlank(lines.at(-2)) ? [] : [ending];
  lines.splice(lines.length - 1, 0, ...spaced, ...block.map(line => line + ending));
}

function indentUnit(lines) {
  const indents = lines.filter(isContent).map(indentOf).filter(n => n > 0);
  return indents.length ? Math.min(...indents) : 2;
}

function valueAfterColon(text) {
  const rest = text.slice(text.indexOf(':') + 1).trim();
  return rest.startsWith('#') ? '' : rest;
}

function plain(item) {
  return PLAIN.test(item) && !SPECIAL.test(item) ? item : `'${item.replace(/'/g, '\'\'')}'`;
}

const bare = line => line.replace(/\r$/, '');
const indentOf = line => /^ */.exec(line)[0].length;
const isBlank = line => !bare(line).trim();
const isComment = line => bare(line).trimStart().startsWith('#');
const isContent = line => !isBlank(line) && !isComment(line);
const isItem = line => /^-(\s|$)/.test(bare(line).trimStart());
const isMap = value => !!value && typeof value === 'object' && !Array.isArray(value);
