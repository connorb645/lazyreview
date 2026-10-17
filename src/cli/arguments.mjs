/**
 * Reads the lazyreview command line.
 *
 * parseArguments(argv) returns {command, ...options}, or throws an error whose
 * message says what was wrong and ends with the usage.
 */
import { english } from '../i18n/english.mjs';

const COMMANDS = {
  load: { valued: ['--account', '--config', '--repo', '--base', '--head', '--title'], flags: ['--context', '--no-context'], check: checkLoad },
  attach: { min: 2, max: 2 },
  list: { max: 0 },
  terminal: { max: 1 },
  serve: { max: 0 },
  start: { max: 0 },
  stop: { max: 0 },
  help: { max: 0 },
};

export function parseArguments(argv) {
  const fail = message => { throw new Error(english.cli.withUsage(message)); };
  const [command = 'help', ...rest] = argv;
  const spec = COMMANDS[command === '--help' || command === '-h' ? 'help' : command];
  if (!spec) fail(english.cli.unknownCommand(command));
  const options = { command: command.startsWith('-') ? 'help' : command, positional: [] };
  for (let i = 0; i < rest.length; i++) {
    const a = rest[i];
    if (a === '--help' || a === '-h') return { command: 'help', positional: [] };
    if ((spec.flags || []).includes(a)) { options[name(a)] = true; continue; }
    if ((spec.valued || []).includes(a)) {
      if (i + 1 >= rest.length || rest[i + 1].startsWith('--')) fail(english.cli.needsValue(a));
      options[name(a)] = rest[++i];
      continue;
    }
    if (a.startsWith('--')) fail(english.cli.unknownOption(a, command));
    options.positional.push(a);
  }
  if (spec.min && options.positional.length < spec.min) fail(english.cli.needsArguments(command, spec.min));
  if (spec.max !== undefined && options.positional.length > spec.max) fail(english.cli.takesArguments(command, spec.max));
  if (spec.check) spec.check(options, fail);
  return options;
}

function checkLoad(options, fail) {
  if (options.repo === undefined) {
    const stray = ['--base', '--head', '--title'].find(option => options[name(option)] !== undefined);
    if (stray) fail(english.cli.needsRepo(stray));
    if (!options.positional.length) fail(english.cli.loadNeedsSomething);
    return;
  }
  if (options.positional.length) fail(english.cli.notBoth);
  if (options.account !== undefined) fail(english.cli.accountForPulls);
  for (const option of ['--base', '--head']) if (options[name(option)] === undefined) fail(english.cli.repoNeeds(option));
}

function name(option) {
  return option.slice(2).replace(/-([a-z])/g, (_, c) => c.toUpperCase());
}
