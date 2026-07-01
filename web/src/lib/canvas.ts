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

const GRID_LINE = 'rgba(255, 255, 255, 0.08)';
const EMPTY_CELL = 'rgba(255, 255, 255, 0.02)';

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
// cells get a faint fill plus grid lines so the empty grid is visible.
export function render(
  ctx: CanvasRenderingContext2D,
  model: PixelCanvas,
  cellSize: number,
): void {
  const { width, height, palette, pixels } = model;

  ctx.clearRect(0, 0, width * cellSize, height * cellSize);

  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const value = pixels[y * width + x];
      ctx.fillStyle = value === EMPTY ? EMPTY_CELL : palette[value];
      ctx.fillRect(x * cellSize, y * cellSize, cellSize, cellSize);
    }
  }

  // Grid lines on top, aligned to cell boundaries (0.5 offset keeps 1px crisp).
  ctx.strokeStyle = GRID_LINE;
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
