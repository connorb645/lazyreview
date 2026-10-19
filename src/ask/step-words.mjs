/**
 * Says what one of Claude's tool calls does, in the words both apps show
 * while Claude works and under its answer after.
 *
 * stepWords(tool, input, folder) is pure. It takes the tool's name, its
 * input as Claude Code gives it, and `folder`, the copy of the code, whose
 * paths it names from the copy's top, and a command that starts by going
 * there is named without it. Long words are cut.
 */
import { isAbsolute, relative, sep } from 'node:path';
import { english } from '../i18n/english.mjs';

const LONG = 160;

export function stepWords(tool, input, folder) {
  const words = english.ask.steps;
  const path = value => shown(String(value || ''), folder);
  switch (tool) {
    case 'Read': case 'NotebookRead': return words.read(path(input.file_path || input.notebook_path));
    case 'Grep': return words.searched(cut(String(input.pattern || '')), input.path ? path(input.path) : null);
    case 'Glob': return words.listed(cut(String(input.pattern || '')));
    case 'Bash': case 'Monitor': return words.ran(cut(inCopy(String(input.command || '').split('\n')[0], folder)));
    case 'Edit': case 'MultiEdit': case 'NotebookEdit': return words.edited(path(input.file_path || input.notebook_path));
    case 'Write': return words.wrote(path(input.file_path));
    case 'WebFetch': return words.fetched(cut(String(input.url || '')));
    case 'WebSearch': return words.searchedWeb(cut(String(input.query || '')));
    case 'Task': case 'Agent': return words.helper(cut(String(input.description || input.prompt || '').toLowerCase()));
    case 'Skill': return words.skill(String(input.skill || input.name || ''));
    case 'TodoWrite': return words.planned;
  }
  if (tool.startsWith('mcp__')) {
    const [, server, name] = tool.split('__');
    return words.mcp(name, server);
  }
  return words.used(tool);
}

function inCopy(command, folder) {
  const quoted = folder.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  return command.replace(new RegExp(`^cd\\s+(["']?)${quoted}/?\\1\\s*(&&|;)\\s*`), '').replaceAll(`${folder}/`, '');
}

function shown(path, folder) {
  if (!path || !isAbsolute(path)) return path;
  const inside = relative(folder, path);
  return inside && !inside.startsWith('..') && !isAbsolute(inside) ? inside.split(sep).join('/') : path;
}

function cut(text) {
  return text.length > LONG ? `${text.slice(0, LONG - 1)}…` : text;
}
