# SSR with Node

Official-style Inertia SSR: a Node SSR server, reached from Go via [`ssr/nodejs`](../../ssr/nodejs).

```go
import ssrnodejs "github.com/kohkimakimoto/inertia-echo/v5/ssr/nodejs"

gateway := ssrnodejs.NewGateway()
renderer.SsrGateway = gateway
```

For in-process SSR without Node, see [`examples/ssr-slowjs`](../ssr-slowjs).

## Run

```sh
make dev    # Vite (includes SSR in hot mode)
# open http://localhost:8080
```

Production-style (Go app + `npm run start-ssr`):

```sh
make run
```
