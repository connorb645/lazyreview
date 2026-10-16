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
      --context          also draw the unchanged code the change touches
      --no-context       stop drawing it, for a review loaded with it
  load --repo <folder> --base <ref> --head <branch>
                         load a branch with no pull request, read only, as
                         its diff from base would show; loading the same
                         branch again refreshes it
      --title <text>     name the review; the branch's name otherwise
      --config, --context and --no-context work as above
  attach <pull link> <file>
                         attach a written review, in Markdown, to the review
                         holding that pull request
  list                   list every review, with where to open it
  serve                  run the app in this terminal
  start                  run the app in the background
  stop                   stop the app running in the background
  help                   show this`;

const COMMANDS = {
  load: { valued: ['--account', '--config', '--repo', '--base', '--head', '--title'], flags: ['--context', '--no-context'], check: checkLoad },
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
  if (spec.min && options.positional.length < spec.min) fail(`${command} needs ${spec.min} arguments`);
  if (spec.max !== undefined && options.positional.length > spec.max) fail(`${command} takes ${spec.max || 'no'} argument${spec.max === 1 ? '' : 's'}`);
  if (spec.check) spec.check(options, fail);
  return options;
}

function checkLoad(options, fail) {
  if (options.repo === undefined) {
    const stray = ['--base', '--head', '--title'].find(option => options[name(option)] !== undefined);
    if (stray) fail(`${stray} needs --repo`);
    if (!options.positional.length) fail('load needs a pull request, or --repo with --base and --head');
    return;
  }
  if (options.positional.length) fail('load takes pull requests or --repo, not both');
  if (options.account !== undefined) fail('--account is for pull requests; a branch loads from its own repository');
  for (const option of ['--base', '--head']) if (options[name(option)] === undefined) fail(`--repo needs ${option}`);
}

function name(option) {
  return option.slice(2).replace(/-([a-z])/g, (_, c) => c.toUpperCase());
}
