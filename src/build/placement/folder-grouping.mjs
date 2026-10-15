/**
 * Groups paths by folder when nothing names their components.
 *
 * groupByFolder(paths, options) starts from the deepest folder that holds
 * every path and splits the largest group into its subfolders while the
 * result stays within `maxGroups`. A group splits only when it holds more
 * than `splitAbove` paths and, with `balanced`, only when at least two of its
 * subfolders hold a real share of it, so a big folder is not split just to
 * peel off a stray file. One exception: a folder where one subfolder holds
 * most of the paths, and that subfolder would itself split by these rules,
 * splits into that subfolder and one group of everything else, so a change
 * inside one part of a large repository is still divided by its parts. With
 * `splitTop`, the top folder always splits into its subfolders. The files
 * directly in a split folder stay together as one group of their own.
 *
 * Returns [{folder, paths, direct}] in folder order: `folder` is '' for the
 * repository root, and null for everything else beside a dominant subfolder,
 * which comes last; `direct` is true for the files left directly in a folder
 * whose subfolders were split off.
 */
export function groupByFolder(paths, { maxGroups = 8, splitAbove = 8, balanced = false, splitTop = false } = {}) {
  const root = buildTree(paths);
  let groups = [collapse(root)];
  if (splitTop && groups[0].node.children.size && parts(groups[0].node).length > 1 && parts(groups[0].node).length <= maxGroups) {
    groups = parts(groups[0].node);
  }
  const pieces = g => (!balanced || isBalanced(g.node) ? parts(g.node) : [collapse(dominant(g.node, splitAbove)), { node: g.node, rest: dominant(g.node, splitAbove) }]);
  for (;;) {
    const candidates = groups
      .filter(g => !g.direct && !g.rest && g.node.children.size && g.node.count > splitAbove && (!balanced || isBalanced(g.node) || dominant(g.node, splitAbove)))
      .sort((a, b) => b.node.count - a.node.count);
    const pick = candidates.find(g => groups.length - 1 + pieces(g).length <= maxGroups);
    if (!pick) break;
    groups = groups.flatMap(g => (g === pick ? pieces(g) : [g]));
  }
  return groups
    .map(g => (g.rest
      ? { folder: null, paths: [...g.node.files, ...[...g.node.children.values()].filter(c => c !== g.rest).flatMap(allPaths)], direct: false }
      : { folder: g.node.path, paths: g.direct ? [...g.node.files] : allPaths(g.node), direct: g.direct }))
    .sort((a, b) => (a.folder === null) - (b.folder === null) || (a.folder || '').localeCompare(b.folder || '') || Number(b.direct) - Number(a.direct));
}

// The subfolder holding most of a folder's paths, when it would itself split cleanly.
function dominant(n, splitAbove) {
  for (const child of n.children.values()) {
    const inner = collapse(child).node;
    if (child.count >= n.count * 0.75 && inner.children.size && inner.count > splitAbove && isBalanced(inner)) return child;
  }
  return null;
}

function buildTree(paths) {
  const root = node('');
  for (const path of paths) {
    const segments = path.split('/');
    let at = root;
    at.count++;
    for (const segment of segments.slice(0, -1)) {
      const next = at.path ? `${at.path}/${segment}` : segment;
      if (!at.children.has(segment)) at.children.set(segment, node(next));
      at = at.children.get(segment);
      at.count++;
    }
    at.files.push(path);
  }
  return root;
}

function node(path) {
  return { path, files: [], children: new Map(), count: 0 };
}

// A folder with nothing but one subfolder is that subfolder.
function collapse(n) {
  let at = n;
  while (!at.files.length && at.children.size === 1) at = [...at.children.values()][0];
  return { node: at, direct: false };
}

function parts(n) {
  const out = [...n.children.values()].map(collapse);
  if (n.files.length) out.push({ node: n, direct: true });
  return out;
}

function isBalanced(n) {
  const share = Math.max(2, n.count * 0.12);
  const big = [...n.children.values()].filter(c => c.count >= share).length + (n.files.length >= share ? 1 : 0);
  return big >= 2;
}

function allPaths(n) {
  return [...n.files, ...[...n.children.values()].flatMap(allPaths)];
}
