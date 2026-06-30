import { useEffect, useState } from 'preact/hooks';
import { isLoggedIn } from '../lib/session';

type Status = 'checking' | 'idle' | 'sending' | 'sent' | 'error';

// MagicLinkForm requests a passwordless login link. The same form both signs in
// and registers: POST /api/auth/magic-link creates the account if the email is
// new, and always answers 202, so the success copy is identical either way.
export default function MagicLinkForm() {
  const [email, setEmail] = useState('');
  const [status, setStatus] = useState<Status>('checking');
  const [error, setError] = useState('');

  // If there is already a valid session, skip the form and go to the dashboard.
  useEffect(() => {
    isLoggedIn().then((authed) => {
      if (authed) {
        window.location.replace('/dashboard');
      } else {
        setStatus('idle');
      }
    });
  }, []);

  async function submit() {
    setStatus('sending');
    setError('');
    try {
      const res = await fetch('/api/auth/magic-link', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: email.trim() }),
      });
      if (res.status === 202) {
        setStatus('sent');
        return;
      }
      setStatus('error');
      setError(
        res.status === 400
          ? 'Ese email no es válido. Revísalo e inténtalo de nuevo.'
          : 'Algo ha ido mal. Inténtalo de nuevo en un momento.',
      );
    } catch {
      setStatus('error');
      setError('No hemos podido conectar. Comprueba tu conexión.');
    }
  }

  if (status === 'checking') {
    return <p class="text-slate-400">Cargando…</p>;
  }

  if (status === 'sent') {
    return (
      <p class="text-amber-200" role="status">
        Te hemos enviado un enlace a <strong>{email.trim()}</strong>. Revisa tu correo para entrar.
      </p>
    );
  }

  return (
    <form
      class="space-y-4"
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
    >
      <div>
        <label for="email" class="block text-sm font-medium text-slate-300">Email</label>
        <input
          id="email"
          name="email"
          type="email"
          required
          autocomplete="email"
          placeholder="tu@email.com"
          value={email}
          onInput={(event) => setEmail(event.currentTarget.value)}
          class="mt-2 w-full rounded-md border border-white/15 bg-white/5 px-4 py-3 text-slate-100 placeholder:text-slate-500 focus:border-amber-400 focus:ring-1 focus:ring-amber-400 focus:outline-none"
        />
      </div>
      <button
        type="submit"
        disabled={status === 'sending'}
        class="w-full rounded-md bg-amber-400 px-6 py-3 font-semibold text-slate-950 transition hover:bg-amber-300 disabled:opacity-60"
      >
        {status === 'sending' ? 'Enviando…' : 'Enviar enlace de acceso'}
      </button>
      {status === 'error' && (
        <p class="text-sm text-rose-300" role="alert">{error}</p>
      )}
    </form>
  );
}
