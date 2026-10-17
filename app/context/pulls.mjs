/**
 * A stack's pull requests on the page: each one's colour and the filter to
 * one of them.
 *
 * createPulls(ctx) reads the stack from ctx.data and ctx.state, and returns
 * what every view shares. On a single pull request's review there is no
 * stack, and no pull request has a colour of its own. ownLines(f) are the
 * [added, removed] lines of a file the filter shows: all of them, or those
 * its pull request changed. componentTotals() are each component's files
 * and lines the filter shows; an unchanged component keeps its own count.
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
  function ownLines(f) {
    if (!ctx.filter.pull) return [f.adds, f.dels];
    return (f.byPull && f.byPull[ctx.filter.pull]) || [0, 0];
  }
  function componentTotals() {
    const { model, compTotals } = ctx.data;
    if (!ctx.filter.pull) return compTotals;
    const totals = {};
    model.components.forEach(c => {
      totals[c.id] = c.context ? compTotals[c.id] : { files: 0, adds: 0, dels: 0 };
    });
    model.files.filter(f => onFilter(f.pulls)).forEach(f => {
      const t = totals[f.component], own = ownLines(f);
      if (!t) return;
      t.files++; t.adds += own[0]; t.dels += own[1];
    });
    return totals;
  }

  return { stack, members, memberOf, pullColor, onFilter, ownLines, componentTotals };
}
