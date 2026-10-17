/**
 * Keys as the config writes them, as lazygit writes its own: a character,
 * or a named key such as <space>, <enter>, <esc>, <tab>, <up>, <down>,
 * <left>, <right> or <c-x> for Ctrl-x. A sequence is its keys in a row,
 * such as gg, ]c or <space>v, and <disabled> alone is no key.
 *
 * readerName(label) returns the sequence as the terminal app's key reader
 * names it, its keys joined by spaces, such as "g g", "space v" or
 * "ctrl+o", or "" for <disabled>. A terminal sends Tab for Ctrl-i, so
 * <c-i> is "tab". labelProblem(label) says what is wrong with a label, in
 * the config's words, or returns null.
 */
import { english } from '../i18n/english.mjs';

const words = english.config.keys;
const NAMED = ['space', 'enter', 'esc', 'tab', 'up', 'down', 'left', 'right'];
export const DISABLED = '<disabled>';

export function readerName(label) {
  if (label === DISABLED) return '';
  return tokens(label).map(token => {
    const named = /^<(.+)>$/.exec(token);
    if (!named) return token;
    const ctrl = /^c-([a-z])$/.exec(named[1]);
    if (ctrl) return ctrl[1] === 'i' ? 'tab' : `ctrl+${ctrl[1]}`;
    return named[1];
  }).join(' ');
}

export function labelProblem(label) {
  if (typeof label !== 'string') return words.notText;
  if (label === DISABLED) return null;
  if (!label) return words.empty;
  for (const token of tokens(label)) {
    if (token === '<') return words.unclosed;
    const named = /^<(.+)>$/.exec(token);
    if (!named) continue;
    if (named[1] === 'disabled') return words.disabledAlone;
    if (!NAMED.includes(named[1]) && !/^c-[a-z]$/.test(named[1])) return words.unknownKey(token);
  }
  return null;
}

// A label's keys: each <name> whole, every other character alone.
function tokens(label) {
  const out = [];
  for (let i = 0; i < label.length;) {
    const close = label[i] === '<' ? label.indexOf('>', i) : -1;
    if (label[i] === '<' && close > i + 1) {
      out.push(label.slice(i, close + 1));
      i = close + 1;
    } else {
      const ch = String.fromCodePoint(label.codePointAt(i));
      out.push(ch);
      i += ch.length;
    }
  }
  return out;
}
