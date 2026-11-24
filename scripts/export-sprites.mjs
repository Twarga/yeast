#!/usr/bin/env node
// export-sprites.mjs — extract the Yeast Cell sprite definition from the
// design system source of truth (pixel.js) and write assets/sprites/pixel.json.
//
// Usage: node scripts/export-sprites.mjs [path/to/pixel.js]
//
// The script loads pixel.js inside a minimal vm sandbox (only a `window`
// object exists), pulls out window.PIXEL_SPRITES.yeast, validates the
// palette and every frame grid, and writes a deterministic JSON artifact
// consumed by cmd/gen-sprites.

import { readFile, writeFile, mkdir } from 'node:fs/promises';
import vm from 'node:vm';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

const defaultSource = path.resolve(
  repoRoot,
  '../../design_system/brand/characters/v2/styles/pixel.js',
);
const sourcePath = path.resolve(process.argv[2] ?? defaultSource);
const outPath = path.join(repoRoot, 'assets', 'sprites', 'pixel.json');

const EXPECTED_FRAMES = [
  'cell',
  'eyes_open',
  'eyes_happy',
  'eyes_closed',
  'eyes_strain',
  'mouth_smile',
  'mouth_open',
  'mouth_frown',
  'blush_big',
  'bud_s',
  'bud_m',
  'baby',
  'bubble',
  'pop',
  'cap',
  'z',
];

const CELL_WIDTH = 16;
const CELL_HEIGHT = 15;
const HEX_RE = /^#[0-9a-fA-F]{6}$/;

function fail(msg) {
  console.error(`export-sprites: ${msg}`);
  process.exit(1);
}

const source = await readFile(sourcePath, 'utf8').catch((err) => {
  fail(`cannot read ${sourcePath}: ${err.message}`);
});

// Minimal sandbox: pixel.js only needs `window` plus the design-system
// registry hook, which we stub out — we only want the sprite data.
const sandbox = { window: {}, registerStyle: () => {} };
vm.createContext(sandbox);
try {
  vm.runInContext(source, sandbox, { filename: 'pixel.js' });
} catch (err) {
  fail(`pixel.js did not evaluate cleanly: ${err.message}`);
}

const sprites = sandbox.window.PIXEL_SPRITES;
if (!sprites || typeof sprites !== 'object') {
  fail('window.PIXEL_SPRITES missing after evaluation');
}
const yeast = sprites.yeast;
if (!yeast) fail('window.PIXEL_SPRITES.yeast missing');

// --- Palette validation -----------------------------------------------------
const palette = yeast.palette;
if (!palette || typeof palette !== 'object') fail('yeast.palette missing');
const paletteKeys = Object.keys(palette).sort();
for (const [key, hex] of Object.entries(palette)) {
  if (key.length !== 1) fail(`palette key ${JSON.stringify(key)} must be one char`);
  if (!HEX_RE.test(hex)) fail(`palette ${key} has invalid hex ${JSON.stringify(hex)}`);
}

// --- Frame validation -------------------------------------------------------
const frames = yeast.frames;
if (!frames || typeof frames !== 'object') fail('yeast.frames missing');
const frameNames = Object.keys(frames).sort();
const missing = EXPECTED_FRAMES.filter((n) => !(n in frames));
const extra = frameNames.filter((n) => !EXPECTED_FRAMES.includes(n));
if (missing.length) fail(`missing expected frames: ${missing.join(', ')}`);
if (extra.length) fail(`unexpected frames (export only yeast): ${extra.join(', ')}`);

const validChars = new Set([...paletteKeys, '.']);
for (const [name, rows] of Object.entries(frames)) {
  if (!Array.isArray(rows)) fail(`frame ${name} is not an array`);
  for (const row of rows) {
    if (typeof row !== 'string') fail(`frame ${name} has non-string row`);
    if (name === 'cell' && row.length !== CELL_WIDTH) {
      fail(`cell frame row must be ${CELL_WIDTH}px wide, got ${row.length}: ${row}`);
    }
    for (const ch of row) {
      if (!validChars.has(ch)) fail(`frame ${name} uses unknown pixel ${JSON.stringify(ch)}`);
    }
  }
}
if (frames.cell.length !== CELL_HEIGHT) {
  fail(`cell frame must be ${CELL_HEIGHT}px tall, got ${frames.cell.length}`);
}

// --- Emit deterministic JSON ------------------------------------------------
const sortObj = (obj) =>
  Object.fromEntries(Object.keys(obj).sort().map((k) => [k, obj[k]]));

const artifact = {
  name: 'yeast',
  title: 'Yeast Cell',
  source: path.relative(repoRoot, sourcePath),
  cell: { width: CELL_WIDTH, height: CELL_HEIGHT },
  palette: sortObj(palette),
  frames: sortObj(frames),
};

await mkdir(path.dirname(outPath), { recursive: true });
await writeFile(outPath, JSON.stringify(artifact, null, 2) + '\n');
console.log(
  `export-sprites: wrote ${path.relative(repoRoot, outPath)} ` +
    `(${frameNames.length} frames, ${paletteKeys.length} colors)`,
);
