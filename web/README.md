# Pixel Present — frontend (Astro)

Frontend del proyecto: sitio **estático** con [Astro](https://astro.build), islas
[Preact](https://preactjs.com) para la UI con estado (editor, dashboard, login,
revelación) y [Tailwind 4](https://tailwindcss.com) para estilos. La lógica de
canvas (render del editor, animaciones de revelación) vive en TS vanilla bajo
`src/lib/`.

El build (`dist/`) se **embebe en el binario Go** (`internal/web`), que en
producción sirve los estáticos y la API JSON en el mismo origen — no hay un
segundo proceso Node.

## Desarrollo

No arranques este paquete por separado: usa las tareas del repo raíz, que
levantan backend + frontend juntos (el dev server de Astro proxea `/api` a la
API Go). Ver [`../README.md`](../README.md).

```bash
task dev-front     # servidor de Astro en :4321 (proxy a la API en :8080)
```

El dev server corre en modo background (`astro dev --background`); gestiónalo con
`astro dev stop|status|logs`. Notas para agentes en [`AGENTS.md`](AGENTS.md).

## Estructura

```text
src/
  pages/         Rutas (index, login, dashboard, editor, g/ = revelación)
  components/    Islas Preact (Editor, DashboardGifts, RevealStage, ...)
  lib/           Canvas y animaciones en TS vanilla (canvas, reveal, confetti)
  layouts/       Layout base
  styles/        Estilos globales / Tailwind
public/          Assets estáticos
```
