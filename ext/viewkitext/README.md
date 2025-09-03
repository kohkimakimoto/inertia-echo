# Echo ViewKit Extension

This is an Inertia Echo extension that provides a renderer for integrating with [Echo ViewKit](https://github.com/kohkimakimoto/echo-viewkit).

## Usage

Go code like the following:

```go
// Echo ViewKit
v := viewkit.New()
v.BaseDir = "views"
// Configure the Echo ViewKit instance
// ...

// Create Echo ViewKit Renderer　and wrap it for Inertia
r := viewkitext.NewRenderer(v.MustRenderer())

// Setup Inertia middleware with the renderer
e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
	Renderer: r,
}))
```

And then, you can use Echo ViewKit template functionality like this:

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  {{- inertiaHead -}}
</head>
<body>
{{- inertia -}}
{{- vite_react_refresh -}}
{{- vite("assets/app.tsx") -}}
</body>
</html>
```

See also [Echo ViewKit official website](https://echo-viewkit.kohkimakimoto.dev/).
