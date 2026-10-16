/**
 * What the apps read of a review's page to draw the diagram and the file
 * lists: everything but the per-file parts, which the file route gives one
 * file at a time.
 *
 * pageView(build, page) is pure. It returns {build, model, layout, stack}:
 * the page's build id, its model, each unchanged file of the context with
 * its count of `lines`, the layout, and for a stack its members and its
 * warnings, without the line map. It leaves out the diff, the whole texts
 * and the code of the context.
 */
export function pageView(build, page) {
  return {
    build,
    model: { ...page.model, contextFiles: page.model.contextFiles.map(f => ({ ...f, lines: (page.context[f.path] || []).length })) },
    layout: page.layout,
    stack: page.stack && { members: page.stack.members, warnings: page.stack.warnings },
  };
}
