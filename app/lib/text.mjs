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

/** Escaped text with break opportunities after slashes and inside long names, so a path never breaks mid-word. */
export function breakable(text) {
  return esc(text).replace(/([/+_-])/g, '$1<wbr>').replace(/([a-z])([A-Z])/g, '$1<wbr>$2');
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
