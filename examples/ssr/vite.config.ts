import inertia from '@inertiajs/vite'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ isSsrBuild }) => {
  return {
    plugins: [inertia({ ssr: 'resources/js/app.tsx' }), react()],
    publicDir: false,
    build: {
      manifest: isSsrBuild ? false : "manifest.json",
      outDir: isSsrBuild ? ".build/ssr" : "resources/public/build",
      rollupOptions: {
        input: ['resources/js/app.tsx'],
      },
    },
  }
})
