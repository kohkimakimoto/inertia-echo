module github.com/kohkimakimoto/inertia-echo/examples/viewkit

go 1.23.0

replace (
	github.com/kohkimakimoto/inertia-echo/ext/viewkitext/v2 => ../../ext/viewkitext
	github.com/kohkimakimoto/inertia-echo/v2 => ../..
)

require (
	github.com/kohkimakimoto/echo-viewkit v0.7.0
	github.com/kohkimakimoto/go-subprocess v0.2.0
	github.com/kohkimakimoto/inertia-echo/ext/viewkitext/v2 v2.0.0
	github.com/kohkimakimoto/inertia-echo/v2 v2.1.0
	github.com/labstack/echo/v4 v4.13.4
)

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	golang.org/x/time v0.12.0 // indirect
)
