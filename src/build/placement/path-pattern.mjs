/**
 * A path pattern from the config: `**` matches across folders, `*` within
 * one, `?` one character. A pattern with no wildcard matches that path and
 * everything under it, and one ending in `/` matches everything under it.
 *
 * pathPattern(text) returns {text, test(path), specificity}: the more literal
 * characters a pattern holds, the more specific it is, and an exact path is
 * the most specific of all.
 */
export function pathPattern(text) {
  const literal = !/[*?]/.test(text);
  const folder = text.endsWith('/') ? text : `${text}/`;
  if (literal) {
    const exact = text.replace(/\/$/, '');
    return {
      text,
      test: path => path === exact || path.startsWith(folder),
      specificity: text.length + (text.endsWith('/') ? 0 : 0.5),
    };
  }
  let source = '';
  for (let i = 0; i < text.length; i++) {
    const c = text[i];
    if (c === '*' && text[i + 1] === '*') {
      if (text[i + 2] === '/') { source += '(?:.*/)?'; i += 2; } else { source += '.*'; i += 1; }
    } else if (c === '*') source += '[^/]*';
    else if (c === '?') source += '[^/]';
    else source += c.replace(/[.+^${}()|[\]\\]/g, '\\$&');
  }
  if (text.endsWith('/')) source += '.*';
  const pattern = new RegExp(`^${source}$`);
  return { text, test: path => pattern.test(path), specificity: text.replace(/[*?]/g, '').length };
}
