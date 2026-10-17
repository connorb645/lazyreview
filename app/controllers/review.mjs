/**
 * One review in the app: the bar, the pull request strip of a stack, and the
 * three views, Review, Diagram and a component's diff, moved between by the
 * address's #fragment so the browser's back button walks them.
 *
 * openReview(id) loads the review's state and page data from the server,
 * draws every view and shows the one the address names, then keeps them up
 * to date with changes made elsewhere. It opens on the diagram, or on the
 * Review when a written review is attached.
 */
import { getJSON, reviewPath } from '../api/server.mjs';
import { views } from '../views.mjs';
import { createReview } from '../context/review.mjs';
import { headingContext, totals as totalsHTML, filteredTotals, allPulls, onePull, pullButtons as pullButtonsHTML, warnLine, backLabel } from '../presentation/bar.mjs';
import { createDiagram } from './diagram.mjs';
import { createDiffs } from './diffs.mjs';
import { createLineNotes } from './line-notes.mjs';
import { createQueuePanel } from './queue.mjs';
import { createOverview } from './overview.mjs';
import { createRefresh } from './refresh.mjs';
import { createPassages } from './passages.mjs';
import { createLiveUpdates } from './live-updates.mjs';
import { english } from '/i18n/english.mjs';

export async function openReview(id) {
  const [state, page] = await Promise.all([getJSON(reviewPath(id)), getJSON(reviewPath(id, '/page'))]);
  const ctx = createReview(state, page);
  const { model, comps, files, totals } = ctx.data;
  const pulls = ctx.pulls;

  const { reviewView, diagramView, diffView, back, viewsNav } = views;
  const totalsEl = views.totals;

  document.title = model.heading.title + ' - lazyreview';
  views.pageTitle.textContent = model.heading.title;
  views.pageTitle.title = model.heading.title;
  views.context.innerHTML = headingContext(model.heading.context);
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
  async function openNode(nodeId) {
    const n = ctx.data.nodes[nodeId];
    if (n.state === 'context') { openComponent(n.component, { file: n.file, line: n.line }); return; }
    // an unchanged type in a changed file opens that file whole
    if (n.state === 'same' && files[n.file] && (await ctx.diffs.load(n.file).catch(() => ({}))).whole) {
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
  const strip = views.stackStrip;
  const pullButtons = views.pullFilter;
  function stripNote() {
    const m = pulls.memberOf[ctx.filter.pull];
    if (!m) return allPulls(pulls, model.heading.base.slice(0, 8));
    return onePull(m, memberTotals());
  }
  function memberTotals() {
    const t = { files: 0, adds: 0, dels: 0 };
    model.files.filter(f => pulls.onFilter(f.pulls)).forEach(f => {
      const own = pulls.ownLines(f);
      t.files++; t.adds += own[0]; t.dels += own[1];
    });
    return t;
  }
  function setPullFilter(pull, quiet) {
    ctx.filter.pull = pulls.memberOf[pull] ? pull : null;
    pullButtons.querySelectorAll('.pf').forEach(b => b.setAttribute('aria-pressed', String((+b.getAttribute('data-pull') || null) === ctx.filter.pull)));
    views.stripNote.innerHTML = stripNote();
    ctx.diagram.filterTo(ctx.filter.pull);
    totalsEl.innerHTML = ctx.filter.pull ? filteredTotals(pulls, ctx.filter.pull, memberTotals()) : combinedTotals;
    if (!diagramView.hidden && !quiet) ctx.diagram.showFiltered();
    else filterMoved = !quiet;
    const cid = ctx.diffs.current();
    ctx.diffs.forget();
    if (!diffView.hidden && cid) ctx.diffs.renderComponent(cid);
  }
  if (pulls.members.length) {
    pullButtons.innerHTML = pullButtonsHTML(pulls);
    pullButtons.addEventListener('click', e => {
      const b = e.target.closest('.pf');
      if (b) setPullFilter(+b.getAttribute('data-pull') || null);
    });
    ctx.data.stack.warnings.forEach(w => strip.insertAdjacentHTML('beforeend', warnLine(w)));
    views.stripNote.innerHTML = stripNote();
  }

  /* ---------- state changes ---------- */
  const toast = views.toast;
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
    badge.title = n ? english.web.counts.queuedComments(n) : '';
  }
  // Which draft has the keyboard, and where its caret is, so a page drawn again from a new state keeps it there.
  function draftFocus() {
    const area = document.activeElement;
    const holder = area && area.tagName === 'TEXTAREA' && area.closest('.qc-item, .queued-note, .composer, .thread');
    if (!holder) return null;
    const of = name => holder.getAttribute(name);
    const selector = holder.matches('.composer') ? '.composer textarea'
      : holder.matches('.thread') ? '.thread[data-thread="' + of('data-thread') + '"] .reply textarea'
        : (holder.matches('.qc-item') ? '.qc-item' : '.queued-note') + '[data-comment="' + of('data-comment') + '"] .edit-box textarea';
    return { selector, start: area.selectionStart, end: area.selectionEnd };
  }
  function drawState(message) {
    const focus = draftFocus();
    ctx.overview.render(message);
    ctx.notes.refresh();
    ctx.queuePanel.restore();
    countQueued();
    const area = focus && document.querySelector(focus.selector);
    if (area && document.activeElement !== area) {
      area.focus({ preventScroll: true });
      area.setSelectionRange(focus.start, focus.end);
    }
  }

  ctx.onChange(drawState);
  Object.assign(ctx, {
    go, setPullFilter, flash,
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
  createPassages(ctx);

  /* ---------- routing ---------- */
  // Shows the view `route` names, and resolves once it is drawn.
  async function show(route) {
    let cid = route.indexOf('diff/') === 0 ? decodeURIComponent(route.slice(5)) : null;
    if (cid && !comps[cid]) cid = null;
    const which = cid ? 'diff' : route === 'review' ? 'review' : route === 'map' ? 'map' : home;
    reviewView.hidden = which !== 'review';
    diagramView.hidden = which !== 'map';
    diffView.hidden = which !== 'diff';
    document.body.classList.toggle('map-mode', which === 'map');
    back.hidden = which !== 'diff';
    views.diagramTools.hidden = which !== 'map';
    views.diffTools.hidden = which !== 'diff';
    strip.hidden = !pulls.members.length || which === 'review';
    viewsNav.querySelectorAll('button').forEach(b => {
      if (b.getAttribute('data-view') === which) b.setAttribute('aria-current', 'page'); else b.removeAttribute('aria-current');
    });
    if (which !== 'diff') from = which;
    back.innerHTML = backLabel(from);
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
    const p = pending;
    pending = null;
    if (ctx.diffs.current() !== cid) await ctx.diffs.renderComponent(cid);
    window.scrollTo(0, 0);
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

  await show(location.hash.slice(1));
  refresh.restore();
  createLiveUpdates(ctx, refresh.reloadHere);
  return ctx;
}
