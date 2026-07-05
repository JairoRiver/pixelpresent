// Canvas data model and rendering for the pixel-art editor (PP-42).
//
// This module is framework-agnostic vanilla TS: the editor's Preact island is
// only chrome around it (architecture §2). It owns the in-memory shape of the
// drawing — the same shape persisted in gifts.pixel_art (JSONB) — and the
// imperative render onto a 2D <canvas>.
//
// The model is size-agnostic: any width×height works (the editor exposes
// selectable sizes in PP-49 and zoom in PP-48). PP-42 only renders an empty
// grid, but rendering already fits the available width so it is usable on
// mobile from the start.

// EMPTY marks a cell with no colour (transparent). The model stores a palette
// index per cell; 255 is reserved as the empty sentinel, so a palette can hold
// up to 255 distinct colours (0..254).
export const EMPTY = 255;

export interface PixelCanvas {
  width: number;
  height: number;
  palette: string[]; // hex colours, e.g. ["#000000", "#ff0044"]; index by pixel value
  pixels: Uint8Array; // length = width * height, row-major; value = palette index or EMPTY
}

// createCanvas returns an empty grid of the given size (all cells EMPTY, no
// palette colours yet).
export function createCanvas(width: number, height: number): PixelCanvas {
  const pixels = new Uint8Array(width * height);
  pixels.fill(EMPTY);
  return { width, height, palette: [], pixels };
}

// fitCellSize picks the largest integer cell size whose grid still fits in
// availableWidth, clamped to [min, max]. Keeping it an integer keeps the pixels
// crisp; recomputing it on resize is what makes the board responsive on mobile.
// Zoom (PP-48) will later drive cellSize directly instead of fitting.
export function fitCellSize(
  availableWidth: number,
  model: PixelCanvas,
  min = 4,
  max = 28,
): number {
  const fit = Math.floor(availableWidth / model.width);
  return Math.max(min, Math.min(max, fit));
}

// SerializedCanvas is the JSON shape persisted in gifts.pixel_art: the same as
// PixelCanvas but with pixels as a plain number array, because a Uint8Array does
// not JSON-stringify to an array (JSON.stringify turns it into an object with
// numeric keys). serializeCanvas / deserializeCanvas convert between the
// in-memory model and this shape (PP-54).
export interface SerializedCanvas {
  width: number;
  height: number;
  palette: string[];
  pixels: number[];
}

// serializeCanvas turns the in-memory model into the plain-JSON shape stored in
// gifts.pixel_art. The palette is copied so later edits can't mutate a payload
// already handed off.
export function serializeCanvas(model: PixelCanvas): SerializedCanvas {
  return {
    width: model.width,
    height: model.height,
    palette: model.palette.slice(),
    pixels: Array.from(model.pixels),
  };
}

// deserializeCanvas rebuilds a PixelCanvas from a parsed gifts.pixel_art value,
// or returns null when the value isn't a well-formed canvas (wrong types, or a
// pixels length that doesn't match width*height). Callers fall back to a blank
// canvas on null rather than trusting malformed data.
export function deserializeCanvas(value: unknown): PixelCanvas | null {
  if (typeof value !== 'object' || value === null) return null;
  const v = value as Record<string, unknown>;
  const { width, height, palette, pixels } = v;
  if (!Number.isInteger(width) || !Number.isInteger(height)) return null;
  const w = width as number;
  const h = height as number;
  if (w <= 0 || h <= 0) return null;
  if (!Array.isArray(palette) || !palette.every((c) => typeof c === 'string')) return null;
  if (!Array.isArray(pixels) || pixels.length !== w * h) return null;
  if (!pixels.every((p) => typeof p === 'number')) return null;
  return { width: w, height: h, palette: palette.slice(), pixels: Uint8Array.from(pixels) };
}

// resizeCanvas returns a new grid of the given size (PP-49), preserving the
// palette and the overlapping top-left region of the current drawing: cells that
// fall outside the new bounds are dropped and any newly exposed cells start
// EMPTY. Resizing this way means changing the grid size (a preset or the custom
// input) never silently wipes the work already on the board.
export function resizeCanvas(model: PixelCanvas, width: number, height: number): PixelCanvas {
  const pixels = new Uint8Array(width * height);
  pixels.fill(EMPTY);
  const copyW = Math.min(width, model.width);
  const copyH = Math.min(height, model.height);
  for (let y = 0; y < copyH; y++) {
    for (let x = 0; x < copyW; x++) {
      pixels[y * width + x] = model.pixels[y * model.width + x];
    }
  }
  return { width, height, palette: model.palette.slice(), pixels };
}

// colorIndex returns the palette index for a hex colour, appending it to the
// palette the first time it is used. The palette holds at most 255 colours
// (0..254; 255 is EMPTY); once full it reuses the last slot rather than grow.
export function colorIndex(model: PixelCanvas, hex: string): number {
  const existing = model.palette.indexOf(hex);
  if (existing !== -1) return existing;
  if (model.palette.length >= EMPTY) return EMPTY - 1;
  model.palette.push(hex);
  return model.palette.length - 1;
}

// setPixel writes a palette index (or EMPTY) at (x, y), ignoring out-of-bounds
// coordinates so callers need not clamp.
export function setPixel(model: PixelCanvas, x: number, y: number, value: number): void {
  if (x < 0 || y < 0 || x >= model.width || y >= model.height) return;
  model.pixels[y * model.width + x] = value;
}

// paintLine sets every cell on the grid line from (x0, y0) to (x1, y1) to value,
// using Bresenham so a fast pointer drag leaves no gaps between sampled points.
export function paintLine(
  model: PixelCanvas,
  x0: number,
  y0: number,
  x1: number,
  y1: number,
  value: number,
): void {
  const dx = Math.abs(x1 - x0);
  const dy = -Math.abs(y1 - y0);
  const sx = x0 < x1 ? 1 : -1;
  const sy = y0 < y1 ? 1 : -1;
  let err = dx + dy;
  let x = x0;
  let y = y0;

  for (;;) {
    setPixel(model, x, y, value);
    if (x === x1 && y === y1) break;
    const e2 = 2 * err;
    if (e2 >= dy) {
      err += dy;
      x += sx;
    }
    if (e2 <= dx) {
      err += dx;
      y += sy;
    }
  }
}

// floodFill replaces the 4-connected region of same-valued cells starting at
// (x, y) with value — the classic bucket fill. It uses an explicit stack (DFS)
// so a large uniform region can't blow the call stack. The `target === value`
// guard makes filling a region with its own colour a no-op, which also prevents
// an infinite loop; filling a grid that is entirely one value fills all of it.
export function floodFill(model: PixelCanvas, x: number, y: number, value: number): void {
  const { width, height, pixels } = model;
  if (x < 0 || y < 0 || x >= width || y >= height) return;

  const target = pixels[y * width + x];
  if (target === value) return;

  const stack: number[] = [y * width + x];
  while (stack.length > 0) {
    const idx = stack.pop() as number;
    if (pixels[idx] !== target) continue;
    pixels[idx] = value;

    const cx = idx % width;
    const cy = (idx - cx) / width;
    if (cx + 1 < width) stack.push(idx + 1);
    if (cx - 1 >= 0) stack.push(idx - 1);
    if (cy + 1 < height) stack.push(idx + width);
    if (cy - 1 >= 0) stack.push(idx - width);
  }
}

// Surface colours for the drawing area (PP-44.5). They are theme-aware: the
// editor reads them from CSS variables (--canvas-empty / --canvas-grid, defined
// in global.css) so empty/erased cells and grid lines follow the light/dark
// theme. These constants are only the light-theme fallback when no colours are
// passed. Painting an EMPTY cell is purely visual; EMPTY still means transparent
// in the saved pixel_art.
export interface SurfaceColors {
  empty: string; // fill for EMPTY cells (the canvas background)
  grid: string; // grid line colour
}

const DEFAULT_COLORS: SurfaceColors = {
  empty: '#ffffff',
  grid: 'rgba(0, 0, 0, 0.12)',
};

// sizeCanvas sets the backing-store resolution to match the grid and the device
// pixel ratio (crisp on retina / mobile), while keeping the CSS size at
// width*cellSize. It returns the 2D context, scaled so callers draw in CSS
// pixels. Returns null if the 2D context is unavailable.
export function sizeCanvas(
  el: HTMLCanvasElement,
  model: PixelCanvas,
  cellSize: number,
): CanvasRenderingContext2D | null {
  const cssWidth = model.width * cellSize;
  const cssHeight = model.height * cellSize;
  const dpr = window.devicePixelRatio || 1;

  el.style.width = `${cssWidth}px`;
  el.style.height = `${cssHeight}px`;
  el.width = Math.round(cssWidth * dpr);
  el.height = Math.round(cssHeight * dpr);

  const ctx = el.getContext('2d');
  if (!ctx) return null;
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  return ctx;
}

// render redraws the whole grid, one fillRect per cell. Even at the 128×128
// maximum this is effectively instant, so there is no partial redraw. Empty
// cells are painted with the surface colour (theme-aware) and grid lines on top.
// `grid` draws the cell borders; the preview (PP-55) passes false for a clean,
// recipient's-eye render at a smaller scale.
export function render(
  ctx: CanvasRenderingContext2D,
  model: PixelCanvas,
  cellSize: number,
  colors: SurfaceColors = DEFAULT_COLORS,
  grid = true,
): void {
  const { width, height, palette, pixels } = model;

  ctx.clearRect(0, 0, width * cellSize, height * cellSize);

  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const value = pixels[y * width + x];
      ctx.fillStyle = value === EMPTY ? colors.empty : palette[value];
      ctx.fillRect(x * cellSize, y * cellSize, cellSize, cellSize);
    }
  }

  if (!grid) return;

  // Grid lines on top, aligned to cell boundaries (0.5 offset keeps 1px crisp).
  ctx.strokeStyle = colors.grid;
  ctx.lineWidth = 1;
  ctx.beginPath();
  for (let x = 0; x <= width; x++) {
    ctx.moveTo(x * cellSize + 0.5, 0);
    ctx.lineTo(x * cellSize + 0.5, height * cellSize);
  }
  for (let y = 0; y <= height; y++) {
    ctx.moveTo(0, y * cellSize + 0.5);
    ctx.lineTo(width * cellSize, y * cellSize + 0.5);
  }
  ctx.stroke();
}
