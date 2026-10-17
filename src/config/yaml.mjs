/**
 * YAML, through the vendored js-yaml, with its core schema: a date such as
 * 2026-10-01 stays text, and a key written twice is refused.
 *
 * parseYAML(text) returns the value the text holds, or throws an error
 * saying why it cannot be read and where, as "<reason> (<line>:<column>)".
 * dumpYAML(value) returns the value as YAML text, quoting a date such as
 * '2026-10-01', so any reader takes it as text.
 */
import { createRequire } from 'node:module';
import { english } from '../i18n/english.mjs';

const yaml = createRequire(import.meta.url)('../../vendor/js-yaml-4.3.2.min.js');

export function parseYAML(text) {
  try {
    return yaml.load(text, { schema: yaml.CORE_SCHEMA });
  } catch (error) {
    if (!error.mark) throw error;
    throw new Error(english.config.yamlError(error.reason, error.mark.line + 1, error.mark.column + 1));
  }
}

export function dumpYAML(value) {
  return yaml.dump(value, { lineWidth: -1 });
}
