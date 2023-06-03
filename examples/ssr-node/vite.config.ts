import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  publicDir: false,
  build: {
    manifest: "manifest.json",
    outDir: "public/dist",
    rollupOptions: {
      input: ['js/app.tsx'],
    },
  },
})
