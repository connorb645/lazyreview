/**
 * One review in the app: the bar, the pull request strip of a stack, and the
 * three views, Review, Diagram and a component's diff, moved between by the
 * address's #fragment so the browser's back button walks them.
 *
 * openReview(id) loads the review's state and page data from the server,
 * draws every view and shows the one the address names. It opens on the
 * diagram, or on the Review when a written review is attached.
 */
import { getJSON, reviewPath } from '../lib/api.mjs';
import { esc, fmt, plural } from '../lib/text.mjs';
import { indexPage } from './review-data.mjs';
import { createPulls } from './pulls.mjs';
import { createDiagram } from './diagram.mjs';
import { createDiffs } from './diffs.mjs';
import { createLineNotes } from './line-notes.mjs';
import { createQueuePanel } from './queue-panel.mjs';
import { createOverview } from './overview.mjs';
import { createRefresh } from './refresh.mjs';

export async function openReview(id) {
  const [state, page] = await Promise.all([getJSON(reviewPath(id)), getJSON(reviewPath(id, '/page'))]);
  const ctx = { state, data: indexPage(page), filter: { pull: null } };
  const { model, comps, files, totals } = ctx.data;
  ctx.pulls = createPulls(ctx);
  const pulls = ctx.pulls;

  const reviewView = document.getElementById('review-view');
  const diagramView = document.getElementById('diagram-view');
  const diffView = document.getElementById('diff-view');
  const back = document.getElementById('back');
  const viewsNav = document.getElementById('views');
  const totalsEl = document.getElementById('totals');

  document.title = model.heading.title + ' - Review Desk';
  document.getElementById('page-title').textContent = model.heading.title;
  document.getElementById('page-title').title = model.heading.title;
  document.getElementById('context').innerHTML = model.heading.context.map(part => (part[1] ? '<b>' + esc(part[0]) + '</b>' : esc(part[0]))).join('');
  const totalsHTML = t => plural(t.files, 'file') + ' &nbsp;<span class="plus">+' + fmt(t.adds) + '</span> <span class="minus">&minus;' + fmt(t.dels) + '</span>';
  const combinedTotals = totalsHTML(totals);
  totalsEl.innerHTML = combinedTotals;
  viewsNav.hidden = false;

  let pending = null;
  let fitted = false;
  let savedView = null;
  let filterMoved = false;
  const home = state.written ? 'review' : 'map';
  let from = home;

  function go(hash) {
    if (location.hash === hash) show(hash.slice(1));
    else location.hash = hash;
  }
  function openComponent(cid, opts) {
    if (!diagramView.hidden) savedView = ctx.diagram.saveView();
    if (ctx.filter.pull && opts && opts.file && files[opts.file] && !pulls.onFilter(files[opts.file].pulls)) setPullFilter(null, true);
    pending = opts || null;
    ctx.diagram.clearHover();
    go('#diff/' + encodeURIComponent(cid));
  }
  function openNode(nodeId) {
    const n = ctx.data.nodes[nodeId];
    if (n.state === 'context') { openComponent(n.component, { file: n.file, line: n.line }); return; }
    // an unchanged type in a changed file opens that file whole
    if (n.state === 'same' && files[n.file] && ctx.diffs.wholeText(ctx.diffs.parsed(n.file))) {
      ctx.diffs.setMode(n.file, 'whole');
      openComponent(n.component, { file: n.file, mode: 'whole', line: n.line });
      return;
    }
    const meta = files[n.file];
    openComponent(n.component, meta ? (meta.status === 'A' && n.line > 1 ? { file: n.file, line: n.line } : { file: n.file }) : null);
  }
  function showOnDiagram(nodeId) {
    pending = { node: nodeId };
    go('#map');
  }

  /* ---------- the pull request filter ---------- */
  const strip = document.getElementById('stack-strip');
  const pullButtons = document.getElementById('pull-filter');
  function stripNote() {
    const m = pulls.memberOf[ctx.filter.pull];
    if (!m) return 'All ' + pulls.members.length + ' pull requests, combined, from <b>' + esc(model.heading.base.slice(0, 8)) + '</b> to the head of <b>#' + esc(pulls.members[pulls.members.length - 1].pull) + '</b>';
    return (m.title ? '<b>' + esc(m.title) + '</b> &middot; ' : '') + totalsHTML(memberTotals()) + ' of the combined diff';
  }
  function memberTotals() {
    const t = { files: 0, adds: 0, dels: 0 };
    model.files.filter(f => pulls.onFilter(f.pulls)).forEach(f => {
      const own = (f.byPull && f.byPull[ctx.filter.pull]) || [0, 0];
      t.files++; t.adds += own[0]; t.dels += own[1];
    });
    return t;
  }
  function setPullFilter(pull, quiet) {
    ctx.filter.pull = pulls.memberOf[pull] ? pull : null;
    pullButtons.querySelectorAll('.pf').forEach(b => b.setAttribute('aria-pressed', String((+b.getAttribute('data-pull') || null) === ctx.filter.pull)));
    document.getElementById('strip-note').innerHTML = stripNote();
    ctx.diagram.filterTo(ctx.filter.pull);
    totalsEl.innerHTML = ctx.filter.pull ? pulls.pullPill(ctx.filter.pull) + ' ' + totalsHTML(memberTotals()) : combinedTotals;
    if (!diagramView.hidden && !quiet) ctx.diagram.showFiltered();
    else filterMoved = !quiet;
    const cid = ctx.diffs.current();
    ctx.diffs.forget();
    if (!diffView.hidden && cid) ctx.diffs.renderComponent(cid);
  }
  if (pulls.members.length) {
    pullButtons.innerHTML = '<button type="button" class="pf" data-pull="" aria-pressed="true">All</button>' + pulls.members.map(m =>
      '<button type="button" class="pf" data-pull="' + m.pull + '" aria-pressed="false" style="--pc:' + pulls.pullColor(m.pull) + '"' +
      (m.title ? ' title="' + esc(m.title) + '"' : '') + '><i></i>#' + m.pull + '</button>').join('');
    pullButtons.addEventListener('click', e => {
      const b = e.target.closest('.pf');
      if (b) setPullFilter(+b.getAttribute('data-pull') || null);
    });
    ctx.data.stack.warnings.forEach(w => strip.insertAdjacentHTML('beforeend', '<p class="warn-line">' + esc(w) + '</p>'));
    document.getElementById('strip-note').innerHTML = stripNote();
  }

  /* ---------- state changes ---------- */
  const toast = document.getElementById('toast');
  function flash(message) {
    toast.textContent = message;
    toast.hidden = false;
    clearTimeout(flash.timer);
    flash.timer = setTimeout(() => { toast.hidden = true; }, 6000);
  }
  function countQueued() {
    const n = ctx.state.queue ? ctx.state.queue.members.reduce((k, m) => k + m.comments.length, 0) : 0;
    const badge = viewsNav.querySelector('.vcount');
    badge.textContent = n ? String(n) : '';
    badge.hidden = !n;
    badge.title = n ? plural(n, 'queued comment') : '';
  }
  function setState(next, message) {
    ctx.state = next;
    ctx.overview.render(message);
    ctx.notes.refresh();
    countQueued();
  }

  Object.assign(ctx, {
    go, setPullFilter, setState, flash,
    open: { component: openComponent, node: openNode, onDiagram: showOnDiagram },
  });
  ctx.notes = createLineNotes(ctx);
  ctx.diagram = createDiagram(ctx);
  ctx.diffs = createDiffs(ctx);
  ctx.queuePanel = createQueuePanel(ctx);
  ctx.queueActions = ctx.queuePanel.actions;
  ctx.overview = createOverview(ctx);
  ctx.overview.render();
  countQueued();
  const refresh = createRefresh(ctx);

  /* ---------- routing ---------- */
  function show(route) {
    let cid = route.indexOf('diff/') === 0 ? decodeURIComponent(route.slice(5)) : null;
    if (cid && !comps[cid]) cid = null;
    const which = cid ? 'diff' : route === 'review' ? 'review' : route === 'map' ? 'map' : home;
    reviewView.hidden = which !== 'review';
    diagramView.hidden = which !== 'map';
    diffView.hidden = which !== 'diff';
    document.body.classList.toggle('map-mode', which === 'map');
    back.hidden = which !== 'diff';
    document.getElementById('diagram-tools').hidden = which !== 'map';
    document.getElementById('diff-tools').hidden = which !== 'diff';
    strip.hidden = !pulls.members.length || which === 'review';
    viewsNav.querySelectorAll('button').forEach(b => {
      if (b.getAttribute('data-view') === which) b.setAttribute('aria-current', 'page'); else b.removeAttribute('aria-current');
    });
    if (which !== 'diff') from = which;
    back.innerHTML = from === 'review' ? '&larr; Review' : '&larr; Diagram';
    if (which === 'review') { window.scrollTo(0, 0); pending = null; return; }
    if (which === 'map') {
      ctx.diagram.fitTexts();
      if (!fitted || !savedView) { ctx.diagram.fit(); fitted = true; }
      else ctx.diagram.restoreView(savedView);
      if (filterMoved) { filterMoved = false; ctx.diagram.showFiltered(); }
      if (pending && pending.node) ctx.diagram.showNode(pending.node);
      pending = null;
      return;
    }
    if (ctx.diffs.current() !== cid) ctx.diffs.renderComponent(cid);
    window.scrollTo(0, 0);
    const p = pending;
    pending = null;
    if (p && p.file && p.mode) {
      const d = ctx.diffs.detailsFor(p.file);
      if (d) ctx.diffs.rerenderFile(d);
    }
    if (p && p.file && p.line) requestAnimationFrame(() => ctx.diffs.focusLine(p.file, p.line, p.side));
    else if (p && p.file) requestAnimationFrame(() => ctx.diffs.focusFile(p.file));
    else if (p && p.pkg) {
      const c = comps[cid];
      const first = (c.context ? model.contextFiles : model.files).filter(f => f.component === cid && f.package === p.pkg)[0];
      if (first) requestAnimationFrame(() => ctx.diffs.focusFile(first.path));
    }
  }

  back.addEventListener('click', () => { pending = null; go('#' + from); });
  viewsNav.addEventListener('click', e => {
    const b = e.target.closest('button[data-view]');
    if (b) { pending = null; go('#' + b.getAttribute('data-view')); }
  });
  window.addEventListener('hashchange', () => show(location.hash.slice(1)));
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape' && !diffView.hidden && !/^(INPUT|TEXTAREA|SELECT)$/.test(e.target.tagName)) go('#' + from);
  });

  show(location.hash.slice(1));
  refresh.restore();
  return ctx;
}
