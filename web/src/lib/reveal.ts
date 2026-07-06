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

// revealCellSize fits the drawing into ~360px on its longest side (the
// recipient's view), at least 1px per cell.
export function revealCellSize(model: PixelCanvas): number {
  return Math.max(1, Math.floor(360 / Math.max(model.width, model.height)));
}
