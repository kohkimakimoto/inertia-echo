# Results

Uses core packages `ssr/slowjs` and `ssr/nodejs`.

## Author's machine

| | |
|---|---|
| Host | Author's development machine |
| OS | macOS 26.6.2 (Build 25G83), Darwin 25.6.0 |
| CPU | Apple M1 Max (arm64), 10 cores |
| Memory | 32 GiB |
| Go | go1.27.3 (`darwin/arm64`), GOMAXPROCS=10 |
| Node | v25.9.0 |

## Latency (sequential)

`make bench-long` — PoolSize=1, 50 warmup + 1000 timed renders, single-threaded Dispatch.

| Engine | pool | init | footprint≈ | avg | p50 | p95 |
|---|---:|---:|---:|---:|---:|---:|
| **slowjs** | 1 | 93ms | **27.9 MiB** | **82µs** | 75µs | 81µs |
| node | — | 827ms | 126.3 MiB | 277µs | 308µs | 417µs |

- slowjs avg ≈ **3.4×** faster than Node HTTP SSR.
- Memory footprint ≈ **4.5×** smaller (one process vs Go harness + Node).

## Throughput (concurrent)

`make bench-throughput` — 10 workers (=GOMAXPROCS), slowjs PoolSize=10, 50 warmup + 5000 timed renders.

| Engine | pool | workers | footprint≈ | wall | rps | avg (under load) | p50 |
|---|---:|---:|---:|---:|---:|---:|---:|
| **slowjs** | 10 | 10 | **93.3 MiB** | 62ms | **~80k** | 124µs | 84µs |
| node | — | 10 | 128.8 MiB | 455ms | ~11k | 908µs | 617µs |

- slowjs ≈ **7.3×** higher RPS at the same worker count.
- Pool of 10 VMs raises slowjs RSS (~93 MiB) vs single-VM latency run (~28 MiB); still under Node’s combined footprint.

## Recommendation

Prefer **slowjs** for in-process Inertia React SSR without a Node sidecar. Keep **ssr-node** for the official Node SSR path.
