/**
 * The whole text of files, as lines, for the page's Whole file, unchanged
 * lines and Rendered views.
 *
 * changedTexts(files) gives each changed or renamed file's new text, leaving
 * out any file too large to carry, or that would take the page past its
 * budget, and returns {texts, left} with the paths it left out. Added files
 * need none: the diff holds their whole text.
 */
const FILE_LIMIT = 1_500_000;
const TOTAL_LIMIT = 8_000_000;

export function changedTexts(files) {
  const texts = {};
  const left = [];
  let total = 0;
  for (const f of files) {
    if ((f.status !== 'M' && f.status !== 'R') || f.binary || f.text == null) continue;
    if (f.text.length > FILE_LIMIT || total + f.text.length > TOTAL_LIMIT) { left.push(f.path); continue; }
    texts[f.path] = lines(f.text);
    total += f.text.length;
  }
  return { texts, left };
}

/** A text as lines, without the empty line its final newline would leave. */
export function lines(text) {
  const out = text.split('\n');
  if (out.length && out[out.length - 1] === '') out.pop();
  return out;
}
