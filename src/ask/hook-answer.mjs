/**
 * Answers Claude Code's PreToolUse hook with the guard's decision.
 *
 * hookAnswer(text, {copy}) takes the hook's input as Claude Code writes it,
 * and `copy`, the folder of the copy of the code, and returns {output,
 * code}: the JSON that allows or refuses the call, and the exit code. When
 * the input cannot be read, it says why with code 2, which Claude Code takes
 * as a refusal, so the guard never lets a call through by failing.
 */
import { existsSync, realpathSync } from 'node:fs';
import { basename, dirname, join, resolve } from 'node:path';
import { decide } from './guard.mjs';
import { english } from '../i18n/english.mjs';

export function hookAnswer(text, { copy }) {
  try {
    const decision = decide(JSON.parse(text), { copy, real });
    return {
      output: JSON.stringify({
        hookSpecificOutput: {
          hookEventName: 'PreToolUse',
          permissionDecision: decision.allow ? 'allow' : 'deny',
          ...(decision.refuse ? { permissionDecisionReason: decision.refuse } : {}),
        },
      }),
      code: 0,
    };
  } catch (error) {
    return { output: english.guard.failed(error.message), code: 2 };
  }
}

function real(path) {
  let at = resolve(path);
  const rest = [];
  while (!existsSync(at) && dirname(at) !== at) {
    rest.unshift(basename(at));
    at = dirname(at);
  }
  return join(realpathSync(at), ...rest);
}
