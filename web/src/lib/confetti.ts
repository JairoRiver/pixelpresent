import { EMPTY, render, type PixelCanvas } from './canvas';
import {
  emptyColor,
  prepareRevealCanvas,
  revealCanvasClass,
  revealCellSize,
  type RevealMechanic,
} from './reveal';

// The confetti reveal mechanic (arquitectura §7, MVP): the pixel art's own cells
// fly in as confetti and assemble into the drawing. This module owns particle
// generation (PP-61), the converging animation (PP-62), and particle grouping on
// large grids (PP-63). Coordinates are in canvas pixels (cell * cellSize).

// A Particle is one flying block: its current position starts scattered and is
// interpolated toward its target (its block's place in the grid). On small grids
// a block is a single cell (w = h = cellSize); on large grids adjacent cells are
// grouped into one block (PP-63) so fewer particles animate per frame.
export interface Particle {
  x: number; // current position (starts at the scattered initial position)
  y: number;
  targetX: number; // final position: the block's top-left in the assembled grid
  targetY: number;
  w: number; // block size in canvas pixels
  h: number;
  color: string; // the block's representative colour (hex, from the palette)
}

// easeOutCubic decelerates toward the end so particles rush in and settle softly.
function easeOutCubic(t: number): number {
  return 1 - Math.pow(1 - t, 3);
}

// blockSize decides how many cells per side are grouped into one flying particle
// (PP-63). Below the threshold every cell flies on its own (1); larger grids fold
// adjacent cells into 2×2 or 4×4 blocks so a 128×128 drawing animates ~1k
// particles instead of ~16k. Grouping only affects the journey — the final frame
// is always drawn per cell. Thresholds are starting values to tune on device.
export function blockSize(model: PixelCanvas): number {
  const maxDim = Math.max(model.width, model.height);
  if (maxDim <= 64) return 1;
  if (maxDim <= 96) return 2;
  return 4;
}

// generateParticles builds one Particle per non-empty block of the drawing (one
// per cell when block === 1). Each block's colour is the most common non-empty
// colour inside it (a fair stand-in while it flies; the real per-cell colours are
// drawn once it lands). Initial positions are scattered uniformly across the
// canvas so the blocks look dispersed before they converge.
export function generateParticles(model: PixelCanvas, cellSize: number): Particle[] {
  const { width, height, palette, pixels } = model;
  const block = blockSize(model);
  const canvasW = width * cellSize;
  const canvasH = height * cellSize;

  const particles: Particle[] = [];
  for (let by = 0; by < height; by += block) {
    for (let bx = 0; bx < width; bx += block) {
      // Tally the non-empty colours in this block to pick a representative and
      // to know whether the block has anything to show at all.
      const counts = new Map<number, number>();
      const blockW = Math.min(block, width - bx);
      const blockH = Math.min(block, height - by);
      for (let dy = 0; dy < blockH; dy++) {
        for (let dx = 0; dx < blockW; dx++) {
          const value = pixels[(by + dy) * width + (bx + dx)];
          if (value === EMPTY) continue;
          counts.set(value, (counts.get(value) ?? 0) + 1);
        }
      }
      if (counts.size === 0) continue; // fully empty block: nothing flies

      let best = EMPTY;
      let bestCount = 0;
      for (const [value, count] of counts) {
        if (count > bestCount) {
          best = value;
          bestCount = count;
        }
      }

      particles.push({
        x: Math.random() * canvasW,
        y: Math.random() * canvasH,
        targetX: bx * cellSize,
        targetY: by * cellSize,
        w: blockW * cellSize,
        h: blockH * cellSize,
        color: palette[best],
      });
    }
  }
  return particles;
}

// confettiMechanic generates the particles (PP-61) and animates them converging
// from their scattered initial positions to their targets with an eased
// requestAnimationFrame loop (PP-62), grouping cells into blocks on large grids
// to stay smooth (PP-63). The journey draws blocks; the final frame is drawn per
// cell (pixel-perfect), matching the static canvas the stage swaps in next.
const DURATION_MS = 1500;

export const confettiMechanic: RevealMechanic = (container, gift, onComplete) => {
  const model = gift.pixelArt;
  const cellSize = revealCellSize(model);
  const particles = generateParticles(model, cellSize);
  const canvasW = model.width * cellSize;
  const canvasH = model.height * cellSize;

  const canvas = document.createElement('canvas');
  canvas.className = revealCanvasClass;
  canvas.style.imageRendering = 'pixelated';
  const ctx = prepareRevealCanvas(canvas, model, cellSize);
  container.appendChild(canvas);

  // Draw every block interpolated `progress` (0..1) of the way to its target.
  const drawFrame = (progress: number) => {
    if (!ctx) return;
    ctx.clearRect(0, 0, canvasW, canvasH);
    for (const p of particles) {
      const x = p.x + (p.targetX - p.x) * progress;
      const y = p.y + (p.targetY - p.y) * progress;
      ctx.fillStyle = p.color;
      ctx.fillRect(x, y, p.w, p.h);
    }
  };

  const start = performance.now();
  let raf = 0;
  let finished = false;

  const tick = (now: number) => {
    const t = Math.min(1, (now - start) / DURATION_MS);
    if (t < 1) {
      drawFrame(easeOutCubic(t));
      raf = requestAnimationFrame(tick);
      return;
    }
    // Landed: draw the exact drawing per cell so grouped blocks resolve to their
    // real colours with no visible pop before the stage shows its static canvas.
    if (ctx) render(ctx, model, cellSize, { empty: emptyColor(), grid: '' }, false);
    finished = true;
    onComplete();
  };
  raf = requestAnimationFrame(tick);

  // Skip: stop the loop and drop the canvas; the stage renders its own static
  // drawing in the revealed state, so nothing is left to show here.
  return () => {
    if (!finished) cancelAnimationFrame(raf);
    canvas.remove();
  };
};
