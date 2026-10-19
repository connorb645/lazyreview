/**
 * The markup of one component's diff: its file list, each file's diff,
 * unified or split, the whole file, the unchanged lines between hunks, and
 * a Rendered view of a Markdown file. Every line number names the line a
 * click on it comments on.
 *
 * displayList(file) gives a file's hunks and the runs of unchanged lines
 * between them, in the order the diff shows them.
 *
 * createDiffMarkup(ctx, view) returns, for the files `view` holds:
 *   fileHTML(path, open)        a file, its body drawn when open
 *   fileBody(path)              a file's body, in its mode
 *   gapHTML(file, gap)          a run of unchanged lines, shown
 *   nav(cid, pkgs, totals)      the component's file list
 *   componentFiles(cid, pkgs)   the component's files, by package
 *   notLoaded(message), fileNotLoaded(message)
 *                               what shows when the server gave no diff
 * `view` holds the files the server gave, by path, in `drawn`; each file's
 * mode, diff, whole or rendered, in `fileModes`; the runs of unchanged lines
 * shown, in `expandedGaps`; and `mode`, unified or split.
 */
import { esc, fmt, baseName, breakable, short } from '../lib/text.mjs';
import { drawRendered } from './rendered-markdown.mjs';
import { pullPill, pullPills } from './pulls.mjs';
import { compColor } from './diagram.mjs';
import { countBadge } from './line-notes.mjs';
import { english } from '/i18n/english.mjs';

const words = english.web.diffs;
const counts = english.web.counts;

export function displayList(f) {
  const items = [];
  const gapBefore = i => f.gaps.filter(g => g.before === i).forEach(g => items.push({ type: 'gap', key: f.path + ':' + g.before, g }));
  f.hunks.forEach((h, i) => {
    gapBefore(i);
    items.push({ type: 'hunk', h });
  });
  gapBefore(f.hunks.length);
  return items;
}

export function createDiffMarkup(ctx, view) {
  const { model, files, contextFiles, nodesByFile, comps } = ctx.data;
  const pulls = ctx.pulls;

  function fileName(p) {
    const i = p.lastIndexOf('/');
    if (i < 0) return esc(p);
    return '<span class="dir">' + breakable(p.slice(0, i + 1)) + '</span>' + esc(p.slice(i + 1));
  }
  function hunkRow(text) { return '<tr class="hunk"><td colspan="4">' + esc(text) + '</td></tr>'; }

  function gapRows(f, g) {
    const rows = [];
    for (let n = g.from; n <= g.to; n++) rows.push({ kind: 'same', old: n + g.delta, new: n, text: f.whole[n - 1] === undefined ? '' : f.whole[n - 1], owner: null, words: null });
    return rows;
  }
  function expanderRow(item) {
    const count = item.g.to - item.g.from + 1;
    return '<tr class="expander"><td colspan="4"><button class="gapbtn" data-gap="' + esc(item.key) + '">&#183; &#183; &#183; ' + words.showUnchanged(count) + ' &#183; &#183; &#183;</button></td></tr>';
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
      if (it.type === 'gap') return view.expandedGaps[it.key] ? unifiedGapHTML(f, it.g) : expanderRow(it);
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
      if (it.type === 'gap') return view.expandedGaps[it.key] ? splitGapHTML(f, it.g) : expanderRow(it);
      return splitHunkHTML(it.h);
    }).join('');
  }
  function gapHTML(f, g) {
    return view.mode === 'split' ? splitGapHTML(f, g) : unifiedGapHTML(f, g);
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
    const f = view.drawn[path];
    if (contextFiles[path]) {
      return '<div class="note">' + words.unchangedAt(esc(short(model.heading.head))) + '</div><div class="diffwrap">' + wholeHTML(f.whole || [], null) + '</div>';
    }
    const meta = files[path], mode = view.fileModes[path] || 'diff';
    if (meta.binary) return '<div class="note">' + words.binary + '</div>';
    if (!f.hunks.length) {
      return '<div class="note">' + (meta.status === 'R' ? words.renamedOnly : words.modeOnly) + '</div>';
    }
    if (mode === 'whole') {
      const added = {};
      if (meta.status !== 'A') f.hunks.forEach(h => h.rows.forEach(r => { if (r.kind === 'add') added[r.new] = r; }));
      return '<div class="diffwrap">' + wholeHTML(f.whole || [], added) + '</div>';
    }
    if (mode === 'rendered') {
      return '<div class="mdview"><p class="md-hint">' + (ctx.state.queue ? words.passageHint : words.passageHintAsk) + '</p>' + drawRendered(f.rendered, meta.status, b => b.pulls.length > 0 && !pulls.onFilter(b.pulls)) + '</div>';
    }
    const split = view.mode === 'split';
    return '<div class="diffwrap"><table class="difftable' + (split ? ' splittable' : '') + '">' +
      (split ? '<colgroup><col class="cnum"><col><col class="cnum"><col></colgroup>' : '') +
      '<tbody>' + (split ? splitBody(f) : unifiedBody(f)) + '</tbody></table>' + (split ? '<div class="split-divider" aria-hidden="true"></div>' : '') + '</div>';
  }

  function chipsFor(path) {
    const types = (nodesByFile[path] || []).filter(n => n.kind !== 'artifact');
    return types.length ? '<span class="types">' + types.map(n =>
      '<button class="chip" data-node="' + esc(n.id) + '" title="' + words.showOnDiagram(esc(n.name)) + '">' + esc(n.name) + '</button>').join('') + '</span>' : '';
  }

  function fileHTML(path, open) {
    const id = contextFiles[path] ? 'f-' + contextFiles[path].index : 'f-' + files[path].index;
    let head;
    if (contextFiles[path]) {
      head = '<span class="badge same">' + words.badges.unchanged + '</span>' + chipsFor(path) + countBadge(ctx.state, path) + '<span class="fstats">' + counts.lines(contextFiles[path].lines) + '</span>';
    } else {
      const meta = files[path], f = view.drawn[path], mode = view.fileModes[path] || 'diff';
      const badge = { A: '<span class="badge new">' + words.badges.added + '</span>', D: '<span class="badge gone">' + words.badges.deleted + '</span>', R: '<span class="badge mod">' + words.badges.renamed + '</span>' }[meta.status] ||
        '<span class="badge mod">' + words.badges.changed + '</span>';
      const from = meta.status === 'R' && meta.oldPath ? '<span class="from">' + words.from(breakable(meta.oldPath)) + '</span>' : '';
      let buttons = '';
      if (!meta.binary && f.hunks.length && f.whole) buttons += '<button class="fbtn" data-mode="whole" aria-pressed="' + (mode === 'whole') + '">' + words.wholeFile + '</button>';
      if (f.rendered) buttons += '<button class="fbtn" data-mode="rendered" aria-pressed="' + (mode === 'rendered') + '">' + words.rendered + '</button>';
      const shown = ctx.filter.pull && meta.byPull ? meta.byPull[ctx.filter.pull] || [0, 0] : [meta.adds, meta.dels];
      head = badge + pullPills(pulls, meta.pulls) + from + chipsFor(path) + buttons + countBadge(ctx.state, path) +
        '<span class="fstats"' + (ctx.filter.pull ? ' title="' + words.pullLines(ctx.filter.pull, fmt(meta.adds), fmt(meta.dels)) + '"' : '') + '>' +
        '<span class="plus">+' + fmt(shown[0]) + '</span> <span class="minus">&minus;' + fmt(shown[1]) + '</span></span>';
    }
    return '<details class="file" ' + (open ? 'open ' : '') + 'data-path="' + esc(path) + '" id="' + id + '"><summary>' +
      '<span class="chev">&#9654;</span><span class="fname">' + fileName(path) + '</span>' + head +
      '</summary>' + (open ? fileBody(path) : '') + '</details>';
  }

  function nav(cid, pkgs, t) {
    const c = comps[cid];
    return '<p class="nav-comp" style="color:' + compColor(comps, cid) + '">' + esc(c.title) + '</p>' +
      '<p class="nav-tag">' + breakable(c.tag) + '</p>' +
      (c.description ? '<p class="nav-blurb">' + esc(c.description) + '</p>' : '') +
      '<p class="nav-totals">' + (c.context ? words.contextTotals(counts.files(t.files), esc(short(model.heading.head)))
        : (ctx.filter.pull ? pullPill(pulls, ctx.filter.pull) + ' ' : '') + counts.files(t.files) + ' &nbsp;<span class="plus">+' + fmt(t.adds) + '</span> <span class="minus">&minus;' + fmt(t.dels) + '</span>') + '</p>' +
      pkgs.map(p =>
        (p.bare ? '' : '<div class="nav-pkg"><div class="nav-pkg-t">' + breakable(p.title) + '</div><div class="nav-pkg-p">' + breakable(p.path || '/') + '</div></div>') +
        p.files.map(f => {
          const own = ctx.filter.pull && f.byPull ? f.byPull[ctx.filter.pull] || [0, 0] : [f.adds, f.dels];
          const dots = f.pulls && f.pulls.length ? '<span class="pdots" aria-label="' + esc(f.pulls.map(p => '#' + p).join(', ')) + '">' +
            f.pulls.map(p => '<i style="--pc:' + pulls.pullColor(p) + '"></i>').join('') + '</span>' : '';
          const stats = c.context ? '<span class="fst">' + fmt(f.lines) + '</span>'
            : '<span class="fst">' + dots + '<span class="plus">+' + fmt(own[0]) + '</span>' + (own[1] ? ' <span class="minus">&minus;' + fmt(own[1]) + '</span>' : '') + '</span>';
          return '<button class="nav-file" data-path="' + esc(f.path) + '"><span class="fnm">' + breakable(baseName(f.path)) + '</span>' + stats + '</button>';
        }).join('')).join('') +
      (model.components.length > 1 ? '<div class="nav-other"><p class="rail-h">' + words.otherComponents + '</p>' + model.components.filter(o => o.id !== cid).map(o =>
        '<button data-comp="' + esc(o.id) + '"><span class="swatch-i" style="background:' + compColor(comps, o.id) + '"></span>' + esc(o.title) + '</button>').join('') + '</div>' : '');
  }

  function componentFiles(cid, pkgs) {
    const c = comps[cid];
    return pkgs.map(p =>
      (p.bare ? '' : '<div class="pkg-sep"><h2>' + breakable(p.title) + '</h2><span>' + breakable(p.path || '/') + '</span>' +
        (p.description ? '<p>' + esc(p.description) + '</p>' : '') + '</div>') +
      p.files.map(f => fileHTML(f.path, !c.context && !f.collapsed)).join('')).join('') ||
      '<p class="filter-empty">' + words.noFileChanged(esc(c.title), esc(ctx.filter.pull)) + ' <button type="button" data-clear-filter>' + words.showEveryPull + '</button></p>';
  }

  function notLoaded(message) {
    return '<p class="filter-empty">' + words.notLoaded(esc(message)) + '</p>';
  }

  function fileNotLoaded(message) {
    return '<div class="note">' + words.fileNotLoaded(esc(message)) + '</div>';
  }

  return { fileHTML, fileBody, gapHTML, nav, componentFiles, notLoaded, fileNotLoaded };
}
