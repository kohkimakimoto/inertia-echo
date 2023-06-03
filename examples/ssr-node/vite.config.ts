import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig(({ command, mode, ssrBuild }) => {
  return {
    plugins: [react()],
    publicDir: false,
    build: {
      manifest: ssrBuild ? false : "manifest.json",
      outDir: ssrBuild ? ".generated/ssr" : "public/dist",
      rollupOptions: {
        input: ssrBuild ? ['js/ssr.tsx'] : ['js/app.tsx'],
      }
    }
  }
})
