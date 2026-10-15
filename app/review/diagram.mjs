/**
 * The class diagram of the change: drawn once, panned and zoomed, with every
 * link of a type shown while it is pointed at, every link at once on request,
 * a search for a type or file, and the rail listing the components and the
 * notation.
 *
 * createDiagram(ctx) draws into the diagram view and returns the controls the
 * other views use: fit, zoomTo, showNode (zoom to a type and light its
 * links), filterTo (dim what the pull request filter leaves out), and
 * saveView and restoreView, so returning to the diagram finds it as it was.
 * Selecting a type or a component calls ctx.open.
 */
import { esc, fmt, plural, baseName, short } from '../lib/text.mjs';

const KIND_WORDS = { has: 'holds', realize: 'conforms to', inherit: 'inherits from', extend: 'extends', use: 'uses', test: 'tests' };
const MIN = 0.03, MAX = 2.6, NAMES_SHOWN = 0.45;
const ZOOM_STRIP = 58;
const STATE_WORDS = { changed: 52, removed: 54, same: 68 };
const NODE_RADIUS = 6;

export function createDiagram(ctx) {
  const { model, layout, nodes, comps, groupsById, compTotals } = ctx.data;
  const pulls = ctx.pulls;
  const svg = document.getElementById('diagram');
  const canvas = document.getElementById('canvas');
  const colors = [];
  model.components.forEach(c => { if (colors.indexOf(c.color) < 0) colors.push(c.color); });

  function compColor(id) { return 'var(--c-' + (comps[id] ? comps[id].color : 'grey') + ')'; }
  function countWords(c, compact) {
    const t = compTotals[c.id] || { files: 0, adds: 0, dels: 0 };
    const n = compact ? fmt(t.files) : plural(t.files, 'file');
    if (c.context) return n + ' · unchanged';
    return n + ' · +' + fmt(t.adds) + (t.dels ? ' −' + fmt(t.dels) : '');
  }

  function markerDefs() {
    return '<defs>' +
      '<marker id="m-tri" viewBox="0 0 12 12" refX="11" refY="6" markerWidth="13" markerHeight="13" markerUnits="userSpaceOnUse" orient="auto">' +
        '<path d="M1,1 L11,6 L1,11 z" fill="var(--surface)" stroke="var(--edge)" stroke-width="1.2"></path></marker>' +
      '<marker id="m-diamond" viewBox="0 0 14 8" refX="0.5" refY="4" markerWidth="15" markerHeight="9" markerUnits="userSpaceOnUse" orient="auto">' +
        '<path d="M1,4 L7,1 L13,4 L7,7 z" fill="var(--edge)"></path></marker>' +
      '<marker id="m-open" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="10" markerHeight="10" markerUnits="userSpaceOnUse" orient="auto">' +
        '<path d="M1,1 L9,5 L1,9" fill="none" stroke="var(--edge)" stroke-width="1.3"></path></marker>' +
      '<marker id="h-tri" viewBox="0 0 12 12" refX="11" refY="6" markerWidth="14" markerHeight="14" markerUnits="userSpaceOnUse" orient="auto">' +
        '<path d="M1,1 L11,6 L1,11 z" fill="var(--surface)" stroke="var(--accent)" stroke-width="1.4"></path></marker>' +
      '<marker id="h-diamond" viewBox="0 0 14 8" refX="0.5" refY="4" markerWidth="16" markerHeight="10" markerUnits="userSpaceOnUse" orient="auto">' +
        '<path d="M1,4 L7,1 L13,4 L7,7 z" fill="var(--accent)"></path></marker>' +
      '<marker id="h-open" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="11" markerHeight="11" markerUnits="userSpaceOnUse" orient="auto">' +
        '<path d="M1,1 L9,5 L1,9" fill="none" stroke="var(--accent)" stroke-width="1.5"></path></marker>' +
      colors.map(color =>
        '<marker id="m-pkg-' + color + '" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="4" markerHeight="4" orient="auto">' +
          '<path d="M0,0 L10,5 L0,10 z" style="fill:var(--c-' + color + ')"></path></marker>' +
        '<marker id="m-pkgs-' + color + '" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="4" markerHeight="4" orient="auto-start-reverse">' +
          '<path d="M0,0 L10,5 L0,10 z" style="fill:var(--c-' + color + ')"></path></marker>').join('') +
      '</defs>';
  }
  function markersFor(kind, prefix) {
    if (kind === 'realize' || kind === 'inherit') return ' marker-end="url(#' + prefix + '-tri)"';
    if (kind === 'has') return ' marker-start="url(#' + prefix + '-diamond)"';
    return ' marker-end="url(#' + prefix + '-open)"';
  }
  function polyPath(pts) {
    return 'M' + pts.map(p => p[0] + ',' + p[1]).join(' L');
  }
  function longestMidpoint(pts) {
    let best = -1, at = pts[0];
    for (let i = 1; i < pts.length; i++) {
      const len = Math.abs(pts[i][0] - pts[i - 1][0]) + Math.abs(pts[i][1] - pts[i - 1][1]);
      if (len > best) { best = len; at = [(pts[i][0] + pts[i - 1][0]) / 2, (pts[i][1] + pts[i - 1][1]) / 2]; }
    }
    return at;
  }
  function boxOf(id) { return layout.boxes[id] || null; }

  function goWords(n) {
    if (n.state === 'context') return 'Unchanged. Select to read its code at ' + short(model.heading.head);
    if (n.state === 'same') return 'Unchanged, in a changed file. Select to read the file whole';
    if (n.state === 'removed') return 'Removed by this change. Select to open the ' + comps[n.component].title + ' diff';
    return 'Select to open the ' + comps[n.component].title + ' diff';
  }

  function pullMarks(n, b) {
    if (!n.pulls || !n.pulls.length) return '';
    let out = '';
    const w = (b.w - NODE_RADIUS * 2) / n.pulls.length;
    n.pulls.forEach((p, i) => {
      out += '<rect class="pstrip" style="--pc:' + pulls.pullColor(p) + '" x="' + (b.x + NODE_RADIUS + i * w) + '" y="' + (b.y + 1) + '" width="' + w + '" height="3.5"></rect>';
    });
    const room = b.w - (STATE_WORDS[n.state] || 0) - 6;
    const width = text => text.length * 6.4 + 10;
    const shown = [];
    let x = b.x;
    for (let i = 0; i < n.pulls.length; i++) {
      const rest = n.pulls.length - i - 1;
      const need = width('#' + n.pulls[i]) + (rest ? width('+' + rest) + 3 : 0);
      if (x - b.x + need > room && shown.length) break;
      shown.push('#' + n.pulls[i]);
      x += width('#' + n.pulls[i]) + 3;
    }
    if (shown.length < n.pulls.length) shown.push('+' + (n.pulls.length - shown.length));
    x = b.x;
    shown.forEach((text, i) => {
      const pw = width(text);
      const color = text.charAt(0) === '#' ? pulls.pullColor(n.pulls[i]) : 'var(--muted)';
      out += '<g class="npill" style="--pc:' + color + '"><rect x="' + x + '" y="' + (b.y - 17) + '" width="' + pw + '" height="14" rx="7"></rect>' +
        '<text x="' + (x + pw / 2) + '" y="' + (b.y - 6.5) + '" text-anchor="middle">' + esc(text) + '</text></g>';
      x += pw + 3;
    });
    return out;
  }

  function nodeSVG(n) {
    const b = boxOf(n.id);
    if (!b) return '';
    const cls = 'node ' + n.state + (n.kind === 'protocol' || n.kind === 'interface' ? ' proto' : '') + (n.test ? ' test' : '') + (n.kind === 'artifact' ? ' artifact' : '');
    const cx = b.x + b.w / 2;
    let out = '<g class="' + cls + '" data-id="' + esc(n.id) + '" tabindex="0" role="link" aria-label="' +
      esc(n.name + ', ' + baseName(n.file) + '. ' + (n.pulls && n.pulls.length ? 'Changed in ' + n.pulls.map(p => '#' + p).join(', ') + '. ' : '') + goWords(n)) + '">' +
      '<rect class="box" x="' + b.x + '" y="' + b.y + '" width="' + b.w + '" height="' + b.h + '" rx="' + NODE_RADIUS + '"></rect>';
    if (n.kind === 'artifact') {
      out += '<text class="st" x="' + cx + '" y="' + (b.y + 17) + '">«file»</text>' +
        '<text class="nm" x="' + cx + '" y="' + (b.y + 35) + '" data-fit="' + (b.w - 14) + '">' + esc(n.name) + '</text>';
    } else {
      const st = n.stereotype;
      out += '<line class="rule" x1="' + b.x + '" x2="' + (b.x + b.w) + '" y1="' + (b.y + b.h - 18) + '" y2="' + (b.y + b.h - 18) + '"></line>';
      if (st) out += '<text class="st" x="' + cx + '" y="' + (b.y + 15) + '" data-fit="' + (b.w - 12) + '">' + esc(st) + '</text>';
      out += '<text class="nm" x="' + cx + '" y="' + (b.y + (st ? 33 : 26)) + '" data-fit="' + (b.w - 14) + '">' + esc(n.name) + '</text>' +
        '<text class="fl" x="' + cx + '" y="' + (b.y + b.h - 5.5) + '" data-fit="' + (b.w - 10) + '">' + esc(baseName(n.file)) + '</text>';
    }
    out += pullMarks(n, b);
    if (n.state === 'changed') out += '<text class="chg" x="' + (b.x + b.w - 4) + '" y="' + (b.y - 4) + '" text-anchor="end">changed</text>';
    if (n.state === 'same') out += '<text class="chg" x="' + (b.x + b.w - 4) + '" y="' + (b.y - 4) + '" text-anchor="end" style="fill:var(--muted)">unchanged</text>';
    if (n.state === 'removed') out += '<text class="chg" x="' + (b.x + b.w - 4) + '" y="' + (b.y - 4) + '" text-anchor="end" style="fill:var(--del-ink)">removed</text>';
    return out + '</g>';
  }

  const pkgList = [];
  model.components.forEach(c => {
    c.packages.forEach(p => {
      const b = layout.boxes['pkg:' + c.id + ':' + p.id];
      if (b) pkgList.push({ comp: c.id, id: p.id, title: p.title, path: p.path, description: p.description, bare: !!p.bare, box: b });
    });
  });

  function summarySVG(s) {
    const color = comps[s.source] ? comps[s.source].color : 'grey';
    const at = s.labelAt || longestMidpoint(s.points);
    const top = at[1] - (s.lines.length - 1) * 15 + 8;
    return '<path class="summary" d="' + polyPath(s.points) + '" style="stroke:var(--c-' + color + ')" marker-end="url(#m-pkg-' + color + ')"' +
      (s.both ? ' marker-start="url(#m-pkgs-' + color + ')"' : '') + '></path>' +
      '<text class="summary-label" x="' + at[0] + '" y="' + top + '" style="fill:var(--c-' + color + ')">' +
      s.lines.map((l, i) => '<tspan x="' + at[0] + '" dy="' + (i ? 30 : 0) + '">' + esc(l) + '</tspan>').join('') + '</text>';
  }
  function pkgArrowSVG(a) {
    const color = comps[a.comp].color;
    const lp = a.labelAt || longestMidpoint(a.points);
    return '<path class="pkg-arrow" d="' + polyPath(a.points) + '" style="stroke:var(--c-' + color + ')" marker-end="url(#m-pkg-' + color + ')"></path>' +
      '<text class="pkg-arrow-label" x="' + lp[0] + '" y="' + (lp[1] + 5) + '" style="fill:var(--c-' + color + ')">' +
      (a.words ? esc(a.words) + ' <tspan class="sub">· ' + esc(a.countText) + '</tspan>' : '<tspan class="sub">' + esc(a.countText) + '</tspan>') + '</text>';
  }

  function buildDiagram() {
    let out = markerDefs() + '<g id="viewport">';
    out += '<g class="areas">' + model.areas.map(a => {
      const b = layout.areas[a.id];
      if (!b) return '';
      return '<rect class="area-frame" x="' + b.x + '" y="' + b.y + '" width="' + b.w + '" height="' + b.h + '" rx="28"></rect>' +
        '<text class="area-title" x="' + (b.x + 44) + '" y="' + (b.y + 96) + '">' + esc(a.title) + '</text>' +
        (a.description ? '<text class="area-tag" x="' + (b.x + 48) + '" y="' + (b.y + 140) + '">' + esc(a.description) + '</text>' : '');
    }).join('') + '</g>';
    out += '<g class="summaries">' + layout.summaries.map(summarySVG).join('') + '</g>';
    out += '<g class="comps">';
    model.components.forEach(c => {
      const b = layout.components[c.id];
      if (!b) return;
      const color = compColor(c.id);
      out += '<rect class="comp-frame" data-comp="' + esc(c.id) + '" x="' + b.x + '" y="' + b.y + '" width="' + b.w + '" height="' + b.h + '" rx="14" style="stroke:' + color +
        ';fill:color-mix(in srgb, ' + color + ' 6%, var(--bg))' + (c.context ? ';stroke-dasharray:14 10' : '') + '"></rect>' +
        '<g class="comp-head" data-comp="' + esc(c.id) + '" tabindex="0" role="link" aria-label="' + (c.context ? 'Read the code of ' : 'Open the diff for ') + esc(c.title) + '">' +
        '<rect x="' + b.x + '" y="' + b.y + '" width="' + b.w + '" height="88" fill="transparent"></rect>' +
        '<text class="comp-title" x="' + (b.x + 24) + '" y="' + (b.y + 44) + '" style="fill:' + color + '">' + esc(c.title) + '</text>' +
        '<text class="comp-tag" x="' + (b.x + 26) + '" y="' + (b.y + 72) + '">' + esc(c.tag + ' · ' + countWords(c)) + '</text>' +
        '<text class="comp-open" x="' + (b.x + b.w - 24) + '" y="' + (b.y + 44) + '" text-anchor="end" style="fill:' + color + '">' + (c.context ? 'Read code' : 'Open diff') + ' →</text>' +
        '</g>';
    });
    out += '</g><g class="pkgs">';
    pkgList.forEach(p => {
      const b = p.box;
      const tab = p.description ? 50 : 34;
      out += '<rect class="pkg-frame" x="' + b.x + '" y="' + b.y + '" width="' + b.w + '" height="' + b.h + '" rx="6"></rect>';
      if (p.bare) return;
      out += '<g class="pkg-head" data-comp="' + esc(p.comp) + '" data-pkg="' + esc(p.id) + '" tabindex="-1">' +
        '<path class="pkg-tab" d="M' + (b.x + 0.6) + ',' + (b.y + tab) + ' L' + (b.x + 0.6) + ',' + (b.y + 6) + ' Q' + (b.x + 0.6) + ',' + (b.y + 0.6) + ' ' + (b.x + 6) + ',' + (b.y + 0.6) +
        ' L' + (b.x + b.w - 6) + ',' + (b.y + 0.6) + ' Q' + (b.x + b.w - 0.6) + ',' + (b.y + 0.6) + ' ' + (b.x + b.w - 0.6) + ',' + (b.y + 6) + ' L' + (b.x + b.w - 0.6) + ',' + (b.y + tab) + ' Z"></path>' +
        '<text class="pkg-title" x="' + (b.x + 12) + '" y="' + (b.y + 22) + '">' + esc(p.title) + '</text>' +
        '<text class="pkg-path" x="' + (b.x + b.w - 12) + '" y="' + (b.y + 22) + '" text-anchor="end">' + esc(p.path || '/') + '</text>' +
        (p.description ? '<text class="pkg-note" x="' + (b.x + 12) + '" y="' + (b.y + 40) + '">' + esc(p.description) + '</text>' : '') +
        '</g>';
    });
    model.groups.forEach(g => {
      const b = boxOf('group:' + g.id);
      if (!b) return;
      out += '<rect class="group-frame" data-group="' + esc(g.id) + '" x="' + b.x + '" y="' + b.y + '" width="' + b.w + '" height="' + b.h + '" rx="6"></rect>' +
        '<text class="group-title" data-group="' + esc(g.id) + '" x="' + (b.x + 12) + '" y="' + (b.y + 20) + '">' + esc(g.title) + '</text>' +
        (g.description ? '<text class="group-sub" data-group="' + esc(g.id) + '" x="' + (b.x + 12) + '" y="' + (b.y + 36) + '">' + esc(g.description) + '</text>' : '');
    });
    out += '</g>';
    out += '<g class="pkg-arrows">' + layout.pkgArrows.map(pkgArrowSVG).join('') + '</g>';
    out += '<g class="edges">';
    layout.routes.forEach((r, i) => {
      out += '<path class="edge ' + r.kind + '" data-i="' + i + '" d="' + polyPath(r.points) + '"' + markersFor(r.kind, 'm') + '></path>';
    });
    layout.routes.forEach((r, i) => {
      if (!r.label) return;
      const lp = r.labelAt || longestMidpoint(r.points);
      out += '<text class="elabel" data-i="' + i + '" x="' + lp[0] + '" y="' + (lp[1] + 4) + '">' + esc(r.label) + '</text>';
    });
    out += '</g><g class="faint-edges"></g><g class="hover-edges"></g>';
    out += '<g class="nodes">' + model.nodes.map(nodeSVG).join('') + '</g>';
    out += '<g class="overlay"></g></g>';
    svg.innerHTML = out;
  }

  buildDiagram();
  const viewport = document.getElementById('viewport');
  const overlay = svg.querySelector('.overlay');
  // A component's far-zoom title may reach past its frame, so a small component's stays as large as any other's.
  const farTitle = layout.farTitle || { size: 150, overhang: 0 };
  overlay.innerHTML = pkgList.filter(p => !p.bare).map(p =>
    '<text data-w="' + p.box.w + '" data-base="15" x="' + (p.box.x + p.box.w / 2) + '" y="' + (p.box.y + p.box.h / 2) + '" style="fill:' + compColor(p.comp) + '">' + esc(p.title) + '</text>'
  ).join('') + model.components.map(c => {
    const b = layout.components[c.id];
    if (!b) return '';
    return '<text class="above" data-comp="' + esc(c.id) + '" data-w="' + (b.w + farTitle.overhang * 2) + '" data-base="26" data-max="' + farTitle.size + '" x="' + (b.x + b.w / 2) + '" y="' + (b.y - 24) + '" style="fill:' + compColor(c.id) + '">' + esc(c.title) + '</text>';
  }).join('') + model.areas.map(a => {
    const b = layout.areas[a.id];
    if (!b) return '';
    return '<text class="above start" data-w="' + b.w + '" data-base="34" data-max="130" x="' + (b.x + 20) + '" y="' + (b.y - 28) + '" style="fill:var(--ink)">' + esc(a.title) + '</text>';
  }).join('');

  // The layout estimates text widths; a name that still overflows its box is fitted to it.
  let textsFitted = false;
  function fitTexts() {
    if (textsFitted) return;
    textsFitted = true;
    svg.querySelectorAll('text[data-fit]').forEach(t => {
      const room = +t.getAttribute('data-fit');
      if (t.getComputedTextLength() > room) {
        t.setAttribute('textLength', room);
        t.setAttribute('lengthAdjust', 'spacingAndGlyphs');
      }
    });
  }

  /* ---------- pan and zoom ---------- */
  let view = { s: 1, x: 0, y: 0 };
  const scaleEl = document.getElementById('scale');
  let isFar = null;
  let atFit = false;

  function apply() {
    atFit = false;
    viewport.setAttribute('transform', 'translate(' + view.x + ' ' + view.y + ') scale(' + view.s + ')');
    scaleEl.textContent = Math.round(view.s * 100) + '%';
    const far = view.s < NAMES_SHOWN - 0.03;
    svg.style.setProperty('--inv', (1 / view.s).toFixed(3));
    if (far !== isFar) { svg.classList.toggle('far', far); isFar = far; }
    svg.classList.toggle('farther', view.s < 0.2);
    if (far) {
      overlay.querySelectorAll('text').forEach(t => {
        const fitSize = +t.getAttribute('data-w') / (t.textContent.length * 0.62);
        const fs = Math.min(+t.getAttribute('data-base') / view.s, fitSize, +(t.getAttribute('data-max') || Infinity));
        t.setAttribute('font-size', fs);
        t.setAttribute('stroke-width', fs * 0.28);
      });
    }
  }
  // At phone width the diagram area fills the screen below it, and is at least as tall as the diagram at a fit to its width.
  const phone = window.matchMedia('(max-width: 760px)');
  function sizeCanvas() {
    if (!phone.matches) { canvas.style.height = ''; return; }
    const s = Math.min(1, (canvas.clientWidth * 0.97) / layout.w);
    const fitted = Math.max(220, Math.min(window.innerHeight * 0.68, layout.h * s + ZOOM_STRIP + 16));
    const free = window.innerHeight - (canvas.getBoundingClientRect().top + window.scrollY);
    canvas.style.height = Math.round(Math.max(fitted, Math.min(free, window.innerHeight))) + 'px';
  }
  function fit() {
    sizeCanvas();
    const r = canvas.getBoundingClientRect();
    // At phone width the zoom buttons get a strip of their own below the diagram.
    const room = r.height - (phone.matches ? ZOOM_STRIP : 0);
    const s = Math.min(1, Math.min(r.width / layout.w, room / layout.h) * 0.97);
    view.s = Math.max(MIN, s);
    view.x = (r.width - layout.w * view.s) / 2;
    view.y = (room - layout.h * view.s) / 2;
    apply();
    atFit = true;
  }
  function zoomAt(px, py, factor) {
    const ns = Math.max(MIN, Math.min(MAX, view.s * factor));
    view.x = px - (px - view.x) * (ns / view.s);
    view.y = py - (py - view.y) * (ns / view.s);
    view.s = ns;
    apply();
  }
  function zoomTo(box, maxScale) {
    const r = canvas.getBoundingClientRect();
    const s = Math.min(maxScale || 1, Math.min(r.width / (box.w + 160), r.height / (box.h + 160)));
    view.s = Math.max(MIN, s);
    view.x = r.width / 2 - (box.x + box.w / 2) * view.s;
    view.y = r.height / 2 - (box.y + box.h / 2) * view.s;
    apply();
  }
  function local(e) {
    const r = canvas.getBoundingClientRect();
    return [e.clientX - r.left, e.clientY - r.top];
  }

  canvas.addEventListener('wheel', e => {
    e.preventDefault();
    const p = local(e);
    if (e.ctrlKey || e.metaKey) {
      zoomAt(p[0], p[1], Math.exp(-e.deltaY * (e.deltaMode === 1 ? 0.05 : 0.0022)));
    } else {
      const k = e.deltaMode === 1 ? 16 : 1;
      view.x -= e.deltaX * k; view.y -= e.deltaY * k;
      apply();
    }
  }, { passive: false });

  const pointers = {};
  let drag = null, moved = false, pinch = null;
  canvas.addEventListener('pointerdown', e => {
    if (e.target.closest('.zoom-tools')) return;
    pointers[e.pointerId] = local(e);
    const ids = Object.keys(pointers);
    if (ids.length === 1) {
      drag = { start: local(e), x: view.x, y: view.y };
      moved = false;
    } else if (ids.length === 2) {
      const a = pointers[ids[0]], b = pointers[ids[1]];
      pinch = { d: Math.hypot(a[0] - b[0], a[1] - b[1]), s: view.s, cx: (a[0] + b[0]) / 2, cy: (a[1] + b[1]) / 2, x: view.x, y: view.y };
      drag = null; moved = true;
    }
  });
  canvas.addEventListener('pointermove', e => {
    if (!(e.pointerId in pointers)) return;
    pointers[e.pointerId] = local(e);
    const ids = Object.keys(pointers);
    if (pinch && ids.length === 2) {
      const a = pointers[ids[0]], b = pointers[ids[1]];
      const d = Math.hypot(a[0] - b[0], a[1] - b[1]);
      const ns = Math.max(MIN, Math.min(MAX, pinch.s * d / pinch.d));
      view.x = pinch.cx - (pinch.cx - pinch.x) * (ns / pinch.s);
      view.y = pinch.cy - (pinch.cy - pinch.y) * (ns / pinch.s);
      view.s = ns;
      apply();
      return;
    }
    if (!drag) return;
    const p = local(e);
    const dx = p[0] - drag.start[0], dy = p[1] - drag.start[1];
    if (!moved && Math.abs(dx) + Math.abs(dy) > 5) {
      moved = true;
      canvas.classList.add('panning');
      canvas.setPointerCapture(e.pointerId);
    }
    if (moved) { view.x = drag.x + dx; view.y = drag.y + dy; apply(); }
  });
  function endPointer(e) {
    delete pointers[e.pointerId];
    if (Object.keys(pointers).length < 2) pinch = null;
    if (!Object.keys(pointers).length) { drag = null; canvas.classList.remove('panning'); }
  }
  canvas.addEventListener('pointerup', endPointer);
  canvas.addEventListener('pointercancel', endPointer);
  canvas.addEventListener('dblclick', e => {
    if (e.target.closest('.node, .comp-head, .zoom-tools')) return;
    const p = local(e);
    zoomAt(p[0], p[1], e.shiftKey ? 1 / 1.8 : 1.8);
  });
  document.getElementById('zoom-in').addEventListener('click', () => { const r = canvas.getBoundingClientRect(); zoomAt(r.width / 2, r.height / 2, 1.4); });
  document.getElementById('zoom-out').addEventListener('click', () => { const r = canvas.getBoundingClientRect(); zoomAt(r.width / 2, r.height / 2, 1 / 1.4); });
  document.getElementById('zoom-fit').addEventListener('click', fit);

  /* ---------- relationships of one type ---------- */
  const hoverLayer = svg.querySelector('.hover-edges');
  function anchor(b, toward) {
    const cx = b.x + b.w / 2, cy = b.y + b.h / 2;
    const dx = toward[0] - cx, dy = toward[1] - cy;
    if (!dx && !dy) return [cx, cy];
    const k = Math.min((b.w / 2) / Math.abs(dx || 1e-9), (b.h / 2) / Math.abs(dy || 1e-9));
    return [cx + dx * k, cy + dy * k];
  }
  function relationsOf(id) {
    const n = nodes[id];
    const group = n && n.group ? 'group:' + n.group : null;
    return model.edges.filter(e => e.source === id || e.target === id || (group && (e.source === group || e.target === group)));
  }
  function drawHover(id) {
    const rels = relationsOf(id);
    const lit = { [id]: true };
    const parts = [];
    rels.forEach(e => {
      const a = boxOf(e.source), b = boxOf(e.target);
      if (!a || !b) return;
      [e.source, e.target].forEach(x => {
        if (x.indexOf('group:') === 0) (groupsById[x.slice(6)] || { members: [] }).members.forEach(m => { lit[m] = true; });
        else lit[x] = true;
      });
      const ca = [a.x + a.w / 2, a.y + a.h / 2], cb = [b.x + b.w / 2, b.y + b.h / 2];
      const p1 = anchor(a, cb), p2 = anchor(b, ca);
      const mx = (p1[0] + p2[0]) / 2, my = (p1[1] + p2[1]) / 2;
      const dx = p2[0] - p1[0], dy = p2[1] - p1[1], len = Math.hypot(dx, dy) || 1;
      const bend = Math.min(80, len * 0.12);
      const qx = mx - dy / len * bend, qy = my + dx / len * bend;
      parts.push('<path class="hover-edge ' + e.kind + '" d="M' + p1[0] + ',' + p1[1] + ' Q' + qx + ',' + qy + ' ' + p2[0] + ',' + p2[1] + '"' + markersFor(e.kind, 'h') + '></path>');
      parts.push('<text class="hover-label" x="' + (mx - dy / len * bend / 2) + '" y="' + (my + dx / len * bend / 2 + 4) + '">' + esc(e.label || KIND_WORDS[e.kind]) + '</text>');
    });
    hoverLayer.innerHTML = parts.join('');
    svg.classList.add('focus');
    svg.querySelectorAll('.node').forEach(g => {
      g.classList.toggle('lit', !!lit[g.getAttribute('data-id')]);
      g.classList.toggle('self', g.getAttribute('data-id') === id);
    });
    return rels;
  }
  function clearHover() {
    hoverLayer.innerHTML = '';
    svg.classList.remove('focus');
    svg.querySelectorAll('.node.lit').forEach(g => g.classList.remove('lit', 'self'));
  }

  let hovered = null;
  svg.addEventListener('pointerover', e => {
    if (drag && moved) return;
    const g = e.target.closest('.node');
    if (!g || g.getAttribute('data-id') === hovered) return;
    hovered = g.getAttribute('data-id');
    drawHover(hovered);
  });
  svg.addEventListener('pointerout', e => {
    const g = e.target.closest('.node');
    if (!g) return;
    const to = e.relatedTarget && e.relatedTarget.closest ? e.relatedTarget.closest('.node') : null;
    if (to === g) return;
    hovered = null;
    clearHover();
    applySearch();
  });
  svg.addEventListener('focusin', e => {
    const g = e.target.closest('.node');
    if (g) drawHover(g.getAttribute('data-id'));
  });
  svg.addEventListener('focusout', () => clearHover());

  function activate(target) {
    const g = target.closest('.node');
    if (g) { ctx.open.node(g.getAttribute('data-id')); return true; }
    const h = target.closest('.comp-head');
    if (h) { ctx.open.component(h.getAttribute('data-comp')); return true; }
    const p = target.closest('.pkg-head');
    if (p) { ctx.open.component(p.getAttribute('data-comp'), { pkg: p.getAttribute('data-pkg') }); return true; }
    return false;
  }
  svg.addEventListener('click', e => {
    if (moved) { moved = false; return; }
    activate(e.target);
  });
  svg.addEventListener('keydown', e => {
    if ((e.key === 'Enter' || e.key === ' ') && activate(e.target)) e.preventDefault();
  });

  /* ---------- every link, faintly ---------- */
  const allLinks = document.getElementById('all-links');
  const faintLayer = svg.querySelector('.faint-edges');
  let faintBuilt = false;
  allLinks.addEventListener('click', () => {
    const on = allLinks.getAttribute('aria-pressed') !== 'true';
    allLinks.setAttribute('aria-pressed', String(on));
    if (on && !faintBuilt) {
      faintBuilt = true;
      const drawn = {};
      layout.routes.forEach(r => { drawn[r.source + '>' + r.target] = true; });
      faintLayer.innerHTML = model.edges.filter(e => !drawn[e.source + '>' + e.target]).map(e => {
        const a = boxOf(e.source), b = boxOf(e.target);
        if (!a || !b) return '';
        const p1 = anchor(a, [b.x + b.w / 2, b.y + b.h / 2]), p2 = anchor(b, [a.x + a.w / 2, a.y + a.h / 2]);
        return '<path class="edge faint ' + e.kind + '" d="M' + p1[0] + ',' + p1[1] + ' L' + p2[0] + ',' + p2[1] + '"></path>';
      }).join('');
    }
    svg.classList.toggle('all-links', on);
  });

  /* ---------- find ---------- */
  const search = document.getElementById('search');
  function matches(q) {
    if (!q) return [];
    q = q.toLowerCase();
    return model.nodes.filter(n => n.name.toLowerCase().indexOf(q) >= 0 || baseName(n.file).toLowerCase().indexOf(q) >= 0);
  }
  function applySearch() {
    const ids = {};
    const found = matches(search.value.trim());
    found.forEach(n => { ids[n.id] = true; });
    svg.querySelectorAll('.node').forEach(g => g.classList.toggle('match', !!ids[g.getAttribute('data-id')]));
    return found;
  }
  search.addEventListener('input', applySearch);
  search.addEventListener('keydown', e => {
    if (e.key !== 'Enter') return;
    const found = applySearch();
    if (!found.length) return;
    const q = search.value.trim().toLowerCase();
    const exact = found.filter(n => n.name.toLowerCase() === q)[0] || found[0];
    zoomTo(boxOf(exact.id), 1.1);
  });

  /* ---------- component list and legend ---------- */
  function swatch(c, cls) {
    return '<span class="' + cls + (c.context ? ' dashed' : '') + '" style="background:' + compColor(c.id) + ';color:' + compColor(c.id) + '"></span>';
  }
  const railGroups = [];
  model.components.forEach(c => { if (railGroups.indexOf(c.group) < 0) railGroups.push(c.group); });
  const areaTitles = model.areas.map(a => a.title);
  const compList = document.getElementById('comp-list');
  compList.innerHTML = railGroups.map(g =>
    (g ? '<li class="comp-group' + (areaTitles.indexOf(g) >= 0 ? ' boxed' : '') + '">' + esc(g) + '</li>' : '') +
    model.components.filter(c => c.group === g).map(c =>
      '<li><button class="comp-item" data-comp="' + esc(c.id) + '">' + swatch(c, 'swatch') +
      '<span class="t">' + esc(c.title) + '</span><span class="n">' + countWords(c, true) + '</span>' +
      (c.description ? '<span class="d">' + esc(c.description) + '</span>' : '') + '</button></li>').join('')
  ).join('');
  compList.addEventListener('click', e => {
    const b = e.target.closest('.comp-item');
    if (b) ctx.open.component(b.getAttribute('data-comp'));
  });
  compList.addEventListener('pointerover', e => {
    const b = e.target.closest('.comp-item');
    if (!b) return;
    svg.querySelectorAll('.comp-frame').forEach(r => { r.style.strokeWidth = r.getAttribute('data-comp') === b.getAttribute('data-comp') ? 6 : ''; });
  });
  compList.addEventListener('pointerleave', () => {
    svg.querySelectorAll('.comp-frame').forEach(r => { r.style.strokeWidth = ''; });
  });

  function legendRow(inner, words) {
    return '<svg width="46" height="18" viewBox="0 0 46 18">' + inner + '</svg><span>' + words + '</span>';
  }
  const states = {};
  model.nodes.forEach(n => { states[n.state] = true; });
  const legend = document.getElementById('legend');
  legend.innerHTML =
    legendRow('<line x1="4" y1="9" x2="40" y2="9" class="edge has" marker-start="url(#m-diamond)"></line>', 'Holds one as a stored property') +
    legendRow('<line x1="2" y1="9" x2="40" y2="9" class="edge realize" marker-end="url(#m-tri)"></line>', 'Conforms to a protocol or interface') +
    legendRow('<line x1="2" y1="9" x2="40" y2="9" class="edge inherit" marker-end="url(#m-tri)"></line>', 'Inherits from a type') +
    legendRow('<line x1="2" y1="9" x2="40" y2="9" class="edge use" marker-end="url(#m-open)"></line>', 'Uses it (named links only, until you point)') +
    legendRow('<rect x="3" y="2" width="38" height="14" rx="4" fill="var(--surface)" stroke="var(--ink)"></rect>', 'New in this change') +
    (states.changed ? legendRow('<rect x="3" y="2" width="38" height="14" rx="4" fill="var(--surface)" stroke="var(--changed)" stroke-width="2"></rect>', 'Existing, changed') : '') +
    (states.same || states.context ? legendRow('<rect x="3" y="2" width="38" height="14" rx="4" fill="var(--bg)" stroke="var(--muted)" stroke-dasharray="4 3"></rect>', 'Unchanged, shown for context') : '') +
    (states.removed ? legendRow('<rect x="3" y="2" width="38" height="14" rx="4" fill="var(--del-bg)" stroke="var(--del-ink)" stroke-dasharray="4 3"></rect>', 'Removed by this change') : '');
  legend.querySelectorAll('svg').forEach(s => s.insertAdjacentHTML('afterbegin', markerDefs()));
  if (pulls.members.length) {
    legend.insertAdjacentHTML('beforeend', '<span class="pps">' + pulls.pullPill(pulls.members[0].pull) + '</span><span>Changed in that pull request of the stack</span>');
  }

  /* ---------- the pull request filter ---------- */
  function filterTo(pull) {
    const on = {};
    model.nodes.forEach(n => { on[n.id] = !!pull && pulls.onFilter(n.pulls); });
    model.groups.forEach(g => { on['group:' + g.id] = g.members.some(id => on[id]); });
    const compOn = {};
    model.nodes.forEach(n => { if (on[n.id]) compOn[n.component] = true; });
    model.files.forEach(f => { if (pull && pulls.onFilter(f.pulls)) compOn[f.component] = true; });
    svg.classList.toggle('pr-filter', !!pull);
    svg.querySelectorAll('.node').forEach(g => g.classList.toggle('pr-on', !!on[g.getAttribute('data-id')]));
    svg.querySelectorAll('.edge[data-i], .elabel[data-i]').forEach(el => {
      const r = layout.routes[+el.getAttribute('data-i')];
      el.classList.toggle('pr-on', !!(r && on[r.source] && on[r.target]));
    });
    svg.querySelectorAll('[data-group]').forEach(el => el.classList.toggle('pr-off', !on['group:' + el.getAttribute('data-group')]));
    svg.querySelectorAll('.comp-frame, .comp-head, .overlay text[data-comp]').forEach(el => {
      el.classList.toggle('pr-off', !!pull && !compOn[el.getAttribute('data-comp')]);
    });
    document.querySelectorAll('.comp-item[data-comp]').forEach(el => {
      el.classList.toggle('pr-off', !!pull && !compOn[el.getAttribute('data-comp')]);
    });
  }
  // Zoom to the filtered pull request's types, at a size whose names read.
  function showFiltered() {
    const boxes = ctx.filter.pull ? model.nodes.filter(n => pulls.onFilter(n.pulls) && boxOf(n.id)).map(n =>
      layout.boxes['pkg:' + n.component + ':' + n.package] || boxOf(n.id)) : [];
    if (!boxes.length) { fit(); return; }
    const x = Math.min(...boxes.map(b => b.x - 24)), y = Math.min(...boxes.map(b => b.y - 24));
    const right = Math.max(...boxes.map(b => b.x + b.w + 24)), bottom = Math.max(...boxes.map(b => b.y + b.h + 24));
    sizeCanvas();
    const r = canvas.getBoundingClientRect();
    const room = r.height - (phone.matches ? ZOOM_STRIP : 0);
    let area = { x, y, w: right - x, h: bottom - y };
    if (Math.min(r.width / area.w, room / area.h) < NAMES_SHOWN) area = busiestPackage(boxes);
    view.s = Math.max(NAMES_SHOWN, Math.min(1, r.width / area.w, room / area.h));
    view.x = area.w * view.s <= r.width ? (r.width - area.w * view.s) / 2 - area.x * view.s : 16 - area.x * view.s;
    view.y = area.h * view.s <= room ? (room - area.h * view.s) / 2 - area.y * view.s : 16 - area.y * view.s;
    apply();
  }
  function busiestPackage(boxes) {
    const counts = [];
    boxes.forEach(b => {
      const seen = counts.filter(c => c.box === b)[0];
      if (seen) seen.n++; else counts.push({ box: b, n: 1 });
    });
    counts.sort((a, c) => c.n - a.n || a.box.y - c.box.y || a.box.x - c.box.x);
    const b = counts[0].box;
    return { x: b.x - 24, y: b.y - 24, w: b.w + 48, h: b.h + 48 };
  }

  function showNode(id) {
    zoomTo(boxOf(id), 1.1);
    const g = svg.querySelector('.node[data-id="' + CSS.escape(id) + '"]');
    if (!g) return;
    g.classList.add('hit');
    drawHover(id);
    setTimeout(() => { g.classList.remove('hit'); clearHover(); }, 2600);
  }

  window.addEventListener('resize', () => {
    if (!document.getElementById('diagram-view').hidden && atFit) fit();
  });

  return {
    fit, fitTexts, apply, showFiltered, filterTo, showNode, clearHover, compColor, countWords, swatch,
    saveView() { return { ...view }; },
    restoreView(saved) { view = { ...saved }; apply(); },
  };
}
