# Upgrading to v4

Inertia Echo v4 is the compatibility line for Echo v4. It requires Go 1.25 or later and uses Echo v4.15.4 or later.

## Update the module path

Update the dependency and every Inertia Echo import from `/v2` to `/v4`:

```sh
go get github.com/kohkimakimoto/inertia-echo/v4@v4
```

```go
import inertia "github.com/kohkimakimoto/inertia-echo/v4"
```

The Echo import path remains unchanged:

```go
import "github.com/labstack/echo/v4"
```

## Removed session integration

The session-backed validation error integration that was available in v2.1.0 has been removed. Echo does not provide a built-in session abstraction, so applications should select and configure their own session implementation and pass validation errors through Inertia props.

The removed API includes:

- `MiddlewareConfig.Session`, `SessionName`, and `SessionOptions`
- `ErrSessionStoreNotRegistered`
- `ErrorMessageMap` and `NewErrorMessageMap`
- `Session`, `MustSession`, `SaveSession`, and `MustSaveSession`
- `ErrorMessages`, `UpdateErrorMessages`, `UpdateErrorMessagesWithSession`, `MustUpdateErrorMessagesWithSession`, and `SyncErrorMessagesSession`

## Removed ViewKit extension

The `github.com/kohkimakimoto/inertia-echo/ext/viewkitext/v2` extension is no longer provided or maintained beginning with Inertia Echo v4. Use the built-in `HTMLRenderer` or implement the `Renderer` interface for another view system.
