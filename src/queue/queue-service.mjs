/**
 * Changes a review's queue, and sends a review to GitHub from it.
 *
 * Every change names `build`, the build id of the page its window was drawn
 * from, and is refused once the review was loaded again since: that
 * window's line numbers may name other code now.
 *
 * add(ref, given) queues a comment on a line the app shows, at the line of
 * the pull request that takes it, once GitHub confirms the line is in that
 * pull request's diff and its head has not moved. edit(ref, id, {body,
 * version, build}) and drop(ref, id, {version, build}) change or remove a
 * queued comment, and are refused once it changed since `version`.
 *
 * post(ref, {pull, action, set, build}) is the only way a review reaches GitHub. It
 * refuses, and posts nothing, unless the queue still gives exactly the set id
 * the app showed for that action, the pull request is open, its head has not
 * moved, every comment sits on a line of its diff and none is stale, and the
 * account is the one the review was loaded as. Before posting it looks for a
 * review of the same account at this head that already does the same thing,
 * so a second press, or one whose answer was lost, never posts twice. GitHub
 * creates a review and its comments in one request, so a failure there posts
 * nothing.
 */
import { GitHub } from '../github/github.mjs';
import { ACTIONS, setId, offered, sent, postedFields, bodyProblems, diffPlaces, placeProblem, liveMember, nextId, findComment, checkVersion } from './queue-rules.mjs';
import { commentTarget } from './comment-target.mjs';
import { checkBuild } from '../store/page-build.mjs';

export class QueueService {
  #store;
  #github;
  #now;

  constructor({ store, github = options => new GitHub(options), now = () => new Date().toISOString() }) {
    this.#store = store;
    this.#github = github;
    this.#now = now;
  }

  add(ref, given) {
    return this.#change(ref, { build: given.build, refused: 'not queued' }, async (review, gh) => {
      const { page } = this.#store.readPage(ref);
      const { pull, target, shown } = commentTarget(review, page.stack, given);
      const member = liveMember(review, pull);
      const problems = bodyProblems(given.body);
      if (problems.length) throw new Error(`not queued: ${problems.join('; ')}`);
      const now = await gh.pull(review.repo, pull);
      if (now.head.sha !== member.head) {
        throw new Error(`not queued: #${pull} moved from ${member.head.slice(0, 8)} to ${now.head.sha.slice(0, 8)}; load it again, then queue`);
      }
      const places = await filePlaces(gh, review.repo, pull);
      const where = `${target.path}:${target.start_line ? `${target.start_line}-` : ''}${target.line}${shown ? ` of #${pull}` : ''}`;
      const problem = placeProblem(places[target.path] || NO_PLACES, target);
      if (problem) throw new Error(`not queued: ${where} ${problem}`);
      const comment = {
        id: nextId(review), path: target.path, line: target.line, side: target.side, body: given.body.trim(), code: given.code || '',
        ...(target.start_line ? { start_line: target.start_line, start_side: target.start_side } : {}), ...(shown ? { shown } : {}),
      };
      member.comments.push(comment);
      return { review, result: { comment, pull } };
    });
  }

  edit(ref, id, { body, version, build }) {
    const problems = bodyProblems(body);
    if (problems.length) return Promise.reject(new Error(`not changed: ${problems.join('; ')}`));
    return this.#change(ref, { build, refused: 'not changed' }, async review => {
      const { member, comment } = findComment(review, id);
      liveMember(review, member.pull);
      checkVersion(comment, version, 'not changed');
      comment.body = body.trim();
      return { review, result: { comment, pull: member.pull } };
    }, false);
  }

  drop(ref, id, { version, build }) {
    return this.#change(ref, { build, refused: 'not dropped' }, async review => {
      const { member, comment } = findComment(review, id);
      checkVersion(comment, version, 'not dropped');
      member.comments = member.comments.filter(c => c !== comment);
      return { review, result: { comment, pull: member.pull } };
    }, false);
  }

  post(ref, { pull, action, set, build }) {
    return this.#change(ref, { build, refused: 'nothing was posted' }, async (review, gh) => {
      if (!ACTIONS[action]) throw new Error(`"${action}" is not an action; nothing was posted`);
      const member = liveMember(review, pull);
      if (!offered(review, member).includes(action)) throw new Error(`"${action}" is not offered for #${pull}; nothing was posted`);
      const current = setId(review, member, action);
      if (set !== current) throw new Error(`the page showed set ${set}, but "${action}" now gives set ${current}; nothing was posted`);
      const me = await gh.login();
      if (me !== review.me) throw new Error(`GitHub answers as ${me} now, but this review was loaded as ${review.me}; nothing was posted`);
      const earlier = await alreadyPosted(gh, review, member, action, me);
      if (earlier) {
        if (!member.posted.some(p => p.review === earlier.html_url)) record(review, member, action, earlier, this.#now());
        return { review, result: { message: `already done as ${earlier.html_url}; posted nothing new`, review: earlier.html_url } };
      }
      const comments = sent(member, action);
      const { problems, now } = await problemsIn(gh, review, member, comments);
      if (ACTIONS[action].event === 'APPROVE' && now.user.login === me) problems.push('this is your own pull request, which GitHub does not let you approve');
      if (problems.length) throw new Error(`nothing was posted: ${problems.join('; ')}`);
      const payload = { commit_id: member.head, event: ACTIONS[action].event, ...(comments.length ? { comments: comments.map(postedFields) } : {}) };
      const posted = await gh.createReview(review.repo, pull, payload);
      record(review, member, action, posted, this.#now());
      const back = (await gh.reviewComments(review.repo, pull)).filter(c => c.pull_request_review_id === posted.id);
      const count = `${comments.length} comment${comments.length === 1 ? '' : 's'}`;
      const message = posted.state !== ACTIONS[action].state || back.length !== comments.length
        ? `posted ${posted.html_url}, but GitHub reads back state ${posted.state} with ${back.length} of ${count}: check it`
        : `${{ 'comment': `posted ${count} as one review`, 'approve-with-comments': `approved with ${count}`, 'approve': 'approved, with no comments' }[action]}: ${posted.html_url}`;
      return { review, result: { message, review: posted.html_url } };
    });
  }

  async #change(ref, { build, refused }, work, online = true) {
    return this.#store.change([ref], async () => {
      const review = this.#store.read(ref);
      if (!review) throw new Error(`there is no review ${ref.id}`);
      checkBuild(this.#store.pageBuild(ref), build, refused);
      const gh = online ? this.#github({ host: review.host, account: review.account }) : null;
      const done = await work(review, gh);
      return { files: [this.#store.recordWrite(ref, done.review)], result: done.result };
    });
  }
}

const NO_PLACES = diffPlaces(null);

async function filePlaces(gh, repo, pull) {
  const files = {};
  for (const item of await gh.files(repo, pull)) files[item.filename] = diffPlaces(item.patch);
  return files;
}

async function problemsIn(gh, review, member, comments) {
  const now = await gh.pull(review.repo, member.pull);
  const problems = [];
  if (now.state !== 'open') problems.push(`#${member.pull} is ${now.merged_at ? 'merged' : now.state}`);
  if (now.head.sha !== member.head) problems.push(`the head moved from ${member.head.slice(0, 9)} to ${now.head.sha.slice(0, 9)}`);
  const places = comments.length ? await filePlaces(gh, review.repo, member.pull) : {};
  for (const comment of comments) {
    const where = `comment ${comment.id} (${comment.path}:${comment.line})`;
    const problem = placeProblem(places[comment.path] || NO_PLACES, comment);
    if (problem) problems.push(`${where} ${problem}`);
    problems.push(...bodyProblems(comment.body).map(p => `${where}: ${p}`));
    if (comment.stale) problems.push(`${where} is stale: ${comment.stale}; queue it again or drop it`);
  }
  return { problems, now };
}

async function alreadyPosted(gh, review, member, action, me) {
  const wanted = JSON.stringify(sent(member, action).map(c => [c.path, c.body.trim()]).sort());
  const mine = (await gh.reviews(review.repo, member.pull))
    .filter(r => r.user.login === me && r.commit_id === member.head && r.state === ACTIONS[action].state);
  if (!mine.length) return null;
  if (wanted === '[]') return mine[0];
  const comments = await gh.reviewComments(review.repo, member.pull);
  return mine.find(r => JSON.stringify(comments.filter(c => c.pull_request_review_id === r.id).map(c => [c.path, c.body.trim()]).sort()) === wanted) || null;
}

function record(review, member, action, posted, at) {
  member.posted.push({ set: setId(review, member, action), action, review: posted.html_url, at, comments: [...sent(member, action)] });
  if (ACTIONS[action].comments) member.comments = [];
}
