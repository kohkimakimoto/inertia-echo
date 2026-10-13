# Inertia Echo

[![test](https://github.com/kohkimakimoto/inertia-echo/actions/workflows/test.yml/badge.svg?branch=master)](https://github.com/kohkimakimoto/inertia-echo/actions/workflows/test.yml?query=branch%3Amaster)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/kohkimakimoto/inertia-echo/v5.svg)](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5)

This is the [Inertia.js](https://inertiajs.com) server-side adapter for [Echo](https://echo.labstack.com/) Go web framework.

[Inertia.js](https://inertiajs.com) lets you build server-driven single-page apps without an API layer.
The official server-side adapter is [inertia-laravel](https://github.com/inertiajs/inertia-laravel); this package brings the same protocol to Echo.
This README assumes you already know [how Inertia works](https://inertiajs.com/docs/v3/core-concepts/how-it-works) and are familiar with Echo.

> [!NOTE]
> Inertia Echo v5.1.0+ supports Echo v5 and Inertia.js v3. v5.0.0 is the final release for Inertia.js v2.

This README follows the same categories as the [Inertia.js v3 docs](https://inertiajs.com/docs/v3/getting-started/index) where the topic is server-side.
Client-only topics (links, forms, layouts, prefetching, and so on) stay in the official docs — use this guide for how to express those features from Go with inertia-echo.

Table of Contents

- [Getting started](#getting-started)
  - [Installation](#installation)
  - [Root template](#root-template)
  - [Write Go code](#write-go-code)
  - [Setup frontend](#setup-frontend)
  - [Run the application](#run-the-application)
  - [Run in Dev mode](#run-in-dev-mode)
  - [Embed assets into a single binary](#embed-assets-into-a-single-binary)
  - [Next steps](#next-steps)
- [Architecture](#architecture)
  - [Middleware](#middleware)
  - [Renderer](#renderer)
  - [StateStore](#statestore)
- [The Basics](#the-basics)
  - [Responses](#responses)
  - [Redirects](#redirects)
  - [Routing](#routing)
  - [Root template data](#root-template-data)
- [Data Props](#data-props)
  - [Shared data](#shared-data)
  - [Flash data](#flash-data)
  - [Partial reloads](#partial-reloads)
  - [Deferred props](#deferred-props)
  - [Merging props](#merging-props)
  - [Once props](#once-props)
  - [Infinite scroll](#infinite-scroll)
- [Security](#security)
  - [CSRF protection](#csrf-protection)
  - [History encryption](#history-encryption)
- [Advanced](#advanced)
  - [Asset versioning](#asset-versioning)
  - [Page URL](#page-url)
  - [Error handling](#error-handling)
  - [Preserving URL fragments](#preserving-url-fragments)
  - [Server-side Rendering (SSR)](#server-side-rendering-ssr)
  - [Vite and embedding](#vite-and-embedding)
- [Compatibility](#compatibility)
- [Author](#author)
- [License](#license)

## Getting started

A minimal app from scratch. Full copies of each step live under [`examples/`](./examples).

### Installation

```sh
go get github.com/kohkimakimoto/inertia-echo/v5
```

```sh
go get github.com/labstack/echo/v5
```

### Root template

Create `resources/views/app.html`.
`{{ vite ... }}` emits your JS/CSS tags; `{{ .inertia }}` / `{{ .inertiaHead }}` emit the Inertia page bootstrap (details in [Renderer](#renderer)):

```html
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    {{ vite "resources/js/app.jsx" }}
    {{- .inertiaHead -}}
  </head>
  <body>
    {{ .inertia }}
  </body>
</html>
```

### Write Go code

Create `main.go`:

```go
package main

import (
	"log/slog"
	"net/http"

	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	r := inertia.NewHTMLRenderer()
	r.ViteBasePath = "/build"
	r.ViteManifest = inertia.MustParseViteManifestFile("resources/public/build/manifest.json")
	r.MustParseGlob("resources/views/*.html")

	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	e.Use(inertia.CSRF())

	e.Static("/", "resources/public")

	e.GET("/", func(c *echo.Context) error {
		return inertia.Render(c, http.StatusOK, "Index", map[string]any{
			"message": "Hello, World!",
		})
	})

	if err := e.Start(":8080"); err != nil {
		slog.Error("failed to start server", "error", err)
	}
}
```

### Setup frontend

This tutorial uses Vite and React.

```sh
npm init -y
npm install @inertiajs/react react react-dom vite @vitejs/plugin-react
```

Create `vite.config.js`:

```js
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  publicDir: false,
  build: {
    manifest: "manifest.json",
    outDir: "resources/public/build",
    rollupOptions: {
      input: ['resources/js/app.jsx'],
    },
  },
})
```

Create `resources/js/app.jsx`:

```js
import { createInertiaApp } from '@inertiajs/react'

createInertiaApp({
  resolve: name => {
    const pages = import.meta.glob('./pages/**/*.jsx')
    return pages[`./pages/${name}.jsx`]()
  },
})
```

Create `resources/js/pages/Index.jsx` ([page components](https://inertiajs.com/docs/v3/the-basics/pages)):

```jsx
import React from 'react';

export default function Index({ message }) {
  return (
    <div>
      <h1>{ message }</h1>
    </div>
  );
}
```

```sh
npx vite build
```

### Run the application

```sh
go run .
```

Open `http://localhost:8080`. You should see "Hello, World!".

Complete code: [`examples/getting-started-step1`](./examples/getting-started-step1).

### Run in Dev mode

For Vite hot reload, use a `BuildMode` flag and start Vite from Go with [go-subprocess](https://github.com/kohkimakimoto/go-subprocess):

```sh
go get github.com/kohkimakimoto/go-subprocess
```

Update `main.go`:

```go
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kohkimakimoto/go-subprocess"
	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// BuildMode is overwritten at build time via -ldflags "-X main.BuildMode=production".
var BuildMode = "debug"

func main() {
	isDebug := BuildMode == "debug"

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	r := inertia.NewHTMLRenderer()
	r.Debug = isDebug
	r.ViteBasePath = "/build"
	if !isDebug {
		r.ViteManifest = inertia.MustParseViteManifestFile("resources/public/build/manifest.json")
	}
	r.MustParseGlob("resources/views/*.html")

	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	e.Use(inertia.CSRF())

	e.Static("/", "resources/public")

	e.GET("/", func(c *echo.Context) error {
		return inertia.Render(c, http.StatusOK, "Index", map[string]any{
			"message": "Hello, World!",
		})
	})

	var vite *subprocess.Process
	if isDebug {
		p, err := subprocess.Start(ctx, subprocess.Config{
			Command:         "npx",
			Args:            []string{"vite"},
			Stdout:          os.Stdout,
			StdoutFormatter: subprocess.PrefixFormatter("[Vite] "),
			Stderr:          os.Stderr,
			StderrFormatter: subprocess.PrefixFormatter("[Vite] "),
			Dir:             ".",
		})
		if err != nil {
			slog.Error("failed to start Vite subprocess", "error", err)
			return
		}
		vite = p
	}

	if err := (echo.StartConfig{Address: ":8080"}).Start(ctx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
	}

	if vite != nil {
		if err := vite.Wait(); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("the Vite subprocess returned an error", "error", err)
		}
	}
}
```

Add `{{ vite_react_refresh }}` for React Fast Refresh ([Vite docs](https://vitejs.dev/guide/backend-integration.html)):

```html
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    {{ vite_react_refresh }}
    {{ vite "resources/js/app.jsx" }}
    {{- .inertiaHead -}}
  </head>
  <body>
    {{ .inertia }}
  </body>
</html>
```

Debug (hot reload):

```sh
go run .
```

Production:

```sh
npx vite build
go run -ldflags="-X main.BuildMode=production" .
```

Complete code: [`examples/getting-started-step2`](./examples/getting-started-step2).

### Embed assets into a single binary

Embed templates and built assets into one binary for production.
See [`examples/getting-started-step3`](./examples/getting-started-step3) and [Vite and embedding](#vite-and-embedding).

```sh
npx vite build
go build -ldflags="-X main.BuildMode=production" -o app
./app
```

### Next steps

- [Architecture](#architecture) — Middleware, Renderer, and StateStore
- [The Basics](#the-basics) onward — Inertia features in Go
- [`examples/notes`](./examples/notes) — authenticated sample app
- [`examples/ssr`](./examples/ssr) — SSR setup

## Architecture

Like [inertia-laravel](https://github.com/inertiajs/inertia-laravel), **middleware is the core**: it prepares each request for the Inertia protocol and lets handlers call `inertia.Render`, `inertia.Share`, and related helpers.

Laravel already provides a view layer and a session. Echo does not, so inertia-echo fills those gaps with two extra pieces wired into the same middleware:

| Piece | Fills the gap for… |
| --- | --- |
| [Renderer](#renderer) | Root HTML (and optional SSR) on first visits |
| [StateStore](#statestore) | Short-lived cross-request values (flash, clear history, preserve fragment) |

### Middleware

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	Renderer: r, // see Renderer below
	// StateStore: optional; defaults to CookieStateStore
}))
```

After this middleware runs, use package helpers (`inertia.Render`, `inertia.Share`, …) or [`inertia.Get`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#Get) / [`MustGet`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#MustGet) on `echo.Context`.

[`MiddlewareConfig`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#MiddlewareConfig) also covers shared props, asset versioning, page URL, SSR exclusions, and related options — see the linked sections under [The Basics](#the-basics), [Data Props](#data-props), and [Advanced](#advanced).

### Renderer

:book: Related: [Server-Side Setup](https://inertiajs.com/docs/v3/installation/server-side-setup)

[`Renderer`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#Renderer) turns a first visit into HTML (root template + Inertia bootstrap, or SSR output).
The built-in [`HTMLRenderer`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#HTMLRenderer) uses `html/template` and Vite helpers (`{{ vite }}`, `{{ .inertia }}`, `{{ .inertiaHead }}`).

```go
r := inertia.NewHTMLRenderer()
r.ViteBasePath = "/build"
r.ViteManifest = inertia.MustParseViteManifestFile("resources/public/build/manifest.json")
r.MustParseGlob("resources/views/*.html")

e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	Renderer: r,
}))
```

You can implement `Renderer` for another template engine and pass it via `MiddlewareConfig.Renderer`.
See [`HTMLRenderer`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#HTMLRenderer) for fields and helpers (Vite, SSR, and so on).

### StateStore

Several Inertia features need a value that survives until a later request ([flash](#flash-data), [clear history](#clearing-history), [preserve fragment](#preserving-url-fragments)).
inertia-laravel keeps those in the Laravel session. Echo has no built-in session, so [`StateStore`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#StateStore) is inertia-echo's small, Inertia-only bag for that gap — **not** a general application session and not part of the Inertia protocol itself.

The default is [`CookieStateStore`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#CookieStateStore).

**Not a secure session.** Treat loaded values like untrusted request input. Do not store credentials, identity, or authorization state. Keep payloads small.

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	Renderer: r,
	StateStore: inertia.NewCookieStateStoreWithConfig(inertia.CookieStateStoreConfig{
		Secure:   true,
		HTTPOnly: true,
	}),
	// Or: StateStore: myStateStore, // implements inertia.StateStore
}))
```

See [`CookieStateStoreConfig`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#CookieStateStoreConfig) for fields and defaults.

## The Basics

Server-side counterparts to [Inertia's the-basics](https://inertiajs.com/docs/v3/the-basics/pages).
Pages, links, forms, and similar client topics are covered in the official docs.

### Responses

:book: [Responses](https://inertiajs.com/docs/v3/the-basics/responses)

```go
func ShowEventHandler(c *echo.Context) error {
	event := // retrieve an event...
	return inertia.Render(c, http.StatusOK, "Event/Show", map[string]any{
		"event": event,
	})
}
```

`Render` accepts a `map[string]any` of props for the page component.
For root-template-only data, see [Root template data](#root-template-data).

### Redirects

:book: [Redirects](https://inertiajs.com/docs/v3/the-basics/redirects)

Use Echo's standard redirect:

```go
return c.Redirect(http.StatusFound, "/")
```

The middleware may rewrite Inertia redirect responses after the handler returns (see below).

#### External redirects

For a full page visit outside the current Inertia app (or a hard reload):

```go
return inertia.Location(c, "/path/to/external")
```

For Inertia requests this returns `409` with `X-Inertia-Location`. For non-Inertia requests it falls back to a normal `302` redirect.

#### Redirect status (302 → 303)

For Inertia requests that use `PUT`, `PATCH`, or `DELETE`, a buffered `302 Found` is rewritten to `303 See Other` so the follow-up visit is a GET. This matches the Inertia protocol.

#### Redirects with URL fragments

When an Inertia request redirects to a `Location` that already contains `#…`, the middleware converts it to:

- status `409 Conflict`
- header `X-Inertia-Redirect` with the original location (including the fragment)
- no `Location` header

Prefetch requests are skipped (`Purpose`, `Sec-Purpose`, or `X-Moz` equal to `prefetch`, case-insensitive).

This is separate from [`PreserveFragment`](#preserving-url-fragments), which asks the client to keep the **current** URL fragment across the next page.

### Routing

:book: [Routing](https://inertiajs.com/docs/v3/the-basics/routing)

#### Shorthand routes

```go
e.GET("/about", inertia.Handler("About"))
```

### Root template data

:book: Related: [Title & Meta](https://inertiajs.com/docs/v3/the-basics/title-and-meta)

Props are available on the root template:

```html
<meta name="twitter:title" content="{{ .page.Props.event.Title }}">
```

To pass data that is **not** sent to the JavaScript page component, use `RenderWithViewData`:

```go
func ShowEventsHandler(c *echo.Context) error {
	event := // retrieve an event...
	return inertia.RenderWithViewData(c, http.StatusOK, "Event/Show", map[string]any{
		"event": event,
	}, map[string]any{
		"meta": "Meta data...",
	})
}
```

```html
<meta name="twitter:title" content="{{ .meta }}">
```

## Data Props

:book: Category: [Data props](https://inertiajs.com/docs/v3/data-props/shared-data)

### Shared data

:book: [Shared data](https://inertiajs.com/docs/v3/data-props/shared-data)

#### Sharing data using middleware

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	Share: func(c *echo.Context) (map[string]any, error) {
		user := // get auth user...
		return map[string]any{
			"appName":  "App Name",
			"authUser": user,
		}, nil
	},
	// Once props remembered by the client across navigations.
	// see https://inertiajs.com/docs/v3/data-props/shared-data#sharing-once-props
	ShareOnce: func(c *echo.Context) (map[string]any, error) {
		return map[string]any{
			"countries": func() (any, error) {
				return loadCountries()
			},
		}, nil
	},
}))
```

You can also put `inertia.Once(...)` in `Share`, or call `inertia.ShareOnce` manually after the middleware has run.

#### Sharing data manually

```go
inertia.Share(c, map[string]any{
	"appName":  "App Name",
	"authUser": user,
})
```

### Flash data

:book: [Flash data](https://inertiajs.com/docs/v3/data-props/flash-data)

`Flash` adds one-time data to the top-level Page `flash` object:

```go
inertia.Flash(c, map[string]any{
	"message": "Profile updated.",
})
return c.Redirect(http.StatusFound, "/profile")
```

Across redirects, flash is persisted via [`StateStore`](#statestore) until the next rendered page.

### Partial reloads

:book: [Partial reloads](https://inertiajs.com/docs/v3/data-props/partial-reloads)

```go
inertia.Render(c, http.StatusOK, "Users/Index", map[string]any{
	// ALWAYS included on standard visits
	// OPTIONALLY included on partial reloads
	// ALWAYS evaluated
	"users": users,

	// ALWAYS included on standard visits
	// OPTIONALLY included on partial reloads
	// ONLY evaluated when needed
	"companies": func() (any, error) {
		return loadCompanies()
	},

	// NEVER included on standard visits
	// OPTIONALLY included on partial reloads
	// ONLY evaluated when needed
	"permissions": inertia.Optional(func() (any, error) {
		return loadPermissions()
	}),

	// ALWAYS included on standard visits
	// ALWAYS included on partial reloads
	// ALWAYS evaluated
	"auth": inertia.Always(authUser),
})
```

### Deferred props

:book: [Deferred props](https://inertiajs.com/docs/v3/data-props/deferred-props)

```go
inertia.Render(c, http.StatusOK, "Users/Index", map[string]any{
	"users": users,
	"roles": roles,
	"permissions": inertia.Defer(func() (any, error) {
		permissions, err := // get permissions...
		if err != nil {
			return nil, err
		}
		return permissions, nil
	}),
})
```

#### Grouping requests

```go
inertia.Render(c, http.StatusOK, "Users/Index", map[string]any{
	"users": users,
	"roles": roles,
	"permissions": inertia.Defer(func() (any, error) {
		permissions, err := // get permissions...
		if err != nil {
			return nil, err
		}
		return permissions, nil
	}),
	"teams": inertia.DeferWithGroup(func() (any, error) {
		teams, err := // get teams...
		if err != nil {
			return nil, err
		}
		return teams, nil
	}, "attributes"),
	"projects": inertia.DeferWithGroup(func() (any, error) {
		projects, err := // get projects...
		if err != nil {
			return nil, err
		}
		return projects, nil
	}, "attributes"),
	"tasks": inertia.DeferWithGroup(func() (any, error) {
		tasks, err := // get tasks...
		if err != nil {
			return nil, err
		}
		return tasks, nil
	}, "attributes"),
})
```

#### Rescuing deferred errors

Deferred callback errors are returned by default. Use `Rescue` to omit a failed deferred prop while allowing the rest of the response to render. Configure `RescueReporter` to observe rescued errors without exposing their contents to the client.

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	Renderer: r,
	RescueReporter: func(path string, err error) {
		logger.Error("deferred prop failed", "path", path, "error", err)
	},
}))

inertia.Render(c, http.StatusOK, "Users/Index", map[string]any{
	"permissions": inertia.Defer(func() (any, error) {
		return loadPermissions()
	}).Rescue(),
})
```

### Merging props

:book: [Merging props](https://inertiajs.com/docs/v3/data-props/merging-props)

#### Shallow merge

```go
inertia.Render(c, http.StatusOK, "Tags/Index", map[string]any{
	"tags": inertia.Merge(tags),
})
```

#### Deep merge

```go
inertia.Render(c, http.StatusOK, "Users/Index", map[string]any{
	"users": inertia.DeepMerge(users),
})
```

You may chain `MatchOn` to determine how existing items should be matched and updated.

```go
inertia.Render(c, http.StatusOK, "Users/Index", map[string]any{
	"users": inertia.DeepMerge(users).MatchOn("data.id"),
})
```

Use `Append` or `Prepend` to select the client-side merge direction at the root or at nested paths:

```go
inertia.Render(c, http.StatusOK, "Users/Index", map[string]any{
	"users": inertia.Merge(users).Append("data").MatchOn("data.id"),
	"activity": inertia.Merge(activity).Prepend(),
})
```

### Once props

:book: [Once props](https://inertiajs.com/docs/v3/data-props/once-props)

Once props let the client reuse a previously loaded value. The server remains stateless; the client reports loaded keys in later requests.

```go
inertia.Render(c, http.StatusOK, "Dashboard", map[string]any{
	"settings": inertia.Once(func() (any, error) {
		return loadSettings()
	}).As("dashboard-settings").For(10 * time.Minute),
})
```

Use `Fresh(true)` to resolve the value even when the client already has it, `Until(time.Time)` to set an absolute expiration, or `For(time.Duration)` to set a relative expiration. `OptionalProp`, `DeferProp`, and `MergeProp` can opt in with `Once()`; calling `As`, `Fresh`, `Until`, or `For` on those prop types also enables once behavior. `ShareOnce` is a shortcut for sharing a callback-backed once prop.

### Infinite scroll

:book: [Infinite Scroll](https://inertiajs.com/docs/v3/data-props/infinite-scroll)

`Scroll` receives the page value and pagination metadata together, so loading the data does not need to run twice. Page identifiers may be numbers, cursor strings, or `nil`.

```go
inertia.Render(c, http.StatusOK, "Users/Index", map[string]any{
	"users": inertia.Scroll(func() (inertia.ScrollResult, error) {
		users, err := loadUsers()
		if err != nil {
			return inertia.ScrollResult{}, err
		}
		return inertia.ScrollResult{
			Value: map[string]any{"data": users},
			Metadata: inertia.ScrollMetadata{
				PageName: "page", PreviousPage: nil,
				CurrentPage: 1, NextPage: 2,
			},
		}, nil
	}),
})
```

The default data path is `data`. Use `WithDataPath` to change it and `Defer` to defer the initial load.

## Security

:book: Category: [Security](https://inertiajs.com/docs/v3/security/authentication)

Authentication and authorization are application concerns (same as with any Echo app). This section covers adapter helpers that map to Inertia's security docs.

### CSRF protection

:book: [CSRF protection](https://inertiajs.com/docs/v3/security/csrf-protection)

Inertia Echo ships CSRF middleware configured for Inertia.js.
It sets an `XSRF-TOKEN` cookie and verifies the `X-XSRF-TOKEN` header.

```go
e.Use(inertia.CSRF())
```

### History encryption

:book: [History encryption](https://inertiajs.com/docs/v3/security/history-encryption)

#### Encrypt middleware

```go
e.Use(inertia.EncryptHistoryMiddleware())
```

Opt out on a specific page:

```go
inertia.EncryptHistory(c, false)
```

#### Per-request encryption

```go
inertia.EncryptHistory(c, true)
```

#### Clearing history

```go
inertia.ClearHistory(c)
```

If the response is a redirect (or otherwise does not consume the flag on a Page render), the flag is persisted via [`StateStore`](#statestore) until the next rendered page.

## Advanced

### Asset versioning

:book: [Asset versioning](https://inertiajs.com/docs/v3/advanced/asset-versioning)

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	VersionFunc: func(c *echo.Context) string { return version },
}))
```

The default `VersionFunc` reads `INERTIA_VERSION`, then `GAE_VERSION`, then a development timestamp. On mismatch the middleware returns an Inertia `Location` for the current URL.

Per request:

```go
inertia.SetVersion(c, func(c *echo.Context) string { return version })
```

### Page URL

:book: [The Page object](https://inertiajs.com/docs/v3/core-concepts/the-protocol#the-page-object)

By default the Page `url` is resolved with `req.URL.String()` (path and query). Customize it when the request URL differs from the URL you want the client to see (for example behind a path prefix):

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	URLResolver: func(c *echo.Context) string {
		return "/app" + c.Request().URL.String()
	},
}))
```

Or set it per request:

```go
inertia.SetURLResolver(c, func(c *echo.Context) string {
	return "/app" + c.Request().URL.String()
})
```

### Error handling

:book: [Error handling](https://inertiajs.com/docs/v3/advanced/error-handling)

This package does not add its own error-handling framework.
Echo already has `HTTPErrorHandler`, and Inertia request state is available on
`echo.Context` after the middleware runs. Prefer those primitives.

**In a handler**, render an error page with a non-200 status via `Render`:

```go
return inertia.Render(c, http.StatusNotFound, "ErrorPage", map[string]any{
	"status": http.StatusNotFound,
})
```

**Centrally**, use Echo's `HTTPErrorHandler`. When the Inertia middleware has
already run, `inertia.Get(c)` succeeds and shared props / renderer settings are
available:

```go
e.HTTPErrorHandler = func(c *echo.Context, err error) {
	status := http.StatusInternalServerError
	var he *echo.HTTPError
	if errors.As(err, &he) && he.Code != 0 {
		status = he.Code
	}

	if i, e := inertia.Get(c); e == nil {
		switch status {
		case http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError, http.StatusServiceUnavailable:
			if renderErr := i.Render(status, "ErrorPage", map[string]any{
				"status": status,
			}); renderErr == nil {
				return
			}
		}
	}

	echo.DefaultHTTPErrorHandler(false)(c, err)
}
```

Errors that occur before the Inertia middleware runs have no Inertia context;
fall back to Echo's default (or your own) handler in that case.

### Preserving URL fragments

Call `PreserveFragment` before a redirect to set the Page `preserveFragment` flag so the client retains the **source** URL fragment on the next rendered page. The flag survives redirects via [`StateStore`](#statestore):

```go
inertia.PreserveFragment(c)
return c.Redirect(http.StatusFound, "/account")
```

If the redirect **destination** already includes `#…`, see [Redirects with URL fragments](#redirects-with-url-fragments) — that is a different middleware rewrite (`409` + `X-Inertia-Redirect`).

### Server-side Rendering (SSR)

:book: [Server-side Rendering (SSR)](https://inertiajs.com/docs/v3/advanced/server-side-rendering)

SSR is pluggable through [`SsrGateway`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#SsrGateway) interface. [`SsrHTTPGateway`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#SsrHTTPGateway) is the built-in HTTP client for a remote SSR server.

```go
// Assign an SsrHTTPGateway to the renderer to enable SSR.
renderer.SsrGateway = inertia.NewSsrHTTPGateway()
```

See [`examples/ssr`](./examples/ssr) for details.

### Vite and embedding

You can bundle frontend builds into a single Go binary using `embed`.
Getting Started covers the full tutorial path in [Embed assets into a single binary](#embed-assets-into-a-single-binary); the snippet below is the production shape.

```go
package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/labstack/echo/v5"
)

//go:embed resources/views/*.html
var viewFiles embed.FS

//go:embed resources/public/*
var publicFiles embed.FS

func main() {
	e := echo.New()
	// ...

	r := inertia.NewHTMLRenderer()
	r.ViteManifest = inertia.MustParseViteManifestFS(publicFiles, "resources/public/build/manifest.json")
	r.MustParseFS(viewFiles, "resources/views/*.html")
	// ...

	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	// ...

	fsys, err := fs.Sub(publicFiles, "resources/public")
	if err != nil {
		panic(err)
	}

	assetHandler := http.FileServer(http.FS(fsys))
	e.GET("/*", echo.WrapHandler(http.StripPrefix("/", assetHandler)))

	e.GET("/", func(c *echo.Context) error {
		return inertia.Render(c, http.StatusOK, "Index", map[string]any{
			"message": "Hello, World!",
		})
	})

	if err := e.Start(":8080"); err != nil {
		slog.Error("failed to start server", "error", err)
	}
}
```

## Compatibility

Inertia Echo aligns its major version with the supported Echo major version.

| Inertia Echo | Echo | Inertia.js | Status |
| --- | --- | --- | --- |
| v5.1.0+ | v5 | v3 | Current |
| v5.0.0 | v5 | v2 | Final |
| v4.0.1 | v4 | v2 | Final |

## Author

Kohki Makimoto <kohki.makimoto@gmail.com>

## License

The MIT License (MIT)
