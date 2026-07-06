import { useState } from 'preact/hooks';

// GiftReactions lets a creator read the reactions left on one of their gifts. It
// lazy-loads GET /gifts/{id}/reactions the first time it is opened (so the
// dashboard doesn't fan out a request per card up front) and shows the emoji
// reactions as a row plus any written messages as dated notes.

interface Reaction {
  id: string;
  kind: string;
  emoji?: string;
  message?: string;
  created_at: string;
}

type State =
  | { kind: 'closed' }
  | { kind: 'loading' }
  | { kind: 'error' }
  | { kind: 'loaded'; reactions: Reaction[] };

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('es-ES', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  });
}

export default function GiftReactions({ giftId }: { giftId: string }) {
  const [state, setState] = useState<State>({ kind: 'closed' });

  async function toggle() {
    if (state.kind !== 'closed') {
      setState({ kind: 'closed' });
      return;
    }
    setState({ kind: 'loading' });
    try {
      const res = await fetch(`/api/gifts/${giftId}/reactions`, {
        headers: { Accept: 'application/json' },
      });
      if (res.status === 401) {
        window.location.replace('/login');
        return;
      }
      if (!res.ok) {
        setState({ kind: 'error' });
        return;
      }
      const data = (await res.json()) as { reactions: Reaction[] };
      setState({ kind: 'loaded', reactions: data.reactions ?? [] });
    } catch {
      setState({ kind: 'error' });
    }
  }

  const open = state.kind !== 'closed';
  const emojis = state.kind === 'loaded' ? state.reactions.filter((r) => r.kind === 'emoji') : [];
  const messages = state.kind === 'loaded' ? state.reactions.filter((r) => r.kind === 'text') : [];

  return (
    <div class="mt-3 border-t border-slate-200 pt-3 dark:border-white/10">
      <button
        type="button"
        onClick={toggle}
        aria-expanded={open}
        class="text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white"
      >
        {open ? 'Ocultar reacciones' : 'Ver reacciones'}
      </button>

      {state.kind === 'loading' && (
        <p class="mt-2 text-xs text-slate-500 dark:text-slate-400">Cargando reacciones…</p>
      )}

      {state.kind === 'error' && (
        <p class="mt-2 text-xs text-rose-600 dark:text-rose-300" role="alert">
          No se pudieron cargar las reacciones.
        </p>
      )}

      {state.kind === 'loaded' && state.reactions.length === 0 && (
        <p class="mt-2 text-xs text-slate-500 dark:text-slate-400">Sin reacciones aún.</p>
      )}

      {state.kind === 'loaded' && state.reactions.length > 0 && (
        <div class="mt-3 flex flex-col gap-3">
          {emojis.length > 0 && (
            <div class="flex flex-wrap gap-1.5" aria-label="Reacciones con emoji">
              {emojis.map((r) => (
                <span key={r.id} title={formatDate(r.created_at)} class="text-xl">
                  {r.emoji}
                </span>
              ))}
            </div>
          )}
          {messages.length > 0 && (
            <ul class="flex flex-col gap-2">
              {messages.map((r) => (
                <li
                  key={r.id}
                  class="rounded-lg bg-white px-3 py-2 text-sm text-slate-700 dark:bg-white/5 dark:text-slate-200"
                >
                  <p class="whitespace-pre-wrap">{r.message}</p>
                  <p class="mt-1 text-xs text-slate-400 dark:text-slate-500">
                    {formatDate(r.created_at)}
                  </p>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
