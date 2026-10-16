/**
 * Calls the app's server. Each call resolves with the answer's JSON, or
 * rejects with an error carrying the server's own words for what went wrong.
 * A DELETE carries no body, so send() puts its fields in the address.
 */
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

/** The address of a review's API, from its id. */
export function reviewPath(id, rest = '') {
  return '/api/reviews/' + id.split('/').map(encodeURIComponent).join('/') + rest;
}

/** The address of a review's page in the app, from its id. */
export function reviewAddress(id) {
  return '/review/' + id.split('/').map(encodeURIComponent).join('/');
}

async function answer(response) {
  let body = null;
  try {
    body = await response.json();
  } catch (error) {
    throw new Error('the server sent something that is not JSON (' + response.status + ')', { cause: error });
  }
  if (!response.ok) throw new Error(body && body.error ? body.error : 'the server answered ' + response.status);
  return body;
}
