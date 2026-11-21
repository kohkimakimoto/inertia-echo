module github.com/kohkimakimoto/inertia-echo/examples/ssr-bench

go 1.26.6

require (
	github.com/kohkimakimoto/inertia-echo/v5 v5.0.0
	github.com/liustack/slowjs v0.31.10
)

require (
	github.com/labstack/echo/v5 v5.4.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/time v0.15.0 // indirect
)

replace github.com/kohkimakimoto/inertia-echo/v5 => ../..
