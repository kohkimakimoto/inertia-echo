import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  publicDir: false,
  build: {
    manifest: 'manifest.json',
    outDir: 'resources/public/build',
    rollupOptions: { input: ['resources/js/app.tsx'] },
  },
})
