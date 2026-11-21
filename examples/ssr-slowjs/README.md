# SSR with slowjs

Inertia SSR inside the Go process via [`ssr/slowjs`](../../ssr/slowjs) ([liustack/slowjs](https://github.com/liustack/slowjs)). No Node SSR HTTP server.

```go
import ssrslowjs "github.com/kohkimakimoto/inertia-echo/v5/ssr/slowjs"

gateway, err := ssrslowjs.New(".build/ssr/ssr.js") // PoolSize defaults to GOMAXPROCS
// gateway, err := ssrslowjs.New(path, ssrslowjs.Config{PoolSize: 1})
renderer.SsrGateway = gateway
defer gateway.Close()
```

See [`ssr/slowjs/README.md`](../../ssr/slowjs/README.md) for the VM pool and Vite IIFE setup.

## How it differs from `examples/ssr-node`

| | `examples/ssr-node` | `examples/ssr-slowjs` |
|---|---|---|
| SSR runtime | Node (`@inertiajs/react` + Vite SSR) | slowjs (Pure Go, in-process) |
| Go package | `ssr/nodejs` | `ssr/slowjs` |
| Transport | HTTP to SSR server | `renderPage(page)` in-process |
| Entry | Vite `--ssr` | thin `resources/js/ssr.tsx` IIFE (`vite build --mode ssr`) |

## Run

```sh
make dev
# open http://localhost:8080
```

```sh
make run          # production-style binary
make ssr-bundle   # rebuild SSR IIFE after page changes
```
