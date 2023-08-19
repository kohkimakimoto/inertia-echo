# inertia-echo

[![test](https://github.com/kohkimakimoto/inertia-echo/actions/workflows/test.yml/badge.svg)](https://github.com/kohkimakimoto/inertia-echo/actions/workflows/test.yml)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/kohkimakimoto/inertia-echo/blob/main/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/kohkimakimoto/inertia-echo.svg)](https://pkg.go.dev/github.com/kohkimakimoto/inertia-echo)

The [Inertia.js](https://inertiajs.com) server-side adapter for [Echo](https://echo.labstack.com/) Go web framework.

[Inertia.js](https://inertiajs.com) is a JavaScript library that allows you to build a fully JavaScript-based single-page app without complexity.
I assume that you are familiar with Inertia.js and [how it works](https://inertiajs.com/how-it-works).
You also need to familiarize yourself with [Echo](https://echo.labstack.com/), a Go web framework. The inertia-echo assists you in developing web applications that leverage both of these technologies.

## Installation

```sh
go get github.com/kohkimakimoto/inertia-echo
```

## Minimum example

Please see [Hello World](https://github.com/kohkimakimoto/inertia-echo/tree/master/examples/helloworld) example.

## Usage

### Shorthand routes

The inertia-echo provides a helper function for shorthand routes like [Official Laravel Adapter](https://inertiajs.com/routing#shorthand-routes).

```go
e.GET("/about", inertia.Handler("About"))
```

See also the official document: [Routing](https://inertiajs.com/routing)

### Responses

Creating responses.

```go
func ShowEventsHandler(c echo.Context) error {
	event := // retrieve a event...
	return inertia.Render(c, http.StatusOK, "Event/Show", map[string]interface{}{
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
	return inertia.RenderWithViewData(c, http.StatusOK, "Event/Show", map[string]interface{}{
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

See also the official document: [Responses](https://inertiajs.com/responses)

### Redirects

You can use Echo's standard way to redirect.

```go
return c.Redirect(http.StatusFound, "/")
```

The following is a way to redirect to an external website in Inertia apps.

```go
return inertia.Location(c, "/path/to/external")
```

See also the official document: [Redirects](https://inertiajs.com/redirects)

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
inertia.Share(c, map[string]interface{}{
	"AppName":  "App Name",
	"AuthUser": user,
})
```

See also the official document: [Shared data](https://inertiajs.com/shared-data)

### Partial reloads

```go
inertia.Render(c, http.StatusOK, "Index", map[string]interface{}{
	// ALWAYS included on first visit
	// OPTIONALLY included on partial reloads
	// ALWAYS evaluated
	"users": users,

	// ALWAYS included on first visit...
	// OPTIONALLY included on partial reloads...
	// ONLY evaluated when needed...
	"users": func() (interface{}, error) {
    users, err := // get users...
    if err != nil {
        return nil, err
    }
		return users
	},

	// NEVER included on first visit
	// OPTIONALLY included on partial reloads
	// ONLY evaluated when needed
	"users": inertia.Lazy(func() (interface{}, error) {
		users, err := // get users...
		if err != nil {
      return nil, err
    }
		return users, nil
	}),
})
```

See also the official document: [Partial reloads](https://inertiajs.com/partial-reloads)

### Asset versioning

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

See also the official document: [Assset versioning](https://inertiajs.com/asset-versioning)

### Server-side Rendering (SSR)

The inertia-echo supports SSR. Please see [SSR Node.js](https://github.com/kohkimakimoto/inertia-echo/tree/master/examples/ssrnodejs) example.

See also the official document: [Server-side Rendering (SSR)](https://inertiajs.com/server-side-rendering)

## Unsupported features

### Validation

The inertia-echo does not support validation, as Echo lacks built-in validation.
The implementation of validation is up to you.
If you wish to handle validation errors with inertia-echo, you will need to implement it yourself.

See also the official document: [Validation](https://inertiajs.com/validation)

## Demo application

- [Hello World](https://github.com/kohkimakimoto/inertia-echo/tree/master/examples/helloworld)
- [SSR Node.js](https://github.com/kohkimakimoto/inertia-echo/tree/master/examples/ssrnodejs)
- [pingcrm-echo](https://github.com/kohkimakimoto/pingcrm-echo) (but it was implemented with the old version of inertia-echo)

## Author

Kohki Makimoto <kohki.makimoto@gmail.com>

## License

The MIT License (MIT)
