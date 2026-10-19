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
 * requests; it says whether it started the app or found it running, and
 * refuses a link that names no review before it starts anything. `load`
 * reads pull requests from GitHub, or with --repo a branch from a
 * repository on this machine. `config` says where the config file
 * is, prints the default config or every setting as read, and makes the
 * one-time move into the config; it reads no other setting first, so a
 * config with a mistake in it can always be found and fixed.
 */
import { readFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { parseArguments } from './arguments.mjs';
import { readTerminalLink } from './terminal-link.mjs';
import { App } from '../app/app.mjs';
import { PullRef } from '../models/pull-ref.mjs';
import { readWrittenReview, checkWrittenReview } from '../review/written-review.mjs';
import { english } from '../i18n/english.mjs';
import { findConfigFile } from '../config/config-file.mjs';
import { readUserConfig, terminalPart } from '../config/user-config.mjs';
import { defaultConfigText, checkedConfigText } from '../config/config-text.mjs';
import { planMove, checkLines, move } from '../config/config-move.mjs';

export async function run(argv, { env = process.env, bin, say = () => {}, terminal }) {
  const options = parseArguments(argv);
  if (options.command === 'help') return [english.cli.usage];
  if (options.command === 'config') return config(options, { env, platform: process.platform, home: homedir() });
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
      const running = await app.background.running();
      say(running ? english.cli.runningAlready(running) : english.cli.started(await app.background.start()));
      await app.terminal.launch({ serverFile: app.home.serverFile, review, env, userConfig: terminalPart(app.config, env), say });
      return [];
    }
    case 'start': {
      const address = (await app.background.running()) || (await app.background.start());
      return [english.cli.running(address)];
    }
    case 'stop':
      return [(await app.background.stop()) ? english.cli.stopped : english.cli.notRunning];
    case 'serve':
      return app.serve();
    default:
      throw new Error(english.cli.notACommand(options.command));
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
  return [
    english.cli.loaded(changeName(loaded.review), loaded.counts),
    english.cli.open(reviewAddress(address, loaded.ref)),
    ...loaded.notes.map(english.cli.note),
  ];
}

async function attach({ store, home, background }, options) {
  const found = store.find(PullRef.parse(options.positional[0]));
  if (!found) throw new Error(english.cli.notLoaded(options.positional[0]));
  const text = readFileSync(options.positional[1], 'utf8');
  await store.change([found.ref], async () => {
    const review = store.read(found.ref);
    if (review.moved) throw new Error(english.cli.reviewMoved(review.moved.to));
    checkWrittenReview(readWrittenReview(text), review);
    return { files: [[home.review(found.ref).written, text]], result: null };
  });
  const address = await background.running();
  return [english.cli.attached(found.ref.id, address && reviewAddress(address, found.ref))];
}

async function list({ store, config, background }) {
  const address = (await background.running()) || 'http://127.0.0.1:' + config.values.server.port;
  const reviews = store.list();
  if (!reviews.length) return [english.cli.noReviews];
  return reviews.map(({ ref, review }) => (review.moved
    ? english.cli.listedMoved(ref.id, review.moved.to)
    : english.cli.listed(reviewAddress(address, ref), changeName(review), review.title)));
}

function config(options, where) {
  switch (options.positional[0]) {
    case 'path': {
      const file = findConfigFile(where);
      return [file.path, file.exists ? english.config.exists : english.config.missing];
    }
    case 'defaults':
      return [defaultConfigText().trimEnd()];
    case 'check': {
      const read = readUserConfig(where);
      return [read.exists ? english.config.checked(read.path) : english.config.unset(read.path), '', checkedConfigText(read).trimEnd()];
    }
    default: {
      const plan = planMove(where);
      return options.check ? checkLines(plan) : move(plan);
    }
  }
}

function changeName(review) {
  return review.kind === 'branch' ? english.review.branchName(review.head.ref, review.base.ref) : english.review.pullsName(review.members.map(m => m.pull));
}

function reviewAddress(address, ref) {
  return `${address}/review/${ref.id.split('/').map(encodeURIComponent).join('/')}`;
}

async function reviewToOpen(text, { app, say }) {
  const { review, pull } = readTerminalLink(text);
  if (review) {
    if (!app.store.read(review)) throw new Error(english.cli.noReview(review.id));
    return review.id;
  }
  const found = app.store.find(pull);
  if (found) return found.ref.id;
  say(english.cli.loading(pull.url));
  const loaded = await app.loader.load({ links: [text] });
  for (const note of loaded.notes) say(english.cli.note(note));
  return loaded.ref.id;
}

function readConfigFile(path) {
  try {
    return JSON.parse(readFileSync(path, 'utf8'));
  } catch (error) {
    throw new Error(english.cli.configUnreadable(path, error.message));
  }
}
