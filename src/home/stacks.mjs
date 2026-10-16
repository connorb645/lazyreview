/**
 * Finds the stacks among a repository's open pull requests.
 *
 * stacks(pulls, repo) takes the open pull requests of `repo` (owner/name),
 * as GitHub's list gives them, and returns a Map from each one's number to
 * the pull requests it is reviewed with, bottom first. A stack is a chain of
 * two or more, each based on the head branch of the one below, where every
 * branch is in `repo` itself. A chain that forks, with two pull requests
 * based on the same branch, is no stack: each of its pull requests is
 * reviewed alone, as is one with nothing stacked on or under it.
 */
export function stacks(pulls, repo) {
  const byHead = new Map(pulls.filter(p => p.head.repo?.full_name === repo).map(p => [p.head.ref, p]));
  const above = new Map();
  for (const p of pulls) {
    const below = byHead.get(p.base.ref);
    if (below && below !== p) above.set(below.number, [...(above.get(below.number) || []), p]);
  }
  const out = new Map();
  for (const p of pulls) {
    let bottom = p;
    const seen = new Set([p.number]);
    while (byHead.has(bottom.base.ref) && !seen.has(byHead.get(bottom.base.ref).number)) {
      bottom = byHead.get(bottom.base.ref);
      seen.add(bottom.number);
    }
    const group = [];
    const todo = [bottom];
    while (todo.length) {
      const next = todo.shift();
      if (group.includes(next)) continue;
      group.push(next);
      todo.push(...(above.get(next.number) || []));
    }
    const forked = group.some(g => (above.get(g.number) || []).length > 1);
    out.set(p.number, group.length > 1 && !forked ? group : [p]);
  }
  return out;
}
