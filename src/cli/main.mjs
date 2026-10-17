/**
 * The lazyreview command.
 *
 * run(argv, {env, bin, say, terminal}) does what the command line asks and
 * returns the lines to print; `say` prints a line at once, such as what is
 * loading before the terminal app opens, and `terminal` is the terminal app
 * to launch. `serve` keeps running until it is stopped, and `terminal` until
 * the terminal app quits. Every other command returns once done; `load` and
 * `terminal` start the app in the background first when it is not running.
 * `terminal` with a link opens that review, loading a pull request link
 * that no review holds yet, and with none the home screen of open pull
 * requests. `load` reads pull requests from GitHub, or with --repo a branch
 * from a repository on this machine.
 */
import { readFileSync } from 'node:fs';
import { parseArguments, USAGE } from './arguments.mjs';
import { App } from '../app/app.mjs';
import { PullRef } from '../models/pull-ref.mjs';
import { reviewRef } from '../models/review-ref.mjs';
import { readWrittenReview, checkWrittenReview } from '../review/written-review.mjs';

export async function run(argv, { env = process.env, bin, say = () => {}, terminal }) {
  const options = parseArguments(argv);
  if (options.command === 'help') return [USAGE];
  const app = new App({ env, bin, terminal });

  switch (options.command) {
    case 'load':
      return load(app, options);
    case 'attach':
      return attach(app, options);
    case 'list':
      return list(app);
    case 'terminal': {
      const review = options.positional.length ? await reviewToOpen(options.positional[0], { app, say }) : null;
      if (!(await app.background.running())) await app.background.start();
      await app.terminal.launch({ serverFile: app.home.serverFile, review, env });
      return [];
    }
    case 'start': {
      const address = (await app.background.running()) || (await app.background.start());
      return [`running at ${address}`];
    }
    case 'stop':
      return [(await app.background.stop()) ? 'stopped' : 'not running'];
    case 'serve':
      return app.serve();
    default:
      throw new Error(`unknown command ${options.command}`);
  }
}

async function load(app, options) {
  const drawing = {
    config: options.config ? readConfigFile(options.config) : undefined,
    context: options.context ? true : options.noContext ? false : undefined,
  };
  const loaded = options.repo !== undefined
    ? await app.branches.load({ repo: options.repo, base: options.base, head: options.head, title: options.title, ...drawing })
    : await app.loader.load({ links: options.positional, account: options.account, ...drawing });
  const address = (await app.background.running()) || (await app.background.start());
  const c = loaded.counts;
  return [
    `loaded ${changeName(loaded.review)}: ${c.files} file${c.files === 1 ? '' : 's'} +${c.adds} -${c.dels}, ${c.components} component${c.components === 1 ? '' : 's'}, ${c.types} type${c.types === 1 ? '' : 's'}`,
    `open ${reviewAddress(address, loaded.ref)}`,
    ...loaded.notes.map(n => `note: ${n}`),
  ];
}

async function attach({ store, home, background }, options) {
  const found = store.find(PullRef.parse(options.positional[0]));
  if (!found) throw new Error(`${options.positional[0]} is not loaded; load it first`);
  const text = readFileSync(options.positional[1], 'utf8');
  await store.change([found.ref], async () => {
    const review = store.read(found.ref);
    if (review.moved) throw new Error(`that review moved to ${review.moved.to}`);
    checkWrittenReview(readWrittenReview(text), review);
    return { files: [[home.review(found.ref).written, text]], result: null };
  });
  const address = await background.running();
  return [`attached to ${found.ref.id}${address ? `; open ${reviewAddress(address, found.ref)}` : ''}`];
}

async function list({ store, settings, background }) {
  const address = (await background.running()) || 'http://127.0.0.1:' + settings.port;
  const reviews = store.list();
  if (!reviews.length) return ['no reviews yet; load one with lazyreview load <pull link>'];
  return reviews.map(({ ref, review }) => (review.moved
    ? `${ref.id}  moved to ${review.moved.to}`
    : `${reviewAddress(address, ref)}  ${changeName(review)}  ${review.title}`));
}

function changeName(review) {
  return review.kind === 'branch' ? `${review.head.ref} against ${review.base.ref}` : review.members.map(m => `#${m.pull}`).join(', ');
}

function reviewAddress(address, ref) {
  return `${address}/review/${ref.id.split('/').map(encodeURIComponent).join('/')}`;
}

async function reviewToOpen(text, { app, say }) {
  const address = /^https?:\/\/(?:127\.0\.0\.1|localhost)(?::\d+)?\/review\/([^?#]+?)\/*(?:[?#].*)?$/.exec(text);
  if (address) return reviewToOpen(address[1].split('/').map(decodeURIComponent).join('/'), { app, say });
  if (/^https?:\/\//.test(text)) {
    const ref = PullRef.parse(text);
    const found = app.store.find(ref);
    if (found) return found.ref.id;
    say(`loading ${ref.url}…`);
    const loaded = await app.loader.load({ links: [text] });
    for (const note of loaded.notes) say(`note: ${note}`);
    return loaded.ref.id;
  }
  const ref = reviewRef(text);
  if (!app.store.read(ref)) throw new Error(`there is no review ${text}; lazyreview list shows every review`);
  return ref.id;
}

function readConfigFile(path) {
  try {
    return JSON.parse(readFileSync(path, 'utf8'));
  } catch (error) {
    throw new Error(`could not read the config ${path}: ${error.message}`);
  }
}
