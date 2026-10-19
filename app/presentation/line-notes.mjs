/**
 * The markup of the notes on the lines of a diff: the composer a line number
 * opens, and each note drawn under its lines.
 *
 * threadsOn(state, path), githubOn(state, path) and queuedOn(state, path)
 * are a file's conversations, GitHub threads and queued comments, and
 * countBadge(state, path) the file header's count of them.
 *
 * createNotesMarkup(ctx, folds) returns, from ctx.state and the folds a
 * person left:
 *   whereOf(composer)        where a comment on the composer's lines goes,
 *                            as far as the server has said: {words, off,
 *                            choose}
 *   chooseHTML(composer, choose)
 *                            the pull request a comment on an unchanged
 *                            line of a stack goes on, as a choice
 *   composerHTML(composer, touch)
 *                            the composer, its hint for a touch screen or not
 *   notesOn(path)            each note of a file, {side, line, html, cls}
 *   noteRow(columns, cls, html), mdNote(cls, html), awayBand(notes)
 *                            a note under a line of a table, under a passage
 *                            of the Rendered view, and the notes on no line
 *                            shown
 */
import { esc } from '../lib/text.mjs';
import { createNoteCards } from './note-cards.mjs';
import { english } from '/i18n/english.mjs';

const QUOTE = 200;
const words = english.web.notes;

export function threadsOn(state, path) {
  return state.threads.filter(t => t.path === path);
}

export function githubOn(state, path) {
  return state.github.filter(t => (t.at ? t.at.path : t.path) === path);
}

export function queuedOn(state, path) {
  const out = [];
  const q = state.queue;
  if (!q) return out;
  q.members.concat(q.earlier).forEach(m => m.comments.forEach(c => {
    const at = c.shown || c;
    if (at.path === path) out.push({ member: m, comment: c, side: at.side, line: at.line, start_line: at.start_line, start_side: at.start_side });
  }));
  return out;
}

export function countBadge(state, path) {
  const g = githubOn(state, path).length, t = threadsOn(state, path).length, q = queuedOn(state, path).length;
  if (!g && !t && !q) return '';
  const counts = english.web.counts;
  const said = [g ? counts.githubThreads(g) : '', t ? counts.conversations(t) : '', q ? counts.queuedComments(q) : ''].filter(Boolean).join(', ');
  return '<span class="ncount" title="' + esc(said) + '">' + esc(said) + '</span>';
}

export function createNotesMarkup(ctx, folds) {
  const cards = createNoteCards(ctx);

  function whereOf(c) {
    if (!ctx.state.queue) return { words: '' };
    if (!c.where) return { words: words.checking, off: words.checkingTitle };
    if (c.where.refused) return { words: c.where.refused, off: c.where.refused };
    return { words: c.where.words, choose: c.where.choose || c.choices };
  }

  function chooseHTML(c, choose) {
    if (!choose) return '';
    return '<label class="composer-choose">' + words.commentOn + '<select data-pull>' + choose.map(x =>
      '<option value="' + x.pull + '"' + (c.pull === x.pull ? ' selected' : '') + '>' + words.option(x.pull, x.line) + '</option>').join('') + '</select></label>';
  }

  function composerHTML(c, touch) {
    const to = whereOf(c);
    const hint = c.quote !== undefined ? words.passageHint
      : touch ? words.tapHint : words.dragHint;
    return '<div class="composer">' +
      '<p class="composer-where"><b>' + esc(english.review.lines(c)) + '</b><span class="where-words">' + (to.words ? ' &middot; ' + esc(to.words) : '') + '</span>' +
      ' <span class="hint-inline">' + hint + '</span></p>' +
      (c.quote ? '<blockquote class="composer-quote">' + esc(c.quote.length > QUOTE ? c.quote.slice(0, QUOTE) + '…' : c.quote) + '</blockquote>' : '') +
      '<div class="choose-slot">' + chooseHTML(c, to.choose) + '</div>' +
      cards.textArea(ctx.state.queue ? words.askOrComment : words.askAbout, c.text) +
      cards.actions(to.off, c.sending ? words.reading : null) + '<button type="button" class="nbtn quiet close" data-act="cancel" aria-label="' + words.close + '"' + (c.sending ? ' disabled' : '') + '>' + words.cancel + '</button></div>';
  }

  function githubCard(t) {
    return cards.github(t, folds.isOpen('g' + t.id, !t.resolved && !t.outdated));
  }

  function notesOn(path) {
    return githubOn(ctx.state, path).map(t => ({ ...t.at, html: githubCard(t), cls: 'gh-row' }))
      .concat(threadsOn(ctx.state, path).map(t => ({ side: t.side, line: t.line, html: cards.conversation(t, folds.isOpen('t' + t.id, true)), cls: 'thread-row' })))
      .concat(queuedOn(ctx.state, path).map(n => ({ side: n.side, line: n.line, html: cards.queued(n), cls: 'queued-row' })));
  }

  function noteRow(columns, cls, html) {
    return '<tr class="note-row ' + cls + '"><td colspan="' + columns + '">' + html + '</td></tr>';
  }

  function mdNote(cls, html) {
    return '<div class="md-note ' + cls + '">' + html + '</div>';
  }

  function awayBand(html) {
    return '<div class="gh-away"><p class="gh-away-h">' + words.onNoLine + '</p>' + html.join('') + '</div>';
  }

  return { whereOf, chooseHTML, composerHTML, notesOn, noteRow, mdNote, awayBand };
}
