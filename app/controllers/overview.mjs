/**
 * The Review view: the review's own page in the app, drawn from ctx.state.
 * Its links open the diffs at their lines, its table of contents scrolls to
 * its sections, and in a written review each pull request it names gets its
 * pill and each `file:line` it names opens there.
 *
 * createOverview(ctx) returns render(message), which draws the view from
 * ctx.state again, with `message` saying what the last action did.
 */
import { views, element } from '../views.mjs';
import { fromHTML } from '../lib/html.mjs';
import { createOverviewMarkup, markedText, locationLink } from '../presentation/overview.mjs';

export function createOverview(ctx) {
  const { files, contextFiles } = ctx.data;
  const pulls = ctx.pulls;
  const markup = createOverviewMarkup(ctx);
  const body = views.reviewBody;
  const side = views.reviewSide;
  const view = views.reviewView;

  function decorate(root) {
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    const hits = [];
    while (walker.nextNode()) {
      const t = walker.currentNode;
      const tags = /\[(Certain|Likely|Guessing)\]/.test(t.nodeValue);
      const mentions = pulls.members.length && /#\d/.test(t.nodeValue) && t.parentNode.closest('.rv-sec') && !t.parentNode.closest('a, .pp');
      if ((tags || mentions) && !t.parentNode.closest('code, pre')) hits.push(t);
    }
    hits.forEach(t => {
      const span = document.createElement('span');
      span.innerHTML = markedText(t.nodeValue, pulls, !!(t.parentNode.closest('.rv-sec') && !t.parentNode.closest('a, .pp')));
      t.replaceWith(...Array.prototype.slice.call(span.childNodes));
    });
    const sections = ctx.state.written ? ctx.state.written.sections : [];
    root.querySelectorAll('.rv-body code').forEach(code => {
      if (code.closest('pre')) return;
      const section = sections.find(s => 'rv-' + s.id === code.closest('.rv-sec').id);
      const at = section && section.links.find(l => l.text === code.textContent.trim());
      if (!at) return;
      code.replaceWith(fromHTML(locationLink(at, code.textContent)));
    });
  }

  function render(message) {
    const state = ctx.state;
    body.innerHTML = markup.body(state);
    decorate(body);
    if (state.queue) ctx.queuePanel.mount(element('queued-comments'), message);
    side.innerHTML = markup.side(state);
  }

  view.addEventListener('click', e => {
    const go = e.target.closest('[data-go-file]');
    if (go) {
      e.preventDefault();
      const path = go.getAttribute('data-go-file');
      const meta = files[path] || contextFiles[path];
      if (meta) ctx.open.component(meta.component, { file: path, line: +go.getAttribute('data-go-line'), side: go.getAttribute('data-go-side') || 'RIGHT' });
      return;
    }
    const toc = e.target.closest('a[data-section]');
    if (toc) {
      e.preventDefault();
      element(toc.getAttribute('data-section')).scrollIntoView({ block: 'start', behavior: 'smooth' });
      return;
    }
    const act = e.target.closest('.qc-item [data-act]');
    if (act) { ctx.queueActions[act.getAttribute('data-act')](+act.closest('.qc-item').getAttribute('data-comment'), act.closest('.qc-item')); return; }
    const item = e.target.closest('.comp-item');
    if (item) { ctx.open.component(item.getAttribute('data-comp')); return; }
    const sidePull = e.target.closest('.side-pull');
    if (sidePull) { ctx.setPullFilter(+sidePull.getAttribute('data-pull')); ctx.go('#map'); return; }
    if (e.target.closest('[data-go="map"]')) ctx.go('#map');
  });

  return { render };
}
