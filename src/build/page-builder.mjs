/**
 * Builds what the app draws for one review: the class diagram's model and
 * layout, the diff split per file, the whole text of changed files, the code
 * of unchanged context, and for a stack, where a comment on each line goes.
 *
 * buildPage({git, base, members, baseBranch, title, subject, config, context, rootTitle, elk})
 * reads the change from the merge base of `base` and the top member's head to
 * that head, as a pull request shows it. `members` are the heads to read,
 * bottom first, each {pull, head}; one member is a single pull request or a
 * branch, more are a stack of pull requests, each based on the one below.
 * `subject` names the change in the line under the title, as heading.mjs
 * gives it. It only reads the repository.
 *
 * It returns {page, starts, counts, notes}: `page` is the data the app
 * renders, `starts` the commit each member's own diff starts from, bottom
 * first, `counts` the files, added and removed lines, components and types
 * it drew, and `notes` lines worth telling the person who loaded it, such as
 * a file whose whole text was left out for its size.
 */
import { readDiff } from './diff/diff-reader.mjs';
import { readChangedTypes } from './model/changed-types.mjs';
import { chooseContext } from './model/context-selection.mjs';
import { buildModel } from './model/model.mjs';
import { findContextCandidates, findContextExtensions } from './context/context-finder.mjs';
import { layoutModel } from './layout/layout.mjs';
import { describeChange } from './page/heading.mjs';
import { readStackSteps } from './stack/stack-reader.mjs';
import { mapStackLines } from './stack/line-map.mjs';
import { nodePulls } from './stack/node-pulls.mjs';
import { changedTexts, lines } from './page/whole-texts.mjs';

export async function buildPage({ git, base, members, baseBranch, title, subject, config, context, rootTitle, elk }) {
  const pulls = members.map(m => m.pull);
  const top = members[members.length - 1];
  const diff = readDiff(git, { base, head: top.head, exact: false });
  const stacked = members.length > 1 ? readStacked(git, diff, members, pulls) : null;
  if (stacked) diff.files = diff.files.map(f => ({ ...f, ...stacked.mapped.files[f.path] }));

  const changed = readChangedTypes(diff.files);
  const found = context
    ? findContextCandidates({ git, head: diff.headCommit, drawn: changed.nodes, changedPaths: diff.files.map(f => f.path), include: config.contextTypes.include })
    : { candidates: [], texts: new Map() };
  const chosen = chooseContext({
    drawn: changed.nodes, untouched: changed.untouched, candidates: found.candidates,
    options: { enabled: context, ...config.contextTypes },
  });
  const extended = context
    ? findContextExtensions({ git, head: diff.headCommit, types: chosen.context, skip: new Set([...diff.files.map(f => f.path), ...chosen.context.map(n => n.file)]) })
    : { nodes: [], texts: new Map() };
  const drawn = [...changed.nodes, ...chosen.same, ...chosen.context, ...extended.nodes];
  const owners = stacked ? nodePulls(drawn, diff.chunks, stacked.mapped, pulls) : [];
  const model = buildModel({
    files: diff.files,
    nodes: stacked ? drawn.map((n, i) => ({ ...n, pulls: owners[i] })) : drawn,
    imports: changed.imports,
    config,
    rootTitle,
  });
  model.heading = {
    title: config.title || title,
    context: config.summary
      ? [[config.summary, false]]
      : describeChange({ subject, base: baseBranch, headCommit: diff.headCommit, mergeBase: diff.mergeBase, commits: git.commitCount(diff.mergeBase, diff.headCommit) }),
    head: diff.headCommit,
    base: diff.mergeBase,
  };
  const layout = await layoutModel(model, elk);
  const whole = changedTexts(diff.files);
  const page = {
    model,
    layout,
    diff: Object.fromEntries(diff.chunks.map(c => [c.path, c.text])),
    sources: whole.texts,
    context: Object.fromEntries(model.contextFiles.map(f => [f.path, lines(found.texts.get(f.path) || extended.texts.get(f.path) || '')])),
    stack: stacked && {
      members: members.map((m, i) => ({ pull: m.pull, head: stacked.heads[i], start: stacked.steps[i].start, ...stacked.counts[i] })),
      lines: stacked.mapped.lines,
      warnings: stacked.warnings,
    },
  };
  const count = (n, word) => `${n} ${word}${n === 1 ? '' : 's'}`;
  const notes = [
    ...(whole.left.length ? [`whole text left out for ${count(whole.left.length, 'large file')}, which show their diff only: ${whole.left.join(', ')}`] : []),
    ...(stacked ? stacked.warnings : []),
  ];
  return { page, starts: stacked ? stacked.steps.map(s => s.start) : [diff.mergeBase], counts: counts(model), notes };
}

function readStacked(git, diff, members, pulls) {
  const read = readStackSteps(git, { combinedBase: diff.mergeBase, baseCommit: diff.baseCommit, top: diff.headCommit, members });
  return { ...read, mapped: mapStackLines(read.steps, diff.chunks, pulls) };
}

function counts(model) {
  return {
    files: model.files.length,
    adds: model.files.reduce((n, f) => n + f.adds, 0),
    dels: model.files.reduce((n, f) => n + f.dels, 0),
    components: model.components.length,
    types: model.nodes.filter(n => n.kind !== 'artifact').length,
  };
}
