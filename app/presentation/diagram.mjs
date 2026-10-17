/**
 * The markup of the class diagram of the change, and of the rail beside it.
 *
 * compColor(comps, id), swatch(comps, c, cls) and countWords(compTotals, c,
 * compact) are a component's colour, its swatch and its count of files and
 * lines, which other views draw too.
 *
 * createDiagramMarkup(ctx) returns, from the page data:
 *   diagram()            the whole diagram, drawn once
 *   overlay()            the titles drawn over it at far zoom
 *   hoverEdges(rels)     a type's links drawn while it is pointed at
 *   properties(id, rels) a type's stored properties, or null for none
 *   faintEdges()         every link the layout did not route, faintly
 *   compList()           the rail's components, by group
 *   legend()             the rail's notation
 */
import { esc, fmt, baseName, short } from '../lib/text.mjs';
import { pullPill } from './pulls.mjs';
import { english } from '/i18n/english.mjs';

const words = english.web.diagram;
const STATE_WORDS = { changed: 52, removed: 54, same: 68 };
const NODE_RADIUS = 6;

export function compColor(comps, id) {
  return 'var(--c-' + (comps[id] ? comps[id].color : 'grey') + ')';
}

export function swatch(comps, c, cls) {
  return '<span class="' + cls + (c.context ? ' dashed' : '') + '" style="background:' + compColor(comps, c.id) + ';color:' + compColor(comps, c.id) + '"></span>';
}

export function countWords(compTotals, c, compact) {
  const t = compTotals[c.id] || { files: 0, adds: 0, dels: 0 };
  const n = compact ? fmt(t.files) : english.web.counts.files(t.files);
  if (c.context) return words.unchangedCount(n);
  return n + ' · +' + fmt(t.adds) + (t.dels ? ' −' + fmt(t.dels) : '');
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

function anchor(b, toward) {
  const cx = b.x + b.w / 2, cy = b.y + b.h / 2;
  const dx = toward[0] - cx, dy = toward[1] - cy;
  if (!dx && !dy) return [cx, cy];
  const k = Math.min((b.w / 2) / Math.abs(dx || 1e-9), (b.h / 2) / Math.abs(dy || 1e-9));
  return [cx + dx * k, cy + dy * k];
}

export function createDiagramMarkup(ctx) {
  const { model, layout, nodes, comps, compTotals } = ctx.data;
  const pulls = ctx.pulls;
  const colors = [];
  model.components.forEach(c => { if (colors.indexOf(c.color) < 0) colors.push(c.color); });
  const color = id => compColor(comps, id);

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
      colors.map(c =>
        '<marker id="m-pkg-' + c + '" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="4" markerHeight="4" orient="auto">' +
          '<path d="M0,0 L10,5 L0,10 z" style="fill:var(--c-' + c + ')"></path></marker>' +
        '<marker id="m-pkgs-' + c + '" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="4" markerHeight="4" orient="auto-start-reverse">' +
          '<path d="M0,0 L10,5 L0,10 z" style="fill:var(--c-' + c + ')"></path></marker>').join('') +
      '</defs>';
  }
  function boxOf(id) { return layout.boxes[id] || null; }

  function goWords(n) {
    if (n.state === 'context') return words.readContext(short(model.heading.head));
    if (n.state === 'same') return words.readWhole;
    if (n.state === 'removed') return words.removedOpen(comps[n.component].title);
    return words.open(comps[n.component].title);
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
      const pc = text.charAt(0) === '#' ? pulls.pullColor(n.pulls[i]) : 'var(--muted)';
      out += '<g class="npill" style="--pc:' + pc + '"><rect x="' + x + '" y="' + (b.y - 17) + '" width="' + pw + '" height="14" rx="7"></rect>' +
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
      esc(words.nodeLabel(n.name, baseName(n.file), n.pulls || [], goWords(n))) + '">' +
      '<rect class="box" x="' + b.x + '" y="' + b.y + '" width="' + b.w + '" height="' + b.h + '" rx="' + NODE_RADIUS + '"></rect>';
    if (n.kind === 'artifact') {
      out += '<text class="st" x="' + cx + '" y="' + (b.y + 17) + '">' + english.diagram.file + '</text>' +
        '<text class="nm" x="' + cx + '" y="' + (b.y + 35) + '" data-fit="' + (b.w - 14) + '">' + esc(n.name) + '</text>';
    } else {
      const st = n.stereotype;
      out += '<line class="rule" x1="' + b.x + '" x2="' + (b.x + b.w) + '" y1="' + (b.y + b.h - 18) + '" y2="' + (b.y + b.h - 18) + '"></line>';
      if (st) out += '<text class="st" x="' + cx + '" y="' + (b.y + 15) + '" data-fit="' + (b.w - 12) + '">' + esc(st) + '</text>';
      out += '<text class="nm" x="' + cx + '" y="' + (b.y + (st ? 33 : 26)) + '" data-fit="' + (b.w - 14) + '">' + esc(n.name) + '</text>' +
        '<text class="fl" x="' + cx + '" y="' + (b.y + b.h - 5.5) + '" data-fit="' + (b.w - 10) + '">' + esc(baseName(n.file)) + '</text>';
    }
    out += pullMarks(n, b);
    if (n.state === 'changed') out += '<text class="chg" x="' + (b.x + b.w - 4) + '" y="' + (b.y - 4) + '" text-anchor="end">' + words.changed + '</text>';
    if (n.state === 'same') out += '<text class="chg" x="' + (b.x + b.w - 4) + '" y="' + (b.y - 4) + '" text-anchor="end" style="fill:var(--muted)">' + words.unchanged + '</text>';
    if (n.state === 'removed') out += '<text class="chg" x="' + (b.x + b.w - 4) + '" y="' + (b.y - 4) + '" text-anchor="end" style="fill:var(--del-ink)">' + words.removed + '</text>';
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
    const c = comps[s.source] ? comps[s.source].color : 'grey';
    const at = s.labelAt || longestMidpoint(s.points);
    const top = at[1] - (s.lines.length - 1) * 15 + 8;
    return '<path class="summary" d="' + polyPath(s.points) + '" style="stroke:var(--c-' + c + ')" marker-end="url(#m-pkg-' + c + ')"' +
      (s.both ? ' marker-start="url(#m-pkgs-' + c + ')"' : '') + '></path>' +
      '<text class="summary-label" x="' + at[0] + '" y="' + top + '" style="fill:var(--c-' + c + ')">' +
      s.lines.map((l, i) => '<tspan x="' + at[0] + '" dy="' + (i ? 30 : 0) + '">' + esc(l) + '</tspan>').join('') + '</text>';
  }
  function pkgArrowSVG(a) {
    const c = comps[a.comp].color;
    const lp = a.labelAt || longestMidpoint(a.points);
    return '<path class="pkg-arrow" d="' + polyPath(a.points) + '" style="stroke:var(--c-' + c + ')" marker-end="url(#m-pkg-' + c + ')"></path>' +
      '<text class="pkg-arrow-label" x="' + lp[0] + '" y="' + (lp[1] + 5) + '" style="fill:var(--c-' + c + ')">' +
      (a.words ? esc(a.words) + ' <tspan class="sub">· ' + esc(a.countText) + '</tspan>' : '<tspan class="sub">' + esc(a.countText) + '</tspan>') + '</text>';
  }

  function diagram() {
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
      const cc = color(c.id);
      out += '<rect class="comp-frame" data-comp="' + esc(c.id) + '" x="' + b.x + '" y="' + b.y + '" width="' + b.w + '" height="' + b.h + '" rx="14" style="stroke:' + cc +
        ';fill:color-mix(in srgb, ' + cc + ' 6%, var(--bg))' + (c.context ? ';stroke-dasharray:14 10' : '') + '"></rect>' +
        '<g class="comp-head" data-comp="' + esc(c.id) + '" tabindex="0" role="link" aria-label="' + (c.context ? words.readCodeOf(esc(c.title)) : words.openDiffFor(esc(c.title))) + '">' +
        '<rect x="' + b.x + '" y="' + b.y + '" width="' + b.w + '" height="88" fill="transparent"></rect>' +
        '<text class="comp-title" x="' + (b.x + 24) + '" y="' + (b.y + 44) + '" style="fill:' + cc + '">' + esc(c.title) + '</text>' +
        '<text class="comp-tag" x="' + (b.x + 26) + '" y="' + (b.y + 72) + '">' + esc(c.tag + ' · ' + countWords(compTotals, c)) + '</text>' +
        '<text class="comp-open" x="' + (b.x + b.w - 24) + '" y="' + (b.y + 44) + '" text-anchor="end" style="fill:' + cc + '">' + (c.context ? words.readCode : english.diagram.openDiff) + '</text>' +
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
    return out;
  }

  function overlay() {
    // A component's far-zoom title may reach past its frame, so a small component's stays as large as any other's.
    const farTitle = layout.farTitle || { size: 150, overhang: 0 };
    return pkgList.filter(p => !p.bare).map(p =>
      '<text data-w="' + p.box.w + '" data-base="15" x="' + (p.box.x + p.box.w / 2) + '" y="' + (p.box.y + p.box.h / 2) + '" style="fill:' + color(p.comp) + '">' + esc(p.title) + '</text>'
    ).join('') + model.components.map(c => {
      const b = layout.components[c.id];
      if (!b) return '';
      return '<text class="above" data-comp="' + esc(c.id) + '" data-w="' + (b.w + farTitle.overhang * 2) + '" data-base="26" data-max="' + farTitle.size + '" x="' + (b.x + b.w / 2) + '" y="' + (b.y - 24) + '" style="fill:' + color(c.id) + '">' + esc(c.title) + '</text>';
    }).join('') + model.areas.map(a => {
      const b = layout.areas[a.id];
      if (!b) return '';
      return '<text class="above start" data-w="' + b.w + '" data-base="34" data-max="130" x="' + (b.x + 20) + '" y="' + (b.y - 28) + '" style="fill:var(--ink)">' + esc(a.title) + '</text>';
    }).join('');
  }

  // A link's words: a has link names the properties that hold its target.
  function linkWords(e) {
    const said = e.label || words.kinds[e.kind];
    return e.kind === 'has' && e.properties ? said + ' ' + e.properties.join(', ') : said;
  }

  function hoverEdges(rels) {
    const parts = [];
    rels.forEach(e => {
      const a = boxOf(e.source), b = boxOf(e.target);
      if (!a || !b) return;
      const ca = [a.x + a.w / 2, a.y + a.h / 2], cb = [b.x + b.w / 2, b.y + b.h / 2];
      const p1 = anchor(a, cb), p2 = anchor(b, ca);
      const mx = (p1[0] + p2[0]) / 2, my = (p1[1] + p2[1]) / 2;
      const dx = p2[0] - p1[0], dy = p2[1] - p1[1], len = Math.hypot(dx, dy) || 1;
      const bend = Math.min(80, len * 0.12);
      const qx = mx - dy / len * bend, qy = my + dx / len * bend;
      parts.push('<path class="hover-edge ' + e.kind + '" d="M' + p1[0] + ',' + p1[1] + ' Q' + qx + ',' + qy + ' ' + p2[0] + ',' + p2[1] + '"' + markersFor(e.kind, 'h') + '></path>');
      parts.push('<text class="hover-label" x="' + (mx - dy / len * bend / 2) + '" y="' + (my + dx / len * bend / 2 + 4) + '">' + esc(linkWords(e)) + '</text>');
    });
    return parts.join('');
  }

  function properties(id, rels) {
    const n = nodes[id];
    if (!n || n.kind === 'artifact' || !n.properties) return null;
    const holds = name => rels.filter(e => e.source === id && e.kind === 'has' && (e.properties || []).includes(name))
      .map(e => (nodes[e.target] ? nodes[e.target].name : e.target));
    return '<p class="props-head"><b>' + esc(n.name) + '</b> &middot; ' + words.storedProperties(n.properties.length) + '</p>' +
      (n.properties.length ? '<table class="props"><tbody>' + n.properties.map(p => '<tr>' +
        '<td class="prop-name">' + esc(p.name) + '</td>' +
        '<td class="prop-type">' + esc(p.type || '') + (p.inferred ? ' <span class="prop-inferred" title="' + words.inferredTitle + '">' + words.inferred + '</span>' : '') + '</td>' +
        '<td class="prop-line">' + words.line(p.line) + '</td>' +
        '<td class="prop-holds">' + holds(p.name).map(name => words.holds(esc(name))).join(', ') + '</td></tr>').join('') + '</tbody></table>' : '');
  }

  function faintEdges() {
    const drawn = {};
    layout.routes.forEach(r => { drawn[r.source + '>' + r.target] = true; });
    return model.edges.filter(e => !drawn[e.source + '>' + e.target]).map(e => {
      const a = boxOf(e.source), b = boxOf(e.target);
      if (!a || !b) return '';
      const p1 = anchor(a, [b.x + b.w / 2, b.y + b.h / 2]), p2 = anchor(b, [a.x + a.w / 2, a.y + a.h / 2]);
      return '<path class="edge faint ' + e.kind + '" d="M' + p1[0] + ',' + p1[1] + ' L' + p2[0] + ',' + p2[1] + '"></path>';
    }).join('');
  }

  function compList() {
    const railGroups = [];
    model.components.forEach(c => { if (railGroups.indexOf(c.group) < 0) railGroups.push(c.group); });
    const areaTitles = model.areas.map(a => a.title);
    return railGroups.map(g =>
      (g ? '<li class="comp-group' + (areaTitles.indexOf(g) >= 0 ? ' boxed' : '') + '">' + esc(g) + '</li>' : '') +
      model.components.filter(c => c.group === g).map(c =>
        '<li><button class="comp-item" data-comp="' + esc(c.id) + '">' + swatch(comps, c, 'swatch') +
        '<span class="t">' + esc(c.title) + '</span><span class="n">' + countWords(compTotals, c, true) + '</span>' +
        (c.description ? '<span class="d">' + esc(c.description) + '</span>' : '') + '</button></li>').join('')
    ).join('');
  }

  function legendRow(inner, said) {
    return '<svg width="46" height="18" viewBox="0 0 46 18">' + markerDefs() + inner + '</svg><span>' + said + '</span>';
  }
  function legend() {
    const states = {};
    model.nodes.forEach(n => { states[n.state] = true; });
    return legendRow('<line x1="4" y1="9" x2="40" y2="9" class="edge has" marker-start="url(#m-diamond)"></line>', words.legend.holds) +
      legendRow('<line x1="2" y1="9" x2="40" y2="9" class="edge realize" marker-end="url(#m-tri)"></line>', words.legend.conforms) +
      legendRow('<line x1="2" y1="9" x2="40" y2="9" class="edge inherit" marker-end="url(#m-tri)"></line>', words.legend.inherits) +
      legendRow('<line x1="2" y1="9" x2="40" y2="9" class="edge use" marker-end="url(#m-open)"></line>', words.legend.uses) +
      legendRow('<rect x="3" y="2" width="38" height="14" rx="4" fill="var(--surface)" stroke="var(--ink)"></rect>', words.legend.added) +
      (states.changed ? legendRow('<rect x="3" y="2" width="38" height="14" rx="4" fill="var(--surface)" stroke="var(--changed)" stroke-width="2"></rect>', words.legend.changed) : '') +
      (states.same || states.context ? legendRow('<rect x="3" y="2" width="38" height="14" rx="4" fill="var(--bg)" stroke="var(--muted)" stroke-dasharray="4 3"></rect>', words.legend.unchanged) : '') +
      (states.removed ? legendRow('<rect x="3" y="2" width="38" height="14" rx="4" fill="var(--del-bg)" stroke="var(--del-ink)" stroke-dasharray="4 3"></rect>', words.legend.removed) : '') +
      (pulls.members.length ? '<span class="pps">' + pullPill(pulls, pulls.members[0].pull) + '</span><span>' + words.legend.pull + '</span>' : '');
  }

  return { diagram, overlay, hoverEdges, properties, faintEdges, compList, legend };
}
