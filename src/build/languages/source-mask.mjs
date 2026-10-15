/**
 * Blanks the comments and the insides of string literals in source text.
 *
 * Every blanked character becomes a space and every newline stays, so offsets
 * and line numbers in the result match the original exactly. String
 * delimiters stay, so a pattern can still find where a literal sits and read
 * its contents from the original text.
 *
 * A syntax describes one language:
 *   line:          markers that start a comment running to the end of the line
 *   block:         [open, close] pairs for block comments
 *   nestedBlocks:  true when block comments nest, as in Swift and Kotlin
 *   strings:       [{open, close, escape, multiline, interpolation}], longest
 *                  opener first; interpolation is '\\(' or '${'
 *   rawHashes:     true for Swift raw strings, #"..."#
 *   regex:         true when a slash can start a regular expression literal
 *   stringPrefix:  letters that may lead a string, such as Python's rb or f
 */
export function maskSource(text, syntax) {
  const n = text.length;
  const out = text.split('');

  const at = (token, k) => text.startsWith(token, k);

  function blank(from, to) {
    for (let k = from; k < to; k++) if (out[k] !== '\n') out[k] = ' ';
  }

  function lineEnd(k) {
    while (k < n && text[k] !== '\n') k++;
    return k;
  }

  function blockEnd(k, open, close) {
    let depth = 1;
    k += open.length;
    while (k < n) {
      if (syntax.nestedBlocks && at(open, k)) { depth++; k += open.length; continue; }
      if (at(close, k)) {
        k += close.length;
        if (--depth === 0) return k;
        continue;
      }
      k++;
    }
    return n;
  }

  // Code inside an interpolation runs to the bracket that closes it. Strings
  // and comments inside it are skipped whole, so their brackets never count.
  function codeEnd(k, open, close) {
    let depth = 1;
    while (k < n) {
      const literal = literalAt(k);
      if (literal) { k = literal.end; continue; }
      if (at(open, k)) depth++;
      else if (at(close, k) && --depth === 0) return k + 1;
      k++;
    }
    return n;
  }

  function stringAt(k, spec, hashes, lead) {
    const innerStart = k + lead + hashes + spec.open.length;
    const close = spec.close + '#'.repeat(hashes);
    const escape = hashes ? '\\' + '#'.repeat(hashes) : spec.escape;
    let j = innerStart;
    while (j < n) {
      if (!spec.multiline && text[j] === '\n') return { start: k, end: j, inner: [innerStart, j] };
      if (escape && at(escape, j)) {
        const next = j + escape.length;
        if (spec.interpolation === '\\(' && text[next] === '(') { j = codeEnd(next + 1, '(', ')'); continue; }
        j = next + 1;
        continue;
      }
      if (spec.interpolation === '${' && at('${', j)) { j = codeEnd(j + 2, '{', '}'); continue; }
      if (at(close, j)) return { start: k, end: j + close.length, inner: [innerStart, j] };
      j++;
    }
    return { start: k, end: n, inner: [innerStart, n] };
  }

  function regexAllowedBefore(k) {
    let j = k - 1;
    while (j >= 0 && (text[j] === ' ' || text[j] === '\t')) j--;
    if (j < 0) return true;
    if ('(,=:[!&|?{};+-*%<>~^\n'.includes(text[j])) return true;
    return /(?:^|[^\w$])(?:return|typeof|case|do|else|in|of|new|delete|void|throw|yield|await)$/.test(text.slice(Math.max(0, j - 9), j + 1));
  }

  function regexAt(k) {
    let j = k + 1;
    let inClass = false;
    while (j < n && text[j] !== '\n') {
      const c = text[j];
      if (c === '\\') { j += 2; continue; }
      if (c === '[') inClass = true;
      else if (c === ']') inClass = false;
      else if (c === '/' && !inClass) return { start: k, end: j + 1, inner: [k + 1, j] };
      j++;
    }
    return null;
  }

  // A comment or literal starting at k: {start, end, inner} where inner is the
  // range to blank inside a literal, or {start, end, comment: true}.
  function literalAt(k) {
    for (const marker of syntax.line || []) {
      if (at(marker, k)) return { start: k, end: lineEnd(k), comment: true };
    }
    for (const [open, close] of syntax.block || []) {
      if (at(open, k)) return { start: k, end: blockEnd(k, open, close), comment: true };
    }
    let hashes = 0;
    if (syntax.rawHashes) while (text[k + hashes] === '#') hashes++;
    let lead = 0;
    if (syntax.stringPrefix && !hashes && (k === 0 || !/\w/.test(text[k - 1]))) {
      const m = syntax.stringPrefix.exec(text.slice(k, k + 3));
      if (m) lead = m[0].length;
    }
    for (const spec of syntax.strings || []) {
      if (at(spec.open, k + lead + hashes)) return stringAt(k, spec, hashes, lead);
    }
    if (syntax.regex && text[k] === '/' && text[k + 1] !== '/' && text[k + 1] !== '*' && regexAllowedBefore(k)) {
      return regexAt(k);
    }
    return null;
  }

  let k = 0;
  while (k < n) {
    const literal = literalAt(k);
    if (!literal) { k++; continue; }
    if (literal.comment) blank(literal.start, literal.end);
    else blank(literal.inner[0], literal.inner[1]);
    k = Math.max(literal.end, k + 1);
  }
  return out.join('');
}

export const SWIFT_SYNTAX = {
  line: ['//'],
  block: [['/*', '*/']],
  nestedBlocks: true,
  rawHashes: true,
  strings: [
    { open: '"""', close: '"""', escape: '\\', multiline: true, interpolation: '\\(' },
    { open: '"', close: '"', escape: '\\', multiline: false, interpolation: '\\(' },
  ],
};

export const KOTLIN_SYNTAX = {
  line: ['//'],
  block: [['/*', '*/']],
  nestedBlocks: true,
  strings: [
    { open: '"""', close: '"""', escape: null, multiline: true, interpolation: '${' },
    { open: '"', close: '"', escape: '\\', multiline: false, interpolation: '${' },
    { open: "'", close: "'", escape: '\\', multiline: false },
  ],
};

export const JAVA_SYNTAX = {
  line: ['//'],
  block: [['/*', '*/']],
  strings: [
    { open: '"""', close: '"""', escape: '\\', multiline: true },
    { open: '"', close: '"', escape: '\\', multiline: false },
    { open: "'", close: "'", escape: '\\', multiline: false },
  ],
};

export const TYPESCRIPT_SYNTAX = {
  line: ['//'],
  block: [['/*', '*/']],
  regex: true,
  strings: [
    { open: '`', close: '`', escape: '\\', multiline: true, interpolation: '${' },
    { open: '"', close: '"', escape: '\\', multiline: false },
    { open: "'", close: "'", escape: '\\', multiline: false },
  ],
};

export const PYTHON_SYNTAX = {
  line: ['#'],
  stringPrefix: /^[rRbBuUfF]{1,2}(?=["'])/,
  strings: [
    { open: '"""', close: '"""', escape: '\\', multiline: true },
    { open: "'''", close: "'''", escape: '\\', multiline: true },
    { open: '"', close: '"', escape: '\\', multiline: false },
    { open: "'", close: "'", escape: '\\', multiline: false },
  ],
};
