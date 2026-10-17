/**
 * Two made-up accounts and the open pull requests that involve them, for
 * tests of the home screen's list, as settings and as routes for the fake gh.
 *
 * SETTINGS are the accounts as settings.json held them: `reviewer`, with the
 * team octo/reviewers counted from 1 Oct 2026, and `reviewer-home`, and
 * accountsConfig(accounts) the config file that names accounts. homeRoutes()
 * returns the routes each account's searches and the lists of open pull
 * requests take, beside the workbench's own octo/widgets:
 *
 *   reviewer        asked to review octo/widgets#22, the top of the stack
 *                   #21 → #22; commented on octo/widgets#30; reviewed
 *                   octo/gadgets#9; asked through its team on octo/gadgets#7
 *   reviewer-home   opened reviewer-home/notes#3 and the draft #4 on it
 *
 * search(query) is the route of one search, as the account's gh calls it.
 */
export const SETTINGS = {
  accounts: [
    { login: 'reviewer', teams: [{ name: 'octo/reviewers', since: '2026-10-01' }] },
    { login: 'reviewer-home' },
  ],
};

export function accountsConfig(accounts) {
  return `github:\n  accounts: ${JSON.stringify(accounts)}\n`;
}

const OPEN = 'is:pr is:open archived:false';

export function search(query) {
  return `GET search/issues?${new URLSearchParams({ q: `${OPEN} ${query}`, per_page: '100', page: '1' })}`;
}

const found = (...pulls) => ({ total_count: pulls.length, items: pulls.map(([repo, number]) => ({ number, repository_url: `https://api.github.com/repos/${repo}` })) });

const pull = (repo, number, { title, author, base = 'main', head, draft = false, body = '' }) => ({
  number, title, body, draft, html_url: `https://github.com/${repo}/pull/${number}`, state: 'open', user: { login: author },
  base: { ref: base, repo: { full_name: repo } }, head: { ref: head, repo: { full_name: repo } },
});

export function homeRoutes() {
  return {
    [`token-reviewer ${search('draft:false user-review-requested:@me')}`]: found(['octo/widgets', 22]),
    [`token-reviewer ${search('draft:false team-review-requested:octo/reviewers created:>=2026-10-01')}`]: found(['octo/gadgets', 7]),
    [`token-reviewer ${search('author:@me')}`]: found(),
    [`token-reviewer ${search('commenter:@me')}`]: found(['octo/widgets', 30]),
    [`token-reviewer ${search('reviewed-by:@me')}`]: found(['octo/gadgets', 9]),
    [`token-reviewer-home ${search('draft:false user-review-requested:@me')}`]: found(),
    [`token-reviewer-home ${search('author:@me')}`]: found(['reviewer-home/notes', 3], ['reviewer-home/notes', 4]),
    [`token-reviewer-home ${search('commenter:@me')}`]: found(),
    [`token-reviewer-home ${search('reviewed-by:@me')}`]: found(),
    'GET repos/octo/gadgets/pulls?state=open&per_page=100&page=1': [
      pull('octo/gadgets', 9, { title: 'Sort the gadgets by price', author: 'kim', head: 'kim/sort' }),
      pull('octo/gadgets', 7, { title: 'Count the gadgets in stock', author: 'kim', head: 'kim/stock', body: 'Counts each gadget once.\r\n\r\n- Adds a stock count\r\n- Shows it on the shelf' }),
    ],
    'GET repos/reviewer-home/notes/pulls?state=open&per_page=100&page=1': [
      pull('reviewer-home/notes', 4, { title: 'Tag the notes', author: 'reviewer-home', base: 'plain-text', head: 'tags', draft: true }),
      pull('reviewer-home/notes', 3, { title: 'Keep the notes in plain text', author: 'reviewer-home', head: 'plain-text' }),
    ],
  };
}
