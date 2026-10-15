/**
 * Calls the app's server. Each call resolves with the answer's JSON, or
 * rejects with an error carrying the server's own words for what went wrong.
 */
export async function getJSON(path) {
  return answer(await fetch(path, { headers: { Accept: 'application/json' } }));
}

export async function send(method, path, body) {
  return answer(await fetch(path, {
    method,
    headers: body === undefined ? { Accept: 'application/json' } : { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  }));
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
    throw new Error('the server sent something that is not JSON (' + response.status + ')', { cause: error });
  }
  if (!response.ok) throw new Error(body && body.error ? body.error : 'the server answered ' + response.status);
  return body;
}
