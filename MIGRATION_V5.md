# Upgrading to v5

Inertia Echo v5 is the compatibility line for Echo v5. It requires Go 1.25 or later and Echo v5.3.1 or later.

Inertia Echo and Echo align their major versions for compatibility. Their minor and patch versions do not need to match.

## Update the module paths

Update the dependency and every Inertia Echo import from `/v4` to `/v5`:

```sh
go get github.com/kohkimakimoto/inertia-echo/v5@v5
```

```go
import inertia "github.com/kohkimakimoto/inertia-echo/v5"
```

Update Echo and its subpackage imports from `/v4` to `/v5`:

```sh
go get github.com/labstack/echo/v5@v5.3.1
```

```go
import (
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)
```

Run `go mod tidy` after updating the imports. Check `go list -m all` and remove dependencies that still introduce `github.com/labstack/echo/v4`; Echo v4 and v5 middleware and handler types are not interchangeable.

## Update Echo context parameters

Echo v5 changed `echo.Context` from an interface to a concrete type. Handlers and callbacks now receive `*echo.Context`:

```go
e.GET("/", func(c *echo.Context) error {
	return inertia.Render(c, "Index", nil)
})
```

This affects Inertia Echo APIs that accept or expose the Echo context, including:

- `SharedDataFunc` and `MiddlewareConfig.Share`
- `Get`, `MustGet`, and `Has`
- `Render`, `RenderWithViewData`, `Location`, `Share`, and the other context helpers
- `Inertia.EchoContext`
- Echo middleware skippers and application handlers passed to Inertia Echo middleware

## Update Echo logging

Echo v5 uses the standard library's `log/slog` package. Replace the removed request logger middleware:

```go
e.Use(middleware.RequestLogger())
```

Replace `middleware.Logger()` and formatted methods such as `Errorf` or `Debugf`. Use structured slog calls instead:

```go
c.Logger().Debug("user authenticated", "email", email)
slog.Error("failed to start server", "error", err)
```

## Configure debug mode outside Echo

Echo v5 removed `Echo.Debug`. Keep the application setting separately when configuring the Inertia renderer:

```go
debug := isDebug()

r := inertia.NewHTMLRenderer()
r.Debug = debug
```

## Review other Echo v5 API changes

Echo v5 also changes response handling, server configuration, routing, and several middleware APIs. Review the official [Echo v5 API changes](https://github.com/labstack/echo/blob/v5.3.1/API_CHANGES_V5.md) and [Echo changelog](https://github.com/labstack/echo/blob/v5.3.1/CHANGELOG.md) for application code that uses Echo directly.

In particular, `Context.Response()` now returns an `http.ResponseWriter`. Code that used fields on Echo v4's response wrapper must use the standard response writer API or `echo.UnwrapResponse` where Echo-specific response state is required.

## Example changes

The `login-form` example is not included in v5 because its session dependency supports Echo v4 only. This does not remove session support from Echo itself or prevent applications from using an Echo v5-compatible session package.

## Changes inherited from v4

The latest v4 behavior fixes are included in v5:

- Existing `Vary` response values are preserved when `X-Inertia` is added.
- Wrapped response writers retain `http.Flusher` and `http.Hijacker` compatibility.
- Asset-version conflict responses preserve the complete request URL, including its query string.
- SSR requests inherit cancellation and deadlines from the current Echo request context.

## Support branches

Inertia Echo v4 remains the maintenance line for Echo v4 on the [`v4` branch](https://github.com/kohkimakimoto/inertia-echo/tree/v4). New Echo v5 development is released as Inertia Echo v5.

Do not merge v5 code into an application's v4 dependency line without also completing the Echo v5 migration.

## Migrating directly from v2

If you are migrating directly from Inertia Echo v2, first review [Upgrading to v4](./MIGRATION_V4.md). The session integration and ViewKit extension removed in v4 remain unavailable in v5. Then apply the v5 changes in this guide.
