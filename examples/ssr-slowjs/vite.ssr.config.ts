import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Browser-style IIFE bundle that slowjs can Eval in-process.
// No Node builtins, no createServer — only globalThis.renderPage.
export default defineConfig({
  plugins: [react()],
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
  resolve: {
    // Avoid react-dom/server.browser, which constructs MessageChannel at load time.
    alias: {
      'react-dom/server': 'react-dom/server.edge',
    },
  },
  build: {
    outDir: 'ssrbundle',
    emptyOutDir: true,
    minify: false,
    rollupOptions: {
      input: 'resources/js/ssr.tsx',
      output: {
        format: 'iife',
        entryFileNames: 'ssr.js',
        inlineDynamicImports: true,
        name: 'InertiaSlowSSR',
      },
    },
  },
})
