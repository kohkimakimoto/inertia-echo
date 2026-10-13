# Notes

A personal notes application demonstrating inertia-echo with React, TypeScript,
Vite, and Tailwind CSS v4. Data is stored in JSON files; no database or SSR is used.

## Run

Requires Go 1.25+, Node.js 20.19+ (or 22.12+), and npm.

```sh
cd examples/notes
make dev
```

Open <http://localhost:8080>. The Go process starts Vite automatically.

| Email | Password |
| --- | --- |
| demo@example.com | password |
| alice@example.com | password |

The demo account has 32 seeded notes for infinite scrolling. Alice has a separate
notebook. Passwords and session secrets are intentionally simple demo values.

```sh
make build
make run
```

Production embeds the compiled assets and views and does not start Vite. Run the
binary from `examples/notes`: it initializes missing `data/*.json` files from
the committed `seed/*.json` files, then reads and writes only `data/` relative
to the working directory. Existing data is never overwritten by initialization.
`data/` is ignored by Git; `seed/` remains unchanged during use.
Keep the working directory writable. Run one application process per data
directory; the mutex coordinates writes within a process only.

`make clean` removes compiled assets and the binary, preserving JSON data.
For a reproducible dependency install, use `npm ci`. To reset demo data, stop
the application and remove `data/`; the next start copies the seeds again.
Back up `data/` to preserve your notes. Distribute `seed/` alongside the binary
when running it from a new directory.

## Feature walkthrough

| UI action | inertia-echo feature |
| --- | --- |
| Log in and navigate | Session authentication; `Share` exposes the current user and theme |
| See the application name | `ShareOnce` caches the app name |
| Log in or log out | `ClearHistory`; all pages use `EncryptHistoryMiddleware` |
| Submit any mutation | `CSRF`; `Flash` displays feedback after redirects |
| Open Dashboard | Separate `DeferWithGroup` requests for statistics and recent notes |
| Select “Try deferred rescue” | `Rescue` keeps the page usable; `RescueReporter` logs the intentional failure |
| Pin a recent note on Dashboard | Partial reload of `recent`; `Merge(...).MatchOn("id")` updates rows without duplicates |
| Scroll through Notes | `Scroll` metadata and React `InfiniteScroll` append more notes |
| Search or filter Notes | Partial reload resets the scroll collection; `Always` supplies filters |
| Open Settings, navigate away, and return | `Once` caches per-user settings for ten minutes |
| Save Settings | Persist JSON and refresh the once prop on the redirected GET |
| Open a note, jump to `#body`, and choose “Open settings with fragment” | `PreserveFragment` keeps `#body` through the redirect |
| Visit `/notes/missing` or an unknown path | Inertia error page with HTTP 404 |
| Create, edit, delete, or pin a note | Per-user JSON persistence via temporary file and rename |

Merge is demonstrated on Dashboard instead of the infinite scroll collection so
pinning does not interfere with pagination. Its partial reload preserves the
existing recent rows and matches returned rows by ID. Reload Dashboard to rebuild
the sorted five-note snapshot after pinning.

## Checks

```sh
go test -race ./...
go vet ./...
make build
```

The store tests cover persistence after reopening, isolation between users,
filtering, pagination, tag normalization, settings, and failed writes. HTTP tests
cover the application routes and Inertia response contracts.
