/**
 * The markup of the Queued comments section: the comments waiting to go on
 * each pull request, exactly as they will read, the reviews sent from them,
 * and the place of the Add Comments, Approve With Comments and Approve
 * buttons.
 *
 * createQueueMarkup(ctx) returns, from ctx.state:
 *   section(message)         the section's insides, as {html, wired}: each
 *                            place for buttons in `html`, in order, is for
 *                            the pull request at the same place in `wired`
 *   offered(data)            the actions on offer for a pull request
 *   actionButton(action, data), yesButton(action), noButton()
 *                            a press, and the press that confirms it or not
 *   editBox(draft)           the box a queued comment is edited in
 * Text that comes from a person or from GitHub is kept as it is, character
 * for character.
 */
import { esc, when } from '../lib/text.mjs';
import { english } from '/i18n/english.mjs';

const ORDER = ['comment', 'approve-with-comments', 'approve'];
const words = english.web.queue;

function nodeText(value) {
  return esc(value).replace(/\r/g, '&#13;');
}

function block(tag, cls, content) {
  return '<' + tag + (cls ? ' class="' + cls + '"' : '') + '>' + (tag === 'pre' && content.charAt(0) === '\n' ? '\n' : '') + nodeText(content) + '</' + tag + '>';
}

function button(label, cls, act) {
  return '<button class="' + cls + '" type="button"' + (act ? ' data-act="' + act + '"' : '') + '>' + nodeText(label) + '</button>';
}

function link(href, content) {
  return '<a href="' + nodeText(href) + '" target="_blank" rel="noopener">' + nodeText(content) + '</a>';
}

export function createQueueMarkup(ctx) {
  const where = data => ctx.state.repo + '#' + data.pull;

  function pill(pull, prefix) {
    return '<span class="qc-pill" style="--qc-pc: ' + ctx.pulls.pullColor(pull) + ';">' + nodeText((prefix || '') + '#' + pull) + '</span>';
  }

  function item(comment, data, stacked) {
    const shown = comment.shown || comment;
    let row = block('span', 'qc-num', String(comment.id));
    if (stacked) row += pill(data.pull, '→ ');
    row += '<a class="qc-path" href="#" data-go-file="' + nodeText(shown.path) + '" data-go-line="' + nodeText(String(shown.line)) + '" data-go-side="' + nodeText(shown.side) + '">' +
      nodeText(comment.path + ', ' + english.review.lines(comment)) + '</a>';
    if (comment.shown && (comment.shown.path !== comment.path || comment.shown.line !== comment.line)) {
      row += block('span', 'qc-shown', words.shownHere(comment.shown.line, comment.shown.path !== comment.path ? comment.shown.path : null));
    }
    if (!data.ended && !ctx.state.moved) {
      row += '<span class="qc-tools">' + button(words.edit, 'nbtn quiet small', 'edit') + button(words.drop, 'nbtn quiet small', 'drop') + '</span>';
    }
    return '<li class="qc-item" data-comment="' + comment.id + '"><div class="qc-where">' + row + '</div>' +
      (comment.code ? block('pre', 'qc-code', comment.code) : '') +
      block('p', 'qc-body', comment.body) +
      (comment.stale ? block('p', 'qc-stale', words.stale(comment.stale)) : '') + '</li>';
  }

  function offered(data) {
    return ORDER.filter(action => data.sets && data.sets[action]);
  }

  function actionButton(action, data) {
    return button(words.labels[action] + (action === 'approve' ? '' : ' (' + data.comments.length + ')'), 'qc-button' + (action === 'comment' ? '' : ' qc-approve'));
  }

  function yesButton(action) {
    return button(words.yes[action], 'qc-button qc-yes' + (action === 'comment' ? '' : ' qc-approve'));
  }

  function noButton() {
    return button(words.cancel, 'qc-quiet');
  }

  function postedList(data) {
    return '<ul class="qc-posted">' + data.posted.map(set =>
      '<li>' + nodeText(words.posted(set.action, set.count, when(set.at))) + link(set.review, words.theReview) + '</li>').join('') + '</ul>';
  }

  function fill(data, stacked, left, wired) {
    const shut = data.ended || left;
    let out = '';
    if (data.comments.length) {
      out += '<ol class="qc-list">' + data.comments.map(c => item(c, data, stacked)).join('') + '</ol>';
      const stale = data.comments.some(c => c.stale);
      if (stale && !shut) out += block('p', 'qc-note', words.staleHere);
      else if (!shut) out += block('p', 'qc-note', words.goesOn(stacked ? '#' + data.pull : where(data)));
    } else if (stacked) {
      out += block('p', 'qc-empty', words.noneFor(data.pull));
    }
    if (data.ended) {
      out += block('p', 'qc-status', words.ended(stacked ? '#' + data.pull : where(data), data.state, data.ended_at ? when(data.ended_at) : null, stacked));
    } else if (!left && !ctx.state.moved) {
      out += '<div class="qc-actions"></div><p class="qc-status" aria-live="polite"></p>';
      if (!stacked) out += block('p', 'qc-note', words.note);
      wired.push(data);
    }
    if (data.posted.length) out += postedList(data);
    return out;
  }

  // The server's words for what a press did, with the review's address as a link.
  function result(message) {
    let out = '';
    words.result(message).split(/(https:\/\/\S+)/).forEach(part => {
      if (!/^https:\/\//.test(part)) { out += nodeText(part); return; }
      const address = part.replace(/[.,;:]+$/, '');
      out += link(address, address) + nodeText(part.slice(address.length));
    });
    return '<p class="qc-result">' + out + '</p>';
  }

  function section(message) {
    const q = ctx.state.queue;
    const wired = [];
    let html = '<div class="qc-head">' + block('h2', null, words.title) + '</div>';
    if (message) html += result(message);
    if (q.moved) {
      html += '<p class="qc-empty">' + nodeText(words.movedTo) + '<a href="' + nodeText('/review/' + q.moved.to) + '">' + nodeText(english.web.review.stacksReview) + '</a>' + nodeText(words.movedThere) + '</p>';
      q.members.forEach(m => { if (m.posted.length) html += postedList(m); });
      return { html, wired };
    }
    const shown = q.members.map(m => ({ ...m, ended: m.state && m.state !== 'open' }));
    const stacked = ctx.pulls.members.length > 0;
    if (!stacked) {
      if (!shown[0].comments.length) html += block('p', 'qc-empty', words.empty);
      html += fill(shown[0], false, false, wired);
      return { html, wired };
    }
    html += block('p', 'qc-note', words.stackNote);
    if (!shown.some(m => m.comments.length)) html += block('p', 'qc-empty', words.empty);
    shown.map(v => [v, false]).concat(q.earlier.map(m => [{ ...m, ended: m.state && m.state !== 'open' }, true])).forEach(([view, left]) => {
      html += '<section class="qc-pr" data-pull="' + view.pull + '"><h3 class="qc-pr-head">' + pill(view.pull) + (view.title ? block('span', 'qc-pr-title', view.title) : '') + '</h3>' +
        (left && !view.ended ? block('p', 'qc-status', words.leftStack(view.pull)) : '') +
        fill(view, true, left, wired) + '</section>';
    });
    html += block('p', 'qc-note', words.note);
    return { html, wired };
  }

  function editBox(draft) {
    return '<div class="edit-box"><textarea rows="3">' + esc(draft.text) + '</textarea><div class="note-actions">' +
      '<button type="button" class="nbtn primary" data-edit="save">' + words.save + '</button><button type="button" class="nbtn quiet" data-edit="cancel">' + words.cancel + '</button>' +
      '<span class="note-status' + (draft.said ? ' bad' : '') + '" aria-live="polite">' + esc(draft.said) + '</span></div></div>';
  }

  return { section, offered, actionButton, yesButton, noButton, editBox };
}
