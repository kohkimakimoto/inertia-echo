module github.com/kohkimakimoto/inertia-echo/examples/getting-started

go 1.23.0

replace github.com/kohkimakimoto/inertia-echo/v2 => ../..


require (
	github.com/kohkimakimoto/go-subprocess v0.2.0
	github.com/kohkimakimoto/inertia-echo/v2 v2.0.0
	github.com/labstack/echo/v4 v4.13.4
)
