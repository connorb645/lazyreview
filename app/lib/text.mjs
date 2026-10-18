/**
 * Small text helpers for building the app's HTML.
 */

/** Text made safe to put in HTML, inside an element or an attribute. */
export function esc(s) {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

export function fmt(n) {
  return Number(n).toLocaleString('en-GB');
}

export function baseName(path) {
  return path.split('/').pop();
}

export function short(sha) {
  return String(sha).slice(0, 8);
}

/**
 * Escaped text with a break opportunity after each slash, so a path wraps
 * between its folders and never splits a name into what looks like two
 * words. A name too long for a whole line is left to the style's
 * overflow-wrap.
 */
export function breakable(text) {
  return esc(text).replace(/\//g, '/<wbr>');
}

/** Markdown read as plain words, for a short preview. */
export function plainText(markdown) {
  return String(markdown).replace(/```[\s\S]*?```/g, ' ').replace(/\[([^\]]*)\]\([^)]*\)/g, '$1').replace(/^\s*[-+]\s+/gm, '')
    .replace(/[`*_>#]/g, '').replace(/\s+/g, ' ').trim();
}

/** A time such as 15 Oct, 19:32. */
export function when(iso) {
  return new Date(iso).toLocaleString('en-GB', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' });
}
