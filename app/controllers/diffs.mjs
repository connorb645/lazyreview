/**
 * The diff of one component at a time: its file list beside it, each file's
 * diff, unified or split, with wrapped lines, the whole file, the unchanged
 * lines between hunks, and a Rendered view of a Markdown file. Each file is
 * drawn from the server's file route, rows and all, for the page's build.
 *
 * createDiffs(ctx) returns renderComponent(id), which resolves once the
 * component is drawn, and the controls the other views use: load(path),
 * focusFile, focusLine, rerenderFile and current(). Every line number takes
 * a click, which opens a comment or a question on that line through
 * ctx.notes, and ctx.notes.decorate(file) adds the conversations and queued
 * comments under their lines whenever a file's diff is drawn.
 */
import { getJSON, reviewPath } from '../api/server.mjs';
import { views, element } from '../views.mjs';
import { createDiffMarkup, displayList } from '../presentation/diffs.mjs';
import { english } from '/i18n/english.mjs';

const GUTTERS = 104;

export function createDiffs(ctx) {
  const { model, files, contextFiles, comps, compTotals } = ctx.data;
  const pulls = ctx.pulls;
  const main = views.main;
  const nav = views.filesNav;
  const asked = {};
  const view = { drawn: {}, expandedGaps: {}, fileModes: {}, mode: 'unified' };
  const markup = createDiffMarkup(ctx, view);
  let currentComp = null;
  let rendering = 0;
  let splitRatio = 0.5;

  // One file as the server draws it: its hunks of rows, the unchanged runs between them and its whole text.
  function load(path) {
    if (!asked[path]) {
      asked[path] = getJSON(reviewPath(ctx.state.id, '/file?' + new URLSearchParams({ path, build: ctx.data.build })))
        .then(file => { view.drawn[path] = file; return file; })
        .catch(error => { delete asked[path]; throw error; });
    }
    return asked[path];
  }

  function compFiles(cid) {
    const c = comps[cid];
    const pkgs = c.packages.map(p => ({ id: p.id, title: p.title, path: p.path, description: p.description, bare: !!p.bare, files: [] }));
    const byId = {};
    pkgs.forEach(p => { byId[p.id] = p; });
    (c.context ? model.contextFiles : model.files).forEach(f => {
      if (f.component === cid && byId[f.package] && (c.context || pulls.onFilter(f.pulls))) byId[f.package].files.push(f);
    });
    return pkgs.filter(p => p.files.length);
  }
  function filteredTotals(list) {
    const t = { files: 0, adds: 0, dels: 0 };
    list.forEach(f => {
      const own = ctx.filter.pull ? (f.byPull && f.byPull[ctx.filter.pull]) || [0, 0] : [f.adds, f.dels];
      t.files++; t.adds += own[0]; t.dels += own[1];
    });
    return t;
  }

  async function renderComponent(cid) {
    currentComp = cid;
    const turn = ++rendering;
    const c = comps[cid];
    let t = compTotals[cid] || { files: 0, adds: 0, dels: 0 };
    const pkgs = compFiles(cid);
    if (!c.context) {
      try {
        await Promise.all(pkgs.flatMap(p => p.files.map(f => load(f.path))));
      } catch (error) {
        if (turn === rendering) main.innerHTML = markup.notLoaded(error.message);
        return;
      }
    }
    if (turn !== rendering) return;
    if (ctx.filter.pull && !c.context) t = filteredTotals([].concat(...pkgs.map(p => p.files)));
    nav.innerHTML = markup.nav(cid, pkgs, t);
    main.innerHTML = markup.componentFiles(cid, pkgs);
    views.toggleView.hidden = !!c.context;
    main.querySelectorAll('details.file[open]').forEach(d => ctx.notes.decorate(d));
    if (view.mode === 'split') applySplitRatio();
  }

  function rerenderFile(details) {
    const tmp = document.createElement('div');
    tmp.innerHTML = markup.fileHTML(details.getAttribute('data-path'), true);
    const fresh = tmp.firstChild;
    details.replaceWith(fresh);
    ctx.notes.decorate(fresh);
    if (view.mode === 'split') applySplitRatio();
    return fresh;
  }

  // Opens a file, drawing its body once the server has given it.
  async function openFile(d) {
    if (!d.open) d.open = true;
    const path = d.getAttribute('data-path');
    try {
      await load(path);
    } catch (error) {
      if (!d.querySelector('.note')) d.insertAdjacentHTML('beforeend', markup.fileNotLoaded(error.message));
      return;
    }
    if (!d.isConnected || d.querySelector('.diffwrap, .mdview, .note')) return;
    d.insertAdjacentHTML('beforeend', markup.fileBody(path));
    ctx.notes.decorate(d);
    if (view.mode === 'split') applySplitRatio();
  }

  main.addEventListener('toggle', e => {
    const d = e.target;
    if (d.classList && d.classList.contains('file') && d.open) openFile(d);
  }, true);

  function expandGap(det, key) {
    view.expandedGaps[key] = true;
    const f = view.drawn[det.getAttribute('data-path')];
    const item = displayList(f).find(it => it.type === 'gap' && it.key === key);
    const btn = det.querySelector('button.gapbtn[data-gap="' + CSS.escape(key) + '"]');
    if (item && btn) btn.closest('tr').outerHTML = markup.gapHTML(f, item.g);
  }
  // Shows every unchanged line a note sits on, so no note hides inside a gap.
  function expandFor(det, line) {
    const path = det.getAttribute('data-path');
    if (!files[path] || !view.drawn[path] || (view.fileModes[path] || 'diff') !== 'diff') return;
    displayList(view.drawn[path]).forEach(it => {
      if (it.type === 'gap' && !view.expandedGaps[it.key] && line >= it.g.from && line <= it.g.to) expandGap(det, it.key);
    });
  }

  main.addEventListener('click', e => {
    const chip = e.target.closest('.chip');
    if (chip) { e.preventDefault(); e.stopPropagation(); ctx.open.onDiagram(chip.getAttribute('data-node')); return; }
    const fb = e.target.closest('.fbtn');
    if (fb) {
      e.preventDefault(); e.stopPropagation();
      const d = fb.closest('details.file'), path = d.getAttribute('data-path'), mode = fb.getAttribute('data-mode');
      view.fileModes[path] = view.fileModes[path] === mode ? 'diff' : mode;
      rerenderFile(d);
      return;
    }
    const btn = e.target.closest('button.gapbtn');
    if (btn) {
      const det = btn.closest('details.file');
      expandGap(det, btn.getAttribute('data-gap'));
      ctx.notes.decorate(det);
      return;
    }
    const ln = e.target.closest('td.ln');
    if (ln) { ctx.notes.pick(ln, e); return; }
    if (e.target.closest('[data-clear-filter]')) ctx.setPullFilter(null);
  });

  nav.addEventListener('click', e => {
    const b = e.target.closest('.nav-file');
    if (b) { focusFile(b.getAttribute('data-path')); return; }
    const o = e.target.closest('[data-comp]');
    if (o) ctx.open.component(o.getAttribute('data-comp'));
  });

  function detailsFor(path) {
    const meta = files[path] || contextFiles[path];
    return meta ? element('f-' + meta.index) : null;
  }
  function markNav(path) {
    nav.querySelectorAll('.nav-file.on').forEach(b => b.classList.remove('on'));
    const nb = nav.querySelector('.nav-file[data-path="' + CSS.escape(path) + '"]');
    if (nb) { nb.classList.add('on'); nb.scrollIntoView({ block: 'nearest' }); }
  }
  async function focusFile(path) {
    const d = detailsFor(path);
    if (!d) return;
    await openFile(d);
    markNav(path);
    d.scrollIntoView({ block: 'start' });
    d.classList.add('flash');
    setTimeout(() => d.classList.remove('flash'), 1400);
  }
  // Scroll to one line of a file, showing it first if it sits among hidden unchanged lines.
  async function focusLine(path, line, side) {
    const d = detailsFor(path);
    if (!d) return;
    await openFile(d);
    const key = (side === 'LEFT' ? 'L:' : 'R:') + line;
    let row = d.querySelector('tr[data-l~="' + key + '"]');
    if (!row && side !== 'LEFT') {
      expandFor(d, line);
      ctx.notes.decorate(d);
      row = d.querySelector('tr[data-l~="' + key + '"]');
    }
    markNav(path);
    if (!row) { focusFile(path); return; }
    d.style.contentVisibility = 'visible';
    row.scrollIntoView({ block: 'center' });
    row.classList.add('spot');
    setTimeout(() => row.classList.remove('spot'), 2200);
  }

  /* ---------- the split view's divider ---------- */
  function applySplitRatio() {
    main.querySelectorAll('table.splittable').forEach(table => {
      const cols = table.querySelectorAll('col');
      const avail = table.clientWidth - GUTTERS;
      if (cols.length !== 4 || avail <= 0) return;
      cols[1].style.width = Math.round(avail * splitRatio) + 'px';
      cols[3].style.width = Math.round(avail * (1 - splitRatio)) + 'px';
      const div = table.parentElement.querySelector('.split-divider');
      const row = Array.prototype.find.call(table.tBodies[0].rows, r => r.cells.length === 4);
      if (div && row) div.style.left = row.cells[2].offsetLeft + 'px';
    });
  }
  main.addEventListener('pointerdown', e => {
    const div = e.target.closest('.split-divider');
    if (!div) return;
    e.preventDefault();
    const table = div.parentElement.querySelector('table.splittable');
    div.setPointerCapture(e.pointerId);
    div.classList.add('active');
    function move(ev) {
      const rect = table.getBoundingClientRect();
      const avail = rect.width - GUTTERS;
      if (avail <= 0) return;
      splitRatio = Math.min(0.8, Math.max(0.2, (ev.clientX - rect.left - 52) / avail));
      applySplitRatio();
    }
    function up() {
      div.classList.remove('active');
      div.removeEventListener('pointermove', move);
      div.removeEventListener('pointerup', up);
      div.removeEventListener('pointercancel', up);
    }
    div.addEventListener('pointermove', move);
    div.addEventListener('pointerup', up);
    div.addEventListener('pointercancel', up);
  });
  window.addEventListener('resize', () => { if (view.mode === 'split') applySplitRatio(); });

  const words = english.web.diffs;
  let allOpen = true;
  views.toggleAll.addEventListener('click', function () {
    allOpen = !allOpen;
    main.querySelectorAll('details.file').forEach(d => { d.open = allOpen; });
    this.textContent = allOpen ? words.collapseAll : words.expandAll;
  });
  views.toggleView.addEventListener('click', function () {
    view.mode = view.mode === 'unified' ? 'split' : 'unified';
    this.textContent = view.mode === 'split' ? words.unifiedView : words.splitView;
    main.querySelectorAll('details.file').forEach(d => {
      const path = d.getAttribute('data-path');
      if (!files[path] || (view.fileModes[path] || 'diff') !== 'diff' || !d.open) return;
      const wrap = d.querySelector('.diffwrap');
      if (wrap) { wrap.outerHTML = markup.fileBody(path); ctx.notes.decorate(d); }
    });
    if (view.mode === 'split') applySplitRatio();
  });
  views.toggleWrap.addEventListener('click', function () {
    document.body.classList.toggle('wrap');
    this.textContent = document.body.classList.contains('wrap') ? words.unwrapLines : words.wrapLines;
  });

  return {
    renderComponent, rerenderFile, focusFile, focusLine, detailsFor, expandFor, load,
    current: () => currentComp,
    forget: () => { currentComp = null; },
    setMode(path, mode) { view.fileModes[path] = mode; },
    isWide: () => view.mode === 'split',
  };
}
