module github.com/kohkimakimoto/inertia-echo/examples/getting-started

go 1.25.0

replace github.com/kohkimakimoto/inertia-echo/v5 => ../..

require (
	github.com/kohkimakimoto/inertia-echo/v5 v5.0.0
	github.com/labstack/echo/v5 v5.3.1
)

require (
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)
