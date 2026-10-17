/**
 * The class diagram of the change: drawn once, panned and zoomed, with every
 * link of a type and its stored properties shown while it is pointed at or
 * has the keyboard, every link at once on request, a search for a type or
 * file, and the rail listing the components and the notation.
 *
 * createDiagram(ctx) draws into the diagram view and returns the controls the
 * other views use: fit, zoomTo, showNode (zoom to a type and light its
 * links), filterTo (dim what the pull request filter leaves out), and
 * saveView and restoreView, so returning to the diagram finds it as it was.
 * Selecting a type or a component calls ctx.open.
 */
import { baseName } from '../lib/text.mjs';
import { views, element } from '../views.mjs';
import { createDiagramMarkup, countWords, summaryLines, packageCount } from '../presentation/diagram.mjs';

const MIN = 0.03, MAX = 2.6, NAMES_SHOWN = 0.45;
const ZOOM_STRIP = 58;

export function createDiagram(ctx) {
  const { model, layout, nodes, groupsById, comps } = ctx.data;
  const pulls = ctx.pulls;
  const markup = createDiagramMarkup(ctx);
  const svg = views.diagram;
  const canvas = views.canvas;
  const propsPanel = views.propsPanel;

  function boxOf(id) { return layout.boxes[id] || null; }

  svg.innerHTML = markup.diagram();
  const viewport = element('viewport');
  const overlay = svg.querySelector('.overlay');
  overlay.innerHTML = markup.overlay();

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
  const scaleEl = views.scale;
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
  views.zoomIn.addEventListener('click', () => { const r = canvas.getBoundingClientRect(); zoomAt(r.width / 2, r.height / 2, 1.4); });
  views.zoomOut.addEventListener('click', () => { const r = canvas.getBoundingClientRect(); zoomAt(r.width / 2, r.height / 2, 1 / 1.4); });
  views.zoomFit.addEventListener('click', fit);

  /* ---------- relationships of one type ---------- */
  const hoverLayer = svg.querySelector('.hover-edges');
  function showProperties(id, rels) {
    const html = markup.properties(id, rels);
    if (html === null) { propsPanel.hidden = true; return; }
    propsPanel.innerHTML = html;
    propsPanel.hidden = false;
  }
  function relationsOf(id) {
    const n = nodes[id];
    const group = n && n.group ? 'group:' + n.group : null;
    return model.edges.filter(e => e.source === id || e.target === id || (group && (e.source === group || e.target === group)));
  }
  function litBy(id, rels) {
    const lit = { [id]: true };
    rels.forEach(e => {
      if (!boxOf(e.source) || !boxOf(e.target)) return;
      [e.source, e.target].forEach(x => {
        if (x.indexOf('group:') === 0) (groupsById[x.slice(6)] || { members: [] }).members.forEach(m => { lit[m] = true; });
        else lit[x] = true;
      });
    });
    return lit;
  }
  function drawHover(id) {
    const rels = relationsOf(id);
    const lit = litBy(id, rels);
    hoverLayer.innerHTML = markup.hoverEdges(rels);
    showProperties(id, rels);
    svg.classList.add('focus');
    svg.querySelectorAll('.node').forEach(g => {
      g.classList.toggle('lit', !!lit[g.getAttribute('data-id')]);
      g.classList.toggle('self', g.getAttribute('data-id') === id);
    });
    return rels;
  }
  function clearHover() {
    hoverLayer.innerHTML = '';
    propsPanel.hidden = true;
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
  const allLinks = views.allLinks;
  const faintLayer = svg.querySelector('.faint-edges');
  let faintBuilt = false;
  allLinks.addEventListener('click', () => {
    const on = allLinks.getAttribute('aria-pressed') !== 'true';
    allLinks.setAttribute('aria-pressed', String(on));
    if (on && !faintBuilt) {
      faintBuilt = true;
      faintLayer.innerHTML = markup.faintEdges();
    }
    svg.classList.toggle('all-links', on);
  });

  /* ---------- find ---------- */
  const search = views.search;
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
  const compList = views.compList;
  compList.innerHTML = markup.compList();
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
  views.legend.innerHTML = markup.legend();

  /* ---------- the pull request filter ---------- */
  function filterTo(pull) {
    const on = {};
    model.nodes.forEach(n => { on[n.id] = !!pull && pulls.shows(n.id); });
    model.groups.forEach(g => { on['group:' + g.id] = g.members.some(id => on[id]); });
    const compOn = {}, pkgOn = {};
    model.components.forEach(c => { if (pull && c.context) compOn[c.id] = true; });
    model.nodes.forEach(n => { if (on[n.id]) compOn[n.component] = pkgOn[n.component + ':' + n.package] = true; });
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
    svg.querySelectorAll('.pkg-frame[data-pkg], .pkg-head[data-pkg]').forEach(el => {
      el.classList.toggle('pr-off', !!pull && !pkgOn[el.getAttribute('data-comp') + ':' + el.getAttribute('data-pkg')]);
    });
    document.querySelectorAll('.comp-item[data-comp]').forEach(el => {
      el.classList.toggle('pr-off', !!pull && !compOn[el.getAttribute('data-comp')]);
    });
    layout.summaries.forEach((s, i) => {
      const counts = pull && s.ways ? s.ways.map(w => pulls.countShown(w.links)) : null;
      const lit = !counts || counts.some(n => n > 0);
      svg.querySelectorAll('.summary[data-i="' + i + '"], .summary-label[data-i="' + i + '"]').forEach(el => el.classList.toggle('pr-on', lit));
      const lines = summaryLines(s, counts);
      svg.querySelectorAll('.summary-label[data-i="' + i + '"] tspan').forEach((el, k) => { el.textContent = lines[k]; });
    });
    layout.pkgArrows.forEach((a, i) => {
      const count = pull && a.links ? pulls.countShown(a.links) : a.count;
      svg.querySelectorAll('.pkg-arrow[data-i="' + i + '"], .pkg-arrow-label[data-i="' + i + '"]').forEach(el => el.classList.toggle('pr-on', count > 0));
      svg.querySelectorAll('.pkg-arrow-label[data-i="' + i + '"] .sub').forEach(el => { el.textContent = (a.words ? '· ' : '') + packageCount(count); });
    });
    const totals = pulls.componentTotals();
    svg.querySelectorAll('.comp-head[data-comp] .comp-tag').forEach(el => {
      const c = comps[el.parentNode.getAttribute('data-comp')];
      el.textContent = c.tag + ' · ' + countWords(totals, c);
    });
    document.querySelectorAll('.comp-item[data-comp] .n').forEach(el => {
      el.textContent = countWords(totals, comps[el.parentNode.getAttribute('data-comp')], true);
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
    if (!views.diagramView.hidden && atFit) fit();
  });

  return {
    fit, fitTexts, apply, showFiltered, filterTo, showNode, clearHover,
    saveView() { return { ...view }; },
    restoreView(saved) { view = { ...saved }; apply(); },
  };
}
