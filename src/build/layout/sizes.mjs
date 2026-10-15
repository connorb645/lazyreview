/**
 * How big each piece of text and each box on the map is, estimated from the
 * page's fonts so the layout leaves room for every label. The page fits any
 * text that still overflows, so an estimate a little wide is the safe side.
 */
const STYLES = {
  name: { char: 7.9, height: 17 },
  fileName: { char: 7.4, height: 16 },
  file: { char: 6.5, height: 13 },
  stereotype: { char: 6.0, height: 14 },
  edge: { char: 6.2, height: 14 },
  packageTitle: { char: 8.6, height: 19 },
  packagePath: { char: 7.2, height: 15 },
  packageNote: { char: 6.6, height: 15 },
  groupTitle: { char: 7.6, height: 16 },
  groupNote: { char: 6.5, height: 15 },
  packageArrow: { char: 8.2, height: 20 },
  componentTitle: { char: 20, height: 40 },
  componentOpen: { char: 8.6, height: 19 },
  componentTag: { char: 9.1, height: 18 },
  summary: { char: 13.6, height: 30 },
  areaTitle: { char: 37, height: 76 },
  areaTag: { char: 13.3, height: 26 },
};

export const PACKAGE_HEAD = 46;
/**
 * A component's title at far zoom: drawn above its frame at up to this many
 * canvas units tall, and free to reach this far past either side of it.
 */
export const FAR_TITLE = { size: 150, overhang: 200 };
export const PACKAGE_HEAD_WITH_NOTE = 64;
export const COMPONENT_HEAD = 104;
export const AREA_HEAD = 200;

export function measure(text, style) {
  const s = STYLES[style];
  return { width: Math.ceil(String(text || '').length * s.char), height: s.height };
}

/** How wide a component must be for its far-zoom title to reach full size. */
export function farTitleRoom(title) {
  return Math.ceil(String(title).length * 0.62 * FAR_TITLE.size) - FAR_TITLE.overhang * 2;
}

/** A type's box: wide enough for its name, its file and its stereotype. */
export function nodeSize(node) {
  const artifact = node.kind === 'artifact';
  const name = measure(node.name, artifact ? 'fileName' : 'name').width;
  const file = artifact ? 0 : measure(node.file.split('/').pop(), 'file').width;
  const stereotype = measure(node.stereotype, 'stereotype').width;
  return { width: Math.max(120, Math.ceil(Math.max(name, file, stereotype) + 28)), height: artifact ? 46 : 62 };
}
