import { useEffect, useState } from 'preact/hooks';

// AuthError surfaces a failed/expired magic link. The backend redirects failed
// verifications to `/?auth_error=invalid_or_expired_link`; this island reads that
// query param, shows a message, and cleans it from the URL so a refresh or a
// shared link does not keep showing the banner.
export default function AuthError() {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (!params.get('auth_error')) return;
    setVisible(true);
    params.delete('auth_error');
    const query = params.toString();
    window.history.replaceState({}, '', window.location.pathname + (query ? `?${query}` : ''));
  }, []);

  if (!visible) return null;

  return (
    <div
      role="alert"
      class="mx-auto mt-4 flex max-w-2xl items-center justify-between gap-4 rounded-lg border border-rose-400/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-200"
    >
      <span>
        Ese enlace no es válido o ha caducado.{' '}
        <a href="/login" class="font-semibold underline">Pídelo de nuevo</a>.
      </span>
      <button
        type="button"
        aria-label="Cerrar"
        onClick={() => setVisible(false)}
        class="shrink-0 text-rose-300 transition hover:text-white"
      >
        ✕
      </button>
    </div>
  );
}
