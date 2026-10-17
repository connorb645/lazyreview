/**
 * The one place lazyreview reads its configuration.
 *
 * readSettings(env) takes the environment and returns
 * {home, port, claude: {command, model}}:
 *
 *   LAZYREVIEW_HOME    where reviews and clones are kept (~/.lazyreview)
 *   LAZYREVIEW_PORT    the port the app listens on, on this machine only (4747)
 *   LAZYREVIEW_CLAUDE  the Claude Code command that answers questions (claude)
 *   LAZYREVIEW_MODEL   the model it answers with (Claude Code's own default)
 *
 * readAccounts(file) reads the GitHub accounts whose open pull requests the
 * terminal app's home screen lists, from the settings file in the data home:
 *
 *   {"accounts": [
 *     {"login": "lee-at-work", "teams": [{"name": "example-org/mobile", "since": "2026-10-01"}]},
 *     {"login": "lee"}
 *   ]}
 *
 * Each account is read by its own gh token. A review asked of one of an
 * account's teams counts as asked of the account, for pull requests opened
 * on or after the team's `since` date when it has one. It returns the
 * accounts as [{login, teams: [{name, since}]}], an empty list when there is
 * no settings file, and throws an error naming the file and what is wrong
 * when the file cannot be read as these settings.
 */
import { homedir } from 'node:os';
import { join, resolve } from 'node:path';
import { readJSON } from '../store/whole-file.mjs';
import { english } from '../i18n/english.mjs';

export const DEFAULT_PORT = 4747;

const LOGIN = /^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?$/;
const TEAM = /^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?\/[A-Za-z0-9_.-]+$/;
const DATE = /^\d{4}-\d{2}-\d{2}$/;

export function readSettings(env = process.env) {
  const port = env.LAZYREVIEW_PORT ? Number(env.LAZYREVIEW_PORT) : DEFAULT_PORT;
  if (!Number.isInteger(port) || port < 1 || port > 65535) throw new Error(english.settings.badPort(env.LAZYREVIEW_PORT));
  return {
    home: resolve(env.LAZYREVIEW_HOME || join(homedir(), '.lazyreview')),
    port,
    claude: { command: env.LAZYREVIEW_CLAUDE || 'claude', model: env.LAZYREVIEW_MODEL || null },
  };
}

export function readAccounts(file) {
  let settings;
  try {
    settings = readJSON(file);
  } catch (error) {
    throw new Error(english.settings.unreadable(file, error.message));
  }
  if (settings === null) return [];
  if (!settings || typeof settings !== 'object' || !Array.isArray(settings.accounts)) throw new Error(english.settings.needsAccounts(file));
  const logins = new Set();
  return settings.accounts.map((account, i) => {
    if (!account || typeof account.login !== 'string' || !LOGIN.test(account.login)) throw new Error(english.settings.noLogin(file, i + 1));
    if (logins.has(account.login)) throw new Error(english.settings.loginTwice(file, account.login));
    logins.add(account.login);
    const teams = account.teams ?? [];
    if (!Array.isArray(teams)) throw new Error(english.settings.teamsNotList(file, account.login));
    return {
      login: account.login,
      teams: teams.map(team => {
        if (!team || typeof team.name !== 'string' || !TEAM.test(team.name)) throw new Error(english.settings.teamNotNamed(file, account.login));
        if (team.since !== undefined && (typeof team.since !== 'string' || !DATE.test(team.since))) throw new Error(english.settings.sinceNotDate(file, team.name));
        return { name: team.name, since: team.since ?? null };
      }),
    };
  });
}
