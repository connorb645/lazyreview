/**
 * A stack's pull requests on the page: each one's colour, its pill and the
 * filter to one of them.
 *
 * createPulls(ctx) reads the stack from ctx.data and ctx.state, and returns
 * the helpers every view shares. On a single pull request's review there is
 * no stack, and no pull request has a colour of its own.
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

  return { stack, members, memberOf, pullColor, pullPill, pullPills, onFilter };
}
