/**
 * Conversations with Claude on the lines of a review.
 *
 * start(ref, {path, side, line, start_line, start_side, text, build})
 * opens a conversation on a line or a range as the app shows it, with the
 * person's question, and reply(ref, id, {text, build}) adds a question to
 * one. Each names `build`, the build id of the page its window was drawn
 * from, and is refused once the review was loaded again since. Each keeps
 * the question at once, then asks Claude in one short call, with only that
 * file's part of the diff and code around the line, and keeps the answer, or
 * why there is none. Both return the conversation as it then stands. Nothing waits
 * between questions: each is its own call.
 */
import { questionFor } from './question.mjs';
import { checkBuild } from '../store/page-build.mjs';
import { english } from '../i18n/english.mjs';

export class ThreadService {
  #store;
  #claude;
  #now;

  constructor({ store, claude, now = () => new Date().toISOString() }) {
    this.#store = store;
    this.#claude = claude;
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
      thread.messages.push({ from: 'you', text: text.trim(), at: this.#now() });
      return { files: [this.#write(ref, thread)], result: null };
    });
    return this.#answer(ref, id);
  }

  async #answer(ref, id) {
    const review = this.#reviewOf(ref);
    const { page } = this.#store.readPage(ref);
    let message;
    try {
      message = { from: 'claude', text: await this.#claude.ask(questionFor({ review, page, thread: this.#thread(ref, id) })), at: this.#now() };
    } catch (error) {
      message = { from: 'claude', error: error.message, at: this.#now() };
    }
    return this.#store.change([ref], async () => {
      const thread = this.#thread(ref, id);
      thread.messages.push(message);
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
