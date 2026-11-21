# SSR with slowjs (PoC)

Inertia SSR inside the Go process via [liustack/slowjs](https://github.com/liustack/slowjs). No Node SSR HTTP server.

## How it differs from `examples/ssr`

| | `examples/ssr` | `examples/ssr-slowjs` |
|---|---|---|
| SSR runtime | Node (`@inertiajs/react/server`) | slowjs (Pure Go, in-process) |
| Transport | HTTP to SSR server | `SsrGateway` → `renderPage(page)` |
| Entry | Vite SSR + `createServer` | thin `resources/js/ssr.tsx` IIFE |

## Run

```sh
make dev    # Vite for client assets; slowjs for SSR
# open http://localhost:8080
```

Production-style binary:

```sh
make run
```

## Layout

- `resources/js/ssr.tsx` — thin entry: `renderPage(page) -> { head, body }`
- `vite.ssr.config.ts` — builds a browser-style IIFE into `ssrbundle/ssr.js`
- `polyfill.js` — JS host stubs (`MessageChannel`, `FormData`, `TextEncoder`, …)
- `gateway.go` — `inertia.SsrGateway` + `slowjs/{url,intl,temporal}.Register` (no `slowjs/tifa`)

Rebuild the SSR bundle after changing pages:

```sh
make ssr-bundle
```
