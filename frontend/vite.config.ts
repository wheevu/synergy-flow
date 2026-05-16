import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [
    react(),
    // Safari rejects <link crossorigin> for stylesheets (blank page).
    // Remove the attribute from stylesheet links while keeping it on module scripts.
    {
      name: 'safari-css-fix',
      enforce: 'post',
      transformIndexHtml(html) {
        return html.replace(
          /<link\s+rel="stylesheet"[^>]*crossorigin[^>]*>/gi,
          (match) => match.replace(/\s*crossorigin(=("[^"]*"|'[^']*'|[^\s>]+))?/gi, ''),
        );
      },
    },
  ],
  server: { port: 5173 },
});