/**
 * The review-desk command.
 *
 * run(argv, {env, bin}) does what the command line asks and returns the lines
 * to print. `serve` keeps running until it is stopped, and `terminal` until
 * the terminal app quits. Every other command returns once done; `load` and
 * `terminal` start the app in the background first when it is not running.
 * The server records its address and a token in server.json, which only its
 * owner can read; a client such as the terminal app sends the token with
 * each change. `load` reads pull requests
 * from GitHub, or with --repo a branch from a repository on this machine.
 */
import { readFileSync, rmSync, mkdirSync } from 'node:fs';
import { randomBytes } from 'node:crypto';
import { parseArguments, USAGE } from './arguments.mjs';
import { readSettings } from '../settings/settings.mjs';
import { DataHome } from '../store/data-home.mjs';
import { ReviewStore } from '../store/review-store.mjs';
import { writeWhole, readJSON } from '../store/whole-file.mjs';
import { Clones } from '../clone/clones.mjs';
import { Loader } from '../loader/loader.mjs';
import { BranchLoader } from '../loader/branch-loader.mjs';
import { QueueService } from '../queue/queue-service.mjs';
import { ThreadService } from '../ask/thread-service.mjs';
import { Claude } from '../ask/claude.mjs';
import { AppServer } from '../server/app-server.mjs';
import { running, start, stop } from '../server/server-process.mjs';
import { PullRef } from '../github/pull-ref.mjs';
import { readWrittenReview, checkWrittenReview } from '../review/written-review.mjs';
import { reviewRef } from '../review/review-ref.mjs';
import { TerminalApp } from '../terminal/terminal-app.mjs';

export async function run(argv, { env = process.env, bin }) {
  const options = parseArguments(argv);
  if (options.command === 'help') return [USAGE];
  const settings = readSettings(env);
  const home = new DataHome(settings.home);
  mkdirSync(home.root, { recursive: true });
  const store = new ReviewStore(home);

  switch (options.command) {
    case 'load': {
      const drawing = {
        config: options.config ? readConfigFile(options.config) : undefined,
        context: options.context ? true : options.noContext ? false : undefined,
      };
      const loaded = options.repo !== undefined
        ? await new BranchLoader({ store }).load({ repo: options.repo, base: options.base, head: options.head, title: options.title, ...drawing })
        : await new Loader({ store, clones: new Clones(home) }).load({ links: options.positional, account: options.account, ...drawing });
      const address = (await running(home)) || (await start({ home, bin, env }));
      const c = loaded.counts;
      return [
        `loaded ${changeName(loaded.review)}: ${c.files} file${c.files === 1 ? '' : 's'} +${c.adds} -${c.dels}, ${c.components} component${c.components === 1 ? '' : 's'}, ${c.types} type${c.types === 1 ? '' : 's'}`,
        `open ${reviewAddress(address, loaded.ref)}`,
        ...loaded.notes.map(n => `note: ${n}`),
      ];
    }
    case 'attach': {
      const found = store.find(PullRef.parse(options.positional[0]));
      if (!found) throw new Error(`${options.positional[0]} is not loaded; load it first`);
      const text = readFileSync(options.positional[1], 'utf8');
      await store.change([found.ref], async () => {
        const review = store.read(found.ref);
        if (review.moved) throw new Error(`that review moved to ${review.moved.to}`);
        checkWrittenReview(readWrittenReview(text), review);
        return { files: [[home.review(found.ref).written, text]], result: null };
      });
      const address = await running(home);
      return [`attached to ${found.ref.id}${address ? `; open ${reviewAddress(address, found.ref)}` : ''}`];
    }
    case 'list': {
      const address = (await running(home)) || 'http://127.0.0.1:' + settings.port;
      const reviews = store.list();
      if (!reviews.length) return ['no reviews yet; load one with review-desk load <pull link>'];
      return reviews.map(({ ref, review }) => (review.moved
        ? `${ref.id}  moved to ${review.moved.to}`
        : `${reviewAddress(address, ref)}  ${changeName(review)}  ${review.title}`));
    }
    case 'terminal': {
      const review = options.positional[0] ? reviewNamed(options.positional[0], store) : null;
      if (!(await running(home))) await start({ home, bin, env });
      await new TerminalApp().launch({ serverFile: home.serverFile, review, env });
      return [];
    }
    case 'start': {
      const address = (await running(home)) || (await start({ home, bin, env }));
      return [`running at ${address}`];
    }
    case 'stop':
      return [(await stop(home)) ? 'stopped' : 'not running'];
    case 'serve':
      return serve({ settings, home, store });
    default:
      throw new Error(`unknown command ${options.command}`);
  }
}

async function serve({ settings, home, store }) {
  if (await running(home)) throw new Error('the app is running already; review-desk stop stops it');
  const token = randomBytes(32).toString('hex');
  const server = new AppServer({
    store,
    loader: new Loader({ store, clones: new Clones(home) }),
    branches: new BranchLoader({ store }),
    queue: new QueueService({ store }),
    threads: new ThreadService({ store, claude: new Claude({ ...settings.claude, folder: home.root }) }),
    token,
    port: settings.port,
  });
  const address = await server.listen();
  writeWhole(home.serverFile, JSON.stringify({ pid: process.pid, address, token }) + '\n', { mode: 0o600 });
  const shut = async () => {
    try {
      const recorded = readJSON(home.serverFile);
      if (recorded?.pid === process.pid) rmSync(home.serverFile, { force: true });
    } catch (error) {
      console.error(`review-desk: ${error.message}`);
    } finally {
      await server.close();
      process.exit(0);
    }
  };
  process.once('SIGINT', shut);
  process.once('SIGTERM', shut);
  return [`Review Desk is running at ${address}`];
}

function changeName(review) {
  return review.kind === 'branch' ? `${review.head.ref} against ${review.base.ref}` : review.members.map(m => `#${m.pull}`).join(', ');
}

function reviewAddress(address, ref) {
  return `${address}/review/${ref.id.split('/').map(encodeURIComponent).join('/')}`;
}

// The id of the review a review id, its address in the app, or a pull request link names.
function reviewNamed(text, store) {
  const address = /^https?:\/\/(?:127\.0\.0\.1|localhost)(?::\d+)?\/review\/(.+?)\/*$/.exec(text);
  if (address) return reviewNamed(address[1].split('/').map(decodeURIComponent).join('/'), store);
  if (/^https?:\/\//.test(text)) {
    const found = store.find(PullRef.parse(text));
    if (!found) throw new Error(`${text} is not loaded; load it first`);
    return found.ref.id;
  }
  const ref = reviewRef(text);
  if (!store.read(ref)) throw new Error(`there is no review ${text}; review-desk list shows every review`);
  return ref.id;
}

function readConfigFile(path) {
  try {
    return JSON.parse(readFileSync(path, 'utf8'));
  } catch (error) {
    throw new Error(`could not read the config ${path}: ${error.message}`);
  }
}
