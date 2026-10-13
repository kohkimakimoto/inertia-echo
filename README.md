# Inertia Echo

[![test](https://github.com/kohkimakimoto/inertia-echo/actions/workflows/test.yml/badge.svg?branch=master)](https://github.com/kohkimakimoto/inertia-echo/actions/workflows/test.yml?query=branch%3Amaster)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/kohkimakimoto/inertia-echo/v5.svg)](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5)

This is the [Inertia.js](https://inertiajs.com) server-side adapter for [Echo](https://echo.labstack.com/) Go web framework.

[Inertia.js](https://inertiajs.com) is a JavaScript library that allows you to build a fully JavaScript-based single-page app without complexity.
I assume that you are familiar with Inertia.js and [how it works](https://inertiajs.com/docs/v3/core-concepts/how-it-works).
You also need to familiarize yourself with [Echo](https://echo.labstack.com/), a Go web framework.
Inertia Echo assists you in developing web applications that leverage both of these technologies.

> [!NOTE]
> Inertia Echo v5.1.0+ supports Echo v5 and Inertia.js v3. v5.0.0 is the final release for Inertia.js v2.

Table of Contents

- [Getting started](#getting-started)
  - [Installation](#installation)
  - [Root template](#root-template)
  - [Write Go code](#write-go-code)
  - [Setup frontend](#setup-frontend)
  - [Run the application](#run-the-application)
  - [Run in Dev mode](#run-in-dev-mode)
  - [Embed assets into a single binary](#embed-assets-into-a-single-binary)
- [Usage](#usage)
  - [Renderer](#renderer)
  - [Middleware](#middleware)
  - [Responses](#responses)
    - [Creating responses](#creating-responses)
    - [Root template data](#root-template-data)
  - [Redirects](#redirects)
    - [External redirects](#external-redirects)
    - [Redirect status (302 → 303)](#redirect-status-302--303)
    - [Redirects with URL fragments](#redirects-with-url-fragments)
  - [Routing](#routing)
    - [Shorthand routes](#shorthand-routes)
  - [Shared data](#shared-data)
    - [Sharing data using middleware](#sharing-data-using-middleware)
    - [Sharing data manually](#sharing-data-manually)
  - [Partial reloads](#partial-reloads)
  - [Deferred props](#deferred-props)
    - [Grouping requests](#grouping-requests)
    - [Rescuing deferred errors](#rescuing-deferred-errors)
  - [Once props](#once-props)
  - [Merging props](#merging-props)
  - [Infinite scroll props](#infinite-scroll-props)
  - [Cross-request state (StateStore)](#cross-request-state-statestore)
  - [Flash data](#flash-data)
  - [Preserving URL fragments](#preserving-url-fragments)
  - [Error responses](#error-responses)
  - [CSRF protection](#csrf-protection)
  - [History encryption](#history-encryption)
  - [Asset versioning](#asset-versioning)
  - [Page URL](#page-url)
  - [Server-side Rendering (SSR)](#server-side-rendering-ssr)
  - [Embed](#embed)
- [Compatibility](#compatibility)
- [Author](#author)
- [License](#license)

## Getting started

In this section, we provide step-by-step instructions on how to get started with Inertia Echo.

### Installation

Inertia Echo is a Go module that you can install with the following command:

```sh
go get github.com/kohkimakimoto/inertia-echo/v5
```

You also need to install Echo like this:

```sh
go get github.com/labstack/echo/v5
```

### Root template

Next, setup the root template that will be loaded on the first page visit to your application. This template should include your site's CSS and JavaScript assets, along with the `.inertia` and `.inertiaHead` variables.

The built-in renderer emits the Inertia v3 initial page as a JSON `script` element followed by the root `div`. The `script` element's `data-page` value and the root element's `id` both use the renderer's `ContainerId`.

In this tutorial, we will create the `resources/views/app.html` file as the root template.

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

Next, you need to implement Go application code with the Echo framework. Create the `main.go` file with the following code:

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

Next, you need to setup the frontend of your application. In this tutorial, we will use Vite and React.

If you don't have a package.json file yet, create one with the following command:

```sh
npm init -y
```

Install the required packages:

```sh
npm install @inertiajs/react react react-dom vite @vitejs/plugin-react
```

Create the `vite.config.js` file with the following content:

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

Create the `resources/js/app.jsx` file with the following content:

```js
import { createInertiaApp } from '@inertiajs/react'

createInertiaApp({
  resolve: name => {
    const pages = import.meta.glob('./pages/**/*.jsx')
    return pages[`./pages/${name}.jsx`]()
  },
})
```

Create a [page component](https://inertiajs.com/docs/v3/the-basics/pages) as the  `resources/js/pages/Index.jsx` file with the following content:

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

Build the frontend assets with the following command:

```sh
npx vite build
```

### Run the application

Now you can run the application with the following command:

```sh
go run .
```

Then, open your browser and navigate to `http://localhost:8080`. You should see the message "Hello, World!" displayed on the page.

You can find the complete code of this example in the [examples/getting-started-step1](./examples/getting-started-step1) directory of this repository.

### Run in Dev mode

If you want to run in dev mode so that you can hot-reload frontend updates, introduce a `BuildMode` variable that is overwritten at build time via `-ldflags`.

Install [kohkimakimoto/go-subprocess](https://github.com/kohkimakimoto/go-subprocess) to start the Vite development server from your Go process:

```sh
go get github.com/kohkimakimoto/go-subprocess
```

Update your `main.go` as follows:

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

This example uses [kohkimakimoto/go-subprocess](https://github.com/kohkimakimoto/go-subprocess) to start the Vite server only when `BuildMode` is `"debug"`.
A shared cancelable context ties the Echo server and Vite process together so both stop on interrupt.

Also add `{{ vite_react_refresh }}` to your root template for React Fast Refresh with `@vitejs/plugin-react`.
For more information, see [Vite docs](https://vitejs.dev/guide/backend-integration.html).

`resources/views/app.html`

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

Run in debug mode. This starts the Vite development server so you can hot-reload frontend updates:

```sh
go run .
```

Run in production mode. This serves the compiled frontend assets produced by `npx vite build`:

```sh
npx vite build
go run -ldflags="-X main.BuildMode=production" .
```

You can find the complete code of this example in the [examples/getting-started-step2](./examples/getting-started-step2) directory of this repository.

### Embed assets into a single binary

For production, you can embed the root template and built frontend assets into the Go binary.
Inertia Echo accepts `fs.FS` for templates and the Vite manifest (`MustParseFS`, `MustParseViteManifestFS`, and Echo's `StaticFS`), so the same application code can serve files from disk in development and from the embedded filesystem in production.

See the complete code in the [examples/getting-started-step3](./examples/getting-started-step3) directory of this repository.

With that example, build the frontend assets, then compile with production `BuildMode` so the binary embeds those assets:

```sh
npx vite build
go build -ldflags="-X main.BuildMode=production" -o app
```

Run the resulting binary:

```sh
./app
```

Open `http://localhost:8080`. The server no longer depends on `resources/` on disk; the views and public assets are inside the binary.

For a complete authenticated application with JSON persistence, React, Tailwind CSS,
and Inertia feature demonstrations, see [Notes](./examples/notes).

## Usage

### Renderer

Unlike Laravel, which is an officially supported framework for Inertia.js, Echo lacks built-in view rendering.
This means you'll have to build your own view system and integrate it with Inertia.js.

Inertia Echo defines [`Renderer`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#Renderer) interface to integrate view system with Inertia.js.
It also provides a built-in renderer implementation based on the `html/template` package.
Use [`ParseGlob`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#HTMLRenderer.ParseGlob), [`ParseFS`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#HTMLRenderer.ParseFS), or [`ParseFiles`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#HTMLRenderer.ParseFiles) (or their `Must*` variants) to load templates with Inertia/Vite helpers applied.

To setup Inertia Echo with your Echo application, you need to initialize the renderer and set it up with the [middleware](#middleware).

```go
// Create and configure the renderer...
r := inertia.NewHTMLRenderer()
r.ViteBasePath = "/build"
r.MustParseGlob("resources/views/*.html")

// Setup the middleware with the renderer
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
  Renderer: r,
}))
```

### Middleware

Add the Inertia middleware after configuring a renderer:

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	Renderer: r,
}))
```

The middleware is the core of this package. For each request it:

1. Creates a request-scoped [`Inertia`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#Inertia) instance and stores it on `echo.Context`
2. Loads the cross-request state bag via [`StateStore`](#cross-request-state-statestore)
3. Checks asset version **before** `Share` / the handler (unlike inertia-laravel, which does this after `$next`). On mismatch it returns an Inertia `Location` response
4. Applies `Share` / `ShareOnce` when configured
5. Runs the next handler (with a response writer wrapper so redirects can be rewritten)
6. On success: sets `Vary: X-Inertia`, rewrites Inertia redirects when needed (302→303, fragment → 409), then commits state via `StateStore`
7. On handler error: returns the error immediately (no redirect rewrite, no state commit). Use Echo's `HTTPErrorHandler` for error pages

Unlike inertia-laravel, empty Inertia `200` responses are **not** automatically converted to a back redirect. Return `Redirect`, `Render`, or an error explicitly.

Configure behavior through [`MiddlewareConfig`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#MiddlewareConfig) (`Renderer`, `Share`, `ShareOnce`, `VersionFunc`, `URLResolver`, `StateStore`, `WithoutSsr`, `RescueReporter`, and others).

### Responses

:book: The related official document: [Responses](https://inertiajs.com/docs/v3/the-basics/responses)

#### Creating responses

The following code shows how to create an Inertia response.
The `Render` function accepts a `map[string]any` as its final argument, which contains the properties to pass to the view.

```go
func ShowEventHandler(c *echo.Context) error {
	event := // retrieve a event...
	return inertia.Render(c, http.StatusOK, "Event/Show", map[string]any{
		"event": event,
	})
}
```

#### Root template data

You can access your properties in the root template.

```html
<meta name="twitter:title" content="{{ .page.Props.event.Title }}">
```

Sometimes you may even want to provide data that will not be sent to your JavaScript component.
In this case, you can use the `RenderWithViewData` function.

```go
func ShowEventsHandler(c *echo.Context) error {
	event := // retrieve a event...
	return inertia.RenderWithViewData(c, http.StatusOK, "Event/Show", map[string]any{
		"event": event,
	}, map[string]any{
		"meta": "Meta data...",
	})
}
```

You can then access this variable like a regular template variable.

```html
<meta name="twitter:title" content="{{ .meta }}">
```

### Redirects

:book: The related official document: [Redirects](https://inertiajs.com/docs/v3/the-basics/redirects)

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

For Inertia requests that use `PUT`, `PATCH`, or `DELETE`, a buffered `302 Found` is rewritten to `303 See Other` so the follow-up visit is a GET. This matches the Inertia protocol and inertia-laravel middleware behavior.

#### Redirects with URL fragments

When an Inertia request redirects to a `Location` that already contains `#…`, the middleware converts it to:

- status `409 Conflict`
- header `X-Inertia-Redirect` with the original location (including the fragment)
- no `Location` header

Prefetch requests are skipped (`Purpose`, `Sec-Purpose`, or `X-Moz` equal to `prefetch`, case-insensitive — same as Laravel's `Request::prefetch()`).

This is separate from [`PreserveFragment`](#preserving-url-fragments), which asks the client to keep the **current** URL fragment across the next page.

### Routing

:book: The related official document: [Routing](https://inertiajs.com/docs/v3/the-basics/routing)

#### Shorthand routes

Inertia Echo provides a helper function for shorthand routes

```go
e.GET("/about", inertia.Handler("About"))
```

### Shared data

:book: The related official document: [Shared data](https://inertiajs.com/docs/v3/data-props/shared-data)

#### Sharing data using middleware

You can set shared data via middleware.

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

Alternatively, you can manually share data using the `Share` function.

```go
inertia.Share(c, map[string]any{
	"appName":  "App Name",
	"authUser": user,
})
```

### Partial reloads

:book: The related official document: [Partial reloads](https://inertiajs.com/docs/v3/data-props/partial-reloads)

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

:book: The related official document: [Deferred props](https://inertiajs.com/docs/v3/data-props/deferred-props)

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

### Once props

Once props let the client reuse a previously loaded value. The server remains stateless; the client reports loaded keys in later requests.

```go
inertia.Render(c, http.StatusOK, "Dashboard", map[string]any{
	"settings": inertia.Once(func() (any, error) {
		return loadSettings()
	}).As("dashboard-settings").For(10 * time.Minute),
})
```

Use `Fresh(true)` to resolve the value even when the client already has it, `Until(time.Time)` to set an absolute expiration, or `For(time.Duration)` to set a relative expiration. `OptionalProp`, `DeferProp`, and `MergeProp` can opt in with `Once()`; calling `As`, `Fresh`, `Until`, or `For` on those prop types also enables once behavior. `ShareOnce` is a shortcut for sharing a callback-backed once prop.

:book: The related official document: [Once props](https://inertiajs.com/docs/v3/data-props/once-props)

### Merging props

:book: The related official document: [Merging props](https://inertiajs.com/docs/v3/data-props/merging-props)

#### Shallow merge

```go
inertia.Render(c, http.StatusOK, "Tags/Index", map[string]any{
	"tags": inertia.Merge(tags),
})
```

#### Deep merge

```go
inertia.Render(c, http.StatusOK, "Users/Index", map[string]any{
	"tags": inertia.DeepMerge(users),
})
```

You may chain the matchOn method to determine how existing items should be matched and updated.

```go
inertia.Render(c, http.StatusOK, "Users/Index", map[string]any{
	"tags": inertia.DeepMerge(users).MatchOn("data.id"),
})
```

Use `Append` or `Prepend` to select the client-side merge direction at the root or at nested paths:

```go
inertia.Render(c, http.StatusOK, "Users/Index", map[string]any{
	"users": inertia.Merge(users).Append("data").MatchOn("data.id"),
	"activity": inertia.Merge(activity).Prepend(),
})
```

### Infinite scroll props

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

:book: The related official document: [Infinite Scroll](https://inertiajs.com/docs/v3/data-props/infinite-scroll)

### Cross-request state (StateStore)

Echo has no built-in session, but several Inertia features need values that survive until a later request (flash, clear history, preserve fragment). inertia-laravel stores those in the Laravel session. **StateStore is inertia-echo's own design** for that gap — not an official Inertia protocol feature and not a general-purpose application session.

Architecture:

- During a request, values live in an in-memory bag on the request-scoped `Inertia` (`Get` / `Set` / `Delete` / `Pull`)
- [`StateStore`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#StateStore) only **Load**s the bag at the start of the middleware and **Save**s it after a successful handler (and on asset version mismatch before `Location`)
- Bag keys are fixed: `flash_data`, `clear_history`, `preserve_fragment` (see `StateKey*`)

The default implementation is [`CookieStateStore`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#CookieStateStore): one unsigned HttpOnly cookie (`inertia.state` by default) holding the whole bag.

**Not a secure session.** Cookie values are client-controlled. Treat loaded data like any other untrusted request input. Do not store credentials, identity, or authorization state. Keep payloads small (cookie size limits).

Tune cookie attributes or replace the backend:

```go
cfg := inertia.DefaultCookieStateStoreConfig
cfg.Secure = true
cfg.Path = "/app"

e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	Renderer:   r,
	StateStore: inertia.NewCookieStateStoreWithConfig(cfg),
	// Or: StateStore: myStateStore, // implements inertia.StateStore (Load/Save)
}))
```

Copy `DefaultCookieStateStoreConfig` before changing a subset of fields so defaults such as `HTTPOnly: true` are preserved.

### Flash data

:book: The related official document: [Flash data](https://inertiajs.com/docs/v3/data-props/flash-data)

`Flash` adds one-time data to the top-level Page `flash` object:

```go
inertia.Flash(c, map[string]any{
	"message": "Profile updated.",
})
return c.Redirect(http.StatusFound, "/profile")
```

For POST-redirect-GET, unconsumed flash is kept in the in-memory bag and committed through [`StateStore`](#cross-request-state-statestore) after the redirect response. The next request loads it and merges it into the Page on render (then clears it).

### Preserving URL fragments

Call `PreserveFragment` before a redirect to set the Page `preserveFragment` flag so the client retains the **source** URL fragment on the next rendered page. The flag survives redirects via [`StateStore`](#cross-request-state-statestore):

```go
inertia.PreserveFragment(c)
return c.Redirect(http.StatusFound, "/account")
```

If the redirect **destination** already includes `#…`, see [Redirects with URL fragments](#redirects-with-url-fragments) — that is a different middleware rewrite (`409` + `X-Inertia-Redirect`).

### Error responses

This package does not provide an inertia-laravel-style error-handling framework
(`handleExceptionsUsing` / `ExceptionResponse`). Echo already has
`HTTPErrorHandler`, and Inertia request state is available on `echo.Context`
after the middleware runs. Prefer those primitives over a library-specific
abstraction.

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

:book: Related official document: [Error handling](https://inertiajs.com/docs/v3/advanced/error-handling)

### CSRF protection

:book: The related official document: [CSRF protection](https://inertiajs.com/docs/v3/security/csrf-protection)

Inertia Echo has CSRF middleware that is configured for Inertia.js.
This middleware provides `XSRF-TOKEN` cookie and verifies the `X-XSRF-TOKEN` header in the request.

The following code shows how to set up the CSRF middleware in your Echo application.

```go
e.Use(inertia.CSRF())
```

### History encryption

:book: The related official document: [History encryption](https://inertiajs.com/docs/v3/security/history-encryption)

#### Encrypt middleware

To encrypt a group of routes, you may use `EncryptHistoryMiddleware`

```go
e.Use(inertia.EncryptHistoryMiddleware())
```

You are able to opt out of encryption on specific pages by calling the `EncryptHistory` function before returning the response.

```go
inertia.EncryptHistory(c, false)
```

#### Per-request encryption

To encrypt the history of an individual request, you can call the `EncryptHistory` function with `true` as the second argument.

```go
inertia.EncryptHistory(c, true)
```

#### Clearing history

```go
inertia.ClearHistory(c)
```

If the response is a redirect (or otherwise does not consume the flag on a Page render), the flag is persisted via [`StateStore`](#cross-request-state-statestore) until the next rendered page.

### Asset versioning

:book: The related official document: [Asset versioning](https://inertiajs.com/docs/v3/advanced/asset-versioning)

Configure asset version via middleware.

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	VersionFunc: func(c *echo.Context) string { return version },
}))
```

The default `VersionFunc` reads `INERTIA_VERSION`, then `GAE_VERSION`, then a development timestamp. Version mismatch is handled **before** `Share` and the handler: the middleware returns an Inertia `Location` for the current URL and sets `X-Inertia-Version`.

Configure asset version manually for the current request:

```go
inertia.SetVersion(c, func(c *echo.Context) string { return version })
```

### Page URL

:book: The related official document: [The Page object](https://inertiajs.com/docs/v3/core-concepts/the-protocol#the-page-object)

By default the Page `url` is resolved with `req.URL.String()` (path and query), via the same middleware defaulting pattern as `VersionFunc`. Customize it when the request URL differs from the URL you want the client to see (for example behind a path prefix):

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

### Server-side Rendering (SSR)

:book: The related official document: [Server-side Rendering (SSR)](https://inertiajs.com/docs/v3/advanced/server-side-rendering)

Inertia Echo supports SSR via an `SsrGateway` abstraction (mirroring inertia-laravel's `Ssr\Gateway`). See [SSR example](./examples/ssr).

`SsrHTTPGateway` posts the Page JSON to a remote SSR server. Production uses `URL + "/render"`. For Vite hot mode set `UseHot` and `HotURL` (requests go to `HotURL + "/__inertia_ssr"`). `Endpoint` overrides both when you need a full URL:

```go
gateway := inertia.NewSsrHTTPGateway()
gateway.UseHot = true
gateway.HotURL = "http://localhost:5173"
renderer.SsrGateway = gateway
```

Exclude paths from SSR (inertia-laravel `$withoutSsr` / `withoutSsr`):

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	WithoutSsr: []string{"admin/*"},
	Renderer:   renderer,
}))
```

Or per request:

```go
inertia.DisableSsr(c)
inertia.WithoutSsr(c, "billing/*")
```

Custom Renderers should call `inertia.SsrEnabled(c)` (or `MustGet(c).SsrEnabled()`) instead of reading Inertia internals. Shared `SsrGateway` remains request-stateless; request skip state lives on Inertia.

For Vite development SSR, install `@inertiajs/vite`, configure the SSR entry explicitly, and let Inertia v3 provide React's default CSR/SSR setup:

```ts
import inertia from '@inertiajs/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [inertia({ ssr: 'resources/js/app.tsx' }), react()],
})
```

```tsx
import { createInertiaApp, type ResolvedComponent } from '@inertiajs/react'

const pages = import.meta.glob<{ default: ResolvedComponent }>('./pages/**/*.tsx', { eager: true })

createInertiaApp({
  resolve: name => pages[`./pages/${name}.tsx`],
})
```

The same `resources/js/app.tsx` entry is used for both the browser bundle and SSR. The Vite plugin wraps `createInertiaApp` for SSR builds and the development `/__inertia_ssr` endpoint.

The default SSR HTTP client does not set a request timeout. The current Echo request context is propagated to the SSR request, so its cancellation and deadline are respected. Configure an overall timeout when required by your application:

```go
gateway := inertia.NewSsrHTTPGateway()
gateway.HttpClient.Timeout = 5 * time.Second
renderer.SsrGateway = gateway
```

The five-second timeout above is only an example, not a library default.

Unlike inertia-laravel (which falls back to CSR and optionally fires `SsrRenderFailed`), `SsrHTTPGateway` returns errors to the caller — Go style, no built-in fallback or error callback. Handle failures in your `HTTPErrorHandler`, or wrap `SsrGateway` / use a custom `Renderer` if you want CSR-on-failure:

```go
type fallbackGateway struct{ inner inertia.SsrGateway }

func (g fallbackGateway) Dispatch(ctx *inertia.RenderContext) (*inertia.SsrResponse, error) {
	ssr, err := g.inner.Dispatch(ctx)
	if err != nil {
		slog.Error("SSR failed; falling back to CSR", "error", err)
		return nil, nil
	}
	return ssr, nil
}

renderer.SsrGateway = fallbackGateway{inner: inertia.NewSsrHTTPGateway()}
```

A `Dispatch` result of `(nil, nil)`, including the Vite development endpoint's temporary JSON `null` response while warming up, is treated as an intentional skip and renders the normal CSR bootstrap.

### Embed

You can bundle frontend builds into single Go binary using embed.

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
