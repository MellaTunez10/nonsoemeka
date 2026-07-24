import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 5173,
    // DEV-ONLY PROXY: forwards /api/* → Go API on localhost:8080 during `npm run dev`.
    // In production the nginx container handles /api reverse-proxying at the same origin.
    // For cross-domain deployments (e.g. Vercel + fly.io), set VITE_API_BASE_URL in the
    // build environment instead of relying on this proxy.
    proxy: {
      '/api': {
        target: process.env.VITE_DEV_API_TARGET ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
