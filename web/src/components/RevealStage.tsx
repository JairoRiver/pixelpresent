import { useEffect, useRef, useState } from 'preact/hooks';
import { render, sizeCanvas, toPNGDataURL, type PixelCanvas } from '../lib/canvas';
import { confettiMechanic } from '../lib/confetti';
import { emptyColor, revealCellSize, type RevealMechanic } from '../lib/reveal';
import { GiftIcon } from './icons';

// RevealStage is the common wrapper for every reveal mechanic (PP-60). It owns
// the shared state cycle and chrome — the idle expectation screen (PP-59), a
// skip control always visible while animating, a prefers-reduced-motion bypass,
// and a safety duration cap — and delegates the transition animation to a
// pluggable RevealMechanic (confetti in PP-61+, a placeholder until then).

export interface RevealGift {
  title: string;
  message: string;
  pixelArt: PixelCanvas;
  revealType: string;
  revealConfig: unknown;
}

// The design's full cycle is idle → interacting → revealing → revealed →
// reacting. 'interacting' is a mechanic-specific gesture state — confetti's
// gesture is the idle tap, so it goes straight to 'revealing' — and 'reacting'
// is the post-reveal reaction UI (PP-65). Both are intentionally not entered yet.
type Phase = 'idle' | 'revealing' | 'revealed';

// A generous ceiling well under the design's 1-minute limit: if a mechanic never
// signals completion, the stage advances anyway so the recipient is never stuck.
const MAX_REVEAL_MS = 20000;

function prefersReducedMotion(): boolean {
  return (
    typeof window !== 'undefined' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
  );
}

// A safe PNG file name derived from the gift title (falls back to a default).
function fileName(title: string): string {
  const slug = title
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/gi, '-')
    .replace(/^-+|-+$/g, '');
  return `${slug || 'pixel-present'}.png`;
}

export default function RevealStage({
  gift,
  mechanic = confettiMechanic,
}: {
  gift: RevealGift;
  mechanic?: RevealMechanic;
}) {
  const [phase, setPhase] = useState<Phase>('idle');
  const mechanicRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  // Start the reveal from the idle screen. Under reduced motion we skip the
  // transition animation entirely and show the final drawing straight away.
  function begin() {
    setPhase(prefersReducedMotion() ? 'revealed' : 'revealing');
  }

  // Skip advances to the final state; the running mechanic is stopped by the
  // revealing effect's cleanup (below), so no animation keeps running.
  function skip() {
    setPhase('revealed');
  }

  // Run the mechanic while revealing. Its cleanup stops the animation on skip or
  // when it completes; a timeout caps the duration as a safety net.
  useEffect(() => {
    if (phase !== 'revealing') return;
    const container = mechanicRef.current;
    if (!container) return;

    const stop = mechanic(
      container,
      { pixelArt: gift.pixelArt, revealType: gift.revealType, revealConfig: gift.revealConfig },
      () => setPhase('revealed'),
    );
    const cap = window.setTimeout(() => setPhase('revealed'), MAX_REVEAL_MS);

    return () => {
      window.clearTimeout(cap);
      stop();
    };
  }, [phase, gift, mechanic]);

  // Render the final drawing once revealed. Reuses the shared render at a
  // smaller, gridless scale (the recipient's view).
  useEffect(() => {
    if (phase !== 'revealed') return;
    const el = canvasRef.current;
    if (el === null) return;
    const cellSize = revealCellSize(gift.pixelArt);
    const ctx = sizeCanvas(el, gift.pixelArt, cellSize);
    if (ctx) render(ctx, gift.pixelArt, cellSize, { empty: emptyColor(), grid: '' }, false);
  }, [phase, gift]);

  // Export the pixel art as a downloadable PNG (transparent background, no grid).
  function downloadPNG() {
    const cellSize = Math.max(1, Math.round(640 / Math.max(gift.pixelArt.width, gift.pixelArt.height)));
    const url = toPNGDataURL(gift.pixelArt, cellSize);
    if (!url) return;
    const a = document.createElement('a');
    a.href = url;
    a.download = fileName(gift.title);
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  }

  // Idle expectation screen (PP-59): build anticipation before revealing. The
  // drawing stays hidden until the recipient taps/clicks; the teaser is
  // deliberately generic (no title, no art) so the reveal keeps its surprise.
  if (phase === 'idle') {
    return (
      <button
        type="button"
        onClick={begin}
        class="group flex w-full max-w-sm cursor-pointer flex-col items-center gap-5 rounded-2xl px-6 py-10 text-center transition"
        aria-label="Descubrir el regalo"
      >
        <span class="flex h-20 w-20 items-center justify-center rounded-full bg-amber-100 text-amber-600 transition group-hover:scale-105 dark:bg-amber-500/15 dark:text-amber-300">
          <GiftIcon class="h-10 w-10" />
        </span>
        <div class="flex flex-col gap-2">
          <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">Hay algo para ti</h1>
          <p class="text-base text-slate-500 dark:text-slate-400">
            Hecho píxel a píxel, para este momento.
          </p>
        </div>
        <span class="mt-2 inline-flex items-center gap-2 rounded-full bg-amber-500 px-6 py-2.5 text-sm font-semibold text-white transition group-hover:bg-amber-400 dark:bg-amber-400 dark:text-slate-900 dark:group-hover:bg-amber-300">
          Descubrir
        </span>
        <span class="text-xs text-slate-400 dark:text-slate-500">Toca para abrir</span>
      </button>
    );
  }

  // Revealing: the mechanic draws into its container; a skip control is always
  // visible so the recipient can jump to the drawing at any moment.
  if (phase === 'revealing') {
    return (
      <div class="flex w-full max-w-md flex-col items-center gap-5">
        <div ref={mechanicRef} class="relative flex items-center justify-center" />
        <button
          type="button"
          onClick={skip}
          class="rounded-md border border-slate-300 px-4 py-1.5 text-sm font-medium text-slate-600 transition hover:border-slate-400 dark:border-white/15 dark:text-slate-300 dark:hover:border-white/30"
        >
          Saltar
        </button>
      </div>
    );
  }

  // Revealed: the final drawing plus the creator's message.
  return (
    <div class="flex w-full max-w-md flex-col items-center gap-5 text-center">
      {gift.title && (
        <h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">{gift.title}</h1>
      )}
      <canvas ref={canvasRef} class="block rounded-lg shadow-md" />
      {gift.message && (
        <p class="text-base whitespace-pre-wrap text-slate-600 dark:text-slate-300">
          {gift.message}
        </p>
      )}
      <button
        type="button"
        onClick={downloadPNG}
        class="rounded-md border border-slate-300 px-4 py-1.5 text-sm font-medium text-slate-600 transition hover:border-slate-400 dark:border-white/15 dark:text-slate-300 dark:hover:border-white/30"
      >
        Descargar PNG
      </button>
    </div>
  );
}
