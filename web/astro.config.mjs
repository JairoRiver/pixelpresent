// @ts-check
import { defineConfig } from 'astro/config';

import tailwindcss from '@tailwindcss/vite';

// https://astro.build/config
export default defineConfig({
  // Static build: HTML/CSS/JS is generated at build time and embedded into the
  // Go binary (no Node in production).
  output: 'static',
  // Build straight into the Go embed package (internal/web/dist) so `task build`
  // can compile the binary with the static site embedded.
  outDir: '../internal/web/dist',
  vite: {
    plugins: [tailwindcss()],
    server: {
      // Dev only: the Astro dev server proxies the API to the local Go server so
      // the browser sees a single origin (no CORS). In production the same Go
      // binary serves both the static assets and the API under /api.
      proxy: {
        '/api': 'http://localhost:8080',
      },
    },
  },
});