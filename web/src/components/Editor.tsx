import { useEffect, useRef, useState } from 'preact/hooks';
import {
  colorIndex,
  createCanvas,
  EMPTY,
  fitCellSize,
  floodFill,
  paintLine,
  render,
  sizeCanvas,
  type PixelCanvas,
  type SurfaceColors,
} from '../lib/canvas';
import { EraserIcon, PaintBucketIcon, PencilIcon, RedoIcon, UndoIcon } from './icons';
import ThemeToggle from './ThemeToggle';

// The colour the pencil starts on before the user picks another (PP-45).
const DEFAULT_COLOR = '#fbbf24';

// Undo/redo depth (PP-47): a full snapshot of the pixels array per discrete
// action. At the 128×128 maximum each snapshot is 16KB, so 50 steps is trivial
// in memory (architecture §editor) while covering any realistic stroke history.
const HISTORY_LIMIT = 50;

// Curated starter swatches (PP-45): a small warm/retro set for one-tap picks,
// aligned with the amber accent. The native colour input covers anything else.
const SWATCHES = [
  '#000000', '#ffffff', '#9ca3af', '#6d4c41',
  '#ef4444', '#f97316', '#fbbf24', '#facc15',
  '#22c55e', '#10b981', '#06b6d4', '#3b82f6',
  '#6366f1', '#a855f7', '#ec4899', '#f9a8d4',
];

// Active drawing tool. Pencil paints the current colour and eraser clears cells
// back to EMPTY — both share the same drag interaction (PP-44). Fill (PP-46)
// flood-fills the clicked region with the current colour on a single click.
type Tool = 'pencil' | 'eraser' | 'fill';

// Shared class for a tool-bar button, theme-aware (PP-44.5). Extracted so the
// three buttons stay identical and the light/dark styling lives in one place.
function toolButtonClass(active: boolean): string {
  return `inline-flex items-center gap-1.5 rounded-md border px-3 py-1.5 text-sm font-medium transition ${
    active
      ? 'border-amber-500 bg-amber-400/20 text-amber-700 dark:border-amber-400 dark:bg-amber-400/15 dark:text-amber-200'
      : 'border-slate-300 text-slate-600 hover:border-slate-400 dark:border-white/15 dark:text-slate-300 dark:hover:border-white/30'
  }`;
}

// Shared class for the title/message text fields (theme-aware, PP-44.5).
const FIELD_CLASS =
  'mt-2 w-full rounded-md border border-slate-300 bg-white px-4 py-2 text-slate-900 placeholder:text-slate-400 focus:border-amber-400 focus:ring-1 focus:ring-amber-400 focus:outline-none dark:border-white/15 dark:bg-white/5 dark:text-slate-100 dark:placeholder:text-slate-500';

// Reads the theme-aware canvas surface colours from the CSS variables that flip
// with the .dark class (defined in global.css), with light-theme fallbacks.
function surfaceColors(): SurfaceColors {
  const s = getComputedStyle(document.documentElement);
  return {
    empty: s.getPropertyValue('--canvas-empty').trim() || '#ffffff',
    grid: s.getPropertyValue('--canvas-grid').trim() || 'rgba(0, 0, 0, 0.12)',
  };
}

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
  const [tool, setTool] = useState<Tool>('pencil');
  const [color, setColor] = useState(DEFAULT_COLOR);

  const canvasRef = useRef<HTMLCanvasElement>(null);
  const boardRef = useRef<HTMLDivElement>(null);
  // The drawing model. A fixed 16×16 for PP-42; selectable sizes arrive in PP-49.
  const modelRef = useRef<PixelCanvas>(createCanvas(16, 16));
  // Mirror the active tool into a ref so the pointer closures below read the
  // current tool without re-running the drawing effect (which would rebind the
  // listeners on every tool switch).
  const toolRef = useRef<Tool>(tool);
  useEffect(() => {
    toolRef.current = tool;
  }, [tool]);
  // Same trick for the active colour: the pencil reads it from a ref so changing
  // colour never rebinds the pointer listeners.
  const colorRef = useRef<string>(color);
  useEffect(() => {
    colorRef.current = color;
  }, [color]);

  // Undo/redo history (PP-47): each stack holds full copies of `pixels` (not the
  // palette, which only ever grows). A discrete action pushes the pre-mutation
  // snapshot; undo/redo swap the live pixels for a snapshot and repaint. The
  // stacks live in refs so the pointer closures below never rebind, while
  // canUndo/canRedo mirror their emptiness into state to drive the buttons.
  const undoStackRef = useRef<Uint8Array[]>([]);
  const redoStackRef = useRef<Uint8Array[]>([]);
  const [canUndo, setCanUndo] = useState(false);
  const [canRedo, setCanRedo] = useState(false);
  // Set by the drawing effect to repaint the current model; called by undo/redo
  // (which run outside the effect) so they don't need the canvas context.
  const repaintRef = useRef<() => void>(() => {});

  function syncHistory() {
    setCanUndo(undoStackRef.current.length > 0);
    setCanRedo(redoStackRef.current.length > 0);
  }

  // Snapshot the pixels *before* a discrete action mutates them, and drop the
  // redo branch (a new action invalidates any undone future). Trims the oldest
  // step once the stack passes the limit.
  function pushHistory() {
    const stack = undoStackRef.current;
    stack.push(modelRef.current.pixels.slice());
    if (stack.length > HISTORY_LIMIT) stack.shift();
    redoStackRef.current = [];
    syncHistory();
  }

  function undo() {
    const stack = undoStackRef.current;
    if (stack.length === 0) return;
    redoStackRef.current.push(modelRef.current.pixels.slice());
    modelRef.current.pixels = stack.pop() as Uint8Array;
    repaintRef.current();
    syncHistory();
  }

  function redo() {
    const stack = redoStackRef.current;
    if (stack.length === 0) return;
    undoStackRef.current.push(modelRef.current.pixels.slice());
    modelRef.current.pixels = stack.pop() as Uint8Array;
    repaintRef.current();
    syncHistory();
  }

  // Picking a colour (swatch or custom) selects it and switches to the pencil —
  // choosing a colour implies you want to draw with it, not erase.
  function pickColor(hex: string) {
    setColor(hex);
    setTool('pencil');
  }

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

    // Theme-aware surface colours, cached so painting doesn't re-read the CSS
    // variables on every pointer move; refreshed on resize and theme change.
    let colors = surfaceColors();
    let cellSize = fitCellSize(board.clientWidth, model);
    let ctx = sizeCanvas(canvas, model, cellSize);
    if (ctx) render(ctx, model, cellSize, colors);

    function redraw() {
      colors = surfaceColors();
      cellSize = fitCellSize(board.clientWidth, model);
      ctx = sizeCanvas(canvas, model, cellSize);
      if (ctx) render(ctx, model, cellSize, colors);
    }

    // Let undo/redo (defined outside this effect) repaint the current model
    // without owning the canvas context, cell size or colours.
    repaintRef.current = () => {
      if (ctx) render(ctx, model, cellSize, colors);
    };

    // Repaint when the theme toggles (the .dark class on <html> changes), so the
    // empty/erased cells and grid lines follow light/dark without a resize.
    function onThemeChange() {
      colors = surfaceColors();
      if (ctx) render(ctx, model, cellSize, colors);
    }
    const themeObserver = new MutationObserver(onThemeChange);
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    });

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
      const ink = toolRef.current === 'eraser' ? EMPTY : colorIndex(model, colorRef.current);
      paintLine(model, from.x, from.y, to.x, to.y, ink);
      if (ctx) render(ctx, model, cellSize, colors);
    }

    function onPointerDown(event: PointerEvent) {
      const cell = cellFromEvent(event);
      if (!cell) return;
      // A discrete action begins here: snapshot the pixels before mutating so the
      // whole gesture (a drag, or a single fill) undoes as one step (PP-47).
      pushHistory();
      // Fill is a one-shot click, not a drag: flood-fill and stop (drawing stays
      // false, so the move handler below is a no-op for this gesture).
      if (toolRef.current === 'fill') {
        floodFill(model, cell.x, cell.y, colorIndex(model, colorRef.current));
        if (ctx) render(ctx, model, cellSize, colors);
        return;
      }
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

    // Keyboard shortcuts: Ctrl/Cmd+Z undoes, Ctrl/Cmd+Shift+Z or Ctrl/Cmd+Y
    // redoes — the platform-standard bindings for undo/redo (PP-47). Ignore key
    // repeat so a held-down combo doesn't blow through the whole history.
    function onKeyDown(event: KeyboardEvent) {
      if (!(event.ctrlKey || event.metaKey) || event.repeat) return;
      // Don't hijack the native undo of the title/message text fields.
      const target = event.target as HTMLElement | null;
      const tag = target?.tagName;
      if (tag === 'INPUT' || tag === 'TEXTAREA' || target?.isContentEditable) return;
      const key = event.key.toLowerCase();
      if (key === 'z' && !event.shiftKey) {
        event.preventDefault();
        undo();
      } else if (key === 'y' || (key === 'z' && event.shiftKey)) {
        event.preventDefault();
        redo();
      }
    }

    canvas.addEventListener('pointerdown', onPointerDown);
    canvas.addEventListener('pointermove', onPointerMove);
    canvas.addEventListener('pointerup', onPointerUp);
    canvas.addEventListener('pointercancel', onPointerUp);
    window.addEventListener('resize', redraw);
    window.addEventListener('keydown', onKeyDown);
    return () => {
      canvas.removeEventListener('pointerdown', onPointerDown);
      canvas.removeEventListener('pointermove', onPointerMove);
      canvas.removeEventListener('pointerup', onPointerUp);
      canvas.removeEventListener('pointercancel', onPointerUp);
      window.removeEventListener('resize', redraw);
      window.removeEventListener('keydown', onKeyDown);
      themeObserver.disconnect();
    };
  }, [status]);

  if (status === 'loading') {
    return <p class="px-6 py-10 text-slate-500 dark:text-slate-400">Cargando el editor…</p>;
  }

  if (status === 'notfound') {
    return (
      <div class="px-6 py-10">
        <p class="text-slate-600 dark:text-slate-300">
          Ese regalo no existe o no es tuyo.{' '}
          <a
            href="/dashboard"
            class="text-amber-600 hover:text-amber-500 dark:text-amber-300 dark:hover:text-amber-200"
          >
            Volver a tus regalos
          </a>
          .
        </p>
      </div>
    );
  }

  if (status === 'error') {
    return (
      <div class="px-6 py-10">
        <p class="text-rose-600 dark:text-rose-300" role="alert">
          No hemos podido cargar el regalo. Inténtalo de nuevo en un momento.
        </p>
      </div>
    );
  }

  return (
    <div class="flex min-h-screen flex-col">
      <header class="flex items-center justify-between gap-4 border-b border-slate-200 px-6 py-4 dark:border-white/10">
        <a
          href="/dashboard"
          class="font-mono text-xs font-bold tracking-widest text-amber-600 dark:text-amber-300"
        >
          ← PIXEL&nbsp;PRESENT
        </a>
        <div class="flex items-center gap-4">
          <span class="text-sm text-slate-500 dark:text-slate-400">Editor</span>
          <ThemeToggle />
        </div>
      </header>

      <div class="grid flex-1 gap-6 px-6 py-6 lg:grid-cols-[1fr_20rem]">
        {/* Canvas area. The board fits its container width (responsive/mobile);
            zoom (PP-48) and selectable sizes (PP-49) come later. */}
        <section class="flex flex-col items-center gap-4 rounded-xl border border-slate-200 bg-slate-100 p-4 dark:border-white/10 dark:bg-white/5">
          {/* Tool bar. Pencil and eraser share the same drag interaction; the
              eraser clears cells back to empty (PP-44). Fill (PP-46) flood-fills
              the clicked region with the current colour. */}
          <div class="flex gap-2 self-start" role="toolbar" aria-label="Herramientas de dibujo">
            <button
              type="button"
              onClick={() => setTool('pencil')}
              aria-pressed={tool === 'pencil'}
              class={toolButtonClass(tool === 'pencil')}
            >
              <PencilIcon class="h-4 w-4" />
              Lápiz
            </button>
            <button
              type="button"
              onClick={() => setTool('eraser')}
              aria-pressed={tool === 'eraser'}
              class={toolButtonClass(tool === 'eraser')}
            >
              <EraserIcon class="h-4 w-4" />
              Borrador
            </button>
            <button
              type="button"
              onClick={() => setTool('fill')}
              aria-pressed={tool === 'fill'}
              class={toolButtonClass(tool === 'fill')}
            >
              <PaintBucketIcon class="h-4 w-4" />
              Relleno
            </button>

            {/* Undo/redo (PP-47): action buttons, not tools, so no aria-pressed.
                Disabled when their history stack is empty; Ctrl/Cmd+Z and
                Ctrl/Cmd+Shift+Z (or +Y) do the same from the keyboard. */}
            <span class="mx-1 w-px self-stretch bg-slate-300 dark:bg-white/15" aria-hidden="true" />
            <button
              type="button"
              onClick={undo}
              disabled={!canUndo}
              aria-label="Deshacer"
              class={`${toolButtonClass(false)} disabled:cursor-not-allowed disabled:opacity-40`}
            >
              <UndoIcon class="h-4 w-4" />
              Deshacer
            </button>
            <button
              type="button"
              onClick={redo}
              disabled={!canRedo}
              aria-label="Rehacer"
              class={`${toolButtonClass(false)} disabled:cursor-not-allowed disabled:opacity-40`}
            >
              <RedoIcon class="h-4 w-4" />
              Rehacer
            </button>
          </div>

          {/* Colour picker (PP-45): curated swatches for quick picks plus a
              native input for anything else. The chosen colour drives the
              pencil; picking one switches away from the eraser. */}
          <div class="flex flex-col gap-2 self-start">
            <div class="flex flex-wrap gap-1.5" role="group" aria-label="Colores">
              {SWATCHES.map((hex) => (
                <button
                  key={hex}
                  type="button"
                  onClick={() => pickColor(hex)}
                  aria-label={`Color ${hex}`}
                  aria-pressed={color === hex}
                  class={`h-6 w-6 rounded border transition ${
                    color === hex
                      ? 'border-slate-900 ring-2 ring-amber-500 dark:border-white dark:ring-amber-400'
                      : 'border-slate-300 hover:border-slate-500 dark:border-white/20 dark:hover:border-white/50'
                  }`}
                  style={{ backgroundColor: hex }}
                />
              ))}
            </div>
            <label class="inline-flex items-center gap-2 text-sm text-slate-600 dark:text-slate-300">
              Personalizado
              <input
                type="color"
                value={color}
                onInput={(event) => pickColor(event.currentTarget.value)}
                aria-label="Elegir un color personalizado"
                class="h-8 w-12 cursor-pointer rounded border border-slate-300 bg-transparent dark:border-white/15"
              />
            </label>
          </div>

          <div ref={boardRef} class="w-full max-w-lg">
            <canvas ref={canvasRef} class="mx-auto block cursor-crosshair touch-none" />
          </div>
        </section>

        {/* Side panel — gift metadata. */}
        <aside class="space-y-4">
          <div>
            <label for="gift-title" class="block text-sm font-medium text-slate-600 dark:text-slate-300">Título</label>
            <input
              id="gift-title"
              type="text"
              value={title}
              onInput={(event) => setTitle(event.currentTarget.value)}
              placeholder="Para alguien especial"
              class={FIELD_CLASS}
            />
          </div>
          <div>
            <label for="gift-message" class="block text-sm font-medium text-slate-600 dark:text-slate-300">Mensaje</label>
            <textarea
              id="gift-message"
              value={message}
              onInput={(event) => setMessage(event.currentTarget.value)}
              rows={4}
              placeholder="El mensaje que se revela al abrir el regalo"
              class={FIELD_CLASS}
            />
          </div>
        </aside>
      </div>
    </div>
  );
}
