/**
 * The one-time move of settings.json and the LAZYREVIEW_ variables into the
 * config file.
 *
 * planMove({env, platform, home, files}) reads what would move and writes
 * nothing: the accounts in settings.json in the data home, and each
 * variable that is lazyreview's own and set. It returns {configPath,
 * configExists, settingsPath, settingsExists, values, from, variables}:
 * `values` is the config to write, holding only what differs from the
 * defaults, `from` where each value came from, and `variables` every one of
 * those variables set, written or not.
 *
 * checkLines(plan) is what the check prints: the file it would write, where
 * each value comes from, the rename it would make, and that the move would
 * refuse when a config file exists. move(plan, {files}) makes the move: it
 * refuses when a config file exists, writes the file under a temporary
 * name, renames settings.json to settings.json.moved, keeping its bytes,
 * then gives the file its name. If a step fails it puts back what was done.
 * It returns the lines saying what it did, and names each variable still
 * set, which still overrides the config while it is set.
 */
import { existsSync, statSync, readFileSync, writeFileSync, renameSync, rmSync, mkdirSync } from 'node:fs';
import { join, resolve, dirname, basename } from 'node:path';
import { english } from '../i18n/english.mjs';
import { findConfigFile } from './config-file.mjs';
import { defaults } from './schema.mjs';
import { movedConfigText } from './config-text.mjs';
import { checkAccounts } from './user-config.mjs';

const words = english.config.move;
const MOVED = 'settings.json.moved';
const VARIABLES = [
  ['LAZYREVIEW_HOME', 'dataHome'],
  ['LAZYREVIEW_PORT', 'server.port'],
  ['LAZYREVIEW_CLAUDE', 'claude.command'],
  ['LAZYREVIEW_MODEL', 'claude.model'],
  ['LAZYREVIEW_EDITOR', 'os.editor'],
];

const nodeFiles = {
  exists: path => existsSync(path) && statSync(path).isFile(),
  read: path => readFileSync(path, 'utf8'),
  write: (path, text) => writeFileSync(path, text, { flag: 'wx' }),
  rename: (from, to) => renameSync(from, to),
  remove: path => rmSync(path, { force: true }),
  folder: path => mkdirSync(path, { recursive: true }),
};

export function planMove({ env, platform, home, files = nodeFiles }) {
  const config = findConfigFile({ env, platform, home, exists: files.exists });
  const dataHome = env.LAZYREVIEW_HOME ? resolve(env.LAZYREVIEW_HOME) : join(home, '.lazyreview');
  const settingsPath = join(dataHome, 'settings.json');
  const settingsExists = files.exists(settingsPath);
  const values = {};
  const from = [];
  const fallback = defaults();
  const give = (key, value, source) => {
    const [section, name] = key.includes('.') ? key.split('.') : [null, key];
    const was = section ? fallback[section][name] : fallback[name];
    if (JSON.stringify(value) === JSON.stringify(was)) return;
    if (section) (values[section] ??= {})[name] = value; else values[name] = value;
    from.push({ key, source });
  };
  for (const [name, key] of VARIABLES) {
    if (!env[name]) continue;
    if (key === 'server.port') {
      const port = Number(env[name]);
      if (!Number.isInteger(port) || port < 1 || port > 65535) throw new Error(english.config.badPort(env[name]));
      give(key, port, name);
    } else {
      give(key, key === 'dataHome' ? dataHome : env[name], name);
    }
  }
  if (settingsExists) give('github.accounts', readSettingsFile(settingsPath, files), settingsPath);
  return Object.freeze({
    configPath: config.path, configExists: config.exists, settingsPath, settingsExists,
    values, from, variables: VARIABLES.map(([name]) => name).filter(name => env[name]),
  });
}

export function checkLines(plan) {
  if (nothing(plan)) return [words.nothing(plan.settingsPath)];
  const out = [words.wouldWrite(plan.configPath), '', ...indented(movedConfigText(plan.values)), ''];
  out.push(...plan.from.map(({ key, source }) => words.from(key, source)));
  if (plan.settingsExists) out.push(words.wouldRename(plan.settingsPath, MOVED));
  if (plan.configExists) out.push(words.refused(plan.configPath));
  out.push(words.nothingWritten);
  return out;
}

export function move(plan, { files = nodeFiles } = {}) {
  if (nothing(plan)) return [words.nothing(plan.settingsPath)];
  if (plan.configExists || files.exists(plan.configPath)) throw new Error(words.refused(plan.configPath));
  const moved = join(dirname(plan.settingsPath), MOVED);
  if (plan.settingsExists && files.exists(moved)) throw new Error(words.movedExists(moved));
  const text = movedConfigText(plan.values);
  const temporary = join(dirname(plan.configPath), `.${basename(plan.configPath)}.${process.pid}.moving`);
  const undo = [];
  try {
    files.folder(dirname(plan.configPath));
    files.write(temporary, text);
    undo.push(() => files.remove(temporary));
    if (plan.settingsExists) {
      files.rename(plan.settingsPath, moved);
      undo.push(() => files.rename(moved, plan.settingsPath));
    }
    files.rename(temporary, plan.configPath);
  } catch (error) {
    const stuck = [];
    for (const step of undo.reverse()) {
      try {
        step();
      } catch (undoing) {
        stuck.push(undoing.message);
      }
    }
    throw new Error(words.failed(error.message, stuck));
  }
  const out = [words.wrote(plan.configPath), '', ...indented(text), ''];
  if (plan.settingsExists) out.push(words.renamed(plan.settingsPath, MOVED));
  for (const name of plan.variables) out.push(words.stillSet(name, VARIABLES.find(([n]) => n === name)[1]));
  return out;
}

// The accounts in settings.json, checked as the config checks them, each written with only what it gives.
function readSettingsFile(path, files) {
  let settings;
  try {
    settings = JSON.parse(files.read(path));
  } catch (error) {
    throw new Error(english.config.unreadable(path, error.message));
  }
  if (!settings || typeof settings !== 'object' || !Array.isArray(settings.accounts)) throw new Error(english.config.problem(path, words.needsAccounts));
  const checked = checkAccounts(settings.accounts, 'accounts');
  if (checked.problems.length === 1) throw new Error(english.config.problem(path, checked.problems[0]));
  if (checked.problems.length) throw new Error(english.config.problems(path, checked.problems));
  return checked.value.map(account => (account.teams.length
    ? { login: account.login, teams: account.teams.map(team => (team.since ? team : { name: team.name })) }
    : { login: account.login }));
}

function nothing(plan) {
  return !plan.settingsExists && !plan.variables.length;
}

function indented(text) {
  return text.trimEnd().split('\n').map(line => `  ${line}`);
}
