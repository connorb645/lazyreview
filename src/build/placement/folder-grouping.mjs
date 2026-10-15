/**
 * Groups paths by folder when nothing names their components.
 *
 * groupByFolder(paths, options) starts from the deepest folder that holds
 * every path and splits the largest group into its subfolders while the
 * result stays within `maxGroups`. A group splits only when it holds more
 * than `splitAbove` paths and, with `balanced`, only when at least two of its
 * subfolders hold a real share of it, so a big folder is not split just to
 * peel off a stray file. With `splitTop`, the top folder always splits into
 * its subfolders. The files directly in a split folder stay together as one
 * group of their own.
 *
 * Returns [{folder, paths, direct}] in folder order: `folder` is '' for the
 * repository root, and `direct` is true for the files left directly in a
 * folder whose subfolders were split off.
 */
export function groupByFolder(paths, { maxGroups = 8, splitAbove = 8, balanced = false, splitTop = false } = {}) {
  const root = buildTree(paths);
  let groups = [collapse(root)];
  if (splitTop && groups[0].node.children.size && parts(groups[0].node).length > 1 && parts(groups[0].node).length <= maxGroups) {
    groups = parts(groups[0].node);
  }
  for (;;) {
    const candidates = groups
      .filter(g => !g.direct && g.node.children.size && g.node.count > splitAbove && (!balanced || isBalanced(g.node)))
      .sort((a, b) => b.node.count - a.node.count);
    const pick = candidates.find(g => groups.length - 1 + parts(g.node).length <= maxGroups);
    if (!pick) break;
    groups = groups.flatMap(g => (g === pick ? parts(g.node) : [g]));
  }
  return groups
    .map(g => ({ folder: g.node.path, paths: g.direct ? [...g.node.files] : allPaths(g.node), direct: g.direct }))
    .sort((a, b) => a.folder.localeCompare(b.folder) || Number(b.direct) - Number(a.direct));
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
