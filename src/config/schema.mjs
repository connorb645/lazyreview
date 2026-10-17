/**
 * Every setting of lazyreview's config, once: its default, its check, the
 * environment variable that overrides it, and the words that say what it
 * does. The reader, the documentation and the editors' schema are all made
 * from this tree, as lazygit makes its own from GetDefaultConfig.
 *
 * SCHEMA is a tree of sections, {about, settings, flow}, whose leaves are
 * settings, {check, default, env, about, example}. `check` names how the
 * reader checks a value; `about` and `example` are lines of the default
 * config's comments, and a section with `flow` is written on one line. defaults() returns the default config as plain values,
 * and leaves(section) every setting with its dotted key, such as
 * "server.port", in the tree's order. PATTERNS are what a login, a team's
 * name and a date must match, as regular expressions' sources.
 */
import { english } from '../i18n/english.mjs';

const words = english.config.about;

export const PATTERNS = Object.freeze({
  colour: '^#[0-9a-fA-F]{6}$',
  login: '^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?$',
  team: '^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?/[A-Za-z0-9_.-]+$',
  date: '^\\d{4}-\\d{2}-\\d{2}$',
});

const setting = fields => Object.freeze({ kind: 'setting', env: null, example: [], ...fields });
const section = (settings, about = [], { flow = false } = {}) => Object.freeze({ kind: 'section', about, flow, settings: Object.freeze(settings) });
const colour = value => setting({ check: 'colour', default: value, about: [] });

// A terminal's colours, one for each role, then a stack's pull requests' and the components'.
const colours = given => section({
  ...Object.fromEntries(Object.entries(given).filter(([name]) => name !== 'pullRequestColors' && name !== 'componentColors').map(([name, value]) => [name, colour(value)])),
  pullRequestColors: setting({ check: 'colours', default: given.pullRequestColors, about: [] }),
  componentColors: section(Object.fromEntries(Object.entries(given.componentColors).map(([name, value]) => [name, colour(value)])), [], { flow: true }),
});

const DARK = {
  defaultFgColor: '#f1f0fb', mutedFgColor: '#a09dbf', faintFgColor: '#6d6a8a', accentColor: '#a796ff', activeBorderColor: '#a796ff', inactiveBorderColor: '#4a4766',
  addedColor: '#7ce2a7', removedColor: '#ff9fb6', warningColor: '#f2b26a', selectedLineBgColor: '#3a3270', inactiveViewSelectedLineBgColor: '#26233d', visualSelectionBgColor: '#45475a',
  pullRequestColors: ['#3987e5', '#d95926', '#199e70', '#c98500', '#d55181', '#008300', '#9085e9', '#e66767'],
  componentColors: { blue: '#7fa8f8', teal: '#4cd3c2', purple: '#b893fb', amber: '#f2b26a', olive: '#b8cc6a', rose: '#f08cb6', indigo: '#a4a9ff', slate: '#a3afc8', grey: '#9d9bb8' },
};

const LIGHT = {
  defaultFgColor: '#17152f', mutedFgColor: '#615e7f', faintFgColor: '#9b98b5', accentColor: '#5233e6', activeBorderColor: '#5233e6', inactiveBorderColor: '#c4c0dc',
  addedColor: '#146b3a', removedColor: '#b21f43', warningColor: '#9c5212', selectedLineBgColor: '#dcd4ff', inactiveViewSelectedLineBgColor: '#efebfb', visualSelectionBgColor: '#c9cbd6',
  pullRequestColors: ['#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#e87ba4', '#008300', '#6250d6', '#e34948'],
  componentColors: { blue: '#2c66a8', teal: '#0b6e6e', purple: '#7448ad', amber: '#9c5212', olive: '#5f6e12', rose: '#a8325e', indigo: '#4a4fb0', slate: '#556170', grey: '#6f7987' },
};

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
  gui: section({
    theme: section({ dark: colours(DARK), light: colours(LIGHT) }, words.theme),
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
