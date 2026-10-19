module github.com/kohkimakimoto/inertia-echo/examples/notes

go 1.26.0

replace github.com/kohkimakimoto/inertia-echo/v5 => ../..

require (
	github.com/kohkimakimoto/echo-session/v5 v5.0.0
	github.com/kohkimakimoto/go-subprocess v0.5.0
	github.com/kohkimakimoto/inertia-echo/v5 v5.0.0
	github.com/labstack/echo/v5 v5.4.0
)

require (
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/gorilla/sessions v1.4.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)
