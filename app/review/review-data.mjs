/**
 * The page data of a review, indexed for drawing.
 *
 * indexPage(page) takes the page data the server built and returns it with
 * lookups by id and path, the totals of the whole change and of each
 * component, and its build id, which every change the page sends names. It changes nothing in
 * `page` but the context files' own index.
 */
export function indexPage(page) {
  const model = page.model;
  const nodes = {}, comps = {}, files = {}, contextFiles = {}, nodesByFile = {}, groupsById = {};
  model.nodes.forEach(n => { nodes[n.id] = n; });
  model.components.forEach(c => { comps[c.id] = c; });
  model.files.forEach(f => { files[f.path] = f; });
  model.contextFiles.forEach((f, i) => {
    f.index = 'c' + i;
    f.lines = (page.context[f.path] || []).length;
    contextFiles[f.path] = f;
  });
  model.nodes.forEach(n => {
    (nodesByFile[n.file] = nodesByFile[n.file] || []).push(n);
    (n.extFiles || []).forEach(p => { (nodesByFile[p] = nodesByFile[p] || []).push(n); });
  });
  model.groups.forEach(g => { groupsById[g.id] = g; });

  const totals = { files: 0, adds: 0, dels: 0 };
  const compTotals = {};
  model.files.forEach(f => {
    totals.files++; totals.adds += f.adds; totals.dels += f.dels;
    const t = compTotals[f.component] = compTotals[f.component] || { files: 0, adds: 0, dels: 0 };
    t.files++; t.adds += f.adds; t.dels += f.dels;
  });
  model.contextFiles.forEach(f => {
    const t = compTotals[f.component] = compTotals[f.component] || { files: 0, adds: 0, dels: 0 };
    t.files++;
  });
  return {
    build: page.build, model, layout: page.layout, diffs: page.diff || {}, sources: page.sources || {}, contextSources: page.context || {}, stack: page.stack,
    nodes, comps, files, contextFiles, nodesByFile, groupsById, totals, compTotals,
  };
}
