// Package slowjs provides an in-process Inertia SSR gateway backed by
// [liustack/slowjs].
//
// The gateway expects a self-contained IIFE that exposes
// globalThis.renderPage(page) -> { head, body }. See the package README and
// examples/ssr-slowjs for the Vite build shape.
package slowjs
