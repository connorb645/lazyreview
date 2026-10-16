/**
 * The app's web server, for this machine only.
 *
 * new AppServer({store, loader, branches, queue, threads, port}) serves the
 * app at http://127.0.0.1:<port>/, each review at /review/<review id>, and
 * the JSON the app reads and changes reviews through under /api. `loader`
 * loads pull requests again and `branches` branches. A branch review takes
 * no comments: only a pull request's review has a queue to change or post.
 * listen() resolves once it is listening, with its address; close() stops it.
 *
 * It answers only requests addressed to this machine by name, so a page from
 * elsewhere cannot reach it through a name it controls, and it takes a change
 * only from the app's own pages: JSON, sent from its own origin.
 */
import { createServer } from 'node:http';
import { createReadStream, readFileSync } from 'node:fs';
import { PullRef } from '../github/pull-ref.mjs';
import { BranchRef } from '../local/branch-ref.mjs';
import { reviewState, reviewSummary } from './review-state.mjs';
import { staticFile, PAGE } from './static-files.mjs';

const PULL = '([^/]+/[^/]+/[^/]+/\\d+)';
const REVIEW = '(local/[^/]+/[^/]+|[^/]+/[^/]+/[^/]+/\\d+)';
const BODY_LIMIT = 1 << 20;
const PAGE_SECURITY = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'";

export class AppServer {
  #store;
  #loader;
  #branches;
  #queue;
  #threads;
  #port;
  #server = null;
  #routes;

  constructor({ store, loader, branches, queue, threads, port }) {
    this.#store = store;
    this.#loader = loader;
    this.#branches = branches;
    this.#queue = queue;
    this.#threads = threads;
    this.#port = port;
    this.#routes = [
      ['GET', /^\/api\/health$/, () => ({ ok: true })],
      ['GET', /^\/api\/reviews$/, () => this.#store.list().map(reviewSummary)],
      ['GET', new RegExp(`^/api/reviews/${REVIEW}$`), ref => this.#state(ref)],
      ['GET', new RegExp(`^/api/reviews/${REVIEW}/page$`), ref => this.#page(ref)],
      ['POST', new RegExp(`^/api/reviews/${REVIEW}/refresh$`), async ref => {
        const loaded = await (ref instanceof BranchRef ? this.#branches : this.#loader).refresh(ref);
        return { id: loaded.ref.id, notes: loaded.notes };
      }],
      ['POST', new RegExp(`^/api/reviews/${PULL}/comments$`), async (ref, body) => {
        const done = await this.#queue.add(ref, {
          path: text(body.path), line: whole(body.line), side: side(body.side), ...range(body),
          body: text(body.body), code: typeof body.code === 'string' ? body.code : '', pull: body.pull ? whole(body.pull) : null,
        });
        return { comment: done.comment, pull: done.pull, state: this.#state(ref) };
      }],
      ['PATCH', new RegExp(`^/api/reviews/${PULL}/comments/(\\d+)$`), async (ref, body, id) => {
        await this.#queue.edit(ref, Number(id), text(body.body));
        return { state: this.#state(ref) };
      }],
      ['DELETE', new RegExp(`^/api/reviews/${PULL}/comments/(\\d+)$`), async (ref, body, id) => {
        await this.#queue.drop(ref, Number(id));
        return { state: this.#state(ref) };
      }],
      ['POST', new RegExp(`^/api/reviews/${PULL}/post$`), async (ref, body) => {
        const done = await this.#queue.post(ref, { pull: whole(body.pull), action: text(body.action), set: text(body.set) });
        return { message: done.message, review: done.review, state: this.#state(ref) };
      }],
      ['POST', new RegExp(`^/api/reviews/${REVIEW}/threads$`), async (ref, body) => ({
        thread: await this.#threads.start(ref, {
          path: text(body.path), side: side(body.side), line: whole(body.line), ...range(body),
          code: typeof body.code === 'string' ? body.code : '', text: text(body.text),
        }),
      })],
      ['POST', new RegExp(`^/api/reviews/${REVIEW}/threads/(\\d+)/messages$`), async (ref, body, id) => ({
        thread: await this.#threads.reply(ref, Number(id), text(body.text)),
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
    return new Promise(resolve => (this.#server ? this.#server.close(() => resolve()) : resolve()));
  }

  async #handle(request, response) {
    const port = this.#server.address().port;
    const hosts = [`127.0.0.1:${port}`, `localhost:${port}`];
    if (!hosts.includes(request.headers.host)) return send(response, 421, { error: 'this server answers only on 127.0.0.1 and localhost' });
    const url = new URL(request.url, `http://${request.headers.host}`);
    const path = url.pathname;

    if (!path.startsWith('/api/')) {
      if (request.method !== 'GET' && request.method !== 'HEAD') return send(response, 405, { error: 'not allowed' });
      if (path === '/' || /^\/review\//.test(path)) return sendFile(response, PAGE, 'text/html; charset=utf-8', true);
      let file = null;
      try { file = staticFile(path); } catch { file = null; }
      return file ? sendFile(response, file.path, file.type) : send(response, 404, { error: 'no such page' });
    }

    if (request.method !== 'GET') {
      const origin = request.headers.origin;
      if (!origin || !hosts.some(h => origin === `http://${h}`)) return send(response, 403, { error: 'changes come only from the app\'s own pages' });
      if (request.method !== 'DELETE' && !String(request.headers['content-type'] || '').startsWith('application/json')) {
        return send(response, 415, { error: 'send JSON' });
      }
    }
    const route = this.#routes.find(([method, pattern]) => method === request.method && pattern.test(path));
    if (!route) return send(response, 404, { error: `nothing at ${request.method} ${path}` });
    const match = route[1].exec(path).slice(1);
    try {
      const ref = match.length ? reviewRef(match[0]) : null;
      const body = request.method === 'GET' || request.method === 'DELETE' ? {} : await readBody(request);
      const answer = await route[2](...(ref ? [ref, body, ...match.slice(1)] : [body]));
      if (typeof answer === 'string') return sendRaw(response, 200, answer);
      if (answer === null || answer === undefined) return send(response, 404, { error: 'no such review' });
      return send(response, 200, answer);
    } catch (error) {
      return send(response, 409, { error: error.message });
    }
  }

  #state(ref) {
    return reviewState(this.#store, ref);
  }

  #page(ref) {
    return this.#store.readPage(ref);
  }
}

function reviewRef(id) {
  const parts = id.split('/').map(decodeURIComponent);
  if (parts[0] === 'local' && parts.length === 3) return BranchRef.parse(parts.join('/'));
  return new PullRef(parts[0], `${parts[1]}/${parts[2]}`, Number(parts[3]));
}

function text(value) {
  if (typeof value !== 'string') throw new Error('a text field is missing');
  return value;
}

function whole(value) {
  const n = Number(value);
  if (!Number.isInteger(n) || n < 1) throw new Error(`"${value}" is not a line or pull request number`);
  return n;
}

function range(body) {
  if (!body.start_line) return { start_line: null, start_side: null };
  return { start_line: whole(body.start_line), start_side: body.start_side ? side(body.start_side) : side(body.side) };
}

function side(value) {
  if (value !== 'RIGHT' && value !== 'LEFT') throw new Error('the side is RIGHT or LEFT');
  return value;
}

function readBody(request) {
  return new Promise((resolve, reject) => {
    let size = 0;
    const chunks = [];
    request.on('data', chunk => {
      size += chunk.length;
      if (size > BODY_LIMIT) { reject(new Error('the request is too large')); request.destroy(); return; }
      chunks.push(chunk);
    });
    request.on('end', () => {
      try {
        const parsed = JSON.parse(Buffer.concat(chunks).toString('utf8') || '{}');
        resolve(parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {});
      } catch {
        reject(new Error('the request is not JSON'));
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
