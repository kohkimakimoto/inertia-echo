import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(({ command, mode }) => ({
  plugins: [react()],
  publicDir: false,
  build: {
    manifest: true,
    outDir: 'gen/dist',
    rollupOptions: {
      input: 'src/main.tsx',
    },
  },
  resolve: {
    alias: {
      '@': '/src',
    },
  },
}));
