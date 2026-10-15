/**
 * Small pure helpers for reading the text of a type declaration's header.
 */

const OPENERS = { '<': '>', '(': ')', '[': ']', '{': '}' };

/** The offset just past the bracket that closes the one at `k`. */
export function skipBalanced(text, k) {
  const open = text[k];
  const close = OPENERS[open];
  let depth = 0;
  for (let j = k; j < text.length; j++) {
    const c = text[j];
    if (c === open) depth++;
    else if (c === close) {
      // `->` inside angle brackets is an arrow, not a closing bracket
      if (open === '<' && text[j - 1] === '-') continue;
      if (--depth === 0) return j + 1;
    }
  }
  return text.length;
}

/** Splits on `separator` wherever no bracket is open. */
export function splitTopLevel(text, separator = ',') {
  const parts = [];
  let depth = 0;
  let from = 0;
  for (let j = 0; j < text.length; j++) {
    const c = text[j];
    if (c === '<' || c === '(' || c === '[' || c === '{') depth++;
    else if ((c === '>' && text[j - 1] !== '-') || c === ')' || c === ']' || c === '}') depth = Math.max(0, depth - 1);
    else if (c === separator && depth === 0) {
      parts.push(text.slice(from, j));
      from = j + 1;
    }
  }
  parts.push(text.slice(from));
  return parts.map(p => p.trim()).filter(Boolean);
}

/**
 * The bare type name a header entry refers to: `@unchecked Sendable` is
 * Sendable, `Base<T>(args)` is Base, `Foundation.NSObject` is NSObject.
 */
export function bareTypeName(entry) {
  let text = entry
    .replace(/@[\w.]+(?:\([^()]*\))?/g, ' ')
    .replace(/\b(?:any|some|out|in|public|private|protected|internal|final|abstract)\b/g, ' ')
    .replace(/[~?!]/g, ' ')
    .trim();
  const cut = text.search(/[<(\[\s{]/);
  if (cut >= 0) text = text.slice(0, cut);
  const dotted = text.split('.').filter(Boolean);
  const name = dotted[dotted.length - 1] || '';
  return /^[A-Za-z_$][\w$]*$/.test(name) ? name : '';
}

/** Every identifier in a piece of code, once each. */
export function identifiers(text) {
  return new Set(text.match(/[A-Za-z_$][\w$]*/g) || []);
}
