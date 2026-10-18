/**
 * Reads lazyreview's config: the file, on top of the defaults, then the
 * LAZYREVIEW_ variables over both.
 *
 * readUserConfig({env, platform, home}) finds config.yml as findConfigFile
 * says, reads it, and returns {path, exists, values, sources}: `values` is
 * the whole config, every setting filled in, and `sources` names where each
 * setting's value came from, by its dotted key: "default", "file", or the
 * variable that set it. A missing file means every default. A file that is
 * not YAML, or that holds a setting lazyreview does not have or a value it
 * cannot take, fails with one error naming the file and every mistake in
 * it; a setting it no longer has is named with what does its work now. A variable with a value it cannot take fails as it always did.
 *
 * Objects merge key by key, at every depth; a list or a single value in
 * the file replaces the default whole, as lazygit's YAML reader merges.
 *
 * terminalPart(config, env) is what the terminal app is handed: the file's
 * path, the editor, chosen from LAZYREVIEW_EDITOR, then os.editor, then
 * VISUAL, then EDITOR, then vi, the theme's colours, each as text, how the
 * diff numbers its lines, and the keys of every place, each action's as a list of its key sequences as the
 * terminal app's key reader names them; a numbered list keeps its places,
 * with "" for a place that has no key.
 */
import { readFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join, resolve, isAbsolute } from 'node:path';
import { english } from '../i18n/english.mjs';
import { SCHEMA, PATTERNS, RETIRED, defaults, leaves } from './schema.mjs';
import { findConfigFile } from './config-file.mjs';
import { parseYAML } from './yaml.mjs';
import { readerName, labelProblem, DISABLED } from './key-labels.mjs';

const words = english.config;
const LOGIN = new RegExp(PATTERNS.login);
const TEAM = new RegExp(PATTERNS.team);
const DATE = new RegExp(PATTERNS.date);
const COLOUR = new RegExp(PATTERNS.colour);
const PULL_COLOURS = 8;

export function readUserConfig({ env = process.env, platform = process.platform, home = homedir() } = {}) {
  const file = findConfigFile({ env, platform, home });
  let text = '';
  if (file.exists) {
    try {
      text = readFileSync(file.path, 'utf8');
    } catch (error) {
      throw new Error(words.unreadable(file.path, error.message));
    }
  }
  const read = configFrom(text, file.path, home);
  return Object.freeze({ path: file.path, exists: file.exists, ...overridden(read, env) });
}

export function terminalPart(config, env) {
  return {
    path: config.path, editor: config.values.os.editor || env.VISUAL || env.EDITOR || 'vi', theme: config.values.gui.theme,
    lineNumbers: config.values.gui.lineNumbers, keybinding: readerNames(config.values.keybinding),
  };
}

/** The config a file's text gives, on top of the defaults, with each setting's source. */
export function configFrom(text, path, home = homedir()) {
  let given;
  try {
    given = parseYAML(text);
  } catch (error) {
    throw new Error(words.unreadable(path, error.message));
  }
  const values = defaults();
  const sources = Object.fromEntries(leaves().map(({ key }) => [key, 'default']));
  if (given === undefined || given === null) return { values: expanded(values, home), sources };
  if (!isMap(given)) throw new Error(words.problem(path, words.notAMap));
  const problems = [];
  merge(SCHEMA, given, values, sources, '', problems);
  if (problems.length === 1) throw new Error(words.problem(path, problems[0]));
  if (problems.length) throw new Error(words.problems(path, problems));
  return { values: expanded(values, home), sources };
}

function merge(section, given, values, sources, prefix, problems) {
  for (const [name, value] of Object.entries(given)) {
    const key = prefix ? `${prefix}.${name}` : name;
    const node = Object.hasOwn(section.settings, name) ? section.settings[name] : null;
    if (!node) { problems.push(Object.hasOwn(RETIRED, key) ? words.retired(key, RETIRED[key]) : words.unknown(key)); continue; }
    if (node.kind === 'section') {
      if (!isMap(value)) { problems.push(words.notASection(key, value)); continue; }
      merge(node, value, values[name], sources, key, problems);
      continue;
    }
    const found = CHECKS[node.check](value, key, node);
    if (found.problems.length) { problems.push(...found.problems); continue; }
    values[name] = found.value;
    sources[key] = 'file';
  }
}

const CHECKS = {
  folder: (value, key) => (typeof value === 'string' && (value === '~' || value.startsWith('~/') || isAbsolute(value))
    ? ok(value) : wrong(words.notAFolder(key, value))),
  port: (value, key) => (Number.isInteger(value) && value >= 1 && value <= 65535 ? ok(value) : wrong(words.notAPort(key, value))),
  command: (value, key) => (typeof value === 'string' && value.trim() ? ok(value) : wrong(words.notACommand(key, value))),
  text: (value, key) => (typeof value === 'string' ? ok(value) : wrong(words.notText(key, value))),
  choice: (value, key, setting) => (setting.choices.includes(value) ? ok(value) : wrong(words.notAChoice(key, value, setting.choices))),
  accounts: (value, key) => checkAccounts(value, key),
  colour: (value, key) => (isColour(value) ? ok(String(value)) : wrong(words.notAColour(key, value))),
  colours: (value, key) => (Array.isArray(value) && value.length === PULL_COLOURS && value.every(isColour)
    ? ok(value.map(String)) : wrong(words.notColours(key, value, PULL_COLOURS))),
  keys: (value, key) => {
    const labels = [value].flat();
    const why = Array.isArray(value) && !value.length ? words.keys.empty : labels.map(labelProblem).find(Boolean);
    return why ? wrong(words.notKeys(key, value, why)) : ok(value);
  },
  numberedKeys: (value, key, setting) => (Array.isArray(value) && value.length === setting.default.length && value.every(label => !labelProblem(label))
    ? ok(value) : wrong(words.notNumberedKeys(key, value, setting.default.length))),
};

function readerNames(keybinding) {
  return Object.fromEntries(Object.entries(keybinding).map(([place, actions]) => [place, Object.fromEntries(Object.entries(actions).map(([action, value]) => [action,
    SCHEMA.settings.keybinding.settings[place].settings[action].check === 'numberedKeys'
      ? value.map(readerName)
      : [value].flat().filter(label => label !== DISABLED).map(readerName)]))]));
}

function isColour(value) {
  return (typeof value === 'string' && COLOUR.test(value)) || (Number.isInteger(value) && value >= 0 && value <= 255);
}

/** The accounts a list gives, as {login, teams: [{name, since}]}, and every problem in it, in the config's words. */
export function checkAccounts(value, key) {
  if (!Array.isArray(value)) return wrong(words.notAccounts(key, value));
  const problems = [];
  const logins = new Set();
  const read = value.map((account, i) => {
    if (!isMap(account) || typeof account.login !== 'string' || !LOGIN.test(account.login)) { problems.push(words.noLogin(i + 1)); return null; }
    for (const field of Object.keys(account)) if (!['login', 'teams'].includes(field)) problems.push(words.accountKey(account.login, field));
    if (logins.has(account.login)) problems.push(words.loginTwice(account.login));
    logins.add(account.login);
    const teams = account.teams ?? [];
    if (!Array.isArray(teams)) { problems.push(words.teamsNotList(account.login)); return null; }
    return {
      login: account.login,
      teams: teams.map(team => {
        if (!isMap(team) || typeof team.name !== 'string' || !TEAM.test(team.name)) { problems.push(words.teamNotNamed(account.login)); return null; }
        for (const field of Object.keys(team)) if (!['name', 'since'].includes(field)) problems.push(words.teamKey(team.name, field));
        if (team.since !== undefined && (typeof team.since !== 'string' || !DATE.test(team.since))) problems.push(words.sinceNotDate(team.name));
        return { name: team.name, since: team.since ?? null };
      }),
    };
  });
  return problems.length ? wrong(...problems) : ok(read);
}

// The variables over the file and the defaults, each as it always read: an empty one is not set.
function overridden({ values, sources }, env) {
  const set = (key, name, value) => { if (env[name]) { value(env[name]); sources[key] = name; } };
  set('dataHome', 'LAZYREVIEW_HOME', given => { values.dataHome = resolve(given); });
  set('server.port', 'LAZYREVIEW_PORT', given => {
    const port = Number(given);
    if (!Number.isInteger(port) || port < 1 || port > 65535) throw new Error(words.badPort(given));
    values.server.port = port;
  });
  set('claude.command', 'LAZYREVIEW_CLAUDE', given => { values.claude.command = given; });
  set('claude.model', 'LAZYREVIEW_MODEL', given => { values.claude.model = given; });
  set('os.editor', 'LAZYREVIEW_EDITOR', given => { values.os.editor = given; });
  return { values, sources };
}

function expanded(values, home) {
  if (values.dataHome === '~') values.dataHome = home;
  else if (values.dataHome.startsWith('~/')) values.dataHome = join(home, values.dataHome.slice(2));
  return values;
}

const ok = value => ({ value, problems: [] });
const wrong = (...problems) => ({ value: null, problems });
const isMap = value => !!value && typeof value === 'object' && !Array.isArray(value);
