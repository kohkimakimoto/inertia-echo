import inertia from '@inertiajs/vite'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Client (browser) build only. SSR for slowjs uses vite.ssr.config.ts.
export default defineConfig({
  // ssr: false — this example uses slowjs in-process, not Vite/Node SSR.
  plugins: [inertia({ ssr: false }), react()],
  publicDir: false,
  build: {
    manifest: 'manifest.json',
    outDir: 'resources/public/build',
    rollupOptions: {
      input: ['resources/js/app.tsx'],
    },
  },
})
