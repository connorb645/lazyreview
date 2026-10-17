/**
 * The markup of the Review view: the pull request, or the stack, at the top;
 * the written review when one is attached; the place for the Queued
 * comments; the review threads already on GitHub; and every conversation
 * with Claude. Beside it, the way to the diagram, each pull request of a
 * stack, the components, and what is on the page. A branch's review shows
 * the branch at the top and its conversations, with no queue: it is read
 * only.
 *
 * createOverviewMarkup(ctx) returns body(state) and side(state), each the
 * HTML of its half of the view for a review's state.
 *
 * markedText(text, pulls, pills) is a written review's text with its
 * confidence tags marked, and with `pills` each pull request of the stack
 * it names as its pill. locationLink(at, text) is a `file:line` of a written
 * review as the link that opens it.
 */
import { esc, fmt, short, when, plainText } from '../lib/text.mjs';
import { trustedMarkdown } from '../lib/markdown.mjs';
import { pullPill } from './pulls.mjs';
import { swatch, countWords } from './diagram.mjs';
import { english } from '/i18n/english.mjs';

const words = english.web.review;
const counts = english.web.counts;

export function createOverviewMarkup(ctx) {
  const { totals, model, comps, compTotals } = ctx.data;
  const pulls = ctx.pulls;

  function changeTotals(label) {
    return '<span>' + label + counts.files(totals.files) + ' <span class="plus">+' + fmt(totals.adds) + '</span> <span class="minus">&minus;' + fmt(totals.dels) + '</span></span>';
  }

  function pullHeader(m) {
    const state = m.state && m.state !== 'open' ? ' &middot; <span class="st-ended">' + esc(m.state) + '</span>' : '';
    return '<header class="pr-head">' +
      '<p class="pr-kicker">' + words.by('<a href="' + esc(m.url) + '" target="_blank" rel="noopener">' + words.pull(esc(m.pull)) + '</a>', esc(m.author)) + state + '</p>' +
      '<h2 class="pr-title">' + esc(m.title) + '</h2>' +
      '<p class="pr-meta"><span>' + words.base(esc(m.base)) + '</span><span>' + words.head(esc(short(m.head))) + '</span>' + changeTotals('') + '</p>' +
      '</header>';
  }

  function stackHeader(title) {
    const top = pulls.members[pulls.members.length - 1];
    return '<header class="pr-head">' +
      '<p class="pr-kicker">' + words.stack(pulls.members.length) + '</p>' +
      '<h2 class="pr-title">' + esc(title) + '</h2>' +
      '<ol class="stack-list">' + pulls.members.map(m => {
        const state = m.state && m.state !== 'open' ? m.state : '';
        const name = m.url ? '<a class="st-title" href="' + esc(m.url) + '" target="_blank" rel="noopener">' + esc(m.title || words.pull(m.pull)) + '</a>'
          : '<span class="st-title">' + esc(m.title || words.pull(m.pull)) + '</span>';
        return '<li>' + pullPill(pulls, m.pull, m.url) + name + '<span class="st-meta">' +
          (m.author ? words.byAuthor(esc(m.author)) + ' &middot; ' : '') + (m.base ? words.onBase(esc(m.base)) + ' &middot; ' : '') +
          words.atHead(esc(short(m.head))) + ' &middot; ' + counts.files(m.files) + ' <span class="plus">+' + fmt(m.adds) + '</span> <span class="minus">&minus;' + fmt(m.dels) + '</span>' +
          (state ? ' &middot; <span class="st-ended">' + esc(state) + '</span>' : '') + '</span></li>';
      }).join('') + '</ol>' +
      '<p class="pr-meta"><span>' + words.base(esc(ctx.state.base.ref)) + '</span><span>' + words.head(esc(short(top.head))) + '</span>' + changeTotals(words.combined) + '</p>' +
      ctx.data.stack.warnings.map(w => '<p class="warn-line">' + esc(w) + '</p>').join('') +
      '</header>';
  }

  function branchHeader(state) {
    return '<header class="pr-head">' +
      '<p class="pr-kicker">' + words.branch(esc(state.head.ref), esc(state.repo)) + '</p>' +
      '<h2 class="pr-title">' + esc(state.title) + '</h2>' +
      '<p class="pr-meta"><span>' + words.base(esc(state.base.ref)) + '</span><span>' + words.head(esc(short(state.head.sha))) + '</span>' + changeTotals('') + '</p>' +
      '</header>';
  }

  function movedNote(moved) {
    const list = (moved.stack || []).map(p => '#' + p);
    return '<p class="moved-note">' + words.moved(list.length ? words.range(esc(list[0]), esc(list[list.length - 1])) : words.itsPull,
      '<a href="/review/' + esc(moved.to) + '">' + words.stacksReview + '</a>') + '</p>';
  }

  function sharedPrefix(titles) {
    if (titles.length < 2 || titles.some(t => !t)) return '';
    let n = 0;
    while (n < titles[0].length && titles.every(t => t[n] === titles[0][n])) n++;
    const cut = titles[0].slice(0, n).lastIndexOf(' ') + 1;
    return titles.every(t => t.length > cut) ? titles[0].slice(0, cut) : '';
  }

  function sectionsHTML(written) {
    if (!ctx.state.queue) {
      return '<section class="rv-sec" id="rv-none"><p class="rv-quiet">' + words.branchReadOnly + '</p></section>';
    }
    if (!written) {
      return '<section class="rv-sec" id="rv-none"><p class="rv-quiet">' + words.noWritten + '</p></section>';
    }
    return (written.stale ? '<p class="warn-line rv-stale">' + words.staleWritten(esc(short(written.head))) + '</p>' : '') +
      written.sections.map(s => '<section class="rv-sec" id="rv-' + esc(s.id) + '"><h2>' + esc(s.title) + '</h2><div class="rv-body">' + trustedMarkdown(s.markdown) + '</div></section>').join('');
  }

  function conversationsHTML() {
    const threads = ctx.state.threads;
    if (!threads.length) return '';
    return '<section class="rv-sec" id="conversations"><h2>' + words.conversations + '</h2><ul class="convo-list">' + threads.map(t => {
      const first = t.messages[0];
      const last = t.messages[t.messages.length - 1];
      const answer = last.from === 'claude' ? (last.error ? english.web.notes.noAnswer(last.error) : plainText(last.text)) : words.waiting;
      const lines = t.start_line ? t.start_line + '-' + t.line : t.line;
      return '<li><a href="#" class="convo-at" data-go-file="' + esc(t.path) + '" data-go-line="' + t.line + '" data-go-side="' + t.side + '">' + esc(t.path) + ':' + lines + '</a>' +
        '<span class="convo-when">' + esc(when(t.created_at)) + ' &middot; ' + counts.messages(t.messages.length) + '</span>' +
        '<p class="convo-q">' + esc(first.text) + '</p><p class="convo-a">' + esc(answer.length > 220 ? answer.slice(0, 220) + '…' : answer) + '</p></li>';
    }).join('') + '</ul></section>';
  }

  function githubHTML() {
    const threads = ctx.state.github;
    if (!threads.length) return '';
    const open = threads.filter(t => !t.resolved).length;
    return '<section class="rv-sec" id="on-github"><h2>' + words.onGitHub + '</h2>' +
      '<p class="rv-quiet">' + words.githubThreads(counts.reviewThreads(threads.length), fmt(open)) + '</p>' +
      '<ul class="convo-list">' + threads.map(t => {
        const first = t.comments[0];
        const at = t.at || { path: t.path, line: t.original_line || 1, side: t.side || 'RIGHT' };
        const lines = t.file ? '' : ':' + (t.at && t.at.start_line ? t.at.start_line + '-' : '') + (t.at ? t.at.line : t.original_line || '');
        const people = [...new Set(t.comments.map(c => c.author))];
        return '<li><a href="#" class="convo-at" data-go-file="' + esc(at.path) + '" data-go-line="' + at.line + '" data-go-side="' + esc(at.side) + '">' + esc(at.path) + lines + '</a>' +
          '<span class="convo-when">' + (pulls.members.length ? pullPill(pulls, t.pull) + ' ' : '') + esc(people.join(', ')) + ' &middot; ' + counts.comments(t.comments.length + t.more) +
          (t.resolved ? ' &middot; ' + words.resolved : '') + (t.outdated ? ' &middot; ' + words.outdated : '') + '</span>' +
          (first ? '<p class="convo-a">' + esc(clip(plainText(first.body))) + '</p>' : '') + '</li>';
      }).join('') + '</ul></section>';
  }

  function clip(text) {
    return text.length > 220 ? text.slice(0, 220) + '…' : text;
  }

  function body(state) {
    const written = state.written;
    const title = (written && written.title) || state.title;
    const branch = state.kind === 'branch';
    return (state.moved ? movedNote(state.moved) : '') +
      (branch ? branchHeader(state) : pulls.members.length ? stackHeader(title) : pullHeader(state.members[0])) +
      (state.notes && state.notes.length ? '<div class="load-notes">' + state.notes.map(n => '<p class="warn-line">' + esc(n) + '</p>').join('') + '</div>' : '') +
      sectionsHTML(written) +
      (state.queue ? '<section id="queued-comments"></section>' : '') +
      githubHTML() +
      conversationsHTML();
  }

  function side(state) {
    const written = state.written;
    const shared = sharedPrefix(pulls.members.map(m => m.title));
    const toc = (written ? written.sections.map(s => ['rv-' + s.id, s.title]) : []).concat(state.queue ? [['queued-comments', words.queuedComments]] : [])
      .concat(ctx.state.github.length ? [['on-github', words.onGitHub]] : [])
      .concat(state.threads.length ? [['conversations', words.conversations]] : []);
    return '<div class="side-open"><button class="back" data-go="map">' + words.openDiagram + '</button></div>' +
      (pulls.members.length ? '<div><p class="rail-h">' + words.showOnDiagram + '</p><div class="side-pulls">' + pulls.members.map(m =>
        '<button class="side-pull" data-pull="' + m.pull + '" title="' + words.showPull(m.pull, m.title ? esc(m.title) : '') + '">' + pullPill(pulls, m.pull) + '<span>' + esc((m.title || '').slice(shared.length)) + '</span></button>').join('') + '</div></div>' : '') +
      '<div><p class="rail-h">' + words.theChange + '</p><ul class="comp-list">' + model.components.map(c =>
        '<li><button class="comp-item" data-comp="' + esc(c.id) + '">' + swatch(comps, c, 'swatch') +
        '<span class="t">' + esc(c.title) + '</span><span class="n">' + countWords(compTotals, c, true) + '</span></button></li>').join('') + '</ul></div>' +
      (toc.length ? '<div class="toc-block"><p class="rail-h">' + words.onThisPage + '</p><ul class="toc">' + toc.map(([id, name]) =>
        '<li><a href="#' + esc(id) + '" data-section="' + esc(id) + '">' + esc(name) + '</a></li>').join('') + '</ul></div>' : '') +
      '<div class="side-tools"><p class="rail-h">' + words.thisReview + '</p>' +
      '<p class="side-line">' + words.loaded(esc(when(state.loaded_at)), state.account ? esc(state.account) : '') + '</p>' +
      '<p class="side-line"><a href="/">' + english.web.page.everyReview + '</a></p></div>';
  }

  return { body, side };
}

export function markedText(text, pulls, pills) {
  return esc(text).replace(/\[(Certain|Likely|Guessing)\]/g, (_, w) => '<span class="conf ' + w.toLowerCase() + '">' + w + '</span>')
    .replace(/#(\d+)(?!\d)/g, (whole, n) => (pulls.memberOf[+n] && pills ? pullPill(pulls, +n) : whole));
}

export function locationLink(at, text) {
  return '<a class="loc" href="#" data-go-file="' + esc(at.path) + '" data-go-line="' + esc(at.line) + '" title="' + esc(words.openAt(at.path, at.line)) + '">' + esc(text) + '</a>';
}
