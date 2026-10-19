/**
 * Which entries of the config's hidden repositories hide a repository.
 *
 * hiddenBy(entries, name) returns the entries, as the config writes them,
 * that hide the repository `name`, written owner/name: the entry naming it,
 * and owner/* for every repository of its owner. Case does not count, as it
 * does not in GitHub's names.
 */
export function hiddenBy(entries, name) {
  const repo = name.toLowerCase();
  const owner = `${repo.split('/')[0]}/*`;
  return entries.filter(entry => [repo, owner].includes(entry.toLowerCase()));
}
