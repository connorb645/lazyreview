/**
 * The one place that finds the page's regions by id.
 *
 * `views` holds each region index.html draws, by name. element(id) finds an
 * element the app drew later, such as a file of a diff, or null.
 */
const byId = id => document.getElementById(id);

export const views = Object.freeze({
  bar: byId('bar'),
  back: byId('back'),
  pageTitle: byId('page-title'),
  context: byId('context'),
  viewsNav: byId('views'),
  diagramTools: byId('diagram-tools'),
  search: byId('search'),
  allLinks: byId('all-links'),
  diffTools: byId('diff-tools'),
  toggleAll: byId('toggle-all'),
  toggleView: byId('toggle-view'),
  toggleWrap: byId('toggle-wrap'),
  totals: byId('totals'),
  refresh: byId('refresh'),
  stackStrip: byId('stack-strip'),
  pullFilter: byId('pull-filter'),
  stripNote: byId('strip-note'),
  loading: byId('loading'),
  homeView: byId('home-view'),
  homeList: byId('home-list'),
  reviewView: byId('review-view'),
  reviewBody: byId('review-body'),
  reviewSide: byId('review-side'),
  diagramView: byId('diagram-view'),
  compList: byId('comp-list'),
  legend: byId('legend'),
  canvas: byId('canvas'),
  diagram: byId('diagram'),
  propsPanel: byId('props-panel'),
  scale: byId('scale'),
  zoomIn: byId('zoom-in'),
  zoomOut: byId('zoom-out'),
  zoomFit: byId('zoom-fit'),
  diffView: byId('diff-view'),
  filesNav: byId('files-nav'),
  main: byId('main'),
  emptyView: byId('empty-view'),
  toast: byId('toast'),
  reloadNote: byId('reload-note'),
});

export function element(id) {
  return byId(id);
}
