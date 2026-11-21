# SSR bench: slowjs vs Node

Compares the two Inertia SSR shapes shipped in this repo:

| | `examples/ssr-slowjs` | `examples/ssr-node` |
|---|---|---|
| Runtime | slowjs in the Go process | Node SSR server |
| Transport | `renderPage(page)` | `SsrHTTPGateway` → `http://127.0.0.1:13714/render` |
| Bundle | thin IIFE (`.build/ssr/ssr.js`) | Vite `--ssr` (`.build/ssr/app.js`) |

Same Index props; architectures differ on purpose (production paths, not a synthetic shared bundle).

```sh
make bench        # -n 200
make bench-long   # -n 1000
```

Requires Node.js and `examples/ssr-node` dependencies (`npm install` there if needed).
