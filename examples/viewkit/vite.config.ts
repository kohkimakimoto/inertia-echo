import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig(({ command, mode, isSsrBuild }) => {
  return {
    plugins: [react()],
    publicDir: false,
    build: {
      manifest: isSsrBuild ? false : "manifest.json",
      outDir: isSsrBuild ? ".build/ssr" : "public/build",
      rollupOptions: {
        input: isSsrBuild ? ['assets/ssr.tsx'] : ['assets/app.tsx'],
      },
    },
  }
})
