/**
 * The `file:line` links of a written review's section: each piece of inline
 * code such as `Basket.swift:12` or `src/log.ts:31-40` that names a file of
 * the review, where exactly one of its paths ends with that name.
 *
 * writtenLinks(markdown, paths) is pure. It returns [{text, path, line}],
 * one for each distinct text, in the order they first appear; code inside a
 * code block is no link.
 */
import { createRequire } from 'node:module';

const { Marked } = createRequire(import.meta.url)('../../vendor/marked-12.0.2.min.js');
const lexer = new Marked({ gfm: true });
const LOCATION = /^(.+?\.[A-Za-z0-9]+):(\d+)(?:[-–](\d+))?$/;

export function writtenLinks(markdown, paths) {
  const links = new Map();
  lexer.walkTokens(lexer.lexer(markdown || ''), token => {
    if (token.type !== 'codespan') return;
    const text = token.raw.replace(/^`+|`+$/g, '').trim();
    const m = LOCATION.exec(text);
    if (!m || links.has(text)) return;
    const hits = paths.filter(p => p === m[1] || p.endsWith(`/${m[1]}`));
    if (hits.length === 1) links.set(text, { text, path: hits[0], line: Number(m[2]) });
  });
  return [...links.values()];
}
