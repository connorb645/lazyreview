/**
 * The line under the page's title that says what the change is.
 *
 * describeChange({subject, base, headCommit, mergeBase, commits}) returns
 * [[text, emphasis]] pieces: the subject, then where it sits. pullsSubject
 * and branchSubject give the subject. For one pull request: Pull request
 * **#12** on **main**: head **3f2a9c1d**, merge base **77b0e412**, 3 commits.
 * For a stack, naming its bottom and top: Stack **#12** to **#14** on
 * **main**: head **9d01c2aa**, merge base **77b0e412**, 8 commits. For a
 * branch: Branch **feature/search** on **main**: and the rest as before.
 */
export function describeChange({ subject, base, headCommit, mergeBase, commits }) {
  const short = sha => sha.slice(0, 8);
  return [...subject, ...(base ? [[' on ', false], [base, true]] : []), [': head ', false], [short(headCommit), true],
    [', merge base ', false], [short(mergeBase), true], [`, ${commits} commit${commits === 1 ? '' : 's'}`, false]];
}

/** The subject of a pull request, or of a stack of them, bottom first. */
export function pullsSubject(pulls) {
  return pulls.length === 1
    ? [['Pull request ', false], [`#${pulls[0]}`, true]]
    : [['Stack ', false], [`#${pulls[0]}`, true], [' to ', false], [`#${pulls[pulls.length - 1]}`, true]];
}

/** The subject of a branch with no pull request. */
export function branchSubject(branch) {
  return [['Branch ', false], [branch, true]];
}
