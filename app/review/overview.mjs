/**
 * The Review view: the pull request, or the stack, at the top; the written
 * review when one is attached; the Queued comments with the review buttons;
 * and every conversation with Claude. Beside it, the way to the diagram, each
 * pull request of a stack, the components, and what is on the page.
 *
 * createOverview(ctx) returns render(message), which draws the view from
 * ctx.state again, with `message` saying what the last action did.
 */
import { esc, fmt, plural, short, when } from '../lib/text.mjs';
import { trustedMarkdown } from '../lib/markdown.mjs';
import { send, reviewPath } from '../lib/api.mjs';

export function createOverview(ctx) {
  const { files, contextFiles, totals, model } = ctx.data;
  const pulls = ctx.pulls;
  const body = document.getElementById('review-body');
  const side = document.getElementById('review-side');
  const view = document.getElementById('review-view');

  // A file:line in the review, such as src/log.ts:31 or log.ts:31-40, points at a file in the diff when exactly one path ends with it.
  function locate(text) {
    const m = /^(.+?\.[A-Za-z0-9]+):(\d+)(?:[-–](\d+))?$/.exec(text.trim());
    if (!m) return null;
    const all = Object.keys(files).concat(Object.keys(contextFiles));
    const hits = all.filter(p => p === m[1] || p.slice(-(m[1].length + 1)) === '/' + m[1]);
    return hits.length === 1 ? { path: hits[0], line: +m[2] } : null;
  }

  function changeTotals(label) {
    return '<span>' + label + plural(totals.files, 'file') + ' <span class="plus">+' + fmt(totals.adds) + '</span> <span class="minus">&minus;' + fmt(totals.dels) + '</span></span>';
  }

  function pullHeader(m) {
    const state = m.state && m.state !== 'open' ? ' &middot; <span class="st-ended">' + esc(m.state) + '</span>' : '';
    return '<header class="pr-head">' +
      '<p class="pr-kicker"><a href="' + esc(m.url) + '" target="_blank" rel="noopener">Pull request #' + esc(m.pull) + '</a> by <b>' + esc(m.author) + '</b>' + state + '</p>' +
      '<h2 class="pr-title">' + esc(m.title) + '</h2>' +
      '<p class="pr-meta"><span>Base <b>' + esc(m.base) + '</b></span><span>Head <b>' + esc(short(m.head)) + '</b></span>' + changeTotals('') + '</p>' +
      '</header>';
  }

  function stackHeader(title) {
    const top = pulls.members[pulls.members.length - 1];
    return '<header class="pr-head">' +
      '<p class="pr-kicker">A stack of ' + pulls.members.length + ' pull requests, each built on the one before, reviewed together</p>' +
      '<h2 class="pr-title">' + esc(title) + '</h2>' +
      '<ol class="stack-list">' + pulls.members.map(m => {
        const state = m.state && m.state !== 'open' ? m.state : '';
        const name = m.url ? '<a class="st-title" href="' + esc(m.url) + '" target="_blank" rel="noopener">' + esc(m.title || 'Pull request #' + m.pull) + '</a>'
          : '<span class="st-title">' + esc(m.title || 'Pull request #' + m.pull) + '</span>';
        return '<li>' + pulls.pullPill(m.pull, m.url) + name + '<span class="st-meta">' +
          (m.author ? 'by <b>' + esc(m.author) + '</b> &middot; ' : '') + (m.base ? 'on <b>' + esc(m.base) + '</b> &middot; ' : '') +
          'head <b>' + esc(short(m.head)) + '</b> &middot; ' + plural(m.files, 'file') + ' <span class="plus">+' + fmt(m.adds) + '</span> <span class="minus">&minus;' + fmt(m.dels) + '</span>' +
          (state ? ' &middot; <span class="st-ended">' + esc(state) + '</span>' : '') + '</span></li>';
      }).join('') + '</ol>' +
      '<p class="pr-meta"><span>Base <b>' + esc(ctx.state.base.ref) + '</b></span><span>Head <b>' + esc(short(top.head)) + '</b></span>' + changeTotals('Combined: ') + '</p>' +
      ctx.data.stack.warnings.map(w => '<p class="warn-line">' + esc(w) + '</p>').join('') +
      '</header>';
  }

  function movedNote(moved) {
    const list = (moved.stack || []).map(p => '#' + p);
    return '<p class="moved-note">This review moved: ' + (list.length ? esc(list[0]) + ' to ' + esc(list[list.length - 1]) : 'its pull request') +
      ' are reviewed together on <a href="/review/' + esc(moved.to) + '">the stack&rsquo;s review</a>. Its queued comments moved there, and nothing can be sent from here.</p>';
  }

  function sharedPrefix(titles) {
    if (titles.length < 2 || titles.some(t => !t)) return '';
    let n = 0;
    while (n < titles[0].length && titles.every(t => t[n] === titles[0][n])) n++;
    const cut = titles[0].slice(0, n).lastIndexOf(' ') + 1;
    return titles.every(t => t.length > cut) ? titles[0].slice(0, cut) : '';
  }

  function sectionsHTML(written) {
    if (!written) {
      return '<section class="rv-sec" id="rv-none"><p class="rv-quiet">No written review is attached. The diagram and the diffs hold the change; select a line number in a diff to ask Claude about it or to queue a comment.</p></section>';
    }
    return (written.stale ? '<p class="warn-line rv-stale">This written review read head ' + esc(short(written.head)) + '; the pull request has moved on since, so it may not match the diff.</p>' : '') +
      written.sections.map(s => '<section class="rv-sec" id="rv-' + esc(s.id) + '"><h2>' + esc(s.title) + '</h2><div class="rv-body">' + trustedMarkdown(s.markdown) + '</div></section>').join('');
  }

  function conversationsHTML() {
    const threads = ctx.state.threads;
    if (!threads.length) return '';
    return '<section class="rv-sec" id="conversations"><h2>Conversations</h2><ul class="convo-list">' + threads.map(t => {
      const first = t.messages[0];
      const last = t.messages[t.messages.length - 1];
      const answer = last.from === 'claude' ? (last.error ? 'No answer: ' + last.error : plainText(last.text)) : 'Waiting for Claude…';
      const lines = t.start_line ? t.start_line + '-' + t.line : t.line;
      return '<li><a href="#" class="convo-at" data-go-file="' + esc(t.path) + '" data-go-line="' + t.line + '" data-go-side="' + t.side + '">' + esc(t.path) + ':' + lines + '</a>' +
        '<span class="convo-when">' + esc(when(t.created_at)) + ' &middot; ' + plural(t.messages.length, 'message') + '</span>' +
        '<p class="convo-q">' + esc(first.text) + '</p><p class="convo-a">' + esc(answer.length > 220 ? answer.slice(0, 220) + '…' : answer) + '</p></li>';
    }).join('') + '</ul></section>';
  }

  // Markdown read as plain words, for a short preview.
  function plainText(markdown) {
    return markdown.replace(/```[\s\S]*?```/g, ' ').replace(/\[([^\]]*)\]\([^)]*\)/g, '$1').replace(/^\s*[-+]\s+/gm, '')
      .replace(/[`*_>#]/g, '').replace(/\s+/g, ' ').trim();
  }

  function decorate(root) {
    const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
    const hits = [];
    while (walker.nextNode()) {
      const t = walker.currentNode;
      const tags = /\[(Certain|Likely|Guessing)\]/.test(t.nodeValue);
      const mentions = pulls.members.length && /#\d/.test(t.nodeValue) && t.parentNode.closest('.rv-sec') && !t.parentNode.closest('a');
      if ((tags || mentions) && !t.parentNode.closest('code, pre')) hits.push(t);
    }
    hits.forEach(t => {
      const span = document.createElement('span');
      span.innerHTML = esc(t.nodeValue).replace(/\[(Certain|Likely|Guessing)\]/g, (_, w) => '<span class="conf ' + w.toLowerCase() + '">' + w + '</span>')
        .replace(/#(\d+)(?!\d)/g, (whole, n) => (pulls.memberOf[+n] && t.parentNode.closest('.rv-sec') && !t.parentNode.closest('a') ? pulls.pullPill(+n) : whole));
      t.replaceWith(...Array.prototype.slice.call(span.childNodes));
    });
    root.querySelectorAll('.rv-body code').forEach(code => {
      if (code.closest('pre')) return;
      const at = locate(code.textContent);
      if (!at) return;
      const a = document.createElement('a');
      a.className = 'loc';
      a.href = '#';
      a.setAttribute('data-go-file', at.path);
      a.setAttribute('data-go-line', at.line);
      a.textContent = code.textContent;
      a.title = 'Open ' + at.path + ' at line ' + at.line;
      code.replaceWith(a);
    });
  }

  function render(message) {
    const state = ctx.state;
    const written = state.written;
    const title = (written && written.title) || state.title;
    body.innerHTML = (state.moved ? movedNote(state.moved) : '') +
      (pulls.members.length ? stackHeader(title) : pullHeader(state.members[0])) +
      (state.notes && state.notes.length ? '<div class="load-notes">' + state.notes.map(n => '<p class="warn-line">' + esc(n) + '</p>').join('') + '</div>' : '') +
      sectionsHTML(written) +
      '<section id="queued-comments"></section>' +
      conversationsHTML();
    decorate(body);
    ctx.queuePanel.mount(document.getElementById('queued-comments'), message);

    const shared = sharedPrefix(pulls.members.map(m => m.title));
    const toc = (written ? written.sections.map(s => ['rv-' + s.id, s.title]) : []).concat([['queued-comments', 'Queued comments']])
      .concat(state.threads.length ? [['conversations', 'Conversations']] : []);
    side.innerHTML =
      '<div class="side-open"><button class="back" data-go="map">Open the diagram &rarr;</button></div>' +
      (pulls.members.length ? '<div><p class="rail-h">Show on the diagram</p><div class="side-pulls">' + pulls.members.map(m =>
        '<button class="side-pull" data-pull="' + m.pull + '" title="Show #' + m.pull + ' on the diagram' + (m.title ? ': ' + esc(m.title) : '') + '">' + pulls.pullPill(m.pull) + '<span>' + esc((m.title || '').slice(shared.length)) + '</span></button>').join('') + '</div></div>' : '') +
      '<div><p class="rail-h">The change</p><ul class="comp-list">' + model.components.map(c =>
        '<li><button class="comp-item" data-comp="' + esc(c.id) + '">' + ctx.diagram.swatch(c, 'swatch') +
        '<span class="t">' + esc(c.title) + '</span><span class="n">' + ctx.diagram.countWords(c, true) + '</span></button></li>').join('') + '</ul></div>' +
      '<div class="toc-block"><p class="rail-h">On this page</p><ul class="toc">' + toc.map(([id, name]) =>
        '<li><a href="#' + esc(id) + '" data-section="' + esc(id) + '">' + esc(name) + '</a></li>').join('') + '</ul></div>' +
      '<div class="side-tools"><p class="rail-h">This review</p>' +
      '<p class="side-line">Loaded ' + esc(when(state.loaded_at)) + (state.account ? ' as <b>' + esc(state.account) + '</b>' : '') + '</p>' +
      (state.moved ? '' : '<button class="btn" data-refresh>Load again from GitHub</button><p class="side-line" data-refresh-status aria-live="polite"></p>') +
      '<p class="side-line"><a href="/">Every review</a></p></div>';
  }

  view.addEventListener('click', async e => {
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
      document.getElementById(toc.getAttribute('data-section')).scrollIntoView({ block: 'start', behavior: 'smooth' });
      return;
    }
    const act = e.target.closest('.qc-item [data-act]');
    if (act) { ctx.queueActions[act.getAttribute('data-act')](+act.closest('.qc-item').getAttribute('data-comment'), act.closest('.qc-item')); return; }
    const item = e.target.closest('.comp-item');
    if (item) { ctx.open.component(item.getAttribute('data-comp')); return; }
    const sidePull = e.target.closest('.side-pull');
    if (sidePull) { ctx.setPullFilter(+sidePull.getAttribute('data-pull')); ctx.go('#map'); return; }
    if (e.target.closest('[data-go="map"]')) { ctx.go('#map'); return; }
    const refresh = e.target.closest('[data-refresh]');
    if (refresh) {
      const status = side.querySelector('[data-refresh-status]');
      refresh.disabled = true;
      status.textContent = 'Fetching the pull request' + (pulls.members.length ? 's' : '') + ' and drawing them again…';
      try {
        const done = await send('POST', reviewPath(ctx.state.id, '/refresh'));
        location.assign('/review/' + done.id + location.hash);
      } catch (error) {
        refresh.disabled = false;
        status.textContent = error.message;
        status.classList.add('bad');
      }
    }
  });

  return { render };
}
