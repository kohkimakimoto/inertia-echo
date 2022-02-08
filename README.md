# inertia-echo

[![test](https://github.com/kohkimakimoto/inertia-echo/actions/workflows/test.yml/badge.svg)](https://github.com/kohkimakimoto/inertia-echo/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/kohkimakimoto/inertia-echo.svg)](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo)

The [Inertia.js](https://inertiajs.com) server-side adapter for [Echo](https://echo.labstack.com/) Go web framework.

[Inertia.js](https://inertiajs.com) is a JavaScript library that allows you to build a fully JavaScript-based single-page app without complexity.
I assume that you know [what Inertia.js is](https://inertiajs.com/who-is-it-for) and [how it works](https://inertiajs.com/how-it-works).
You also need to know [Echo](https://echo.labstack.com/) that is a Go web framework. The inertia-echo helps you to develop web apps based on them.

## Installation

```sh
go get github.com/kohkimakimoto/inertia-echo
```

## Usage

### Minimum example

Create the root template `views/app.html`.

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body>
<div id="app" data-page="{{ json_marshal .page }}"></div>
<!--
Echo does not have built-in frontend library.
You have to set up frontend environment yourself such as Webpack, Vite, etc.
Replace the following script tag with your environment.
-->
<script src="/path/to/bundle.js"></script>
</body>
</html>
```

Next, write server-side app with [Echo](https://echo.labstack.com/)

```go
package main

import (
	"net/http"

	"github.com/kohkimakimoto/inertia-echo"
	"github.com/labstack/echo/v4"
)

// I recommend to define the alias of `inertia.MustGet` to simplify the code.
// The code examples in this document are written with this alias.
var Inertia = inertia.MustGet

func main() {
	e := echo.New()
	// setup renderer to load the root template.
	e.Renderer = inertia.NewRenderer("views/*.html", nil)

	// The middleware is needed to handle inertia protocol.
	e.Use(inertia.Middleware())
	e.Use(inertia.CSRF())

	// handlers
	e.GET("/", func(c echo.Context) error {
		// Instead of using c.Render(), the following code can render inertia response.
		return Inertia(c).Render(http.StatusOK, "Index", map[string]interface{}{
			"message": "Hello, World!",
		})
	})

	// start server
	e.Logger.Fatal(e.Start(":8080"))
}
```

You need to setup your client-side app. See [Client-side setup](https://inertiajs.com/client-side-setup) in official document.

### Routing

Inertia-echo provides route helpers like [Official Laravel Adapter](https://inertiajs.com/routing#route-helpers).

```go
e.GET("/about", inertia.Handler("About"))
```

See also official document: [Routing](https://inertiajs.com/routing)

### Responses

Creating responses.


```go
func ShowEventsHandler(c echo.Context) error {
	event := // retrieve a event...
	return Inertia(c).Render(http.StatusOK, "Event/Show", map[string]interface{}{
		"Event": event,
	})
}
```

Root template data.

```html
<meta name="twitter:title" content="{{ .page.Props.Event.Title }}">
```

Sometimes you may even want to provide data that will not be sent to your JavaScript component.

```go
func ShowEventsHandler(c echo.Context) error {
	event := // retrieve a event...
	return Inertia(c).RenderWithViewData(http.StatusOK, "Event/Show", map[string]interface{}{
		"Event": event,
	}, map[string]interface{}{
		"Meta": "Meta data...",
	})
}
```

You can then access this variable like a regular template variable.

```html
<meta name="twitter:title" content="{{ .Meta }}">
```

See also official document: [Responses](https://inertiajs.com/responses)

### Redirects

You can use Echo's standard way to redirect.

```go
return c.Redirect(http.StatusFound, "/")
```

The following is a way to redirect to an external website in Inertia apps.

```go
return Inertia(c).Location("/path/to/external")
```

See also official document: [Redirects](https://inertiajs.com/redirects)

### Shared data

Set shared data via middleware.

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	Share: func(c echo.Context) (map[string]interface{}, error) {
		user := // get auth user...
		return map[string]interface{}{
			"AppName":  "App Name",
			"AuthUser": user,
		}, nil
	},
}))
```

Set shared data manually.

```go
Inertia(c).Share(map[string]interface{}{
	"AppName":  "App Name",
	"AuthUser": user,
})
```

See also official document: [Shared data](https://inertiajs.com/shared-data)

### Partial reloads

```go

Inertia(c).Render(http.StatusOK, "Index", map[string]interface{}{
	// ALWAYS included on first visit
	// OPTIONALLY included on partial reloads
	// ALWAYS evaluated
	"users": users,
	"users": func() interface{} {
		users := // get users...
		return users
	},

	// NEVER included on first visit
	// OPTIONALLY included on partial reloads
	// ONLY evaluated when needed
	"users": inertia.Lazy(func() interface{} {
		users := // get users...
		return users
	}),
})
```

See also official document: [Partial reloads](https://inertiajs.com/partial-reloads)

### Asset versioning

Configure asset version via middleware.

```go
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	VersionFunc: func() string { return version },
}))
```

Configure asset version manually.

```go
Inertia(c).SetVersion(func() string { return version })
```

See also official document: [Assset versioning](https://inertiajs.com/asset-versioning)

## Unsupported features

### Validation

The inertia-echo does not support validation because echo does not have built-in validation.
Validation implementation is up to you. If you want to handle validation errors with inertia-echo, You have to implement it yourself.

See also official document: [Validation](https://inertiajs.com/validation)

## Author

Kohki Makimoto <kohki.makimoto@gmail.com>

## License

The MIT License (MIT)
