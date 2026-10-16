/**
 * A stack's pull requests on the page: each one's colour, its pill, the
 * filter to one of them, and where a comment on each line goes.
 *
 * createPulls(ctx) reads the stack from ctx.data and ctx.state, and returns
 * the helpers every view shares. On a single pull request's review there is
 * no stack, and every helper answers as if no line had an owner.
 */
import { esc } from '../lib/text.mjs';

export function createPulls(ctx) {
  const stack = ctx.data.stack;
  const byPull = {};
  (ctx.state.members || []).concat(ctx.state.earlier || []).forEach(m => { byPull[m.pull] = m; });
  const members = stack ? stack.members.map((m, i) => ({ ...byPull[m.pull], ...m, slot: (byPull[m.pull] && byPull[m.pull].color) || i % 8 + 1 })) : [];
  const memberOf = {};
  members.forEach(m => { memberOf[m.pull] = m; });

  function pullColor(pull) {
    return memberOf[pull] ? 'var(--p' + memberOf[pull].slot + ')' : 'var(--muted)';
  }
  function pullPill(pull, href) {
    const style = ' style="--pc:' + pullColor(pull) + '"';
    return href
      ? '<a class="pp" href="' + esc(href) + '" target="_blank" rel="noopener"' + style + '>#' + esc(pull) + '</a>'
      : '<span class="pp"' + style + '>#' + esc(pull) + '</span>';
  }
  function pullPills(pulls) {
    return pulls && pulls.length ? '<span class="pps">' + pulls.map(p => pullPill(p)).join('') + '</span>' : '';
  }
  function onFilter(pulls) {
    return !ctx.filter.pull || (pulls || []).indexOf(ctx.filter.pull) >= 0;
  }
  function lineTarget(path, side, n) {
    const e = stack && stack.lines[path] && stack.lines[path][side] && stack.lines[path][side][n];
    if (!e) return null;
    if (Array.isArray(e)) return { pull: e[0], line: e[1], path: e[2] || path };
    if (e.c) return { candidates: e.c.map(c => ({ pull: c[0], line: c[1], path: c[2] || path })) };
    return { none: e.x };
  }
  function targetWords(t, changed, path) {
    if (!t) return '';
    const at = x => '#' + x.pull + ', line ' + x.line + (x.path !== path ? ' of ' + x.path : '');
    if (t.pull) return (changed ? 'Changed in #' + t.pull + '. ' : 'Unchanged here; #' + t.pull + ' last changed it. ') + 'A comment here goes on ' + at(t) + '.';
    if (t.none) return 'No pull request in the stack can take a comment here: ' + t.none + '.';
    if (!t.candidates.length) return 'Unchanged, and no pull request in the stack shows this line, so it takes no comment.';
    if (t.candidates.length === 1) return 'Unchanged. A comment here goes on ' + at(t.candidates[0]) + '.';
    return 'Unchanged, and shown by ' + t.candidates.map(at).join(' and ') + '. A comment here names which one it is for.';
  }
  function rowMark(path, r) {
    if (!stack || !path || r.t === 'meta') return null;
    const changed = r.t !== 'ctx';
    const t = r.t === 'del' ? lineTarget(path, 'LEFT', r.o) : lineTarget(path, 'RIGHT', r.n);
    if (!t) return null;
    const owner = changed && t.pull ? t.pull : null;
    return { owner, title: ' title="' + esc(targetWords(t, changed, path)) + '"', other: !!(changed && ctx.filter.pull && owner !== ctx.filter.pull) };
  }

  return { stack, members, memberOf, pullColor, pullPill, pullPills, onFilter, rowMark };
}
