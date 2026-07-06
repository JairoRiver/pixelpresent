import { EMPTY, sizeCanvas, type PixelCanvas } from './canvas';
import { revealCellSize, type RevealMechanic } from './reveal';

// The confetti reveal mechanic (arquitectura §7, MVP): the pixel art's own cells
// fly in as confetti and assemble into the drawing. This module owns both halves
// from the backlog — particle generation (PP-61) and the converging animation
// (PP-62). Coordinates are in canvas pixels (cell * cellSize).

// easeOutCubic decelerates toward the end so particles rush in and settle softly.
function easeOutCubic(t: number): number {
  return 1 - Math.pow(1 - t, 3);
}

// A Particle is one non-empty cell: its current position starts scattered and is
// interpolated toward its final cell in the grid during the reveal.
export interface Particle {
  x: number; // current position (starts at the scattered initial position)
  y: number;
  targetX: number; // final position: the cell's top-left in the assembled grid
  targetY: number;
  color: string; // the cell colour (hex, from the palette)
}

// generateParticles builds one Particle per non-empty cell of the drawing. The
// final position is the cell's place in the grid; the initial position is
// scattered uniformly across the canvas so the cells look dispersed before they
// converge. Pure and deterministic apart from the random initial scatter.
export function generateParticles(model: PixelCanvas, cellSize: number): Particle[] {
  const { width, height, palette, pixels } = model;
  const canvasW = width * cellSize;
  const canvasH = height * cellSize;

  const particles: Particle[] = [];
  for (let y = 0; y < height; y++) {
    for (let x = 0; x < width; x++) {
      const value = pixels[y * width + x];
      if (value === EMPTY) continue;
      particles.push({
        x: Math.random() * canvasW,
        y: Math.random() * canvasH,
        targetX: x * cellSize,
        targetY: y * cellSize,
        color: palette[value],
      });
    }
  }
  return particles;
}

// confettiMechanic is the reveal mechanic: it generates one particle per cell
// (PP-61) and animates them converging from their scattered initial positions to
// their target cells with an eased requestAnimationFrame loop (PP-62), assembling
// the drawing over ~1.5s. On completion the particles sit exactly on their
// targets (the finished drawing); the stage then swaps in its static canvas.
const DURATION_MS = 1500;

export const confettiMechanic: RevealMechanic = (container, gift, onComplete) => {
  const model = gift.pixelArt;
  const cellSize = revealCellSize(model);
  const particles = generateParticles(model, cellSize);
  const canvasW = model.width * cellSize;
  const canvasH = model.height * cellSize;

  const canvas = document.createElement('canvas');
  canvas.className = 'block rounded-lg shadow-md';
  const ctx = sizeCanvas(canvas, model, cellSize);
  container.appendChild(canvas);

  // Draw every particle interpolated `progress` (0..1) of the way to its target.
  const drawFrame = (progress: number) => {
    if (!ctx) return;
    ctx.clearRect(0, 0, canvasW, canvasH);
    for (const p of particles) {
      const x = p.x + (p.targetX - p.x) * progress;
      const y = p.y + (p.targetY - p.y) * progress;
      ctx.fillStyle = p.color;
      ctx.fillRect(x, y, cellSize, cellSize);
    }
  };

  const start = performance.now();
  let raf = 0;
  let finished = false;

  const tick = (now: number) => {
    const t = Math.min(1, (now - start) / DURATION_MS);
    drawFrame(easeOutCubic(t));
    if (t < 1) {
      raf = requestAnimationFrame(tick);
      return;
    }
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
