import { useEffect, useRef, useState } from 'preact/hooks';
import {
  colorIndex,
  createCanvas,
  fitCellSize,
  paintLine,
  render,
  sizeCanvas,
  type PixelCanvas,
} from '../lib/canvas';

// Default drawing colour until the palette / colour picker lands (PP-45).
const DRAW_COLOR = '#fbbf24';

// Editor is the shell of the gift editor (PP-41). It resolves which gift to work
// on — an existing one when the URL carries ?id=<uuid>, or a fresh empty one
// otherwise — and lays out the editor chrome. The pixel canvas itself, its data
// model and the drawing tools arrive in the following tasks (PP-42+); for now the
// canvas area is a placeholder.

type Status = 'loading' | 'ready' | 'notfound' | 'error';

interface GiftData {
  id: string;
  title: string;
  message: string;
  pixel_art: unknown;
  reveal_type: string;
}

function giftIdFromURL(): string | null {
  if (typeof window === 'undefined') return null;
  return new URLSearchParams(window.location.search).get('id');
}

export default function Editor() {
  const [status, setStatus] = useState<Status>('loading');
  const [title, setTitle] = useState('');
  const [message, setMessage] = useState('');

  const canvasRef = useRef<HTMLCanvasElement>(null);
  const boardRef = useRef<HTMLDivElement>(null);
  // The drawing model. A fixed 16×16 for PP-42; selectable sizes arrive in PP-49.
  const modelRef = useRef<PixelCanvas>(createCanvas(16, 16));

  useEffect(() => {
    const id = giftIdFromURL();

    // No id: this is a brand-new gift, nothing to load.
    if (!id) {
      setStatus('ready');
      return;
    }

    fetch(`/api/gifts/${id}`, { headers: { Accept: 'application/json' } })
      .then((res) => {
        if (res.status === 401) {
          window.location.replace('/login');
          return null;
        }
        if (res.status === 404 || res.status === 403) {
          setStatus('notfound');
          return null;
        }
        if (!res.ok) {
          setStatus('error');
          return null;
        }
        return res.json() as Promise<GiftData>;
      })
      .then((gift) => {
        if (!gift) return;
        setTitle(gift.title);
        setMessage(gift.message);
        setStatus('ready');
      })
      .catch(() => setStatus('error'));
  }, []);

  // Draw the grid once it is in the DOM and wire the pencil. The board stays
  // fitted to the available width (responsive/mobile) and Pointer Events unify
  // mouse and touch; pointer capture keeps a stroke going if the finger leaves
  // the canvas. Pan mode for grids too big to fit is a later task (PP-50).
  useEffect(() => {
    if (status !== 'ready') return;
    if (canvasRef.current === null || boardRef.current === null) return;
    // Non-null declared types so the narrowing survives into the closures below.
    const canvas: HTMLCanvasElement = canvasRef.current;
    const board: HTMLDivElement = boardRef.current;
    const model = modelRef.current;

    let cellSize = fitCellSize(board.clientWidth, model);
    let ctx = sizeCanvas(canvas, model, cellSize);
    if (ctx) render(ctx, model, cellSize);

    function redraw() {
      cellSize = fitCellSize(board.clientWidth, model);
      ctx = sizeCanvas(canvas, model, cellSize);
      if (ctx) render(ctx, model, cellSize);
    }

    // Map a pointer position to a grid cell, or null if outside the grid.
    function cellFromEvent(event: PointerEvent): { x: number; y: number } | null {
      const rect = canvas.getBoundingClientRect();
      const x = Math.floor((event.clientX - rect.left) / cellSize);
      const y = Math.floor((event.clientY - rect.top) / cellSize);
      if (x < 0 || y < 0 || x >= model.width || y >= model.height) return null;
      return { x, y };
    }

    let drawing = false;
    let last: { x: number; y: number } | null = null;

    function paint(from: { x: number; y: number }, to: { x: number; y: number }) {
      const ink = colorIndex(model, DRAW_COLOR);
      paintLine(model, from.x, from.y, to.x, to.y, ink);
      if (ctx) render(ctx, model, cellSize);
    }

    function onPointerDown(event: PointerEvent) {
      const cell = cellFromEvent(event);
      if (!cell) return;
      drawing = true;
      last = cell;
      canvas.setPointerCapture(event.pointerId);
      paint(cell, cell);
    }

    function onPointerMove(event: PointerEvent) {
      if (!drawing) return;
      const cell = cellFromEvent(event);
      if (!cell) return;
      paint(last ?? cell, cell);
      last = cell;
    }

    function onPointerUp(event: PointerEvent) {
      drawing = false;
      last = null;
      if (canvas.hasPointerCapture(event.pointerId)) {
        canvas.releasePointerCapture(event.pointerId);
      }
    }

    canvas.addEventListener('pointerdown', onPointerDown);
    canvas.addEventListener('pointermove', onPointerMove);
    canvas.addEventListener('pointerup', onPointerUp);
    canvas.addEventListener('pointercancel', onPointerUp);
    window.addEventListener('resize', redraw);
    return () => {
      canvas.removeEventListener('pointerdown', onPointerDown);
      canvas.removeEventListener('pointermove', onPointerMove);
      canvas.removeEventListener('pointerup', onPointerUp);
      canvas.removeEventListener('pointercancel', onPointerUp);
      window.removeEventListener('resize', redraw);
    };
  }, [status]);

  if (status === 'loading') {
    return <p class="px-6 py-10 text-slate-400">Cargando el editor…</p>;
  }

  if (status === 'notfound') {
    return (
      <div class="px-6 py-10">
        <p class="text-slate-300">
          Ese regalo no existe o no es tuyo.{' '}
          <a href="/dashboard" class="text-amber-300 hover:text-amber-200">Volver a tus regalos</a>.
        </p>
      </div>
    );
  }

  if (status === 'error') {
    return (
      <div class="px-6 py-10">
        <p class="text-rose-300" role="alert">
          No hemos podido cargar el regalo. Inténtalo de nuevo en un momento.
        </p>
      </div>
    );
  }

  return (
    <div class="flex min-h-screen flex-col">
      <header class="flex items-center justify-between gap-4 border-b border-white/10 px-6 py-4">
        <a href="/dashboard" class="font-mono text-xs font-bold tracking-widest text-amber-300">
          ← PIXEL&nbsp;PRESENT
        </a>
        <span class="text-sm text-slate-400">Editor</span>
      </header>

      <div class="grid flex-1 gap-6 px-6 py-6 lg:grid-cols-[1fr_20rem]">
        {/* Canvas area. The board fits its container width (responsive/mobile);
            zoom (PP-48) and selectable sizes (PP-49) come later. */}
        <section class="flex items-start justify-center rounded-xl border border-white/10 bg-white/5 p-4">
          <div ref={boardRef} class="w-full max-w-lg">
            <canvas ref={canvasRef} class="mx-auto block cursor-crosshair touch-none" />
          </div>
        </section>

        {/* Side panel — gift metadata. */}
        <aside class="space-y-4">
          <div>
            <label for="gift-title" class="block text-sm font-medium text-slate-300">Título</label>
            <input
              id="gift-title"
              type="text"
              value={title}
              onInput={(event) => setTitle(event.currentTarget.value)}
              placeholder="Para alguien especial"
              class="mt-2 w-full rounded-md border border-white/15 bg-white/5 px-4 py-2 text-slate-100 placeholder:text-slate-500 focus:border-amber-400 focus:ring-1 focus:ring-amber-400 focus:outline-none"
            />
          </div>
          <div>
            <label for="gift-message" class="block text-sm font-medium text-slate-300">Mensaje</label>
            <textarea
              id="gift-message"
              value={message}
              onInput={(event) => setMessage(event.currentTarget.value)}
              rows={4}
              placeholder="El mensaje que se revela al abrir el regalo"
              class="mt-2 w-full rounded-md border border-white/15 bg-white/5 px-4 py-2 text-slate-100 placeholder:text-slate-500 focus:border-amber-400 focus:ring-1 focus:ring-amber-400 focus:outline-none"
            />
          </div>
        </aside>
      </div>
    </div>
  );
}
