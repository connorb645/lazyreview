/**
 * The parts of the config's documentation made from the schema, as lazygit
 * writes its own from its config struct.
 *
 * configSchema() is the JSON schema editors read for config.yml, as
 * schema/config.json holds it. documented(markdown) is docs/Config.md with
 * its generated parts written again from the schema: the table of the
 * variables that override a setting, and the default config. Each part sits
 * between its markers, which stay as they are.
 */
import { english } from '../i18n/english.mjs';
import { SCHEMA, PATTERNS, leaves } from './schema.mjs';
import { defaultConfigText } from './config-text.mjs';

const words = english.config.docs;

export function configSchema() {
  return {
    $schema: 'http://json-schema.org/draft-07/schema#',
    title: words.schemaTitle,
    description: words.schemaDescription,
    ...sectionSchema(SCHEMA),
  };
}

export function documented(markdown) {
  return [
    ['VARIABLES', variablesTable()],
    ['DEFAULT CONFIG', ['```yaml', ...defaultConfigText().trimEnd().split('\n'), '```']],
  ].reduce((text, [name, lines]) => between(text, name, lines), markdown);
}

function between(markdown, name, lines) {
  const start = `<!-- START ${name}: ${words.generated} -->`;
  const end = `<!-- END ${name} -->`;
  const from = markdown.indexOf(start);
  const to = markdown.indexOf(end);
  if (from < 0 || to < from) throw new Error(words.noMarkers(name));
  return `${markdown.slice(0, from + start.length)}\n${lines.join('\n')}\n${markdown.slice(to)}`;
}

function variablesTable() {
  return [
    `| ${words.variable} | ${words.overrides} |`,
    '| --- | --- |',
    ...leaves().filter(({ setting }) => setting.env).map(({ key, setting }) => `| \`${setting.env}\` | \`${key}\` |`),
  ];
}

function sectionSchema(section) {
  return {
    type: 'object',
    ...(section.about.length ? { description: section.about.join(' ') } : {}),
    properties: Object.fromEntries(Object.entries(section.settings).map(([name, node]) => [name, node.kind === 'section' ? sectionSchema(node) : settingSchema(node)])),
    additionalProperties: false,
  };
}

function settingSchema(setting) {
  const counted = setting.check === 'numberedKeys' ? { minItems: setting.default.length, maxItems: setting.default.length } : {};
  return { ...TYPES[setting.check], ...counted, ...(setting.about.length ? { description: setting.about.join(' ') } : {}), default: setting.default };
}

const TYPES = {
  folder: { type: 'string', pattern: '^(~|~/.*|/.*)$' },
  port: { type: 'integer', minimum: 1, maximum: 65535 },
  command: { type: 'string', minLength: 1 },
  text: { type: 'string' },
  colour: { oneOf: [{ type: 'string', pattern: PATTERNS.colour }, { type: 'integer', minimum: 0, maximum: 255 }] },
  keys: { oneOf: [{ type: 'string', minLength: 1 }, { type: 'array', minItems: 1, items: { type: 'string', minLength: 1 } }] },
  numberedKeys: { type: 'array', items: { type: 'string', minLength: 1 } },
  colours: { type: 'array', minItems: 8, maxItems: 8, items: { oneOf: [{ type: 'string', pattern: PATTERNS.colour }, { type: 'integer', minimum: 0, maximum: 255 }] } },
  accounts: {
    type: 'array',
    items: {
      type: 'object',
      required: ['login'],
      properties: {
        login: { type: 'string', pattern: PATTERNS.login },
        teams: {
          type: 'array',
          items: {
            type: 'object',
            required: ['name'],
            properties: { name: { type: 'string', pattern: PATTERNS.team }, since: { type: 'string', pattern: PATTERNS.date } },
            additionalProperties: false,
          },
        },
      },
      additionalProperties: false,
    },
  },
};
