import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Client build only. Same shape as examples/helloworld-typescript:
// no @inertiajs/vite — official Node SSR is not used here (slowjs handles SSR).
export default defineConfig({
  plugins: [react()],
  publicDir: false,
  build: {
    manifest: 'manifest.json',
    outDir: 'resources/public/build',
    rollupOptions: {
      input: ['resources/js/app.tsx'],
    },
  },
})
