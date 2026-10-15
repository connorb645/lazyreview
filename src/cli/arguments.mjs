/**
 * Reads the review-desk command line.
 *
 * parseArguments(argv) returns {command, ...options}, or throws an error whose
 * message says what was wrong and ends with the usage.
 */
export const USAGE = `usage: review-desk <command> [options]

  load <pull link>...    load a pull request into the app, or a stack of
                         them, bottom first, and print where to open it;
                         loading one again refreshes it
      --account <login>  act as this GitHub account, not gh's active one
      --config <file>    JSON naming the components and packages to draw
      --no-context       leave out the unchanged code the change touches
  attach <pull link> <file>
                         attach a written review, in Markdown, to the review
                         holding that pull request
  list                   list every review, with where to open it
  serve                  run the app in this terminal
  start                  run the app in the background
  stop                   stop the app running in the background
  help                   show this`;

const COMMANDS = {
  load: { min: 1, valued: ['--account', '--config'], flags: ['--no-context'] },
  attach: { min: 2, max: 2 },
  list: { max: 0 },
  serve: { max: 0 },
  start: { max: 0 },
  stop: { max: 0 },
  help: { max: 0 },
};

export function parseArguments(argv) {
  const fail = message => { throw new Error(`${message}\n\n${USAGE}`); };
  const [command = 'help', ...rest] = argv;
  const spec = COMMANDS[command === '--help' || command === '-h' ? 'help' : command];
  if (!spec) fail(`unknown command "${command}"`);
  const options = { command: command.startsWith('-') ? 'help' : command, positional: [] };
  for (let i = 0; i < rest.length; i++) {
    const a = rest[i];
    if (a === '--help' || a === '-h') return { command: 'help', positional: [] };
    if ((spec.flags || []).includes(a)) { options[name(a)] = true; continue; }
    if ((spec.valued || []).includes(a)) {
      if (i + 1 >= rest.length || rest[i + 1].startsWith('--')) fail(`${a} needs a value`);
      options[name(a)] = rest[++i];
      continue;
    }
    if (a.startsWith('--')) fail(`unknown option ${a} for ${command}`);
    options.positional.push(a);
  }
  if (spec.min && options.positional.length < spec.min) fail(`${command} needs ${spec.min === 1 ? 'a pull request' : `${spec.min} arguments`}`);
  if (spec.max !== undefined && options.positional.length > spec.max) fail(`${command} takes ${spec.max || 'no'} argument${spec.max === 1 ? '' : 's'}`);
  return options;
}

function name(option) {
  return option.slice(2).replace(/-([a-z])/g, (_, c) => c.toUpperCase());
}
