import { useEffect, useRef, useState } from 'preact/hooks';
import {
  colorIndex,
  createCanvas,
  deserializeCanvas,
  EMPTY,
  fitCellSize,
  floodFill,
  paintLine,
  render,
  resizeCanvas,
  serializeCanvas,
  sizeCanvas,
  type PixelCanvas,
  type SurfaceColors,
} from '../lib/canvas';
import { EraserIcon, MoveIcon, PaintBucketIcon, PencilIcon, RedoIcon, UndoIcon } from './icons';
import ThemeToggle from './ThemeToggle';

// The colour the pencil starts on before the user picks another (PP-45).
const DEFAULT_COLOR = '#fbbf24';

// Undo/redo depth (PP-47): a full snapshot of the pixels array per discrete
// action. At the 128×128 maximum each snapshot is 16KB, so 50 steps is trivial
// in memory (architecture §editor) while covering any realistic stroke history.
const HISTORY_LIMIT = 50;

// Zoom (PP-48): a multiplier over the fit-to-width cell size, so 100% keeps the
// existing responsive behaviour and zooming in/out just scales cellSize and
// redraws (architecture §editor: "cellSize controla el zoom"). Panning a grid
// too big to fit is a separate task (PP-50); here the board simply scrolls.
const ZOOM_MIN = 0.5;
const ZOOM_MAX = 4;
const ZOOM_STEP = 0.25;

// Rounds a zoom factor to 2 decimals so repeated ±0.25 steps don't drift into
// float noise (e.g. 0.7500000001) in the percentage label.
function roundZoom(z: number): number {
  return Math.round(z * 100) / 100;
}

// Grid size: square grids from 8×8 up to the architecture's 128×128 cap. PP-49
// added the presets and custom input (capped at 64); PP-50 raises the ceiling to
// 128 now that large grids can be panned. Two quick presets plus a custom
// numeric input drive the size.
const SIZE_MIN = 8;
const SIZE_MAX = 128;
const SIZE_PRESETS = [16, 32];

// Clamps a requested size into range and to a whole number of cells, falling
// back to 16 when the input is empty/NaN (e.g. the custom field cleared).
function clampSize(n: number): number {
  if (!Number.isFinite(n)) return 16;
  return Math.max(SIZE_MIN, Math.min(SIZE_MAX, Math.round(n)));
}

// Curated starter swatches (PP-45): a small warm/retro set for one-tap picks,
// aligned with the amber accent. The native colour input covers anything else.
const SWATCHES = [
  '#000000', '#ffffff', '#9ca3af', '#6d4c41',
  '#ef4444', '#f97316', '#fbbf24', '#facc15',
  '#22c55e', '#10b981', '#06b6d4', '#3b82f6',
  '#6366f1', '#a855f7', '#ec4899', '#f9a8d4',
];

// Active tool. Pencil paints the current colour and eraser clears cells back to
// EMPTY — both share the same drag interaction (PP-44). Fill (PP-46) flood-fills
// the clicked region on a single click. Pan (PP-50) is a separate "move view"
// mode: a drag scrolls the board instead of painting, so large grids that don't
// fit on screen (up to 128×128) can be navigated with touch precision.
type Tool = 'pencil' | 'eraser' | 'fill' | 'pan';

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
  reveal_config?: unknown;
  // RFC3339 instants from the API, absent (omitempty) when unset (PP-52).
  scheduled_open_at?: string | null;
  expires_at?: string | null;
  // Always present in the API response (PP-53).
  single_open?: boolean;
}

function giftIdFromURL(): string | null {
  if (typeof window === 'undefined') return null;
  return new URLSearchParams(window.location.search).get('id');
}

// isoToDateInput turns an RFC3339 instant from the API into the value an
// <input type="date"> expects — a local calendar date "YYYY-MM-DD" — or '' when
// absent/unparseable. The time of day is dropped on purpose: the scheduling
// gates are date-only (a gift opens/expires on a day, not at a given minute).
// Saving (PP-54) turns the date back into an absolute instant.
function isoToDateInput(iso: string | null | undefined): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

// dateInputToISO is the inverse of isoToDateInput for saving (PP-54): it turns a
// date-only "YYYY-MM-DD" into an absolute RFC3339 instant, or null when empty.
// The gate boundaries (gifts/visibility.go) are scheduled_open_at inclusive and
// expires_at exclusive, so an open date maps to the local start of that day and
// an expiry date to the local end of it (23:59:59.999) — the gift is then
// viewable through the whole expiry day, and both round-trip back to the same
// day via isoToDateInput (which reads the local calendar date).
function dateInputToISO(date: string, endOfDay: boolean): string | null {
  if (!date) return null;
  const [y, m, d] = date.split('-').map(Number);
  if (!y || !m || !d) return null;
  const dt = endOfDay
    ? new Date(y, m - 1, d, 23, 59, 59, 999)
    : new Date(y, m - 1, d, 0, 0, 0, 0);
  if (Number.isNaN(dt.getTime())) return null;
  return dt.toISOString();
}

export default function Editor() {
  const [status, setStatus] = useState<Status>('loading');
  const [title, setTitle] = useState('');
  const [message, setMessage] = useState('');
  // Optional scheduling gates (PP-52), held as date strings "YYYY-MM-DD" ('' = unset).
  const [scheduledOpenAt, setScheduledOpenAt] = useState('');
  const [expiresAt, setExpiresAt] = useState('');
  // Single-open toggle (PP-53): the gift can be opened only once when set.
  const [singleOpen, setSingleOpen] = useState(false);

  // Save wiring (PP-54). giftId is the gift being edited: null for a brand-new
  // one (save does POST), set once it exists (save does PUT). It seeds from the
  // ?id URL param and is filled in after the first create.
  const [giftId, setGiftId] = useState<string | null>(() => giftIdFromURL());
  const [saveState, setSaveState] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const [saveError, setSaveError] = useState('');
  // Preview modal (PP-55): renders the live model at a smaller scale, so no extra
  // pixel state — it reuses the same model and render function.
  const [showPreview, setShowPreview] = useState(false);
  const previewCanvasRef = useRef<HTMLCanvasElement>(null);
  // reveal_type/reveal_config aren't editable yet (confetti is the only MVP
  // mechanic). Held so a full-replace update (PUT) preserves whatever a loaded
  // gift already had instead of resetting it. New gifts default to confetti.
  const revealTypeRef = useRef<string>('confetti');
  const revealConfigRef = useRef<unknown>({});
  const [tool, setTool] = useState<Tool>('pencil');
  const [color, setColor] = useState(DEFAULT_COLOR);
  const [zoom, setZoom] = useState(1);
  const [size, setSize] = useState(16);

  const canvasRef = useRef<HTMLCanvasElement>(null);
  const boardRef = useRef<HTMLDivElement>(null);
  // The drawing model. Starts 16×16; the size controls (PP-49) swap it for a new
  // grid of the chosen size, preserving the overlapping region.
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

  // Zoom factor mirrored into a ref so the drawing effect reads the current
  // value without rebinding, plus a redraw hook the effect fills in: changing
  // zoom recomputes cellSize and re-sizes the canvas (PP-48).
  const zoomRef = useRef<number>(zoom);
  const redrawRef = useRef<() => void>(() => {});
  useEffect(() => {
    zoomRef.current = zoom;
    redrawRef.current();
  }, [zoom]);

  function zoomIn() {
    setZoom((z) => Math.min(ZOOM_MAX, roundZoom(z + ZOOM_STEP)));
  }
  function zoomOut() {
    setZoom((z) => Math.max(ZOOM_MIN, roundZoom(z - ZOOM_STEP)));
  }
  function resetZoom() {
    setZoom(1);
  }

  // Change the grid size (PP-49). Recreates the model at the new square size
  // (keeping the overlapping drawing) and clears undo/redo, whose snapshots are
  // sized to the old grid and can't be replayed onto the new one. Updating `size`
  // re-runs the drawing effect, which re-sizes the canvas and repaints.
  function changeSize(next: number) {
    const clamped = clampSize(next);
    if (clamped !== modelRef.current.width || clamped !== modelRef.current.height) {
      modelRef.current = resizeCanvas(modelRef.current, clamped, clamped);
      undoStackRef.current = [];
      redoStackRef.current = [];
      syncHistory();
    }
    setSize(clamped);
  }

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

  // Persist the gift (PP-54): serialize the editor state to the write payload and
  // POST a new gift or PUT the existing one. On the first create we capture the
  // id and reflect it in the URL so a reload edits the same gift (the round-trip
  // the DoD asks for). Save is disabled while in flight to avoid double submits.
  async function save() {
    if (title.trim() === '') {
      setSaveError('El título es obligatorio.');
      setSaveState('error');
      return;
    }
    setSaveState('saving');
    setSaveError('');

    const body = {
      title: title.trim(),
      message,
      pixel_art: serializeCanvas(modelRef.current),
      reveal_type: revealTypeRef.current,
      reveal_config: revealConfigRef.current,
      recipient_email: null,
      scheduled_open_at: dateInputToISO(scheduledOpenAt, false),
      scheduled_send_at: null,
      single_open: singleOpen,
      expires_at: dateInputToISO(expiresAt, true),
    };

    try {
      const res = await fetch(giftId ? `/api/gifts/${giftId}` : '/api/gifts', {
        method: giftId ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify(body),
      });

      if (res.status === 401) {
        window.location.replace('/login');
        return;
      }
      if (res.status === 400) {
        setSaveError('Revisa los campos del regalo.');
        setSaveState('error');
        return;
      }
      if (res.status === 403 || res.status === 404) {
        setSaveError('Ese regalo no existe o no es tuyo.');
        setSaveState('error');
        return;
      }
      if (!res.ok) {
        setSaveError('No se pudo guardar. Inténtalo de nuevo.');
        setSaveState('error');
        return;
      }

      // POST returns {id, view_token}; PUT returns the full gift. Adopt the id on
      // first create so later saves update in place, and mirror it into the URL.
      const data = (await res.json()) as { id?: string };
      if (!giftId && data.id) {
        setGiftId(data.id);
        const next = new URL(window.location.href);
        next.searchParams.set('id', data.id);
        window.history.replaceState(null, '', next);
      }
      setSaveState('saved');
    } catch {
      setSaveError('No se pudo guardar. Comprueba tu conexión.');
      setSaveState('error');
    }
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
        setScheduledOpenAt(isoToDateInput(gift.scheduled_open_at));
        setExpiresAt(isoToDateInput(gift.expires_at));
        setSingleOpen(gift.single_open ?? false);
        if (gift.reveal_type) revealTypeRef.current = gift.reveal_type;
        if (gift.reveal_config !== undefined) revealConfigRef.current = gift.reveal_config;
        // Load the saved drawing into the model (PP-54 round-trip). On malformed
        // pixel_art, keep the blank starter grid rather than trust bad data.
        // Set the model before status flips to 'ready' so the drawing effect
        // renders it; sync the size control to the loaded grid (assumed square,
        // which is all this editor produces).
        const loaded = deserializeCanvas(gift.pixel_art);
        if (loaded) {
          modelRef.current = loaded;
          setSize(loaded.width);
        }
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

    // Effective cell size = fit-to-width base × zoom (PP-48). At zoom 1 this is
    // the plain responsive fit; zooming in makes the canvas overflow the board,
    // which scrolls. Never smaller than 1px so the grid stays valid.
    function computeCellSize(): number {
      return Math.max(1, Math.round(fitCellSize(board.clientWidth, model) * zoomRef.current));
    }

    // Theme-aware surface colours, cached so painting doesn't re-read the CSS
    // variables on every pointer move; refreshed on resize and theme change.
    let colors = surfaceColors();
    let cellSize = computeCellSize();
    let ctx = sizeCanvas(canvas, model, cellSize);
    if (ctx) render(ctx, model, cellSize, colors);

    function redraw() {
      colors = surfaceColors();
      cellSize = computeCellSize();
      ctx = sizeCanvas(canvas, model, cellSize);
      if (ctx) render(ctx, model, cellSize, colors);
    }
    // Let the zoom control (outside this effect) re-size and repaint on change.
    redrawRef.current = redraw;

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
    // Pan mode (PP-50): a drag scrolls the board instead of painting. We scroll
    // it manually (rather than relying on native touch scroll) so mouse and touch
    // behave identically and touch-action:none can stay on to keep the page from
    // scrolling/zooming underneath. panStart captures the pointer and scroll
    // position at the gesture's start; scroll = start − finger movement.
    let panning = false;
    let panStart: { x: number; y: number; left: number; top: number } | null = null;

    function paint(from: { x: number; y: number }, to: { x: number; y: number }) {
      const ink = toolRef.current === 'eraser' ? EMPTY : colorIndex(model, colorRef.current);
      paintLine(model, from.x, from.y, to.x, to.y, ink);
      if (ctx) render(ctx, model, cellSize, colors);
    }

    function onPointerDown(event: PointerEvent) {
      // Pan mode: start a drag-to-scroll gesture; no cell, no history, no paint.
      if (toolRef.current === 'pan') {
        panning = true;
        panStart = {
          x: event.clientX,
          y: event.clientY,
          left: board.scrollLeft,
          top: board.scrollTop,
        };
        canvas.setPointerCapture(event.pointerId);
        canvas.style.cursor = 'grabbing';
        return;
      }
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
      if (panning && panStart) {
        board.scrollLeft = panStart.left - (event.clientX - panStart.x);
        board.scrollTop = panStart.top - (event.clientY - panStart.y);
        return;
      }
      if (!drawing) return;
      const cell = cellFromEvent(event);
      if (!cell) return;
      paint(last ?? cell, cell);
      last = cell;
    }

    function onPointerUp(event: PointerEvent) {
      if (panning) {
        panning = false;
        panStart = null;
        // Drop the inline grabbing cursor; the class falls back to grab.
        canvas.style.cursor = '';
      }
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
  }, [status, size]);

  // Render the preview when the modal opens (PP-55). It reads the live model, so
  // it always shows the latest drawing; a smaller cellSize fits it into the modal
  // and grid=false gives the clean look the recipient will see — no extra state,
  // just the shared render function.
  useEffect(() => {
    if (!showPreview) return;
    const el = previewCanvasRef.current;
    if (el === null) return;
    const model = modelRef.current;
    // Fit the whole drawing into a ~320px box; never below 1px per cell.
    const cellSize = Math.max(1, Math.floor(320 / Math.max(model.width, model.height)));
    const ctx = sizeCanvas(el, model, cellSize);
    if (ctx) render(ctx, model, cellSize, surfaceColors(), false);

    // Close on Escape while the modal is open.
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') setShowPreview(false);
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [showPreview]);

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
        <div class="flex items-center gap-3">
          {saveState === 'error' && saveError && (
            <span class="text-xs text-rose-600 dark:text-rose-300" role="alert">
              {saveError}
            </span>
          )}
          {saveState === 'saved' && (
            <span class="text-xs text-emerald-600 dark:text-emerald-300">Guardado ✓</span>
          )}
          <button
            type="button"
            onClick={() => setShowPreview(true)}
            class="rounded-md border border-slate-300 px-4 py-1.5 text-sm font-medium text-slate-600 transition hover:border-slate-400 dark:border-white/15 dark:text-slate-300 dark:hover:border-white/30"
          >
            Vista previa
          </button>
          <button
            type="button"
            onClick={save}
            disabled={saveState === 'saving'}
            class="rounded-md bg-amber-500 px-4 py-1.5 text-sm font-medium text-white transition hover:bg-amber-400 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-amber-400 dark:text-slate-900 dark:hover:bg-amber-300"
          >
            {saveState === 'saving' ? 'Guardando…' : 'Guardar'}
          </button>
          <ThemeToggle />
        </div>
      </header>

      <div class="grid flex-1 gap-6 px-6 py-6 lg:grid-cols-[1fr_20rem]">
        {/* Canvas area. The board fits its container width (responsive/mobile);
            zoom (PP-48) scales from there and the size control (PP-49) picks the
            grid dimensions. */}
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
            <button
              type="button"
              onClick={() => setTool('pan')}
              aria-pressed={tool === 'pan'}
              class={toolButtonClass(tool === 'pan')}
            >
              <MoveIcon class="h-4 w-4" />
              Mover
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

          {/* Grid size (PP-49): two quick presets plus a custom numeric input
              (8–128, raised from 64 by PP-50). Changing size recreates the grid —
              keeping the overlapping drawing — and clears undo/redo. Grids too big
              to fit are navigated with the Mover (pan) tool (PP-50). */}
          <div
            class="flex flex-wrap items-center gap-2 self-start"
            role="group"
            aria-label="Tamaño del lienzo"
          >
            <span class="text-sm text-slate-600 dark:text-slate-300">Tamaño</span>
            {SIZE_PRESETS.map((preset) => (
              <button
                key={preset}
                type="button"
                onClick={() => changeSize(preset)}
                aria-pressed={size === preset}
                class={toolButtonClass(size === preset)}
              >
                {preset}×{preset}
              </button>
            ))}
            <label class="inline-flex items-center gap-1.5 text-sm text-slate-600 dark:text-slate-300">
              Personalizado
              <input
                type="number"
                min={SIZE_MIN}
                max={SIZE_MAX}
                value={size}
                onChange={(event) => changeSize(event.currentTarget.valueAsNumber)}
                aria-label="Tamaño personalizado del lienzo (entre 8 y 128)"
                class="w-16 rounded-md border border-slate-300 bg-white px-2 py-1.5 text-slate-900 focus:border-amber-400 focus:ring-1 focus:ring-amber-400 focus:outline-none dark:border-white/15 dark:bg-white/5 dark:text-slate-100"
              />
            </label>
          </div>

          {/* Zoom control (PP-48): scales cellSize and redraws. The percentage
              doubles as a reset-to-100% button; the buttons disable at the
              limits. Panning a zoomed-in grid by touch is a later task (PP-50). */}
          <div class="flex items-center gap-2 self-start" role="group" aria-label="Zoom">
            <span class="text-sm text-slate-600 dark:text-slate-300">Zoom</span>
            <button
              type="button"
              onClick={zoomOut}
              disabled={zoom <= ZOOM_MIN}
              aria-label="Alejar"
              class={`${toolButtonClass(false)} px-2.5 disabled:cursor-not-allowed disabled:opacity-40`}
            >
              −
            </button>
            <button
              type="button"
              onClick={resetZoom}
              aria-label="Restablecer zoom al 100%"
              class="min-w-[3.25rem] rounded-md px-2 py-1.5 text-center text-sm tabular-nums text-slate-600 hover:text-amber-600 dark:text-slate-300 dark:hover:text-amber-300"
            >
              {Math.round(zoom * 100)}%
            </button>
            <button
              type="button"
              onClick={zoomIn}
              disabled={zoom >= ZOOM_MAX}
              aria-label="Acercar"
              class={`${toolButtonClass(false)} px-2.5 disabled:cursor-not-allowed disabled:opacity-40`}
            >
              +
            </button>
          </div>

          {/* The board is the measured container (its width drives the fit-to-
              width base); it scrolls when the canvas overflows (zoom or a large
              grid), and the Mover tool (PP-50) pans it by setting its scroll. */}
          <div ref={boardRef} class="max-h-[70vh] w-full max-w-lg overflow-auto">
            <canvas
              ref={canvasRef}
              class={`mx-auto block touch-none ${tool === 'pan' ? 'cursor-grab' : 'cursor-crosshair'}`}
            />
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

          {/* Scheduling gates (PP-52): both optional and date-only (the day is
              enough for a gift; picking a time is needless friction). An empty
              input means unset (no gate). The backend applies scheduled_open_at /
              expires_at as visibility gates (PP-24). Wiring these into the
              save/load round-trip is PP-54. */}
          <div>
            <label for="gift-open-at" class="block text-sm font-medium text-slate-600 dark:text-slate-300">
              Apertura programada{' '}
              <span class="font-normal text-slate-400 dark:text-slate-500">(opcional)</span>
            </label>
            <input
              id="gift-open-at"
              type="date"
              value={scheduledOpenAt}
              onInput={(event) => setScheduledOpenAt(event.currentTarget.value)}
              class={FIELD_CLASS}
            />
            <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
              El regalo no podrá abrirse antes de este día.
            </p>
          </div>
          <div>
            <label for="gift-expires-at" class="block text-sm font-medium text-slate-600 dark:text-slate-300">
              Caducidad{' '}
              <span class="font-normal text-slate-400 dark:text-slate-500">(opcional)</span>
            </label>
            <input
              id="gift-expires-at"
              type="date"
              value={expiresAt}
              onInput={(event) => setExpiresAt(event.currentTarget.value)}
              class={FIELD_CLASS}
            />
            <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
              Tras este día el regalo dejará de estar disponible.
            </p>
          </div>

          {/* Single-open toggle (PP-53): when checked, opening the gift once marks
              it consumed (opened_at) and further views show "ya abierto". Wiring
              this into the save round-trip is PP-54. */}
          <div>
            <label class="flex items-start gap-3">
              <input
                type="checkbox"
                checked={singleOpen}
                onChange={(event) => setSingleOpen(event.currentTarget.checked)}
                class="mt-0.5 h-4 w-4 rounded border-slate-300 text-amber-500 focus:ring-amber-400 dark:border-white/20 dark:bg-white/5"
              />
              <span>
                <span class="block text-sm font-medium text-slate-600 dark:text-slate-300">
                  Apertura única
                </span>
                <span class="mt-0.5 block text-xs text-slate-500 dark:text-slate-400">
                  El regalo solo podrá abrirse una vez; después dejará de estar disponible.
                </span>
              </span>
            </label>
          </div>
        </aside>
      </div>

      {/* Preview modal (PP-55): the recipient's-eye view of the gift, drawn from
          the live model with the shared render function at a smaller, gridless
          scale. Title and message reuse the existing state — no preview-only
          data. Click the backdrop or press Escape to close. */}
      {showPreview && (
        <div
          class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
          role="dialog"
          aria-modal="true"
          aria-label="Vista previa del regalo"
          onClick={() => setShowPreview(false)}
        >
          <div
            class="w-full max-w-md rounded-xl border border-slate-200 bg-white p-6 shadow-xl dark:border-white/10 dark:bg-slate-900"
            onClick={(event) => event.stopPropagation()}
          >
            <div class="mb-4 flex items-center justify-between">
              <h2 class="text-sm font-semibold text-slate-700 dark:text-slate-200">Vista previa</h2>
              <button
                type="button"
                onClick={() => setShowPreview(false)}
                aria-label="Cerrar vista previa"
                class="rounded-md px-2 py-1 text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-100"
              >
                ✕
              </button>
            </div>
            {title && (
              <p class="mb-3 text-center text-lg font-semibold text-slate-900 dark:text-slate-100">
                {title}
              </p>
            )}
            <div class="flex justify-center">
              <canvas ref={previewCanvasRef} class="block rounded-md shadow-sm" />
            </div>
            {message && (
              <p class="mt-4 text-center text-sm whitespace-pre-wrap text-slate-600 dark:text-slate-300">
                {message}
              </p>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
