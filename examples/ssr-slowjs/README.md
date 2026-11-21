# SSR with slowjs (PoC)

Inertia SSR inside the Go process via [liustack/slowjs](https://github.com/liustack/slowjs). No Node SSR HTTP server.

## How it differs from `examples/ssr-node`

| | `examples/ssr-node` | `examples/ssr-slowjs` |
|---|---|---|
| SSR runtime | Node (`@inertiajs/react` + Vite SSR) | slowjs (Pure Go, in-process) |
| Transport | HTTP to SSR server (`SsrHTTPGateway`) | `SsrGateway` → `renderPage(page)` |
| Entry | Vite `--ssr` + Inertia SSR server | thin `resources/js/ssr.tsx` IIFE |

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
- `vite.config.ts` — client build + `vite build --mode ssr` → `.build/ssr/ssr.js`
- `polyfill.js` — JS host stubs (`MessageChannel`, `FormData`, `TextEncoder`, …)
- `gateway.go` — `inertia.SsrGateway` + `slowjs/{url,intl,temporal}.Register` (no `slowjs/tifa`)

Rebuild the SSR bundle after changing pages:

```sh
make ssr-bundle
```
