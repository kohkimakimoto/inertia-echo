# SSR with Node

Official-style Inertia SSR: a Node SSR server, reached from Go via [`SsrHTTPGateway`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#SsrHTTPGateway).

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
