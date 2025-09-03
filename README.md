# Inertia Echo

[![test](https://github.com/kohkimakimoto/inertia-echo/actions/workflows/test.yml/badge.svg)](https://github.com/kohkimakimoto/inertia-echo/actions/workflows/test.yml)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/kohkimakimoto/inertia-echo/blob/master/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/kohkimakimoto/inertia-echo.svg)](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v2)

This is the [Inertia.js](https://inertiajs.com) server-side adapter for [Echo](https://echo.labstack.com/) Go web framework.

[Inertia.js](https://inertiajs.com) is a JavaScript library that allows you to build a fully JavaScript-based single-page app without complexity.
I assume that you are familiar with Inertia.js and [how it works](https://inertiajs.com/how-it-works).
You also need to familiarize yourself with [Echo](https://echo.labstack.com/), a Go web framework.
Inertia Echo assists you in developing web applications that leverage both of these technologies.

Table of Contents

- [Getting started](#getting-started)
  - [Installation](#installation)
  - [Root template](#root-template)
  - [Write Go code](#write-go-code)
  - [Setup frontend](#setup-frontend)
  - [Run the application](#run-the-application)
- [Usage](#usage)
  - [Renderer](#renderer)
  - [Middleware](#middleware)
  - [Responses](#responses)
    - [Creating responses](#creating-responses)
    - [Creating responses using structs](#creating-responses-using-structs)
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
  - [Merging props](#merging-props)
  - [CSRF protection](#csrf-protection)
  - [History encryption](#history-encryption)
  - [Asset versioning](#asset-versioning)
  - [Server-side Rendering (SSR)](#server-side-rendering-ssr)
- [Author](#author)
- [License](#license)

## Getting started

In this section, we provide step-by-step instructions on how to get started with Inertia Echo.

### Installation

Inertia Echo is a Go module that you can install with the following command:

```sh
go get github.com/kohkimakimoto/inertia-echo/v2
```

You also need to install Echo like this:

```sh
go get github.com/labstack/echo/v4
```

### Root template

Next, setup the root template that will be loaded on the first page visit to your application. This template should include your site's CSS and JavaScript assets, along with the `.inertia` and `.inertiaHead` variables.

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
	"net/http"

	inertia "github.com/kohkimakimoto/inertia-echo/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	r := inertia.NewHTMLRenderer()
	r.MustParseGlob("views/*.html")
	r.ViteBasePath = "/build"
	r.MustParseViteManifestFile("public/build/manifest.json")

	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	e.Use(inertia.CSRF())

	e.Static("/", "public")

	e.GET("/", func(c echo.Context) error {
		return inertia.Render(c, "Index", map[string]any{
			"message": "Hello, World!",
		})
	})

	e.Logger.Fatal(e.Start(":8080"))
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
npm install -D @inertiajs/react react react-dom vite @vitejs/plugin-react
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
import { createRoot } from 'react-dom/client'

createInertiaApp({
  resolve: name => {
    const pages = import.meta.glob('./pages/**/*.jsx', { eager: true })
    return pages[`./pages/${name}.jsx`]
  },
  setup({ el, App, props }) {
    createRoot(el).render(<App {...props} />)
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

## Usage

### Renderer

Unlike Laravel, which is an officially supported framework for Inertia.js, Echo lacks built-in view rendering.
This means you'll have to build your own view system and integrate it with Inertia.js.

Inertia Echo defines [`Renderer`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v2#Renderer) interface to integrate view system with Inertia.js.
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

> [!NOTE]
> We also officially provide [`Echo Viewkit`](https://github.com/kohkimakimoto/echo-viewkit) renderer as an additional module.
> It is a recommended renderer because it provides more powerful Vite support.
> For more information see [viewkitext](./ext/viewkitext) module and the [example code](./examples/viewkit).

### Middleware

After setting up the renderer, you need to add the Inertia middleware to your Echo application.

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	Renderer: r,
}))
```

The middleware handles Inertia requests, a foundational functionality of this package.
You can pass a configuration to customize its behavior.
For more details, see the [`MiddlewareConfig`](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo/v2#MiddlewareConfig) documentation.

### Responses

:book: The related official document: [Responses](https://inertiajs.com/responses)

#### Creating responses

The following code shows how to create an Inertia response.
The `Render` function accepts a `map[string]any` as its final argument, which contains the properties to pass to the view.

```go
func ShowEventHandler(c echo.Context) error {
	event := // retrieve a event...
	return inertia.Render(c, "Event/Show", map[string]any{
		"event": event,
	})
}
```

#### Creating responses using structs

You can also pass a struct with the `prop` struct tag.

```go
type ShowEventProps struct {
	Event *Event `prop:"event"`
}

func ShowEventHandler(c echo.Context) error {
	event := // retrieve a event...
	return inertia.Render(c, "Event/Show", &ShowEventProps{
		Event: event,
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
func ShowEventsHandler(c echo.Context) error {
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
	Share: func(c echo.Context) (map[string]any, error) {
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
	"tags": inertia.DeepMerge(users).MatchesOn("data.id"),
})
```

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

## Author

Kohki Makimoto <kohki.makimoto@gmail.com>

## License

The MIT License (MIT)
