/**
 * A stack's pull requests on the page: each one's colour and the filter to
 * one of them.
 *
 * createPulls(ctx) reads the stack from ctx.data and ctx.state, and returns
 * what every view shares. On a single pull request's review there is no
 * stack, and no pull request has a colour of its own.
 */
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
  function onFilter(pulls) {
    return !ctx.filter.pull || (pulls || []).indexOf(ctx.filter.pull) >= 0;
  }

  return { stack, members, memberOf, pullColor, onFilter };
}
