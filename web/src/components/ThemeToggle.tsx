import { useEffect, useState } from 'preact/hooks';
import { MoonIcon, SunIcon } from './icons';

// ThemeToggle flips the app between light and dark (PP-44.5). The active theme
// lives as a `.dark` class on <html>, applied before paint by the inline script
// in Layout.astro; this island reads that initial state on mount, and on click
// flips the class and persists the choice in localStorage (`pp_theme`). Other
// theme-aware pieces (the editor canvas) observe the class rather than coupling
// to this component.
export default function ThemeToggle(props: { class?: string }) {
  const [dark, setDark] = useState(false);

  // Sync with the class the no-flash script already set (avoids a wrong initial
  // icon during hydration).
  useEffect(() => {
    setDark(document.documentElement.classList.contains('dark'));
  }, []);

  function toggle() {
    const next = !dark;
    document.documentElement.classList.toggle('dark', next);
    localStorage.setItem('pp_theme', next ? 'dark' : 'light');
    setDark(next);
  }

  return (
    <button
      type="button"
      onClick={toggle}
      aria-label={dark ? 'Cambiar a tema claro' : 'Cambiar a tema oscuro'}
      title={dark ? 'Tema claro' : 'Tema oscuro'}
      class={`inline-flex h-9 w-9 items-center justify-center rounded-md border border-slate-200 text-slate-600 transition hover:border-slate-300 hover:text-slate-900 dark:border-white/15 dark:text-slate-300 dark:hover:border-white/30 dark:hover:text-white ${props.class ?? ''}`}
    >
      {dark ? <SunIcon class="h-4 w-4" /> : <MoonIcon class="h-4 w-4" />}
    </button>
  );
}
