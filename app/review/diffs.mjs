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
import { esc, fmt, plural, baseName, breakable, short } from '../lib/text.mjs';
import { getJSON, reviewPath } from '../lib/api.mjs';
import { drawRendered } from '../lib/rendered-markdown.mjs';

const GUTTERS = 104;

export function createDiffs(ctx) {
  const { model, files, contextFiles, nodesByFile, comps, compTotals } = ctx.data;
  const pulls = ctx.pulls;
  const main = document.getElementById('main');
  const nav = document.getElementById('files-nav');
  const asked = {};
  const drawn = {};
  const expandedGaps = {};
  const fileModes = {};
  let viewMode = 'unified';
  let currentComp = null;
  let rendering = 0;
  let splitRatio = 0.5;

  // One file as the server draws it: its hunks of rows, the unchanged runs between them and its whole text.
  function load(path) {
    if (!asked[path]) {
      asked[path] = getJSON(reviewPath(ctx.state.id, '/file?' + new URLSearchParams({ path, build: ctx.data.build })))
        .then(file => { drawn[path] = file; return file; })
        .catch(error => { delete asked[path]; throw error; });
    }
    return asked[path];
  }

  function fileName(p) {
    const i = p.lastIndexOf('/');
    if (i < 0) return esc(p);
    return '<span class="dir">' + breakable(p.slice(0, i + 1)) + '</span>' + esc(p.slice(i + 1));
  }
  function hunkRow(text) { return '<tr class="hunk"><td colspan="4">' + esc(text) + '</td></tr>'; }

  function displayList(f) {
    const items = [];
    const gapBefore = i => f.gaps.filter(g => g.before === i).forEach(g => items.push({ type: 'gap', key: f.path + ':' + g.before, g }));
    f.hunks.forEach((h, i) => {
      gapBefore(i);
      items.push({ type: 'hunk', h });
    });
    gapBefore(f.hunks.length);
    return items;
  }
  function gapRows(f, g) {
    const rows = [];
    for (let n = g.from; n <= g.to; n++) rows.push({ kind: 'same', old: n + g.delta, new: n, text: f.whole[n - 1] === undefined ? '' : f.whole[n - 1], owner: null, words: null });
    return rows;
  }
  function expanderRow(item) {
    const count = item.g.to - item.g.from + 1;
    return '<tr class="expander"><td colspan="4"><button class="gapbtn" data-gap="' + esc(item.key) + '">&#183; &#183; &#183; show ' + plural(count, 'unchanged line') + ' &#183; &#183; &#183;</button></td></tr>';
  }

  // A row's pull request stripe and the server's words for it, on a stack.
  function mark(r) {
    if (!r || !r.words) return null;
    return { owner: r.owner, title: ' title="' + esc(r.words) + '"', other: !!(r.kind !== 'same' && ctx.filter.pull && r.owner !== ctx.filter.pull) };
  }
  function num(n) { return n === null || n === undefined ? '' : n; }
  // A line number cell. `at` names the line a click on it comments on, as R:n for the new side or L:o for the removed side.
  function linenoTd(text, m, extra, at) {
    const own = m && m.owner ? ' own' : '';
    return '<td class="lineno' + (extra ? ' ' + extra : '') + own + (m && m.other ? ' pr-other' : '') + (at ? ' ln' : '') + '"' +
      (own ? ' style="--pc:' + pulls.pullColor(m.owner) + '"' : '') + (at ? ' data-at="' + at + '"' : '') + '>' + num(text) + '</td>';
  }
  function rowKeys(r) {
    if (r.kind === 'del') return 'L:' + r.old;
    if (r.kind === 'add') return 'R:' + r.new;
    return 'L:' + r.old + ' R:' + r.new;
  }
  function atOf(r) { return r.kind === 'del' ? 'L:' + r.old : 'R:' + r.new; }
  function unifiedRow(r) {
    if (r.kind === 'note') return hunkRow(r.text);
    const cls = r.kind === 'add' ? 'add' : r.kind === 'del' ? 'del' : '';
    const sign = r.kind === 'add' ? '+' : r.kind === 'del' ? '-' : '';
    const m = mark(r);
    return '<tr class="' + cls + (m && m.other ? ' pr-other' : '') + '"' + (r.new !== null ? ' data-n="' + r.new + '"' : '') + ' data-l="' + rowKeys(r) + '"' + (m ? m.title : '') + '>' +
      linenoTd(r.old, m, '', atOf(r)) + linenoTd(r.new, null, '', atOf(r)) +
      '<td class="sign">' + sign + '</td><td class="code">' + esc(r.text) + '</td></tr>';
  }
  function unifiedGapHTML(f, g) { return gapRows(f, g).map(unifiedRow).join(''); }
  function unifiedBody(f) {
    return displayList(f).map(it => {
      if (it.type === 'gap') return expandedGaps[it.key] ? unifiedGapHTML(f, it.g) : expanderRow(it);
      return hunkRow(it.h.header) + it.h.rows.map(unifiedRow).join('');
    }).join('');
  }
  function splitTd(r, no, cls) {
    if (!r) return '<td class="lineno void"></td><td class="code void"></td>';
    const m = cls ? mark(r) : null;
    return linenoTd(no, m, cls, atOf(r)) + '<td class="code ' + cls + (m && m.other ? ' pr-other' : '') + '"' + (m ? m.title : '') + '>' + esc(r.text) + '</td>';
  }
  function splitSameRow(r) {
    const m = mark(r);
    return '<tr data-n="' + r.new + '" data-l="' + rowKeys(r) + '"' + (m ? m.title : '') + '>' + splitTd(r, r.old, '') + splitTd(r, r.new, '') + '</tr>';
  }
  function splitGapHTML(f, g) { return gapRows(f, g).map(splitSameRow).join(''); }
  function splitHunkHTML(h) {
    let out = hunkRow(h.header), dels = [], adds = [];
    function flush() {
      const n = Math.max(dels.length, adds.length);
      for (let i = 0; i < n; i++) {
        const keys = [dels[i] ? 'L:' + dels[i].old : '', adds[i] ? 'R:' + adds[i].new : ''].filter(Boolean).join(' ');
        out += '<tr' + (adds[i] ? ' data-n="' + adds[i].new + '"' : '') + ' data-l="' + keys + '">' + splitTd(dels[i], dels[i] ? dels[i].old : '', 'cdel') + splitTd(adds[i], adds[i] ? adds[i].new : '', 'cadd') + '</tr>';
      }
      dels = []; adds = [];
    }
    h.rows.forEach(r => {
      if (r.kind === 'del') { dels.push(r); return; }
      if (r.kind === 'add') { adds.push(r); return; }
      flush();
      if (r.kind === 'note') { out += hunkRow(r.text); return; }
      out += splitSameRow(r);
    });
    flush();
    return out;
  }
  function splitBody(f) {
    return displayList(f).map(it => {
      if (it.type === 'gap') return expandedGaps[it.key] ? splitGapHTML(f, it.g) : expanderRow(it);
      return splitHunkHTML(it.h);
    }).join('');
  }
  function wholeHTML(lines, added) {
    return '<table class="difftable wholetable"><tbody>' + lines.map((s, i) => {
      const row = added && added[i + 1];
      const m = mark(row);
      return '<tr data-n="' + (i + 1) + '" data-l="R:' + (i + 1) + '"' + (row ? ' class="mark' + (m && m.other ? ' pr-other' : '') + '"' : '') + (m ? m.title : '') + '>' +
        linenoTd(i + 1, m, '', 'R:' + (i + 1)) + '<td class="code">' + esc(s) + '</td></tr>';
    }).join('') + '</tbody></table>';
  }
  function fileBody(path) {
    const f = drawn[path];
    if (contextFiles[path]) {
      return '<div class="note">Unchanged at ' + esc(short(model.heading.head)) + '. Read only: select a line number to ask about it.</div><div class="diffwrap">' + wholeHTML(f.whole || [], null) + '</div>';
    }
    const meta = files[path], mode = fileModes[path] || 'diff';
    if (meta.binary) return '<div class="note">A binary file. Its contents are not shown.</div>';
    if (!f.hunks.length) {
      return '<div class="note">' + (meta.status === 'R' ? 'Renamed, with no change to its contents.' : 'No change to its contents, only to its mode.') + '</div>';
    }
    if (mode === 'whole') {
      const added = {};
      if (meta.status !== 'A') f.hunks.forEach(h => h.rows.forEach(r => { if (r.kind === 'add') added[r.new] = r; }));
      return '<div class="diffwrap">' + wholeHTML(f.whole || [], added) + '</div>';
    }
    if (mode === 'rendered') {
      return '<div class="mdview"><p class="md-hint">' + (ctx.state.queue ? 'Select a passage to comment on it or ask Claude about it.' : 'Select a passage to ask Claude about it.') + '</p>' + drawRendered(f.rendered, meta.status) + '</div>';
    }
    const split = viewMode === 'split';
    return '<div class="diffwrap"><table class="difftable' + (split ? ' splittable' : '') + '">' +
      (split ? '<colgroup><col class="cnum"><col><col class="cnum"><col></colgroup>' : '') +
      '<tbody>' + (split ? splitBody(f) : unifiedBody(f)) + '</tbody></table>' + (split ? '<div class="split-divider" aria-hidden="true"></div>' : '') + '</div>';
  }

  function chipsFor(path) {
    const types = (nodesByFile[path] || []).filter(n => n.kind !== 'artifact');
    return types.length ? '<span class="types">' + types.map(n =>
      '<button class="chip" data-node="' + esc(n.id) + '" title="Show ' + esc(n.name) + ' on the diagram">' + esc(n.name) + '</button>').join('') + '</span>' : '';
  }

  function fileHTML(path, open) {
    const id = contextFiles[path] ? 'f-' + contextFiles[path].index : 'f-' + files[path].index;
    let head;
    if (contextFiles[path]) {
      head = '<span class="badge same">unchanged</span>' + chipsFor(path) + ctx.notes.countBadge(path) + '<span class="fstats">' + plural(contextFiles[path].lines, 'line') + '</span>';
    } else {
      const meta = files[path], f = drawn[path], mode = fileModes[path] || 'diff';
      const badge = { A: '<span class="badge new">new</span>', D: '<span class="badge gone">deleted</span>', R: '<span class="badge mod">renamed</span>' }[meta.status] ||
        '<span class="badge mod">changed</span>';
      const from = meta.status === 'R' && meta.oldPath ? '<span class="from">from ' + breakable(meta.oldPath) + '</span>' : '';
      let buttons = '';
      if (!meta.binary && f.hunks.length && f.whole) buttons += '<button class="fbtn" data-mode="whole" aria-pressed="' + (mode === 'whole') + '">Whole file</button>';
      if (f.rendered) buttons += '<button class="fbtn" data-mode="rendered" aria-pressed="' + (mode === 'rendered') + '">Rendered</button>';
      const counts = ctx.filter.pull && meta.byPull ? meta.byPull[ctx.filter.pull] || [0, 0] : [meta.adds, meta.dels];
      head = badge + pulls.pullPills(meta.pulls) + from + chipsFor(path) + buttons + ctx.notes.countBadge(path) +
        '<span class="fstats"' + (ctx.filter.pull ? ' title="#' + ctx.filter.pull + '&rsquo;s lines; the file changes +' + fmt(meta.adds) + ' &minus;' + fmt(meta.dels) + ' in all"' : '') + '>' +
        '<span class="plus">+' + fmt(counts[0]) + '</span> <span class="minus">&minus;' + fmt(counts[1]) + '</span></span>';
    }
    return '<details class="file" ' + (open ? 'open ' : '') + 'data-path="' + esc(path) + '" id="' + id + '"><summary>' +
      '<span class="chev">&#9654;</span><span class="fname">' + fileName(path) + '</span>' + head +
      '</summary>' + (open ? fileBody(path) : '') + '</details>';
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
        if (turn === rendering) main.innerHTML = '<p class="filter-empty">The diff could not load: ' + esc(error.message) + '</p>';
        return;
      }
    }
    if (turn !== rendering) return;
    if (ctx.filter.pull && !c.context) t = filteredTotals([].concat(...pkgs.map(p => p.files)));
    nav.innerHTML = '<p class="nav-comp" style="color:' + ctx.diagram.compColor(cid) + '">' + esc(c.title) + '</p>' +
      '<p class="nav-tag">' + breakable(c.tag) + '</p>' +
      (c.description ? '<p class="nav-blurb">' + esc(c.description) + '</p>' : '') +
      '<p class="nav-totals">' + (c.context ? plural(t.files, 'file') + ', unchanged at ' + esc(short(model.heading.head)) + ', read only'
        : (ctx.filter.pull ? pulls.pullPill(ctx.filter.pull) + ' ' : '') + plural(t.files, 'file') + ' &nbsp;<span class="plus">+' + fmt(t.adds) + '</span> <span class="minus">&minus;' + fmt(t.dels) + '</span>') + '</p>' +
      pkgs.map(p =>
        (p.bare ? '' : '<div class="nav-pkg"><div class="nav-pkg-t">' + esc(p.title) + '</div><div class="nav-pkg-p">' + breakable(p.path || '/') + '</div></div>') +
        p.files.map(f => {
          const own = ctx.filter.pull && f.byPull ? f.byPull[ctx.filter.pull] || [0, 0] : [f.adds, f.dels];
          const dots = f.pulls && f.pulls.length ? '<span class="pdots" aria-label="' + esc(f.pulls.map(p => '#' + p).join(', ')) + '">' +
            f.pulls.map(p => '<i style="--pc:' + pulls.pullColor(p) + '"></i>').join('') + '</span>' : '';
          const stats = c.context ? '<span class="fst">' + fmt(f.lines) + '</span>'
            : '<span class="fst">' + dots + '<span class="plus">+' + fmt(own[0]) + '</span>' + (own[1] ? ' <span class="minus">&minus;' + fmt(own[1]) + '</span>' : '') + '</span>';
          return '<button class="nav-file" data-path="' + esc(f.path) + '"><span class="fnm">' + breakable(baseName(f.path)) + '</span>' + stats + '</button>';
        }).join('')).join('') +
      (model.components.length > 1 ? '<div class="nav-other"><p class="rail-h">Other components</p>' + model.components.filter(o => o.id !== cid).map(o =>
        '<button data-comp="' + esc(o.id) + '"><span class="swatch-i" style="background:' + ctx.diagram.compColor(o.id) + '"></span>' + esc(o.title) + '</button>').join('') + '</div>' : '');
    main.innerHTML = pkgs.map(p =>
      (p.bare ? '' : '<div class="pkg-sep"><h2>' + esc(p.title) + '</h2><span>' + breakable(p.path || '/') + '</span>' +
        (p.description ? '<p>' + esc(p.description) + '</p>' : '') + '</div>') +
      p.files.map(f => fileHTML(f.path, !c.context && !f.collapsed)).join('')).join('') ||
      '<p class="filter-empty">No file in ' + esc(c.title) + ' changed in #' + esc(ctx.filter.pull) + '. <button type="button" data-clear-filter>Show every pull request</button></p>';
    document.getElementById('toggle-view').hidden = !!c.context;
    main.querySelectorAll('details.file[open]').forEach(d => ctx.notes.decorate(d));
    if (viewMode === 'split') applySplitRatio();
  }

  function rerenderFile(details) {
    const tmp = document.createElement('div');
    tmp.innerHTML = fileHTML(details.getAttribute('data-path'), true);
    const fresh = tmp.firstChild;
    details.replaceWith(fresh);
    ctx.notes.decorate(fresh);
    if (viewMode === 'split') applySplitRatio();
    return fresh;
  }

  // Opens a file, drawing its body once the server has given it.
  async function openFile(d) {
    if (!d.open) d.open = true;
    const path = d.getAttribute('data-path');
    try {
      await load(path);
    } catch (error) {
      if (!d.querySelector('.note')) d.insertAdjacentHTML('beforeend', '<div class="note">This file could not load: ' + esc(error.message) + '</div>');
      return;
    }
    if (!d.isConnected || d.querySelector('.diffwrap, .mdview, .note')) return;
    d.insertAdjacentHTML('beforeend', fileBody(path));
    ctx.notes.decorate(d);
    if (viewMode === 'split') applySplitRatio();
  }

  main.addEventListener('toggle', e => {
    const d = e.target;
    if (d.classList && d.classList.contains('file') && d.open) openFile(d);
  }, true);

  function expandGap(det, key) {
    expandedGaps[key] = true;
    const f = drawn[det.getAttribute('data-path')];
    const item = displayList(f).find(it => it.type === 'gap' && it.key === key);
    const btn = det.querySelector('button.gapbtn[data-gap="' + CSS.escape(key) + '"]');
    if (item && btn) btn.closest('tr').outerHTML = viewMode === 'split' ? splitGapHTML(f, item.g) : unifiedGapHTML(f, item.g);
  }
  // Shows every unchanged line a note sits on, so no note hides inside a gap.
  function expandFor(det, line) {
    const path = det.getAttribute('data-path');
    if (!files[path] || !drawn[path] || (fileModes[path] || 'diff') !== 'diff') return;
    displayList(drawn[path]).forEach(it => {
      if (it.type === 'gap' && !expandedGaps[it.key] && line >= it.g.from && line <= it.g.to) expandGap(det, it.key);
    });
  }

  main.addEventListener('click', e => {
    const chip = e.target.closest('.chip');
    if (chip) { e.preventDefault(); e.stopPropagation(); ctx.open.onDiagram(chip.getAttribute('data-node')); return; }
    const fb = e.target.closest('.fbtn');
    if (fb) {
      e.preventDefault(); e.stopPropagation();
      const d = fb.closest('details.file'), path = d.getAttribute('data-path'), mode = fb.getAttribute('data-mode');
      fileModes[path] = fileModes[path] === mode ? 'diff' : mode;
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
    return meta ? document.getElementById('f-' + meta.index) : null;
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
  window.addEventListener('resize', () => { if (viewMode === 'split') applySplitRatio(); });

  let allOpen = true;
  document.getElementById('toggle-all').addEventListener('click', function () {
    allOpen = !allOpen;
    main.querySelectorAll('details.file').forEach(d => { d.open = allOpen; });
    this.textContent = allOpen ? 'Collapse all' : 'Expand all';
  });
  document.getElementById('toggle-view').addEventListener('click', function () {
    viewMode = viewMode === 'unified' ? 'split' : 'unified';
    this.textContent = viewMode === 'split' ? 'Unified view' : 'Split view';
    main.querySelectorAll('details.file').forEach(d => {
      const path = d.getAttribute('data-path');
      if (!files[path] || (fileModes[path] || 'diff') !== 'diff' || !d.open) return;
      const wrap = d.querySelector('.diffwrap');
      if (wrap) { wrap.outerHTML = fileBody(path); ctx.notes.decorate(d); }
    });
    if (viewMode === 'split') applySplitRatio();
  });
  document.getElementById('toggle-wrap').addEventListener('click', function () {
    document.body.classList.toggle('wrap');
    this.textContent = document.body.classList.contains('wrap') ? 'Unwrap lines' : 'Wrap lines';
  });

  return {
    renderComponent, rerenderFile, focusFile, focusLine, detailsFor, expandFor, load,
    current: () => currentComp,
    forget: () => { currentComp = null; },
    setMode(path, mode) { fileModes[path] = mode; },
    isWide: () => viewMode === 'split',
  };
}
