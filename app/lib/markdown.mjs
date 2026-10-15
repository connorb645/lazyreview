/**
 * Markdown for the app, on top of marked 12.0.2, loaded as a global first.
 *
 * safeMarkdown(text) renders text the app did not write itself, such as an
 * answer from Claude: raw HTML in it shows as text, an image shows as its
 * description and address and loads nothing, and a link keeps only a web,
 * mail, anchor or relative address. trustedMarkdown(text) renders a written
 * review, which may hold inline SVG for its diagrams.
 */
export function escapeHTML(text) {
  return String(text)
    .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;').replace(/'/g, '&#39;');
}

function allowedHref(href) {
  const scheme = /^\s*([a-z][a-z0-9+.-]*):/i.exec(href || '');
  return !scheme || /^(https?|mailto)$/i.test(scheme[1]);
}

export const safe = new window.marked.Marked({
  gfm: true,
  renderer: {
    html(html) { return escapeHTML(html); },
    link(href, title, text) { return allowedHref(href) ? false : text; },
    image(href, title, text) {
      return '<span class="mdx-image">[image: ' + text + (href ? ' - ' + escapeHTML(href) : '') + ']</span>';
    },
  },
});

const trusted = new window.marked.Marked({ gfm: true });

export function safeMarkdown(text) {
  return safe.parse(text || '');
}

export function trustedMarkdown(text) {
  return trusted.parse(text || '');
}
