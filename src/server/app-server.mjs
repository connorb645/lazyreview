/**
 * The app's web server, for this machine only.
 *
 * new AppServer({store, loader, branches, queue, threads, pulls, events, token, port}) serves the
 * app at http://127.0.0.1:<port>/, each review at /review/<review id>, and
 * the JSON the app reads and changes reviews through under /api. `loader`
 * loads pull requests, new or again, and `branches` branches again; `pulls`
 * lists the open pull requests that involve the user's accounts, for the
 * terminal app's home screen. A branch review takes
 * no comments: only a pull request's review has a queue to change or post.
 * A review's page comes with its build id, without its per-file parts, which
 * the file route gives one file at a time, for the build a window names.
 * Every change but a refresh names the build its window was drawn from. A
 * GET or a DELETE takes its fields from its query. What the page and file
 * routes answer is kept in memory for each build. A review's events route is
 * a stream of its change notices, as `events` gives them, one JSON object
 * to each server-sent event.
 * listen() resolves once it is listening, with its address; close() stops it.
 *
 * It answers only requests addressed to this machine by name, so a page from
 * elsewhere cannot reach it through a name it controls, and it takes a change
 * only as JSON, from the app's own pages, by their origin, or from a client
 * carrying `token` as its bearer token, such as the terminal app.
 */
import { createServer } from 'node:http';
import { createReadStream, readFileSync } from 'node:fs';
import { timingSafeEqual } from 'node:crypto';
import { BranchRef } from '../models/branch-ref.mjs';
import { reviewRef } from '../models/review-ref.mjs';
import { reviewState, reviewSummary } from './review-state.mjs';
import { pageView } from '../review/page-view.mjs';
import { fileView } from '../review/file-view.mjs';
import { checkBuild } from '../store/page-build.mjs';
import { ReviewEvents } from './review-events.mjs';
import { staticFile, PAGE } from './static-files.mjs';
import { english } from '../i18n/english.mjs';

const PULL = '([^/]+/[^/]+/[^/]+/\\d+)';
const REVIEW = '(local/[^/]+/[^/]+|[^/]+/[^/]+/[^/]+/\\d+)';
const EVENTS = new RegExp(`^/api/reviews/${REVIEW}/events$`);
const BODY_LIMIT = 1 << 20;
const PAGE_SECURITY = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'";

export class AppServer {
  #store;
  #loader;
  #branches;
  #queue;
  #threads;
  #pulls;
  #port;
  #events;
  #server = null;
  #routes;
  #views = new WeakMap();
  #streams = new Set();
  #token;

  constructor({ store, loader, branches, queue, threads, pulls, events = new ReviewEvents({ store }), token = null, port }) {
    this.#store = store;
    this.#token = token;
    this.#events = events;
    this.#loader = loader;
    this.#branches = branches;
    this.#queue = queue;
    this.#threads = threads;
    this.#pulls = pulls;
    this.#port = port;
    this.#routes = [
      ['GET', /^\/api\/health$/, () => ({ ok: true })],
      ['GET', /^\/api\/reviews$/, () => this.#store.list().map(reviewSummary)],
      ['GET', /^\/api\/pulls$/, () => this.#pulls.list()],
      ['POST', /^\/api\/load$/, async body => {
        const loaded = await this.#loader.load({ links: links(body.links), account: typeof body.account === 'string' ? body.account : undefined });
        return { id: loaded.ref.id, notes: loaded.notes };
      }],
      ['GET', new RegExp(`^/api/reviews/${REVIEW}$`), ref => this.#state(ref)],
      ['GET', new RegExp(`^/api/reviews/${REVIEW}/page$`), ref => this.#page(ref)],
      ['GET', new RegExp(`^/api/reviews/${REVIEW}/file$`), (ref, query) => this.#file(ref, text(query.path), build(query))],
      ['GET', new RegExp(`^/api/reviews/${REVIEW}/where$`), (ref, query) => this.#queue.where(ref, {
        path: text(query.path), side: side(query.side), line: whole(query.line), ...range(query), pull: query.pull ? whole(query.pull) : null, build: build(query),
      })],
      ['POST', new RegExp(`^/api/reviews/${REVIEW}/refresh$`), async ref => {
        const loaded = await (ref instanceof BranchRef ? this.#branches : this.#loader).refresh(ref);
        return { id: loaded.ref.id, notes: loaded.notes };
      }],
      ['POST', new RegExp(`^/api/reviews/${PULL}/comments$`), async (ref, body) => {
        const done = await this.#queue.add(ref, {
          path: text(body.path), line: whole(body.line), side: side(body.side), ...range(body),
          body: text(body.body), pull: body.pull ? whole(body.pull) : null, build: build(body),
        });
        return { comment: done.comment, pull: done.pull, state: this.#state(ref) };
      }],
      ['PATCH', new RegExp(`^/api/reviews/${PULL}/comments/(\\d+)$`), async (ref, body, id) => {
        await this.#queue.edit(ref, Number(id), { body: text(body.body), version: version(body), build: build(body) });
        return { state: this.#state(ref) };
      }],
      ['DELETE', new RegExp(`^/api/reviews/${PULL}/comments/(\\d+)$`), async (ref, body, id) => {
        await this.#queue.drop(ref, Number(id), { version: version(body), build: build(body) });
        return { state: this.#state(ref) };
      }],
      ['POST', new RegExp(`^/api/reviews/${PULL}/post$`), async (ref, body) => {
        const done = await this.#queue.post(ref, { pull: whole(body.pull), action: text(body.action), set: text(body.set), build: build(body) });
        return { message: done.message, review: done.review, state: this.#state(ref) };
      }],
      ['POST', new RegExp(`^/api/reviews/${REVIEW}/threads$`), async (ref, body) => ({
        thread: await this.#threads.start(ref, {
          path: text(body.path), side: side(body.side), line: whole(body.line), ...range(body),
          text: text(body.text), build: build(body),
        }),
      })],
      ['POST', new RegExp(`^/api/reviews/${REVIEW}/threads/(\\d+)/messages$`), async (ref, body, id) => ({
        thread: await this.#threads.reply(ref, Number(id), { text: text(body.text), build: build(body) }),
      })],
    ];
  }

  listen() {
    this.#server = createServer((request, response) => this.#handle(request, response).catch(error => {
      if (!response.headersSent) send(response, 500, { error: error.message });
      else response.end();
    }));
    return new Promise((resolve, reject) => {
      this.#server.once('error', reject);
      this.#server.listen(this.#port, '127.0.0.1', () => resolve(`http://127.0.0.1:${this.#server.address().port}`));
    });
  }

  close() {
    for (const response of this.#streams) response.end();
    this.#events.close();
    return new Promise(resolve => (this.#server ? this.#server.close(() => resolve()) : resolve()));
  }

  async #handle(request, response) {
    const port = this.#server.address().port;
    const hosts = [`127.0.0.1:${port}`, `localhost:${port}`];
    if (!hosts.includes(request.headers.host)) return send(response, 421, { error: english.server.onlyHere });
    const url = new URL(request.url, `http://${request.headers.host}`);
    const path = url.pathname;

    if (!path.startsWith('/api/')) {
      if (request.method !== 'GET' && request.method !== 'HEAD') return send(response, 405, { error: english.server.notAllowed });
      if (path === '/' || /^\/review\//.test(path)) return sendFile(response, PAGE, 'text/html; charset=utf-8', true);
      let file = null;
      try { file = staticFile(path); } catch { file = null; }
      return file ? sendFile(response, file.path, file.type) : send(response, 404, { error: english.server.noSuchPage });
    }

    if (request.method !== 'GET') {
      const origin = request.headers.origin;
      const own = !!origin && hosts.some(h => origin === `http://${h}`);
      if (!own && !this.#carriesToken(request)) return send(response, 403, { error: english.server.changesFrom });
      if (request.method !== 'DELETE' && !String(request.headers['content-type'] || '').startsWith('application/json')) {
        return send(response, 415, { error: english.server.sendJSON });
      }
    }
    const stream = EVENTS.exec(path);
    if (stream && request.method === 'GET') return this.#stream(request, response, stream[1]);
    const route = this.#routes.find(([method, pattern]) => method === request.method && pattern.test(path));
    if (!route) return send(response, 404, { error: english.server.nothingAt(request.method, path) });
    const match = route[1].exec(path).slice(1);
    try {
      const ref = match.length ? reviewRef(match[0]) : null;
      const body = request.method === 'GET' || request.method === 'DELETE' ? Object.fromEntries(url.searchParams) : await readBody(request);
      const answer = await route[2](...(ref ? [ref, body, ...match.slice(1)] : [body]));
      if (typeof answer === 'string') return sendRaw(response, 200, answer);
      if (answer === null || answer === undefined) return send(response, 404, { error: english.server.noSuchReview });
      return send(response, 200, answer);
    } catch (error) {
      return send(response, 409, { error: error.message });
    }
  }

  #carriesToken(request) {
    const given = /^Bearer (\S+)$/.exec(request.headers.authorization || '');
    if (!this.#token || !given) return false;
    const a = Buffer.from(given[1]), b = Buffer.from(this.#token);
    return a.length === b.length && timingSafeEqual(a, b);
  }

  #stream(request, response, id) {
    let ref;
    try {
      ref = reviewRef(id);
    } catch (error) {
      return send(response, 409, { error: error.message });
    }
    if (!this.#store.read(ref)) return send(response, 404, { error: english.server.noSuchReview });
    response.writeHead(200, headers('text/event-stream; charset=utf-8'));
    response.write('retry: 1000\n\n');
    this.#streams.add(response);
    const stop = this.#events.subscribe(ref, notice => response.write(`data: ${JSON.stringify(notice)}\n\n`));
    response.on('close', () => {
      stop();
      this.#streams.delete(response);
    });
  }

  #state(ref) {
    return reviewState(this.#store, ref);
  }

  #page(ref) {
    const read = this.#store.readPage(ref);
    return read && this.#viewsOf(read).page;
  }

  #file(ref, path, build) {
    const read = this.#store.readPage(ref);
    if (!read) return null;
    checkBuild(read.build, build, english.server.noFile);
    const files = this.#viewsOf(read).files;
    if (!files.has(path)) files.set(path, fileView(read.page, path));
    const file = files.get(path);
    if (!file) throw new Error(english.server.notAFile(path));
    return file;
  }

  #viewsOf(read) {
    if (!this.#views.has(read)) this.#views.set(read, { page: pageView(read.build, read.page), files: new Map() });
    return this.#views.get(read);
  }
}

function text(value) {
  if (typeof value !== 'string') throw new Error(english.server.textMissing);
  return value;
}

function links(value) {
  if (!Array.isArray(value) || !value.length || value.some(link => typeof link !== 'string')) throw new Error(english.server.giveLinks);
  return value;
}

function whole(value) {
  const n = Number(value);
  if (!Number.isInteger(n) || n < 1) throw new Error(english.server.notANumber(value));
  return n;
}

function range(body) {
  if (!body.start_line) return { start_line: null, start_side: null };
  return { start_line: whole(body.start_line), start_side: body.start_side ? side(body.start_side) : side(body.side) };
}

function build(body) {
  return typeof body.build === 'string' ? body.build : null;
}

function version(body) {
  return typeof body.version === 'string' ? body.version : null;
}

function side(value) {
  if (value !== 'RIGHT' && value !== 'LEFT') throw new Error(english.server.sides);
  return value;
}

function readBody(request) {
  return new Promise((resolve, reject) => {
    let size = 0;
    const chunks = [];
    request.on('data', chunk => {
      size += chunk.length;
      if (size > BODY_LIMIT) { reject(new Error(english.server.tooLarge)); request.destroy(); return; }
      chunks.push(chunk);
    });
    request.on('end', () => {
      try {
        const parsed = JSON.parse(Buffer.concat(chunks).toString('utf8') || '{}');
        resolve(parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {});
      } catch {
        reject(new Error(english.server.notJSON));
      }
    });
    request.on('error', reject);
  });
}

function headers(type, extra = {}) {
  return { 'Content-Type': type, 'Cache-Control': 'no-store', 'X-Content-Type-Options': 'nosniff', 'Referrer-Policy': 'no-referrer', ...extra };
}

function send(response, status, value) {
  sendRaw(response, status, JSON.stringify(value));
}

function sendRaw(response, status, json) {
  response.writeHead(status, headers('application/json; charset=utf-8'));
  response.end(json);
}

function sendFile(response, path, type, page = false) {
  response.writeHead(200, headers(type, page ? { 'Content-Security-Policy': PAGE_SECURITY } : {}));
  if (page) response.end(readFileSync(path));
  else createReadStream(path).pipe(response);
}
