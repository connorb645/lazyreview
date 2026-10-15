/**
 * The one place Review Desk reads its configuration.
 *
 * readSettings(env) takes the environment and returns
 * {home, port, claude: {command, model}}:
 *
 *   REVIEW_DESK_HOME    where reviews and clones are kept (~/.review-desk)
 *   REVIEW_DESK_PORT    the port the app listens on, on this machine only (4747)
 *   REVIEW_DESK_CLAUDE  the Claude Code command that answers questions (claude)
 *   REVIEW_DESK_MODEL   the model it answers with (Claude Code's own default)
 */
import { homedir } from 'node:os';
import { join, resolve } from 'node:path';

export const DEFAULT_PORT = 4747;

export function readSettings(env = process.env) {
  const port = env.REVIEW_DESK_PORT ? Number(env.REVIEW_DESK_PORT) : DEFAULT_PORT;
  if (!Number.isInteger(port) || port < 1 || port > 65535) throw new Error(`REVIEW_DESK_PORT must be a port number, not "${env.REVIEW_DESK_PORT}"`);
  return {
    home: resolve(env.REVIEW_DESK_HOME || join(homedir(), '.review-desk')),
    port,
    claude: { command: env.REVIEW_DESK_CLAUDE || 'claude', model: env.REVIEW_DESK_MODEL || null },
  };
}
