# Results

Isolated processes, Index page, 50 warmup + 1000 timed renders.

## Author's machine

| | |
|---|---|
| Host | Author's development machine |
| OS | macOS 26.6.2 (Build 25G83), Darwin 25.6.0 |
| CPU | Apple M1 Max (arm64), 10 cores |
| Memory | 32 GiB |
| Go | go1.27.3 (`darwin/arm64`) |
| Node | v25.9.0 |

## Numbers

| Engine | init | Go HeapInuse | Go MaxRSS | Node RSS | footprint≈ | avg | p50 | p95 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| **slowjs** | 93ms | 1.7 MiB | **28.2 MiB** | — | **28.2 MiB** | **84µs** | 76µs | 87µs |
| node | 778ms | 1.6 MiB | 18.9 MiB | 107.8 MiB | 126.7 MiB | 276µs | 316µs | 418µs |

- slowjs avg render ≈ **3.3× faster** than Node HTTP SSR (localhost).
- Memory footprint ≈ **4.5× smaller** for slowjs (one process vs Go harness + Node).
- Architectures differ on purpose: thin in-process IIFE vs official Vite `--ssr` Node server.

## Recommendation

Prefer **slowjs** for in-process Inertia React SSR without a Node sidecar. Keep **ssr-node** for the official Node SSR path.
