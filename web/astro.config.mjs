// @ts-check
import { defineConfig } from 'astro/config';

import tailwindcss from '@tailwindcss/vite';

import preact from '@astrojs/preact';

// https://astro.build/config
export default defineConfig({
  // Static build: HTML/CSS/JS is generated at build time and embedded into the
  // Go binary (no Node in production).
  output: 'static',

  // Build straight into the Go embed package (internal/web/dist) so `task build`
  // can compile the binary with the static site embedded.
  outDir: '../internal/web/dist',

  vite: {
    plugins: [
      tailwindcss(),
      // Dev only: mirror the Go binary's /g/{token} catch-all. The static build
      // emits a single /g/index.html and the Go server serves it for every
      // tokenized reveal URL; the Astro dev server would 404 on /g/{token}, so
      // rewrite those requests to /g/ (the browser keeps the tokenized URL, which
      // the reveal island reads from the pathname).
      {
        name: 'pp-reveal-dev-rewrite',
        configureServer(server) {
          server.middlewares.use((req, _res, next) => {
            if (req.url && /^\/g\/[^/?#]+/.test(req.url)) req.url = '/g/';
            next();
          });
        },
      },
    ],
    server: {
      // Dev only: the Astro dev server proxies the API to the local Go server so
      // the browser sees a single origin (no CORS). In production the same Go
      // binary serves both the static assets and the API under /api.
      proxy: {
        '/api': 'http://localhost:8080',
      },
    },
  },

  integrations: [preact()],
});