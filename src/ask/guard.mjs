/**
 * The guard on Claude Code while it answers in a throwaway copy of the code,
 * with nobody there to approve anything. Anything that would change
 * something outside the copy, or leave the machine as a change, is refused;
 * reading stays allowed.
 *
 * guardSettings({hook, domains}) is pure. It returns the settings lazyreview
 * hands Claude Code over the user's own, which outrank them: `hook`, a
 * command, is asked about every tool call, and hooks stay on whatever the
 * user's settings say. Shell commands run in Claude Code's sandbox, with no
 * way out of it: they write only in the copy and the temp folders, and reach
 * the network only for `domains`, the package registries, and any host the
 * user's own settings allow. `gh` runs outside the sandbox, so it can sign
 * in, and the hook lets it only read.
 *
 * decide({tool_name, tool_input, cwd}, {copy, real}) is pure. It takes one
 * tool call as Claude Code's PreToolUse hook gives it and returns {allow:
 * true} or {refuse: why}. `copy` is the copy's folder, and `real(path)`
 * names where a path really leads, through any link. It allows tools that
 * only read, file changes inside the copy, shell commands, which the
 * sandbox holds in, and the tools of an MCP server whose names say they only
 * read. It refuses a file change outside the copy, a shell command that
 * pushes, posts, publishes or reaches another machine, `gh` doing anything
 * but read, an MCP tool that may write, and any tool it does not know.
 */
import { isAbsolute, join, sep } from 'node:path';
import { english } from '../i18n/english.mjs';

export const REGISTRIES = [
  'registry.npmjs.org', 'registry.yarnpkg.com', 'pypi.org', 'files.pythonhosted.org', 'proxy.golang.org', 'sum.golang.org',
  'index.crates.io', 'static.crates.io', 'crates.io', 'rubygems.org', 'repo1.maven.org', 'repo.maven.apache.org',
  'plugins.gradle.org', 'services.gradle.org', 'dl.google.com',
];

const READS = new Set(['Read', 'Grep', 'Glob', 'LS', 'NotebookRead', 'WebFetch', 'WebSearch', 'Task', 'Agent', 'Skill', 'ToolSearch', 'LSP',
  'TodoWrite', 'TaskStop', 'TaskOutput', 'BashOutput', 'KillShell', 'ListAgents', 'CronList', 'ReportFindings',
  'ListMcpResourcesTool', 'ReadMcpResourceTool', 'ReadMcpResourceDirTool']);
const FILE_CHANGES = { Write: 'file_path', Edit: 'file_path', MultiEdit: 'file_path', NotebookEdit: 'notebook_path' };
const SHELLS = { Bash: 'command', Monitor: 'command' };

const READ_WORDS = new Set(['get', 'list', 'search', 'read', 'fetch', 'find', 'query', 'view', 'describe', 'show', 'lookup', 'inspect', 'count', 'browse', 'retrieve']);
const WRITE_WORDS = new Set(['create', 'update', 'delete', 'remove', 'post', 'send', 'write', 'edit', 'add', 'set', 'put', 'patch', 'upload', 'publish',
  'share', 'comment', 'reply', 'merge', 'close', 'assign', 'move', 'rename', 'archive', 'trash', 'label', 'mark', 'approve', 'submit', 'cancel', 'run',
  'execute', 'trigger', 'schedule', 'save', 'apply', 'transition', 'invite', 'react', 'resolve', 'enable', 'disable', 'start', 'stop', 'generate',
  'import', 'push', 'sync', 'insert', 'modify', 'change', 'draft', 'forward', 'respond', 'book', 'pay', 'deploy', 'restore', 'duplicate', 'convert']);

const COMMAND_START = String.raw`(?:^|[;&|(` + '`' + String.raw`\n]|\$\()\s*(?:(?:sudo|env|exec|command|nohup|time|xargs)\s+(?:-\S+\s+|\w+=\S*\s+)*)*`;
const run = (programs, rest = '') => new RegExp(COMMAND_START + String.raw`(?:\S*/)?(?:${programs})\b` + rest, 'i');

const OUTWARD = [
  [run('git', String.raw`[^;&|\n]*\s(push|send-email)\b`), 'push'],
  [run('curl|wget', String.raw`[^;&|\n]*(\s-X\s*['"]?(POST|PUT|PATCH|DELETE)\b|\s--request\b|\s-d\b|\s--data|\s-F\b|\s--form\b|\s-T\b|\s--upload-file\b|\s--json\b|\s--post-|\s--method\b)`), 'post'],
  [run('https?|xh', String.raw`\s+(POST|PUT|PATCH|DELETE)\b`), 'post'],
  [run('npm|yarn|pnpm|bun|cargo|poetry|flit|twine|gem|docker|podman|helm', String.raw`[^;&|\n]*\s(publish|unpublish|upload|push)\b`), 'publish'],
  [run('mvn|gradle|gradlew', String.raw`[^;&|\n]*\s(deploy|publish)\b`), 'publish'],
  [run('ssh|scp|sftp|ftp|rsync|nc|ncat|telnet|osascript|open|claude'), 'elsewhere'],
];
const GH = run('gh');
const GH_READS = /^\s*gh\s+((pr|issue|run|release|repo|workflow|gist|label|cache|ruleset|project)\s+(view|list|diff|checks|status)\b|search\s|auth\s+status\b|--version\b|api\s)/;

export function guardSettings({ hook, domains = [] }) {
  return {
    disableAllHooks: false,
    enableAllProjectMcpServers: false,
    hooks: { PreToolUse: [{ matcher: '*', hooks: [{ type: 'command', command: hook }] }] },
    sandbox: {
      enabled: true, autoAllowBashIfSandboxed: true, allowUnsandboxedCommands: false, failIfUnavailable: true, excludedCommands: ['gh'],
      network: { strictAllowlist: true, allowedDomains: [...REGISTRIES, ...domains] },
    },
  };
}

export function decide({ tool_name: tool, tool_input: input = {}, cwd }, { copy, real }) {
  if (READS.has(tool)) return { allow: true };
  if (FILE_CHANGES[tool]) {
    const path = String(input[FILE_CHANGES[tool]] || '');
    return inside(real(isAbsolute(path) ? path : join(cwd || copy, path)), real(copy)) ? { allow: true } : { refuse: english.guard.outside };
  }
  if (SHELLS[tool]) return shell(String(input[SHELLS[tool]] || ''));
  if (tool.startsWith('mcp__')) {
    const [, server, name = ''] = tool.split('__');
    const words = name.replace(/([a-z])([A-Z])/g, '$1 $2').toLowerCase().split(/[^a-z]+/).filter(Boolean);
    return words.some(w => READ_WORDS.has(w)) && !words.some(w => WRITE_WORDS.has(w)) ? { allow: true } : { refuse: english.guard.mcp(server, name) };
  }
  return { refuse: english.guard.tool(tool) };
}

function shell(command) {
  if (GH.test(command)) {
    if (/[;&|<>`\n]|\$\(/.test(command) || !GH_READS.test(command) || !ghApiReads(command)) return { refuse: english.guard.gh };
    return { allow: true };
  }
  const outward = OUTWARD.find(([pattern]) => pattern.test(command));
  return outward ? { refuse: english.guard.shell(outward[1]) } : { allow: true };
}

function ghApiReads(command) {
  if (!/^\s*gh\s+api\s/.test(command)) return true;
  const method = /\s(-X|--method)[\s=]*['"]?(\w+)/.exec(command);
  return (!method || method[2].toUpperCase() === 'GET') && !/\s(-f|-F|--field|--raw-field|--input)\b/.test(command);
}

function inside(path, folder) {
  return path === folder || path.startsWith(folder.endsWith(sep) ? folder : folder + sep);
}
