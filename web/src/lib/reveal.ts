import type { PixelCanvas } from './canvas';

// The reveal mechanic contract (arquitectura §7). A RevealStage (Preact chrome)
// drives the common idle → revealing → revealed cycle; the actual transition
// animation is a pluggable RevealMechanic kept as framework-agnostic imperative
// canvas code, so new mechanics can be added without touching the stage.

// RevealGiftData is what a mechanic needs to animate the transition: the drawing
// (already deserialized) plus its reveal type/config from gifts.reveal_config.
export interface RevealGiftData {
  pixelArt: PixelCanvas;
  revealType: string;
  revealConfig: unknown;
}

// A RevealMechanic animates the "revealing" state inside `container` and calls
// onComplete() once it settles. It returns a stop() function the stage calls to
// skip: stop() must cancel the animation and release resources WITHOUT calling
// onComplete (the stage advances to "revealed" itself). start() is only invoked
// when motion is allowed — under prefers-reduced-motion the stage skips it.
export type RevealMechanic = (
  container: HTMLElement,
  gift: RevealGiftData,
  onComplete: () => void,
) => () => void;

// emptyColor reads the theme-aware empty-cell colour so a revealed drawing sits
// on the page background instead of a hard white block in dark mode.
export function emptyColor(): string {
  const v = getComputedStyle(document.documentElement).getPropertyValue('--canvas-empty');
  return v.trim() || '#ffffff';
}

// revealCellSize sizes the confetti animation backing: kept modest (~360px on
// the longest side) so large grids stay smooth while the cells are in motion.
// The crisp final frame and the download render at higher resolutions (below).
export function revealCellSize(model: PixelCanvas): number {
  return Math.max(1, Math.floor(360 / Math.max(model.width, model.height)));
}

// On-screen display box for the revealed drawing. It is a fixed CSS size, decoupled
// from the backing resolution, so the confetti animation and the crisp final frame
// share one size (no size jump at the swap) while each renders at its own
// resolution. Pair it with image-rendering: pixelated so upscaling stays sharp.
export const revealCanvasClass = 'block w-full max-w-[640px] rounded-lg shadow-md';

// prepareRevealCanvas sizes the canvas backing store to cellSize*dims (display
// size is handled by CSS, so no devicePixelRatio here) and returns its 2D context.
export function prepareRevealCanvas(
  el: HTMLCanvasElement,
  model: PixelCanvas,
  cellSize: number,
): CanvasRenderingContext2D | null {
  el.width = model.width * cellSize;
  el.height = model.height * cellSize;
  return el.getContext('2d');
}

// finalCellSize renders the revealed drawing at ~768px on its longest side, so a
// large grid stays crisp when shown up to 640px and on high-DPI screens (a 128²
// grid jumps from ~2px/cell to 6px/cell).
const FINAL_TARGET_PX = 768;
export function finalCellSize(model: PixelCanvas): number {
  return Math.max(1, Math.ceil(FINAL_TARGET_PX / Math.max(model.width, model.height)));
}

// downloadCellSize targets ~1536px on the longest side for a high-resolution PNG,
// capped at 2048px so toPNGDataURL stays light on memory and time on low-end
// devices (a 128² grid downloads at ~12px/cell instead of ~5).
const DOWNLOAD_TARGET_PX = 1536;
const DOWNLOAD_MAX_PX = 2048;
export function downloadCellSize(model: PixelCanvas): number {
  const maxDim = Math.max(model.width, model.height);
  const target = Math.max(1, Math.round(DOWNLOAD_TARGET_PX / maxDim));
  const cap = Math.max(1, Math.floor(DOWNLOAD_MAX_PX / maxDim));
  return Math.min(target, cap);
}
