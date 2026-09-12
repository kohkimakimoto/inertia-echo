import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  publicDir: false,
  build: {
    manifest: "manifest.json",
    outDir: "resources/public/build",
    rollupOptions: {
      input: ['resources/js/app.tsx'],
    },
  },
})
