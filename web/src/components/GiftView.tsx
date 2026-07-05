import { useEffect, useRef, useState } from 'preact/hooks';
import { deserializeCanvas, render, sizeCanvas, toPNGDataURL } from '../lib/canvas';
import ThemeToggle from './ThemeToggle';

// GiftView is the public, recipient-facing page (PP-57): it reads the view token
// from the /g/{token} URL and consumes GET /api/g/{view_token}, which applies the
// visibility gate and returns a state discriminator. For now it renders the gift
// directly when visible and a short message for each gate state; the reveal
// animation (Fase 6, PP-59+) and richer gate screens (PP-58) build on top.

type State =
  | 'loading'
  | 'visible'
  | 'not_yet_open'
  | 'expired'
  | 'already_opened'
  | 'notfound'
  | 'error';

interface PublicGift {
  title: string;
  message: string;
  pixel_art: unknown;
  reveal_type: string;
  reveal_config?: unknown;
}

interface PublicResponse {
  state: 'visible' | 'not_yet_open' | 'expired' | 'already_opened';
  gift?: PublicGift;
  scheduled_open_at?: string;
}

// tokenFromPath extracts {token} from a /g/{token} pathname ('' if the shape
// doesn't match, which is treated as "not found").
function tokenFromPath(): string {
  if (typeof window === 'undefined') return '';
  const parts = window.location.pathname.split('/').filter(Boolean);
  return parts.length >= 2 && parts[0] === 'g' ? decodeURIComponent(parts[1]) : '';
}

// The empty-cell colour from the theme-aware CSS variable, so a revealed drawing
// sits on the page background rather than a hard white block in dark mode.
function emptyColor(): string {
  const v = getComputedStyle(document.documentElement).getPropertyValue('--canvas-empty');
  return v.trim() || '#ffffff';
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: 'numeric' });
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

export default function GiftView() {
  const [state, setState] = useState<State>('loading');
  const [gift, setGift] = useState<PublicGift | null>(null);
  const [openAt, setOpenAt] = useState<string | null>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const token = tokenFromPath();
    if (!token) {
      setState('notfound');
      return;
    }

    fetch(`/api/g/${encodeURIComponent(token)}`, { headers: { Accept: 'application/json' } })
      .then((res) => {
        if (res.status === 404) {
          setState('notfound');
          return null;
        }
        if (!res.ok) {
          setState('error');
          return null;
        }
        return res.json() as Promise<PublicResponse>;
      })
      .then((data) => {
        if (!data) return;
        if (data.state === 'visible' && data.gift) {
          setGift(data.gift);
          setState('visible');
          return;
        }
        if (data.scheduled_open_at) setOpenAt(data.scheduled_open_at);
        setState(data.state);
      })
      .catch(() => setState('error'));
  }, []);

  // Render the pixel art once the gift is visible and its canvas is in the DOM.
  // Reuses the shared render at a smaller, gridless scale (the recipient's view).
  useEffect(() => {
    if (state !== 'visible' || gift === null) return;
    const el = canvasRef.current;
    if (el === null) return;
    const model = deserializeCanvas(gift.pixel_art);
    if (model === null) return;
    const cellSize = Math.max(1, Math.floor(360 / Math.max(model.width, model.height)));
    const ctx = sizeCanvas(el, model, cellSize);
    if (ctx) render(ctx, model, cellSize, { empty: emptyColor(), grid: '' }, false);
  }, [state, gift]);

  // Export the pixel art as a downloadable PNG (transparent background, no grid).
  // Rendered fresh from the model at an integer scale so pixels stay crisp.
  function downloadPNG() {
    if (gift === null) return;
    const model = deserializeCanvas(gift.pixel_art);
    if (model === null) return;
    const cellSize = Math.max(1, Math.round(640 / Math.max(model.width, model.height)));
    const url = toPNGDataURL(model, cellSize);
    if (!url) return;
    const a = document.createElement('a');
    a.href = url;
    a.download = fileName(gift.title);
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  }

  return (
    <main class="flex min-h-screen flex-col items-center justify-center gap-6 px-6 py-12">
      <div class="fixed top-4 right-4">
        <ThemeToggle />
      </div>

      {state === 'loading' && (
        <p class="text-slate-500 dark:text-slate-400">Cargando el regalo…</p>
      )}

      {state === 'visible' && gift && (
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
      )}

      {state === 'not_yet_open' && (
        <div class="max-w-md text-center">
          <p class="text-lg font-semibold text-slate-800 dark:text-slate-100">
            Este regalo aún no está disponible.
          </p>
          {openAt && (
            <p class="mt-2 text-sm text-slate-500 dark:text-slate-400">
              Podrás abrirlo a partir del {formatDate(openAt)}.
            </p>
          )}
        </div>
      )}

      {state === 'expired' && (
        <p class="max-w-md text-center text-lg font-semibold text-slate-800 dark:text-slate-100">
          Este regalo ha caducado.
        </p>
      )}

      {state === 'already_opened' && (
        <p class="max-w-md text-center text-lg font-semibold text-slate-800 dark:text-slate-100">
          Este regalo ya se abrió.
        </p>
      )}

      {state === 'notfound' && (
        <p class="max-w-md text-center text-lg font-semibold text-slate-800 dark:text-slate-100">
          No encontramos este regalo.
        </p>
      )}

      {state === 'error' && (
        <p class="max-w-md text-center text-rose-600 dark:text-rose-300" role="alert">
          No hemos podido cargar el regalo. Inténtalo de nuevo en un momento.
        </p>
      )}
    </main>
  );
}
