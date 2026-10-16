/**
 * The cards drawn under the lines of a diff: a conversation with Claude, a
 * comment queued for the pull request, and a review thread already on
 * GitHub. A conversation and a GitHub thread fold to one line, which still
 * says who wrote what and how it stands.
 *
 * createNoteCards(ctx) returns:
 *   conversation(t, open)  a conversation with Claude, open or folded
 *   queued(n)              a queued comment, with Edit and Drop
 *   github(t, open)        a GitHub thread: its comments, whether it is
 *                          resolved or outdated, and its address there
 *   textArea(placeholder, value), actions(queueOff)
 *                          the box and the buttons a composer and a
 *                          conversation's reply share
 */
import { esc, plural, when, plainText } from '../lib/text.mjs';
import { safeMarkdown } from '../lib/markdown.mjs';
import { linesWords } from './line-range.mjs';

const PREVIEW = 140;

export function createNoteCards(ctx) {
  function textArea(placeholder, value) {
    return '<textarea rows="3" placeholder="' + esc(placeholder) + '">' + esc(value || '') + '</textarea>';
  }

  function actions(queueOff) {
    return '<div class="note-actions">' +
      '<button type="button" class="nbtn primary" data-act="ask">Ask Claude</button>' +
      (ctx.state.queue ? '<button type="button" class="nbtn" data-act="queue"' + (queueOff ? ' disabled title="' + esc(queueOff) + '"' : '') + '>Queue comment</button>' : '') +
      '<span class="note-status" aria-live="polite"></span></div>';
  }

  function head(parts, preview) {
    return '<summary class="note-head"><span class="chev" aria-hidden="true">&#9654;</span>' + parts.join('') +
      (preview ? '<span class="note-preview">' + esc(preview.length > PREVIEW ? preview.slice(0, PREVIEW) + '…' : preview) + '</span>' : '') + '</summary>';
  }

  function message(m) {
    if (m.from === 'you') return '<div class="msg you"><p class="who">You</p><div class="msg-body plain">' + esc(m.text) + '</div></div>';
    if (m.error) return '<div class="msg claude failed"><p class="who">Claude</p><div class="msg-body">No answer: ' + esc(m.error) + '</div></div>';
    return '<div class="msg claude"><p class="who">Claude</p><div class="msg-body md">' + safeMarkdown(m.text) + '</div>' +
      (ctx.state.queue ? '<button type="button" class="nbtn quiet small" data-act="use">Use as comment</button>' : '') + '</div>';
  }

  function conversation(t, open) {
    const waiting = t.messages.length && t.messages[t.messages.length - 1].from === 'you';
    const asked = t.messages.filter(m => m.from === 'you').length;
    return '<details class="thread" data-thread="' + t.id + '"' + (open ? ' open' : '') + '>' +
      head([
        '<span class="note-tag">Claude</span>',
        '<span class="note-what">' + esc(linesWords(t)) + '</span>',
        '<span class="note-count">' + (waiting ? 'waiting for an answer' : plural(asked, 'question')) + '</span>',
      ], t.messages[0].text) +
      '<div class="note-body">' + t.messages.map(message).join('') +
      (waiting ? '<p class="thinking">Claude is reading the file and the diff&hellip;</p>' : '') +
      '<div class="reply">' + textArea(ctx.state.queue ? 'Ask a follow-up, or write a comment for the pull request' : 'Ask a follow-up') +
      actions(false) + '</div></div></details>';
  }

  function queued(n) {
    const c = n.comment;
    const stacked = ctx.pulls.members.length > 0;
    return '<div class="queued-note" data-comment="' + c.id + '">' +
      '<div class="qn-head"><span class="qn-tag">Queued ' + c.id + '</span><span class="qn-lines">' + esc(linesWords(n)) + '</span>' +
      (stacked ? '<span class="qn-to">&rarr; ' + ctx.pulls.pullPill(n.member.pull) + '</span>' : '') +
      (c.stale ? '<span class="qn-stale" title="' + esc(c.stale) + '">stale</span>' : '') +
      '<span class="qn-tools"><button type="button" class="nbtn quiet small" data-act="edit">Edit</button>' +
      '<button type="button" class="nbtn quiet small" data-act="drop">Drop</button></span></div>' +
      '<p class="qn-body">' + esc(c.body) + '</p></div>';
  }

  function author(c) {
    return '<b>' + esc(c.author) + '</b>' + (c.bot ? '<span class="gh-bot">bot</span>' : '') + (c.author === ctx.state.me ? '<span class="gh-you">you</span>' : '');
  }

  function where(t) {
    if (t.file) return 'the whole file';
    if (t.outdated || !t.line) return 'was on line ' + (t.original_line || '?');
    return linesWords(t);
  }

  function github(t, open) {
    const people = [...new Set(t.comments.map(c => c.author))];
    const others = people.length > 1 ? ' and ' + plural(people.length - 1, 'other') : '';
    const count = t.comments.length + t.more;
    const first = t.comments[0];
    return '<details class="gh-thread' + (t.resolved || t.outdated ? ' gh-quiet' : '') + '" data-gh="' + esc(t.id) + '"' + (open ? ' open' : '') + '>' +
      head([
        '<span class="note-tag gh-tag">GitHub</span>',
        '<span class="gh-who">' + esc(people[0] || 'ghost') + esc(others) + '</span>',
        ctx.pulls.members.length ? ctx.pulls.pullPill(t.pull) : '',
        '<span class="note-what">' + esc(where(t)) + '</span>',
        '<span class="note-count">' + plural(count, 'comment') + '</span>',
        t.resolved ? '<span class="gh-state">Resolved</span>' : '',
        t.outdated ? '<span class="gh-state">Outdated</span>' : '',
      ], first ? plainText(first.body) : '') +
      '<div class="note-body">' +
      (t.hunk && t.outdated ? '<pre class="gh-hunk">' + esc(t.hunk) + '</pre>' : '') +
      '<ol class="gh-comments">' + t.comments.map(c =>
        '<li class="gh-comment"><p class="who">' + author(c) + '<span class="gh-when">' + esc(when(c.at)) + '</span></p>' +
        '<div class="msg-body md">' + safeMarkdown(c.body) + '</div></li>').join('') + '</ol>' +
      '<p class="gh-foot">' + (t.more ? esc(plural(t.more, 'more comment')) + ' on GitHub. ' : '') +
      (t.resolved && t.resolved_by ? 'Resolved by ' + esc(t.resolved_by) + '. ' : '') +
      (first && /^https:\/\//.test(first.url) ? '<a href="' + esc(first.url) + '" target="_blank" rel="noopener">Open on GitHub</a>' : '') + '</p>' +
      '</div></details>';
  }

  return { conversation, queued, github, textArea, actions };
}
