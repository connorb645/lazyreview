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
import { english } from '../../i18n/english.mjs';

export function describeChange({ subject, base, headCommit, mergeBase, commits }) {
  const short = sha => sha.slice(0, 8);
  return english.diagram.change({ subject, base, head: short(headCommit), mergeBase: short(mergeBase), commits });
}

/** The subject of a pull request, or of a stack of them, bottom first. */
export function pullsSubject(pulls) {
  return pulls.length === 1 ? english.diagram.pullSubject(pulls[0]) : english.diagram.stackSubject(pulls[0], pulls[pulls.length - 1]);
}

/** The subject of a branch with no pull request. */
export function branchSubject(branch) {
  return english.diagram.branchSubject(branch);
}
