import { useEffect, useState } from 'preact/hooks';

// Mirror of the API's giftSummary (the light list shape).
interface GiftSummary {
  id: string;
  title: string;
  reveal_type: string;
  view_token: string;
  scheduled_send_at?: string;
  sent_at?: string;
  single_open: boolean;
  opened_at?: string;
  created_at: string;
}

type State =
  | { kind: 'loading' }
  | { kind: 'error' }
  | { kind: 'ready'; gifts: GiftSummary[] };

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('es-ES', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

function statusLabel(gift: GiftSummary): string {
  if (gift.opened_at) return 'Abierto';
  if (gift.sent_at) return 'Enviado';
  if (gift.scheduled_send_at) return 'Programado';
  return 'Borrador';
}

export default function DashboardGifts() {
  const [state, setState] = useState<State>({ kind: 'loading' });

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await fetch('/api/gifts', { headers: { Accept: 'application/json' } });
        if (res.status === 401) {
          // Not logged in (or the session expired): send them to sign in.
          window.location.replace('/login');
          return;
        }
        if (!res.ok) {
          if (!cancelled) setState({ kind: 'error' });
          return;
        }
        const data = (await res.json()) as { gifts: GiftSummary[] };
        if (!cancelled) setState({ kind: 'ready', gifts: data.gifts ?? [] });
      } catch {
        if (!cancelled) setState({ kind: 'error' });
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  if (state.kind === 'loading') {
    return <p class="text-slate-400">Cargando tus regalos…</p>;
  }

  if (state.kind === 'error') {
    return <p class="text-rose-300">No hemos podido cargar tus regalos. Recarga la página.</p>;
  }

  if (state.gifts.length === 0) {
    return (
      <div class="rounded-xl border border-white/10 bg-white/5 p-10 text-center">
        <p class="text-lg text-slate-200">Aún no tienes regalos.</p>
        <p class="mt-2 text-slate-400">Crea el primero, píxel a píxel.</p>
        <a
          href="/editor"
          class="mt-6 inline-block rounded-md bg-amber-400 px-5 py-2.5 font-semibold text-slate-950 transition hover:bg-amber-300"
        >
          Crear un regalo
        </a>
      </div>
    );
  }

  return (
    <ul class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {state.gifts.map((gift) => (
        <li class="rounded-xl border border-white/10 bg-white/5 p-5">
          <div class="flex items-start justify-between gap-3">
            <h3 class="font-semibold text-slate-100">{gift.title}</h3>
            <span class="shrink-0 rounded-full bg-amber-400/10 px-2 py-0.5 font-mono text-xs text-amber-200">
              {gift.reveal_type}
            </span>
          </div>
          <p class="mt-2 text-xs text-slate-400">
            {statusLabel(gift)} · creado el {formatDate(gift.created_at)}
          </p>
          <div class="mt-4 flex items-center gap-4 text-sm">
            <a href={`/editor?id=${gift.id}`} class="font-medium text-amber-300 hover:text-amber-200">
              Editar
            </a>
            <a href={`/g/${gift.view_token}`} class="text-slate-300 hover:text-white">
              Ver regalo →
            </a>
          </div>
        </li>
      ))}
    </ul>
  );
}
