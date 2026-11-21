# SSR bench: slowjs vs Node

Compares the two Inertia SSR shapes shipped in this repo:

| | `examples/ssr-slowjs` | `examples/ssr-node` |
|---|---|---|
| Runtime | [`ssr/slowjs`](../../ssr/slowjs) (in-process) | [`ssr/nodejs`](../../ssr/nodejs) + Node SSR server |
| Transport | `renderPage(page)` | HTTP → `http://127.0.0.1:13714/render` |
| Bundle | thin IIFE (`.build/ssr/ssr.js`) | Vite `--ssr` (`.build/ssr/app.js`) |

Same Index props; architectures differ on purpose (production paths, not a synthetic shared bundle).

Uses core packages `ssr/slowjs` and `ssr/nodejs`.

## Modes

| Mode | What it measures | slowjs pool default |
|---|---|---|
| `latency` | sequential Dispatch latency + RSS | `1` (single warm VM) |
| `throughput` | concurrent workers → wall-clock RPS | `0` → library `GOMAXPROCS` |
| `both` | runs latency then throughput | per mode above |

```sh
make bench              # latency -n 200 -pool 1
make bench-long         # latency -n 1000 -pool 1
make bench-throughput   # throughput -n 5000, workers=GOMAXPROCS, pool=GOMAXPROCS
make bench-all          # both modes -n 1000

go run . -mode throughput -n 5000 -workers 8 -pool 4
```

Requires Node.js and `examples/ssr-node` dependencies (`npm install` there if needed).
