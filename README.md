# Inertia Echo

[![test](https://github.com/kohkimakimoto/inertia-echo/actions/workflows/test.yml/badge.svg?branch=master)](https://github.com/kohkimakimoto/inertia-echo/actions/workflows/test.yml?query=branch%3Amaster)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/kohkimakimoto/inertia-echo/v5.svg)](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5)

This is the [Inertia.js](https://inertiajs.com) server-side adapter for [Echo](https://echo.labstack.com/) Go web framework.

[Inertia.js](https://inertiajs.com) is a JavaScript library that allows you to build a fully JavaScript-based single-page app without complexity.
I assume that you are familiar with Inertia.js and [how it works](https://inertiajs.com/how-it-works).
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
- [Usage](#usage)
  - [Renderer](#renderer)
  - [Middleware](#middleware)
  - [Responses](#responses)
    - [Creating responses](#creating-responses)
    - [Root template data](#root-template-data)
  - [Redirects](#redirects)
    - [External redirects](#external-redirects)
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
  - [Flash data](#flash-data)
  - [Preserving URL fragments](#preserving-url-fragments)
  - [Error responses](#error-responses)
  - [CSRF protection](#csrf-protection)
  - [History encryption](#history-encryption)
  - [Asset versioning](#asset-versioning)
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

In this tutorial, we will create the `views/app.html` file as the root template.

```html
<!DOCTYPE html>
<html>
  <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    {{- .inertiaHead -}}
  </head>
  <body>
  {{ .inertia }}
  {{ vite "js/app.jsx" }}
  </body>
</html>

```

### Write Go code

Next, you need to implement Go application code with the Echo framework. Create the `main.go` file with the following code:


```go
package main

import (
	"log/slog"

	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	r := inertia.NewHTMLRenderer()
	r.MustParseGlob("views/*.html")
	r.ViteBasePath = "/build"
	r.MustParseViteManifestFile("public/build/manifest.json")

	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	e.Use(inertia.CSRF())

	e.Static("/", "public")

	e.GET("/", func(c *echo.Context) error {
		return inertia.Render(c, "Index", map[string]any{
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
npm install -D @inertiajs/react@3.7.0 react@19 react-dom@19 vite@7 @vitejs/plugin-react@5
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
    outDir: "public/build",
    rollupOptions: {
      input: ['js/app.jsx'],
    },
  },
})
```

Create the `js/app.jsx` file with the following content:

```js
import { createInertiaApp } from '@inertiajs/react'

createInertiaApp({
  resolve: name => {
    const pages = import.meta.glob('./pages/**/*.jsx', { eager: true })
    return pages[`./pages/${name}.jsx`]
  },
})
```

Create a [page component](https://inertiajs.com/pages) as the  `js/pages/Index.jsx` file with the following content:

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

Then, open your browser and navigate to `http://localhost:8080`.
You should see the message "Hello, World!" displayed on the page.

You can find the complete code of this example in the [examples/getting-started](./examples/getting-started) directory of this repository.

### Run in Dev mode

If you want to run in dev mode so that you can hot-reload frontend updates, set `Debug` to `true`.

```go
r := inertia.NewHTMLRenderer()
// ...
r.Debug = true
r.ViteDevServerURL = "http://localhost:5173" // set this to change your Vite server host/port.
```

You need to run Vite server while you running in dev mode. To do so, you also can use [kohkimakimoto/go-subprocess](https://github.com/kohkimakimoto/go-subprocess) to simplify your operation with Vite server.

```go
import (
	"log/slog"
	"os"

	"github.com/kohkimakimoto/go-subprocess"
)

// ...
func main(){
	// ...

	e.GET("/", func(c *echo.Context) error {
		return inertia.Render(c, "Index", map[string]any{
			"message": "Hello, World!",
		})
	})

	// Add this to run Vite server.
	go func() {
		if err := subprocess.Run(&subprocess.Config{
			Command:         "bun",
			Args:            []string{"run", "dev"},
			Stdout:          os.Stdout,
			StdoutFormatter: subprocess.PrefixFormatter("[Vite] "),
			Stderr:          os.Stderr,
			StderrFormatter: subprocess.PrefixFormatter("[Vite] "),
			Dir:             ".",
		}); err != nil {
			slog.Error("the Vite subprocess returned an error", "error", err)
		}
	}()

	if err := e.Start(":8080"); err != nil {
		slog.Error("failed to start server", "error", err)
	}
}


```

> [!NOTE]
> If you're using React with `@vitejs/plugin-react`, you have to add `{{ vite_react_refresh }}` on your view file as well.  
> For more information, see [Vite docs](https://vitejs.dev/guide/backend-integration.html).
>
> `views/app.html`
>
> ```html
> <!DOCTYPE html>
> <html>
>   <head>
>     <meta charset="UTF-8" />
>     <meta name="viewport" content="width=device-width, initial-scale=1.0" />
>     {{- .inertiaHead -}}
>   </head>
>   <body>
>     {{ .inertia }} {{ vite_react_refresh }} {{ vite "js/app.jsx" }}
>   </body>
> </html>
> ```

## Usage

### Renderer

Unlike Laravel, which is an officially supported framework for Inertia.js, Echo lacks built-in view rendering.
This means you'll have to build your own view system and integrate it with Inertia.js.

Inertia Echo defines [`Renderer`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#Renderer) interface to integrate view system with Inertia.js.
It also provides a built-in renderer implementation based on the `html/template` package.

To setup Inertia Echo with your Echo application, you need to initialize the renderer and set it up with the [middleware](#middleware).

```go
// Create and configure the renderer...
r := inertia.NewHTMLRenderer()
r.MustParseGlob("views/*.html")
r.ViteBasePath = "/build"

// Setup the middleware with the renderer
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
  Renderer: r,
}))
```

### Middleware

After setting up the renderer, you need to add the Inertia middleware to your Echo application.

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	Renderer: r,
}))
```

The middleware handles Inertia requests, a foundational functionality of this package.
You can pass a configuration to customize its behavior.
For more details, see the [`MiddlewareConfig`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v5#MiddlewareConfig) documentation.

### Responses

:book: The related official document: [Responses](https://inertiajs.com/responses)

#### Creating responses

The following code shows how to create an Inertia response.
The `Render` function accepts a `map[string]any` as its final argument, which contains the properties to pass to the view.

```go
func ShowEventHandler(c *echo.Context) error {
	event := // retrieve a event...
	return inertia.Render(c, "Event/Show", map[string]any{
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
	return inertia.RenderWithViewData(c, "Event/Show", map[string]any{
		"event": event,
	}, map[string]interface{}{
		"meta": "Meta data...",
	})
}
```

You can then access this variable like a regular template variable.

```html
<meta name="twitter:title" content="{{ .meta }}">
```

### Redirects

:book: The related official document: [Redirects](https://inertiajs.com/redirects)

You can use Echo's standard way to redirect.

```go
return c.Redirect(http.StatusFound, "/")
```

#### External redirects

The following is a way to redirect to an external website in Inertia apps.

```go
return inertia.Location(c, "/path/to/external")
```

### Routing

:book: The related official document: [Routing](https://inertiajs.com/routing)

#### Shorthand routes

Inertia Echo provides a helper function for shorthand routes

```go
e.GET("/about", inertia.Handler("About"))
```

### Shared data

:book: The related official document: [Shared data](https://inertiajs.com/shared-data)

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
}))
```

#### Sharing data manually

Alternatively, you can manually share data using the `Share` function.

```go
inertia.Share(c, map[string]any{
	"appName":  "App Name",
	"authUser": user,
})
```

### Partial reloads

:book: The related official document: [Partial reloads](https://inertiajs.com/partial-reloads)

```go
inertia.Render(c, "Users/Index", map[string]any{
	// ALWAYS included on standard visits
	// OPTIONALLY included on partial reloads
	// ALWAYS evaluated
	"users": users,

	// ALWAYS included on standard visits
	// OPTIONALLY included on partial reloads
	// ONLY evaluated when needed
	"users": func() (any, error) {
		users, err := // get users...
		if err != nil {
			return nil, err
		}
		return users
	},

	// NEVER included on standard visits
	// OPTIONALLY included on partial reloads
	// ONLY evaluated when needed
	"users": inertia.Optional(func() (any, error) {
		users, err := // get users...
		if err != nil {
			return nil, err
		}
		return users, nil
	}),

	// ALWAYS included on standard visits
	// ALWAYS included on partial reloads
	// ALWAYS evaluated
	"users": inertia.Always(users),
})
```

### Deferred props

:book: The related official document: [Deferred props](https://inertiajs.com/deferred-props)

```go
inertia.Render(c, "Users/Index", map[string]any{
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
inertia.Render(c, "Users/Index", map[string]any{
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

inertia.Render(c, "Users/Index", map[string]any{
	"permissions": inertia.Defer(func() (any, error) {
		return loadPermissions()
	}).Rescue(),
})
```

### Once props

Once props let the client reuse a previously loaded value. The server remains stateless; the client reports loaded keys in later requests.

```go
inertia.Render(c, "Dashboard", map[string]any{
	"settings": inertia.Once(func() (any, error) {
		return loadSettings()
	}).As("dashboard-settings").For(10 * time.Minute),
})
```

Use `Fresh(true)` to resolve the value even when the client already has it, `Until(time.Time)` to set an absolute expiration, or `For(time.Duration)` to set a relative expiration. `OptionalProp`, `DeferProp`, and `MergeProp` can opt in with `Once()`; calling `As`, `Fresh`, `Until`, or `For` on those prop types also enables once behavior. `ShareOnce` is a shortcut for sharing a callback-backed once prop.

### Merging props

:book: The related official document: [Merging props](https://inertiajs.com/merging-props)

#### Shallow merge

```go
inertia.Render(c, "Tags/Index", map[string]any{
	"tags": inertia.Merge(tags),
})
```

#### Deep merge

```go
inertia.Render(c, "Users/Index", map[string]any{
	"tags": inertia.DeepMerge(users),
})
```

You may chain the matchOn method to determine how existing items should be matched and updated.

```go
inertia.Render(c, "Users/Index", map[string]any{
	"tags": inertia.DeepMerge(users).MatchOn("data.id"),
})
```

Use `Append` or `Prepend` to select the client-side merge direction at the root or at nested paths:

```go
inertia.Render(c, "Users/Index", map[string]any{
	"users": inertia.Merge(users).Append("data").MatchOn("data.id"),
	"activity": inertia.Merge(activity).Prepend(),
})
```

### Infinite scroll props

`Scroll` receives the page value and pagination metadata together, so loading the data does not need to run twice. Page identifiers may be numbers, cursor strings, or `nil`.

```go
inertia.Render(c, "Users/Index", map[string]any{
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

### Flash data

`Flash` adds one-time data to the top-level Page `flash` object for the current render:

```go
inertia.Flash(c, map[string]any{
	"message": "Profile updated.",
})
```

The adapter does not persist arbitrary flash values in cookies. For POST-redirect-GET flows, store the data in the application's session implementation and connect it with the middleware's `FlashData` and `Reflash` callbacks:

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	Renderer:  r,
	FlashData: loadFlashFromSession,
	Reflash:   reflashSessionData,
}))
```

`FlashData` has the signature `func(*echo.Context) (map[string]any, error)`. `Reflash` has the signature `func(*echo.Context) error`.

### Preserving URL fragments

Call `PreserveFragment` before a redirect to ask the client to retain the source visit's URL fragment on the next rendered page:

```go
inertia.PreserveFragment(c)
return c.Redirect(http.StatusFound, "/account")
```

For Inertia requests, redirects whose destination already contains a fragment are converted to the v3 `409` and `X-Inertia-Redirect` response. Prefetch requests are excluded from this conversion.

### Error responses

Use `RenderWithStatus` when an Inertia page must retain an HTTP error status:

```go
return inertia.RenderWithStatus(c, http.StatusNotFound, "ErrorPage", map[string]any{
	"status": http.StatusNotFound,
})
```

For centralized error handling, reuse the middleware configuration so the error page has the same renderer, root view, version, and SSR settings:

```go
inertiaConfig := inertia.MiddlewareConfig{Renderer: r}
e.Use(inertia.MiddlewareWithConfig(inertiaConfig))

e.HTTPErrorHandler = inertia.HTTPErrorHandlerWithConfig(inertia.ErrorHandlerConfig{
	Middleware: inertiaConfig,
	Component:  "ErrorPage",
	Statuses: []int{
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
		http.StatusServiceUnavailable,
	},
})
```

The default props contain `status`. Set `Props` to build application-specific props, `ResolveShared` to load shared props when no Inertia request context exists, and `Fallback` for statuses not handled as Inertia pages. If the application keeps its own Echo error handler, wrap it with `WrapHTTPErrorHandler` so Inertia redirect conversion and one-shot state finalization also run for error-handler responses.

### CSRF protection

:book: The related official document: [CSRF protection](https://inertiajs.com/csrf-protection)

Inertia Echo has CSRF middleware that is configured for Inertia.js.
This middleware provides `XSRF-TOKEN` cookie and verifies the `X-XSRF-TOKEN` header in the request.

The following code shows how to set up the CSRF middleware in your Echo application.

```go
e.Use(inertia.CSRF())
```

### History encryption

:book: The related official document: [History encryption](https://inertiajs.com/history-encryption)

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

### Asset versioning

:book: The related official document: [Asset versioning](https://inertiajs.com/asset-versioning)

Configure asset version via middleware.

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	VersionFunc: func() string { return version },
}))
```

Configure asset version manually.

```go
inertia.SetVersion(c, func() string { return version })
```

### Server-side Rendering (SSR)

:book: The related official document: [Server-side Rendering (SSR)](https://inertiajs.com/server-side-rendering)

Inertia Echo supports SSR. See [SSR example](./examples/ssr).

The HTTP gateway uses `URL + "/render"` for a standalone production SSR server. Set `Endpoint` when the complete SSR endpoint differs, including the Inertia Vite plugin's development endpoint:

```go
gateway := inertia.NewSsrEngineHTTPGateway()
gateway.Endpoint = "http://localhost:5173/__inertia_ssr"
renderer.SsrEngine = gateway
```

For Vite development SSR, install `@inertiajs/vite@3.7.0`, configure the SSR entry explicitly, and let Inertia v3 provide React's default CSR/SSR setup:

```ts
import inertia from '@inertiajs/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [inertia({ ssr: 'assets/ssr.tsx' }), react()],
})
```

```tsx
import { createInertiaApp, type ResolvedComponent } from '@inertiajs/react'

const pages = import.meta.glob<{ default: ResolvedComponent }>('./pages/**/*.tsx', { eager: true })

createInertiaApp({
  resolve: name => pages[`./pages/${name}.tsx`],
})
```

The default SSR HTTP client does not set a request timeout. The current Echo request context is propagated to the SSR request, so its cancellation and deadline are respected. Configure an overall timeout when required by your application:

```go
gateway := inertia.NewSsrEngineHTTPGateway()
gateway.HttpClient.Timeout = 5 * time.Second
renderer.SsrEngine = gateway
```

The five-second timeout above is only an example, not a library default. SSR request errors are returned by default. Enable an explicit CSR fallback and register an error reporter when the application should remain available after an SSR failure:

```go
renderer.SsrFallbackOnError = true
renderer.SsrErrorReporter = func(ctx *inertia.RenderContext, err error) {
	logger.Error("SSR failed; falling back to CSR",
		"component", ctx.Page.Component,
		"url", ctx.Page.URL,
		"error", err,
	)
}
```

Request cancellation is always returned instead of being hidden by the fallback. An SSR engine response of `(nil, nil)`, including the Vite development endpoint's temporary JSON `null` response while warming up, is treated as an intentional skip and renders the normal CSR bootstrap.

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

//go:embed views/*.html
var viewFiles embed.FS

//go:embed public/*
var publicFiles embed.FS

func main() {
	e := echo.New()
	// ...

	r := inertia.NewHTMLRenderer()
	r.MustParseFS(viewFiles, "views/*.html")
	r.MustParseViteManifestFS(publicFiles, "public/build/manifest.json")
	// ...

	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	// ...

	fsys, err := fs.Sub(publicFiles, "public")
	if err != nil {
		panic(err)
	}

	assetHandler := http.FileServer(http.FS(fsys))
	e.GET("/*", echo.WrapHandler(http.StripPrefix("/", assetHandler)))

	e.GET("/", func(c *echo.Context) error {
		return inertia.Render(c, "Index", map[string]any{
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
