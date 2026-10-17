/**
 * The one way to the app's server. Each call resolves with the answer's
 * JSON, or rejects with an error carrying the server's own words for what
 * went wrong. A DELETE carries no body, so send() puts its fields in the
 * address. events(path) opens the stream of a review's change notices.
 */
import { english } from '/i18n/english.mjs';

export async function getJSON(path) {
  return answer(await fetch(path, { headers: { Accept: 'application/json' } }));
}

export async function send(method, path, body = {}) {
  const json = method !== 'DELETE';
  return answer(await fetch(json ? path : path + '?' + new URLSearchParams(body), {
    method,
    headers: json ? { 'Content-Type': 'application/json', Accept: 'application/json' } : { Accept: 'application/json' },
    body: json ? JSON.stringify(body) : undefined,
  }));
}

export function events(path) {
  return new EventSource(path);
}

/** The address of a review's API, from its id. */
export function reviewPath(id, rest = '') {
  return '/api/reviews/' + id.split('/').map(encodeURIComponent).join('/') + rest;
}

async function answer(response) {
  let body = null;
  try {
    body = await response.json();
  } catch (error) {
    throw new Error(english.web.server.notJSON(response.status), { cause: error });
  }
  if (!response.ok) throw new Error(body && body.error ? body.error : english.web.server.answered(response.status));
  return body;
}
