/**
 * The lazyreview command.
 *
 * run(argv, {env, say, terminal}) does what the command line asks and
 * returns the lines to print; `say` prints a line at once, such as what is
 * loading before the terminal app opens, and `terminal` is the terminal app
 * to launch. `terminal` runs until the terminal app quits, and every other
 * command returns once done. `terminal` with an address opens that review,
 * loading a pull request link that no review holds yet, and with none the
 * home screen of open pull requests; it refuses an address that names no
 * review before it opens anything. `load` has the terminal app's program
 * read pull requests from GitHub, or with --repo a branch from a repository
 * on this machine, and so does `terminal` with a link no review holds. A
 * review's address is lazyreview://<review id>. lazyreview runs no server:
 * `start` says so, and `stop` asks every answer Claude is still giving to
 * end, and stops a server an older lazyreview left running. `config` says
 * where the config file is, prints the default config or every setting as
 * read, and makes the one-time move into the config; it reads no other
 * setting first, so a config with a mistake in it can always be found and
 * fixed.
 */
import { readFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { parseArguments } from './arguments.mjs';
import { readTerminalLink } from './terminal-link.mjs';
import { App } from '../app/app.mjs';
import { stopAnswers } from '../ask/answering.mjs';
import { PullRef } from '../models/pull-ref.mjs';
import { readWrittenReview, checkWrittenReview } from '../review/written-review.mjs';
import { english } from '../i18n/english.mjs';
import { findConfigFile } from '../config/config-file.mjs';
import { readUserConfig } from '../config/user-config.mjs';
import { UNUSED } from '../config/schema.mjs';
import { defaultConfigText, checkedConfigText } from '../config/config-text.mjs';
import { planMove, checkLines, move } from '../config/config-move.mjs';

export async function run(argv, { env = process.env, say = () => {}, terminal }) {
  const options = parseArguments(argv);
  if (options.command === 'help') return [english.cli.usage];
  if (options.command === 'config') return config(options, { env, platform: process.platform, home: homedir() });
  const app = new App({ env, terminal });

  switch (options.command) {
    case 'load':
      return load(app, options, { env, say });
    case 'attach':
      return attach(app, options);
    case 'list':
      return list(app);
    case 'terminal': {
      const review = options.positional.length ? await reviewToOpen(options.positional[0], { app, env, say }) : null;
      await app.terminal.launch({ review, env, say });
      return [];
    }
    case 'start':
      return [english.cli.noServer];
    case 'stop':
      return stop(app);
    default:
      throw new Error(english.cli.notACommand(options.command));
  }
}

async function load(app, options, { env, say }) {
  if (options.config) readConfigFile(options.config);
  const loaded = await app.terminal.load([
    ...(options.account !== undefined ? ['--account', options.account] : []),
    ...(options.config ? ['--config', options.config] : []),
    ...(options.context ? ['--context'] : options.noContext ? ['--no-context'] : []),
    ...(options.repo !== undefined ? ['--repo', options.repo, '--base', options.base, '--head', options.head] : []),
    ...(options.title !== undefined ? ['--title', options.title] : []),
    ...options.positional,
  ], { env, say });
  return [
    english.cli.loaded(changeName(loaded.review), loaded.counts),
    english.cli.open(reviewAddress(loaded.id)),
    ...loaded.notes.map(english.cli.note),
  ];
}

async function attach({ store, home }, options) {
  const found = store.find(PullRef.parse(options.positional[0]));
  if (!found) throw new Error(english.cli.notLoaded(options.positional[0]));
  const text = readFileSync(options.positional[1], 'utf8');
  await store.change([found.ref], async () => {
    const review = store.read(found.ref);
    if (review.moved) throw new Error(english.cli.reviewMoved(review.moved.to));
    checkWrittenReview(readWrittenReview(text), review);
    return { files: [[home.review(found.ref).written, text]], result: null };
  });
  return [english.cli.attached(found.ref.id, reviewAddress(found.ref.id))];
}

function list({ store }) {
  const reviews = store.list();
  if (!reviews.length) return [english.cli.noReviews];
  return reviews.map(({ ref, review }) => (review.moved
    ? english.cli.listedMoved(ref.id, review.moved.to)
    : english.cli.listed(reviewAddress(ref.id), changeName(review), review.title)));
}

async function stop({ home, oldServer }) {
  const answers = await stopAnswers(home);
  const server = await oldServer.stop();
  if (!answers && !server) return [english.cli.nothingRunning];
  return [...(answers ? [english.cli.answersStopped(answers)] : []), ...(server ? [english.cli.oldServerStopped] : [])];
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
      const unused = read.unused.map(key => english.config.unused(key, UNUSED[key]));
      return [read.exists ? english.config.checked(read.path) : english.config.unset(read.path), '', checkedConfigText(read).trimEnd(), ...(unused.length ? ['', ...unused] : [])];
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

function reviewAddress(id) {
  return `lazyreview://${id.split('/').map(encodeURIComponent).join('/')}`;
}

async function reviewToOpen(text, { app, env, say }) {
  const { review, pull } = readTerminalLink(text);
  if (review) {
    if (!app.store.read(review)) throw new Error(english.cli.noReview(review.id));
    return review.id;
  }
  const found = app.store.find(pull);
  if (found) return found.ref.id;
  say(english.cli.loading(pull.url));
  const loaded = await app.terminal.load([text], { env, say });
  for (const note of loaded.notes) say(english.cli.note(note));
  return loaded.id;
}

function readConfigFile(path) {
  try {
    return JSON.parse(readFileSync(path, 'utf8'));
  } catch (error) {
    throw new Error(english.cli.configUnreadable(path, error.message));
  }
}
