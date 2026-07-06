import type { JSX } from 'preact';
import { useEffect, useState } from 'preact/hooks';
import { deserializeCanvas } from '../lib/canvas';
import { CalendarXIcon, ClockIcon, GiftIcon, MailOpenIcon } from './icons';
import RevealStage from './RevealStage';
import ThemeToggle from './ThemeToggle';

// GiftView is the public, recipient-facing page (PP-57): it reads the view token
// from the /g/{token} URL and consumes GET /api/g/{view_token}, which applies the
// visibility gate and returns a state discriminator. When the gift is visible it
// hands off to RevealStage (PP-60) for the idle → reveal cycle; each non-visible
// outcome gets its own dedicated gate screen (PP-58).

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

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '';
  return d.toLocaleDateString(undefined, { day: 'numeric', month: 'long', year: 'numeric' });
}

// GateScreen is the shared layout for every non-visible outcome (PP-58): an icon
// in a tinted disc, a headline and a supporting line. `tone` picks the accent so
// each state reads differently at a glance — neutral slate for a normal "not yet"
// or "already opened", a warmer amber for the terminal "expired".
type Tone = 'neutral' | 'warn';

function GateScreen({
  icon,
  title,
  children,
  tone = 'neutral',
}: {
  icon: JSX.Element;
  title: string;
  children?: JSX.Element | string;
  tone?: Tone;
}) {
  const disc =
    tone === 'warn'
      ? 'bg-amber-100 text-amber-600 dark:bg-amber-500/15 dark:text-amber-300'
      : 'bg-slate-100 text-slate-500 dark:bg-white/10 dark:text-slate-300';
  return (
    <div class="flex max-w-sm flex-col items-center gap-4 text-center">
      <span class={`flex h-16 w-16 items-center justify-center rounded-full ${disc}`}>
        {icon}
      </span>
      <h1 class="text-xl font-semibold text-slate-800 dark:text-slate-100">{title}</h1>
      {children && (
        <p class="text-sm leading-relaxed text-slate-500 dark:text-slate-400">{children}</p>
      )}
    </div>
  );
}

export default function GiftView() {
  const [state, setState] = useState<State>('loading');
  const [gift, setGift] = useState<PublicGift | null>(null);
  const [openAt, setOpenAt] = useState<string | null>(null);

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

  // The visible gift's drawing, deserialized for RevealStage. null if the stored
  // pixel_art is malformed — surfaced as an error rather than a broken reveal.
  const model = gift ? deserializeCanvas(gift.pixel_art) : null;

  return (
    <main class="flex min-h-screen flex-col items-center justify-center gap-6 px-6 py-12">
      <div class="fixed top-4 right-4">
        <ThemeToggle />
      </div>

      {state === 'loading' && (
        <p class="text-slate-500 dark:text-slate-400">Cargando el regalo…</p>
      )}

      {/* Visible: hand off the whole idle → reveal → revealed cycle to
          RevealStage (PP-60). A malformed drawing falls through to an error. */}
      {state === 'visible' && gift && model && (
        <RevealStage
          viewToken={tokenFromPath()}
          gift={{
            title: gift.title,
            message: gift.message,
            pixelArt: model,
            revealType: gift.reveal_type,
            revealConfig: gift.reveal_config,
          }}
        />
      )}

      {state === 'visible' && gift && !model && (
        <p class="max-w-md text-center text-rose-600 dark:text-rose-300" role="alert">
          No hemos podido mostrar este regalo. Inténtalo de nuevo en un momento.
        </p>
      )}

      {state === 'not_yet_open' && (
        <GateScreen icon={<ClockIcon class="h-8 w-8" />} title="Este regalo aún no está listo">
          {openAt
            ? `Vuelve el ${formatDate(openAt)} para abrirlo. Alguien lo preparó para que llegue en su momento.`
            : 'Todavía no ha llegado el momento de abrirlo. Vuelve un poco más tarde.'}
        </GateScreen>
      )}

      {state === 'expired' && (
        <GateScreen
          icon={<CalendarXIcon class="h-8 w-8" />}
          title="Este regalo ha caducado"
          tone="warn"
        >
          El plazo para abrirlo ya pasó y su contenido dejó de estar disponible.
        </GateScreen>
      )}

      {state === 'already_opened' && (
        <GateScreen icon={<MailOpenIcon class="h-8 w-8" />} title="Este regalo ya se abrió">
          Era un regalo de un solo uso y ya se disfrutó una vez. Ese momento fue único.
        </GateScreen>
      )}

      {state === 'notfound' && (
        <GateScreen icon={<GiftIcon class="h-8 w-8" />} title="No encontramos este regalo">
          El enlace puede estar incompleto o el regalo ya no existe. Comprueba que lo copiaste
          entero.
        </GateScreen>
      )}

      {state === 'error' && (
        <p class="max-w-md text-center text-rose-600 dark:text-rose-300" role="alert">
          No hemos podido cargar el regalo. Inténtalo de nuevo en un momento.
        </p>
      )}
    </main>
  );
}
