/**
 * Every setting of lazyreview's config, once: its default, its check, the
 * environment variable that overrides it, and the words that say what it
 * does. The reader, the documentation and the editors' schema are all made
 * from this tree, as lazygit makes its own from GetDefaultConfig.
 *
 * SCHEMA is a tree of sections, {about, settings}, whose leaves are
 * settings, {check, default, env, about, example}. `check` names how the
 * reader checks a value; `about` and `example` are lines of the default
 * config's comments. defaults() returns the default config as plain values,
 * and leaves(section) every setting with its dotted key, such as
 * "server.port", in the tree's order. PATTERNS are what a login, a team's
 * name and a date must match, as regular expressions' sources.
 */
import { english } from '../i18n/english.mjs';

const words = english.config.about;

export const PATTERNS = Object.freeze({
  login: '^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?$',
  team: '^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?/[A-Za-z0-9_.-]+$',
  date: '^\\d{4}-\\d{2}-\\d{2}$',
});

const setting = fields => Object.freeze({ kind: 'setting', env: null, example: [], ...fields });
const section = (settings, about = []) => Object.freeze({ kind: 'section', about, settings: Object.freeze(settings) });

export const SCHEMA = section({
  dataHome: setting({ check: 'folder', default: '~/.lazyreview', env: 'LAZYREVIEW_HOME', about: words.dataHome }),
  server: section({
    port: setting({ check: 'port', default: 4747, env: 'LAZYREVIEW_PORT', about: words.port }),
  }),
  claude: section({
    command: setting({ check: 'command', default: 'claude', env: 'LAZYREVIEW_CLAUDE', about: words.claudeCommand }),
    model: setting({ check: 'text', default: '', env: 'LAZYREVIEW_MODEL', about: words.claudeModel }),
  }),
  os: section({
    editor: setting({ check: 'text', default: '', env: 'LAZYREVIEW_EDITOR', about: words.editor }),
  }),
  github: section({
    accounts: setting({ check: 'accounts', default: [], about: words.accounts, example: words.accountsExample }),
  }),
});

export function defaults(node = SCHEMA) {
  if (node.kind === 'setting') return structuredClone(node.default);
  return Object.fromEntries(Object.entries(node.settings).map(([name, inner]) => [name, defaults(inner)]));
}

export function leaves(node = SCHEMA, prefix = '') {
  if (node.kind === 'setting') return [{ key: prefix, setting: node }];
  return Object.entries(node.settings).flatMap(([name, inner]) => leaves(inner, prefix ? `${prefix}.${name}` : name));
}
