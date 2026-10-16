/**
 * The answers GitHub would give about a throwaway repository's pull requests,
 * as routes for the fake gh.
 *
 * githubRoutes({repo, name, me, pulls, reviews, extra}) reads the throwaway
 * repository `repo` and returns the routes: the repository, with the origin
 * as its clone address; the account `me`; the list of its open pull
 * requests; and for each pull request {number, base, head, title, author,
 * state, merged, draft, body}, where base and head are branches, its details
 * at its current head and its files with their patches, as GitHub shows
 * them, from the merge base. `reviews` gives
 * each pull request's existing reviews, review comments and review threads,
 * the threads as GitHub's GraphQL API gives them, and `extra` routes are
 * added as they are.
 */
import { splitDiff } from '../../src/build/diff/diff-parser.mjs';

export function githubRoutes({ repo, name = 'octo/widgets', me = 'reviewer', pulls, reviews = {}, extra = {} }) {
  const routes = {
    [`GET repos/${name}`]: { full_name: name, clone_url: repo.origin },
    'GET user': { login: me },
  };
  const open = [];
  for (const p of pulls) {
    const head = repo.sha(p.head);
    const base = repo.sha(p.base);
    const start = repo.git('merge-base', base, head);
    routes[`GET repos/${name}/pulls/${p.number}`] = {
      number: p.number, title: p.title || `Pull request ${p.number}`, body: p.body || '', html_url: `https://github.com/${name}/pull/${p.number}`,
      state: p.state || 'open', merged_at: p.merged ? '2026-10-15T12:00:00Z' : null, closed_at: p.state === 'closed' || p.merged ? '2026-10-15T12:00:00Z' : null,
      user: { login: p.author || 'author' }, draft: !!p.draft,
      base: { ref: p.base, sha: base, repo: { full_name: name } },
      head: { ref: p.head, sha: head, repo: { full_name: name } },
    };
    if (!p.state || p.state === 'open') open.push(routes[`GET repos/${name}/pulls/${p.number}`]);
    const diff = repo.git('diff', '--no-color', '--find-renames', start, head);
    routes[`GET repos/${name}/pulls/${p.number}/files?per_page=100&page=1`] = splitDiff(diff).map(chunk => {
      const lines = chunk.text.split('\n');
      const first = lines.findIndex(l => l.startsWith('@@'));
      return { filename: chunk.path, ...(first >= 0 ? { patch: lines.slice(first).join('\n') } : {}) };
    });
    const existing = { reviews: [], comments: [], threads: [], ...reviews[p.number] };
    routes[`GET repos/${name}/pulls/${p.number}/reviews?per_page=100&page=1`] = existing.reviews;
    routes[`GET repos/${name}/pulls/${p.number}/comments?per_page=100&page=1`] = existing.comments;
    routes[`POST graphql ${name}#${p.number}`] = { data: { repository: { pullRequest: { reviewThreads: { pageInfo: { hasNextPage: false, endCursor: null }, nodes: existing.threads } } } } };
  }
  routes[`GET repos/${name}/pulls?state=open&per_page=100&page=1`] = open;
  return { ...routes, ...extra };
}
