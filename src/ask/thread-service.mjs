/**
 * Conversations with Claude on the lines of a review.
 *
 * start(ref, {path, side, line, start_line, start_side, text, build})
 * opens a conversation on a line or a range as the app shows it, with the
 * person's question, and reply(ref, id, {text, build}) adds a question to
 * one. Each names `build`, the build id of the page its window was drawn
 * from, and is refused once the review was loaded again since. Each keeps
 * the question at once, then asks Claude, and keeps the answer, or why there
 * is none, with the steps Claude took. Both return the conversation as it
 * then stands.
 *
 * Claude works as the user's own Claude Code, under the guard, in a
 * throwaway copy of the code at the commit the lines belong to: one copy per
 * conversation, made the first time from the review's own clone, or for a
 * branch review from a clone that lazyreview fetches the branch into from
 * its repository. Copies no question used for a day are removed. Each
 * conversation keeps one Claude Code session, so a follow-up goes on from
 * what Claude read and found. While Claude works, the conversation holds
 * the steps so far as `working`, which both apps show.
 *
 * stop(ref, id) stops Claude working on a conversation and returns it once
 * it says so. When nothing works on it, as after the server was stopped
 * mid-answer, a conversation still waiting is told it was stopped, so it
 * waits no longer. stopAll() stops Claude on every conversation it is
 * answering, as when the server stops.
 */
import { randomUUID } from 'node:crypto';
import { mkdirSync, realpathSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { headAsked, copyBranches, questionFor, followUpFor } from './question.mjs';
import { guardSettings } from './guard.mjs';
import { stepWords } from './step-words.mjs';
import { checkBuild } from '../store/page-build.mjs';
import { withLocks } from '../store/file-lock.mjs';
import { Copies } from '../commands/git/copies.mjs';
import { Clones } from '../commands/git/clones.mjs';
import { BranchRef } from '../models/branch-ref.mjs';
import { english } from '../i18n/english.mjs';

const UNUSED = 24 * 60 * 60_000;
const COPY_WAIT = 10 * 60_000;
const STEPS = 200;
const WRITE_EVERY = 400;
const GUARD = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'bin', 'lazyreview-guard');

export class ThreadService {
  #store;
  #claude;
  #copies;
  #clones;
  #hook;
  #session;
  #now;
  #working = new Map();

  constructor({ store, claude, copies = new Copies(), clones = new Clones(), hook = guardCommand, session = randomUUID, now = () => new Date().toISOString() }) {
    this.#store = store;
    this.#claude = claude;
    this.#copies = copies;
    this.#clones = clones;
    this.#hook = hook;
    this.#session = session;
    this.#now = now;
  }

  async start(ref, { path, side, line, start_line, start_side, text, build }) {
    if (!String(text || '').trim()) throw new Error(english.ask.askFirst);
    if (!path || !['RIGHT', 'LEFT'].includes(side) || !Number.isInteger(line) || line < 1) throw new Error(english.ask.needsLine);
    const thread = await this.#store.change([ref], async () => {
      this.#reviewOf(ref);
      checkBuild(this.#store.pageBuild(ref), build, english.ask.notAsked);
      const id = Math.max(0, ...this.#store.threads(ref).map(t => t.id)) + 1;
      const fresh = {
        id, path, side, line, ...(start_line ? { start_line, start_side: start_side || side } : {}), created_at: this.#now(),
        messages: [{ from: 'you', text: text.trim(), at: this.#now() }],
      };
      return { files: [this.#write(ref, fresh)], result: fresh };
    });
    return this.#answer(ref, thread.id);
  }

  async reply(ref, id, { text, build }) {
    if (!String(text || '').trim()) throw new Error(english.ask.askFirst);
    await this.#store.change([ref], async () => {
      this.#reviewOf(ref);
      checkBuild(this.#store.pageBuild(ref), build, english.ask.notAsked);
      const thread = this.#thread(ref, id);
      if (this.#working.has(key(ref, id))) throw new Error(english.ask.stillWorking);
      thread.messages.push({ from: 'you', text: text.trim(), at: this.#now() });
      return { files: [this.#write(ref, thread)], result: null };
    });
    return this.#answer(ref, id);
  }

  async stop(ref, id) {
    const working = this.#working.get(key(ref, id));
    if (working) {
      working.controller.abort();
      await working.done;
      return this.#thread(ref, id);
    }
    return this.#change(ref, id, thread => {
      const last = thread.messages[thread.messages.length - 1];
      if (last?.from === 'you') thread.messages.push({ from: 'claude', error: english.claude.stopped, at: this.#now() });
      delete thread.working;
    });
  }

  async stopAll() {
    const working = [...this.#working.values()];
    for (const w of working) w.controller.abort();
    await Promise.all(working.map(w => w.done));
  }

  async #answer(ref, id) {
    const controller = new AbortController();
    let finished;
    const done = new Promise(resolve => { finished = resolve; });
    this.#working.set(key(ref, id), { controller, done });
    const steps = [];
    let message, progress;
    try {
      const review = this.#reviewOf(ref);
      const { page } = this.#store.readPage(ref);
      const thread = this.#thread(ref, id);
      const folder = await this.#copy(ref, review, page, thread);
      progress = this.#progress(ref, id, steps, realpathSync(folder));
      const ask = (session, resume) => this.#claude.ask({
        ...(resume ? followUpFor({ thread }) : questionFor({ review, page, thread, folder })), folder, session, resume,
        settings: guardSettings({ hook: this.#hook(folder) }), onStep: progress.heard, signal: controller.signal,
      });
      let text;
      try {
        text = await ask(await this.#sessionOf(ref, id, thread.session), !!thread.session);
      } catch (error) {
        if (!error.noSession) throw error;
        text = await ask(await this.#sessionOf(ref, id, null), false);
      }
      message = { from: 'claude', text, ...kept(steps), at: this.#now() };
    } catch (error) {
      message = { from: 'claude', error: error.message, ...kept(steps), at: this.#now() };
    }
    try {
      await progress?.settled();
      return await this.#change(ref, id, thread => {
        thread.messages.push(message);
        delete thread.working;
      });
    } finally {
      this.#working.delete(key(ref, id));
      finished();
    }
  }

  async #sessionOf(ref, id, session) {
    if (session) return session;
    const fresh = this.#session();
    await this.#change(ref, id, thread => { thread.session = fresh; });
    return fresh;
  }

  #progress(ref, id, steps, folder) {
    let queued = null, writing = Promise.resolve(), over = false;
    const write = () => {
      queued = null;
      writing = writing.then(() => (over ? null : this.#change(ref, id, thread => { thread.working = { steps: steps.slice(-STEPS).map(shown) }; }))).catch(() => {});
    };
    return {
      heard: step => {
        if (step.refused) {
          const taken = steps.find(s => s.id === step.id);
          if (taken) taken.refused = step.refused;
        } else steps.push({ id: step.id, tool: step.tool, words: stepWords(step.tool, step.input, folder) });
        queued ??= setTimeout(write, WRITE_EVERY);
      },
      settled: async () => {
        over = true;
        clearTimeout(queued);
        await writing;
      },
    };
  }

  async #copy(ref, review, page, thread) {
    const home = this.#store.home;
    const { commit } = headAsked({ review, page, thread });
    const { branches, branch } = copyBranches({ review, page, thread });
    try {
      const source = review.kind === 'branch' ? await this.#localClone(review) : home.clone(review.host, review.repo);
      mkdirSync(home.copies, { recursive: true });
      return await withLocks([`${home.copies}.lock`], async () => {
        const folder = await this.#copies.prepare(home.copy(ref, thread.id), { source, commit, branches, branch });
        this.#copies.prune(home.copies, { unused: UNUSED, keep: folder });
        return folder;
      }, { wait: COPY_WAIT });
    } catch (error) {
      throw new Error(english.ask.noCodebase(error.message));
    }
  }

  async #localClone(review) {
    const path = this.#store.home.localClone(BranchRef.parse(review.id).repoKey);
    mkdirSync(dirname(path), { recursive: true });
    return withLocks([`${path}.lock`], () => this.#clones.copyFrom(path, { from: review.path, commits: [review.head.sha, review.base.sha] }), { wait: COPY_WAIT });
  }

  #change(ref, id, edit) {
    return this.#store.change([ref], async () => {
      const thread = this.#thread(ref, id);
      edit(thread);
      return { files: [this.#write(ref, thread)], result: thread };
    });
  }

  #reviewOf(ref) {
    const review = this.#store.read(ref);
    if (!review) throw new Error(english.review.none(ref.id));
    return review;
  }

  #thread(ref, id) {
    const thread = this.#store.threads(ref).find(t => t.id === id);
    if (!thread) throw new Error(english.ask.noConversation(id, ref.id));
    return thread;
  }

  #write(ref, thread) {
    return [this.#store.home.review(ref).thread(thread.id), JSON.stringify(thread, null, 2) + '\n'];
  }
}

function key(ref, id) {
  return `${ref.id}#${id}`;
}

function shown({ words, refused }) {
  return { words, ...(refused ? { refused } : {}) };
}

function kept(steps) {
  return steps.length ? { steps: steps.slice(-STEPS).map(shown) } : {};
}

function guardCommand(folder) {
  return [process.execPath, GUARD, folder].map(quoted).join(' ');
}

function quoted(text) {
  return `'${text.replaceAll("'", `'\\''`)}'`;
}
