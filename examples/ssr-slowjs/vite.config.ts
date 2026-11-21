import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// One config, two builds — same idea as examples/ssr's isSsrBuild branch.
// Official Node SSR uses `vite build --ssr` (isSsrBuild). This PoC cannot:
// slowjs needs a client-environment IIFE of ssr.tsx, not Vite's Node SSR graph.
// So the switch is `--mode ssr` instead of `--ssr`.
export default defineConfig(({ mode }) => {
  const isSsr = mode === 'ssr'

  return {
    plugins: [react()],
    publicDir: false,
    define: isSsr
      ? { 'process.env.NODE_ENV': JSON.stringify('production') }
      : undefined,
    resolve: isSsr
      ? {
          // Avoid react-dom/server.browser, which constructs MessageChannel at load time.
          alias: { 'react-dom/server': 'react-dom/server.edge' },
        }
      : undefined,
    build: {
      manifest: isSsr ? false : 'manifest.json',
      outDir: isSsr ? '.build/ssr' : 'resources/public/build',
      rollupOptions: isSsr
        ? {
            input: 'resources/js/ssr.tsx',
            output: {
              format: 'iife',
              entryFileNames: 'ssr.js',
              inlineDynamicImports: true,
              name: 'InertiaSlowSSR',
            },
          }
        : {
            input: ['resources/js/app.tsx'],
          },
    },
  }
})
