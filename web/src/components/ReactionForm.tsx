import { useState } from 'preact/hooks';

// ReactionForm is the recipient's response after the reveal (PP-65): a row of
// quick emoji plus an optional short message, posted to the public
// POST /g/{view_token}/reactions. It mirrors the API's limits (emoji up to 32
// bytes, message up to 500 chars) and maps its error codes to friendly copy.

const EMOJIS = ['❤️', '🎉', '😍', '🥹', '😮', '👏', '🔥', '✨'];
const MAX_MESSAGE = 500;

type Status = 'idle' | 'sending' | 'sent' | 'error';

type Payload = { kind: 'emoji'; emoji: string } | { kind: 'text'; message: string };

export default function ReactionForm({ viewToken }: { viewToken: string }) {
  const [status, setStatus] = useState<Status>('idle');
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  // Remembered only to echo the chosen emoji back in the thank-you note.
  const [lastEmoji, setLastEmoji] = useState<string | null>(null);

  async function send(payload: Payload) {
    setStatus('sending');
    setError('');
    try {
      const res = await fetch(`/api/g/${encodeURIComponent(viewToken)}/reactions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify(payload),
      });
      if (res.status === 201) {
        setStatus('sent');
        return;
      }
      if (res.status === 409 || res.status === 404) {
        setError('Este regalo ya no está disponible para reaccionar.');
      } else if (res.status === 400) {
        setError('Revisa tu reacción e inténtalo de nuevo.');
      } else {
        setError('No se pudo enviar. Inténtalo de nuevo.');
      }
      setStatus('error');
    } catch {
      setError('No se pudo enviar. Comprueba tu conexión.');
      setStatus('error');
    }
  }

  function sendEmoji(emoji: string) {
    setLastEmoji(emoji);
    send({ kind: 'emoji', emoji });
  }

  function submitText(event: Event) {
    event.preventDefault();
    const text = message.trim();
    if (text === '') return;
    send({ kind: 'text', message: text });
  }

  const sending = status === 'sending';

  if (status === 'sent') {
    return (
      <div class="flex w-full max-w-sm flex-col items-center gap-3 border-t border-slate-200 pt-6 text-center dark:border-white/10">
        <p class="text-base font-medium text-slate-700 dark:text-slate-200">
          ¡Gracias por tu reacción! {lastEmoji ?? '🎉'}
        </p>
        <button
          type="button"
          onClick={() => {
            setStatus('idle');
            setMessage('');
            setLastEmoji(null);
          }}
          class="text-sm font-medium text-amber-600 hover:text-amber-500 dark:text-amber-300 dark:hover:text-amber-200"
        >
          Enviar otra
        </button>
      </div>
    );
  }

  return (
    <div class="flex w-full max-w-sm flex-col items-center gap-4 border-t border-slate-200 pt-6 dark:border-white/10">
      <p class="text-sm font-medium text-slate-600 dark:text-slate-300">¿Qué te ha parecido?</p>

      <div class="flex flex-wrap justify-center gap-2" role="group" aria-label="Reacciones rápidas">
        {EMOJIS.map((emoji) => (
          <button
            key={emoji}
            type="button"
            disabled={sending}
            onClick={() => sendEmoji(emoji)}
            aria-label={`Reaccionar con ${emoji}`}
            class="flex h-10 w-10 items-center justify-center rounded-full border border-slate-200 text-xl transition hover:scale-110 hover:border-amber-400 disabled:cursor-not-allowed disabled:opacity-50 dark:border-white/15 dark:hover:border-amber-300"
          >
            {emoji}
          </button>
        ))}
      </div>

      <form onSubmit={submitText} class="flex w-full flex-col items-center gap-2">
        <textarea
          value={message}
          maxLength={MAX_MESSAGE}
          rows={2}
          disabled={sending}
          onInput={(event) => setMessage((event.target as HTMLTextAreaElement).value)}
          placeholder="O escribe un mensaje…"
          aria-label="Mensaje de reacción"
          class="w-full resize-none rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-800 placeholder:text-slate-400 focus:border-amber-400 focus:outline-none disabled:opacity-60 dark:border-white/15 dark:bg-white/5 dark:text-slate-100 dark:placeholder:text-slate-500"
        />
        <button
          type="submit"
          disabled={sending || message.trim() === ''}
          class="rounded-md bg-amber-500 px-4 py-1.5 text-sm font-medium text-white transition hover:bg-amber-400 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-amber-400 dark:text-slate-900 dark:hover:bg-amber-300"
        >
          {sending ? 'Enviando…' : 'Enviar'}
        </button>
      </form>

      {status === 'error' && error && (
        <p class="text-sm text-rose-600 dark:text-rose-300" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}
