/**
 * The config written out as YAML, in the schema's order.
 *
 * defaultConfigText() is the default config with the words saying what each
 * setting does and, beside it, the variable that overrides it: what
 * `lazyreview config defaults` prints and docs/Config.md shows.
 * checkedConfigText(config) is a config as read, every setting with where
 * its value came from beside it. movedConfigText(values) is the file the
 * move writes: the line that gives an editor the schema, then the values.
 */
import { english } from '../i18n/english.mjs';
import { SCHEMA } from './schema.mjs';
import { dumpYAML } from './yaml.mjs';

const COMMENT_COLUMN = 39;
const words = english.config;

export const SCHEMA_ADDRESS = 'https://raw.githubusercontent.com/connorb645/lazyreview/main/schema/config.json';

export function defaultConfigText() {
  return lines(SCHEMA, null, '', {
    about: setting => setting.about,
    beside: setting => (setting.env ? `# ${setting.env}` : ''),
    after: setting => setting.example.map(line => `# ${line}`),
  }).join('\n') + '\n';
}

export function checkedConfigText(config) {
  return lines(SCHEMA, config.values, '', {
    about: () => [],
    beside: (setting, key) => `# ${words.source(config.sources[key])}`,
    after: () => [],
  }).join('\n') + '\n';
}

export function movedConfigText(values) {
  return [`# yaml-language-server: $schema=${SCHEMA_ADDRESS}`, `# ${words.movedHeader}`, dumpYAML(values).trimEnd()].join('\n') + '\n';
}

function lines(section, values, indent, show, prefix = '') {
  const out = [];
  for (const [name, node] of Object.entries(section.settings)) {
    const key = prefix ? `${prefix}.${name}` : name;
    if (!indent && out.length) out.push('');
    if (node.kind === 'section') {
      out.push(...node.about.map(line => `${indent}# ${line}`), `${indent}${name}:`);
      out.push(...lines(node, values?.[name], `${indent}  `, show, key));
      continue;
    }
    const value = values ? values[name] : node.default;
    out.push(...show.about(node).map(line => `${indent}# ${line}`));
    const block = isBlock(value);
    out.push(besides(`${indent}${name}:${block ? '' : ` ${scalar(value)}`}`, show.beside(node, key)));
    if (block) out.push(...dumpYAML(value).trimEnd().split('\n').map(line => `${indent}  ${line}`));
    out.push(...show.after(node).map(line => `${indent}${line}`));
  }
  return out;
}

function besides(text, comment) {
  return comment ? `${text.padEnd(COMMENT_COLUMN - 1)} ${comment}` : text;
}

function isBlock(value) {
  return Array.isArray(value) && value.some(item => item && typeof item === 'object');
}

function scalar(value) {
  if (Array.isArray(value)) return `[${value.map(scalar).join(', ')}]`;
  if (typeof value !== 'string') return String(value);
  return /^[A-Za-z0-9~/][A-Za-z0-9~/._-]*$/.test(value) && !/^(true|false|null|~|[0-9][0-9._]*)$/i.test(value) ? value : `'${value.replace(/'/g, "''")}'`;
}
