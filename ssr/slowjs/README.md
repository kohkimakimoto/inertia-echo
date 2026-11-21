# ssr/slowjs

In-process Inertia SSR gateway backed by [liustack/slowjs](https://github.com/liustack/slowjs).

```go
import (
	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	ssrslowjs "github.com/kohkimakimoto/inertia-echo/v5/ssr/slowjs"
)

gateway, err := ssrslowjs.New(".build/ssr/ssr.js")
if err != nil {
	return err
}
defer gateway.Close()
renderer.SsrGateway = gateway
```

### VM pool

A slowjs `VM` is not safe for concurrent goroutines. The gateway keeps a **fixed pool** of warm VMs and lends one per `Dispatch` (buffered channel acquire/release).

| `Config.PoolSize` | Behaviour |
|---|---|
| `1` | Same concurrency as a single mutex-guarded VM (serial SSR) |
| unset / `0` | Default `max(1, GOMAXPROCS)` — good starting point for CPU-bound SSR |
| `N` | Up to `N` concurrent renders; memory ≈ `N ×` one VM |

```go
gateway, err := ssrslowjs.New(".build/ssr/ssr.js", ssrslowjs.Config{
	PoolSize: 4, // or 1 for lowest memory
})
```

Tune with load tests: raise while SSR CPU is saturated and RSS stays acceptable; lower if memory bound.

## Frontend: thin entry + Vite IIFE

Unlike the official Node path (`vite build --ssr`), slowjs needs a **client-environment IIFE** of a thin entry that exposes `globalThis.renderPage(page) -> { head, body }`.

Build with `vite build --mode ssr` (not `--ssr`). Full example: [`examples/ssr-slowjs`](../../examples/ssr-slowjs).

```ts
// vite.config.ts (excerpt)
export default defineConfig(({ mode }) => {
  const isSsr = mode === 'ssr'
  return {
    plugins: [react()],
    define: isSsr ? { 'process.env.NODE_ENV': JSON.stringify('production') } : undefined,
    resolve: isSsr
      ? { alias: { 'react-dom/server': 'react-dom/server.edge' } }
      : undefined,
    build: {
      outDir: isSsr ? '.build/ssr' : 'resources/public/build',
      chunkSizeWarningLimit: isSsr ? 2000 : undefined,
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
        : { input: ['resources/js/app.tsx'] },
    },
  }
})
```
