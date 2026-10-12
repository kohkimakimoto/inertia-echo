package inertia

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
)

const (
	HeaderXInertia                          = "X-Inertia"
	HeaderXInertiaErrorBag                  = "X-Inertia-Error-Bag"
	HeaderXInertiaLocation                  = "X-Inertia-Location"
	HeaderXInertiaVersion                   = "X-Inertia-Version"
	HeaderXInertiaPartialComponent          = "X-Inertia-Partial-Component"
	HeaderXInertiaPartialData               = "X-Inertia-Partial-Data"
	HeaderXInertiaPartialExcept             = "X-Inertia-Partial-Except"
	HeaderXInertiaReset                     = "X-Inertia-Reset"
	HeaderXInertiaExceptOnceProps           = "X-Inertia-Except-Once-Props"
	HeaderXInertiaInfiniteScrollMergeIntent = "X-Inertia-Infinite-Scroll-Merge-Intent"
	HeaderXInertiaRedirect                  = "X-Inertia-Redirect"
)

// Inertia is a echo.Context wrapper that handles Inertia.js protocol.
type Inertia struct {
	echoContext      *echo.Context
	rootView         string
	sharedProps      map[string]any
	sharedPropsMutex sync.RWMutex
	version          VersionFunc
	urlResolver      URLResolverFunc
	renderer         Renderer
	encryptHistory   bool
	stateStore       StateStore
	state            map[string]string
	stateDirty       bool
	clearHistory     bool
	preserveFragment bool
	flash            map[string]any
	flashLoaded      bool
	rescueReporter   RescueReporter
	now              func() time.Time
	withoutSsr       []string
	ssrDisabled      bool
	ssrDisableWhen   func(c *echo.Context) bool
	partialComponent string
	onlyProps        []string
	exceptProps      []string
	resetProps       []string
}

func (i *Inertia) EchoContext() *echo.Context {
	return i.echoContext
}

func (i *Inertia) SetRenderer(r Renderer) {
	i.renderer = r
}

func (i *Inertia) Renderer() Renderer {
	return i.renderer
}

func (i *Inertia) EncryptHistory(encrypt bool) {
	i.encryptHistory = encrypt
}

// ClearHistory clears the history.
// see https://inertiajs.com/docs/v3/security/history-encryption
func (i *Inertia) ClearHistory() {
	i.clearHistory = true
}

// pullClearHistory pulls the clear-history flag from state or the current field.
// Laravel uses session for this; Echo has no built-in session, so the default
// StateStore uses an unsigned cookie (see StateStore security notes).
func (i *Inertia) pullClearHistory() bool {
	return i.pullBooleanFlag(StateKeyClearHistory, &i.clearHistory)
}

// PreserveFragment asks the client to carry the source URL fragment through
// the next rendered page. The flag survives across requests via StateStore.
func (i *Inertia) PreserveFragment() {
	i.preserveFragment = true
}

func (i *Inertia) pullPreserveFragment() bool {
	return i.pullBooleanFlag(StateKeyPreserveFragment, &i.preserveFragment)
}

func (i *Inertia) stateGet(key string) (string, bool) {
	if i.state == nil {
		return "", false
	}
	value, ok := i.state[key]
	return value, ok
}

func (i *Inertia) stateSet(key string, value string) {
	if i.state == nil {
		i.state = map[string]string{}
	}
	i.state[key] = value
	i.stateDirty = true
}

func (i *Inertia) stateDelete(key string) {
	if i.state == nil {
		return
	}
	if _, ok := i.state[key]; !ok {
		return
	}
	delete(i.state, key)
	i.stateDirty = true
}

// statePull is Get + Delete — the flash-style one-shot helper.
func (i *Inertia) statePull(key string) (string, bool) {
	value, ok := i.stateGet(key)
	if !ok {
		return "", false
	}
	i.stateDelete(key)
	return value, true
}

func (i *Inertia) pullBooleanFlag(key string, current *bool) bool {
	value := *current
	*current = false
	pulled, ok := i.statePull(key)
	if !ok {
		return value
	}
	return pulled == "true" || value
}

func (i *Inertia) saveBooleanFlag(key string, value bool) {
	if !value {
		return
	}
	i.stateSet(key, "true")
}

// persistState writes unconsumed one-shot values (clear history, preserve
// fragment, flash) into the in-memory bag, then Save via StateStore when dirty.
func (i *Inertia) persistState() error {
	i.saveBooleanFlag(StateKeyClearHistory, i.clearHistory)
	i.saveBooleanFlag(StateKeyPreserveFragment, i.preserveFragment)
	if err := i.persistFlash(); err != nil {
		return err
	}
	if !i.stateDirty {
		return nil
	}
	if i.stateStore == nil {
		return nil
	}
	if i.state == nil {
		i.state = map[string]string{}
	}
	return i.stateStore.Save(i.echoContext, i.state)
}

// WithoutSsr excludes matching paths from SSR for this request.
// Mirrors inertia-laravel Inertia::withoutSsr / Middleware::$withoutSsr.
func (i *Inertia) WithoutSsr(paths ...string) {
	i.withoutSsr = append(i.withoutSsr, paths...)
}

// DisableSsr disables SSR for this request.
// Mirrors inertia-laravel Inertia::disableSsr(true).
func (i *Inertia) DisableSsr() {
	i.ssrDisabled = true
}

// DisableSsrWhen disables SSR for this request when fn returns true at dispatch time.
// Mirrors inertia-laravel Inertia::disableSsr(Closure).
func (i *Inertia) DisableSsrWhen(fn func(c *echo.Context) bool) {
	i.ssrDisableWhen = fn
}

// SsrEnabled reports whether this request may attempt SSR from Inertia's point of view.
// Evaluates DisableSsr / DisableSsrWhen / WithoutSsr. SsrHTTPGateway may still skip when
// Enabled is false or no SSR bundle exists. Custom Renderers should call this instead of
// reading Inertia internals.
func (i *Inertia) SsrEnabled() bool {
	if i.ssrDisabled {
		return false
	}
	if i.ssrDisableWhen != nil && i.ssrDisableWhen(i.echoContext) {
		return false
	}
	path := ""
	if i.echoContext != nil && i.echoContext.Request() != nil {
		path = i.echoContext.Request().URL.Path
	}
	return !matchSSRPaths(path, i.withoutSsr)
}

func (i *Inertia) SetRootView(name string) {
	i.rootView = name
}

func (i *Inertia) RootView() string {
	return i.rootView
}

func (i *Inertia) Share(props map[string]any) {
	i.sharedPropsMutex.Lock()
	defer i.sharedPropsMutex.Unlock()

	if i.sharedProps == nil {
		i.sharedProps = map[string]any{}
	}
	for k, v := range props {
		i.sharedProps[k] = v
	}
}

// ShareOnce shares a callback-backed prop that the client can retain across
// visits under the supplied top-level prop name.
func (i *Inertia) ShareOnce(key string, callback func() (any, error)) *OnceProp {
	prop := Once(callback)
	i.Share(map[string]any{key: prop})
	return prop
}

func (i *Inertia) Shared() map[string]any {
	i.sharedPropsMutex.RLock()
	defer i.sharedPropsMutex.RUnlock()

	props := make(map[string]any, len(i.sharedProps))
	for key, value := range i.sharedProps {
		props[key] = value
	}
	return props
}

// Flash adds data to the top-level Page flash object for the current render.
// For POST-redirect-GET, unconsumed flash is persisted via StateStore (default:
// unsigned cookie). Treat loaded flash like untrusted request input.
func (i *Inertia) Flash(data map[string]any) {
	if i.flash == nil {
		i.flash = map[string]any{}
	}
	for key, value := range data {
		i.flash[key] = value
	}
}

func (i *Inertia) resolveFlash() (map[string]any, error) {
	if !i.flashLoaded {
		local := i.flash
		i.flash = map[string]any{}
		raw, ok := i.statePull(StateKeyFlashData)
		if ok && raw != "" {
			if stored, err := decodeFlashState(raw); err == nil {
				for key, value := range stored {
					i.flash[key] = value
				}
			}
			// Broken payloads are ignored after Pull already removed them.
		}
		for key, value := range local {
			i.flash[key] = value
		}
		i.flashLoaded = true
	}
	if len(i.flash) == 0 {
		return nil, nil
	}
	result := make(map[string]any, len(i.flash))
	for key, value := range i.flash {
		result[key] = value
	}
	return result, nil
}

func (i *Inertia) persistFlash() error {
	if i.flashLoaded || len(i.flash) == 0 {
		return nil
	}
	encoded, err := encodeFlashState(i.flash)
	if err != nil {
		return err
	}
	i.stateSet(StateKeyFlashData, encoded)
	i.flash = map[string]any{}
	return nil
}

func encodeFlashState(data map[string]any) (string, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeFlashState(raw string) (map[string]any, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (i *Inertia) FlushShared() {
	i.sharedPropsMutex.Lock()
	defer i.sharedPropsMutex.Unlock()

	i.sharedProps = map[string]any{}
}

// VersionFunc resolves the asset version.
type VersionFunc func(c *echo.Context) string

func (i *Inertia) SetVersion(version VersionFunc) {
	i.version = version
}

func (i *Inertia) Version() string {
	return i.version(i.echoContext)
}

// URLResolverFunc resolves the Page object URL.
// Corresponding specification: https://inertiajs.com/docs/v3/core-concepts/the-protocol#the-page-object
// Mirrors inertia-laravel's resolveUrlUsing / Middleware::urlResolver.
type URLResolverFunc func(c *echo.Context) string

func (i *Inertia) SetURLResolver(resolver URLResolverFunc) {
	i.urlResolver = resolver
}

func (i *Inertia) resolveURL() string {
	return i.urlResolver(i.echoContext)
}

// Location generates 409 response for external redirects
// see https://inertiajs.com/redirects#external-redirects
func (i *Inertia) Location(url string) error {
	if i.echoContext.Request().Header.Get(HeaderXInertia) != "" {
		res := i.echoContext.Response()
		res.Header().Set(HeaderXInertiaLocation, url)
		res.WriteHeader(http.StatusConflict)
		return nil
	} else {
		return i.echoContext.Redirect(http.StatusFound, url)
	}
}

func (i *Inertia) isPartial(component string) bool {
	return i.partialComponent == component
}

type Page struct {
	Component        string                    `json:"component"`
	Props            map[string]any            `json:"props"`
	URL              string                    `json:"url"`
	Version          string                    `json:"version"`
	EncryptHistory   bool                      `json:"encryptHistory,omitempty"`
	ClearHistory     bool                      `json:"clearHistory,omitempty"`
	DeferredProps    map[string]any            `json:"deferredProps,omitempty"`
	SharedProps      []string                  `json:"sharedProps,omitempty"`
	MergeProps       []string                  `json:"mergeProps,omitempty"`
	PrependProps     []string                  `json:"prependProps,omitempty"`
	DeepMergeProps   []string                  `json:"deepMergeProps,omitempty"`
	MatchPropsOn     []string                  `json:"matchPropsOn,omitempty"`
	ScrollProps      map[string]ScrollMetadata `json:"scrollProps,omitempty"`
	OnceProps        map[string]OnceMetadata   `json:"onceProps,omitempty"`
	Flash            map[string]any            `json:"flash,omitempty"`
	PreserveFragment bool                      `json:"preserveFragment,omitempty"`
	RescuedProps     []string                  `json:"rescuedProps,omitempty"`
}

type RenderContext struct {
	Inertia *Inertia
	Page    *Page
	// ViewName is the name of the view to render.
	ViewName string
	// ViewData is optional data passed to the root view template.
	// HTMLRenderer copies it and adds page, inertia, and inertiaHead.
	ViewData map[string]any
	Writer   io.Writer
}

func (i *Inertia) Render(component string, props map[string]any) error {
	return i.render(http.StatusOK, component, props, nil)
}

func (i *Inertia) RenderWithViewData(component string, props map[string]any, viewData map[string]any) error {
	return i.render(http.StatusOK, component, props, viewData)
}

// RenderWithStatus renders an Inertia page with a non-200 HTTP status.
// This package does not provide an inertia-laravel-style error-handling
// framework; Echo and Go favor explicit application logic. For centralized
// error pages, use your application's echo.HTTPErrorHandler with Get.
func (i *Inertia) RenderWithStatus(status int, component string, props map[string]any) error {
	return i.render(status, component, props, nil)
}

func (i *Inertia) RenderWithStatusAndViewData(status int, component string, props map[string]any, viewData map[string]any) error {
	return i.render(status, component, props, viewData)
}

func (i *Inertia) render(status int, component string, props map[string]any, viewData map[string]any) error {
	if i.renderer == nil {
		return ErrRendererNotRegistered
	}
	req := i.echoContext.Request()
	res := i.echoContext.Response()
	resolver := newPropsResolver(i, component)
	validProps, metadata, err := resolver.resolve(i.Shared(), props)
	if err != nil {
		return err
	}
	if validProps == nil {
		validProps = map[string]any{}
	}
	flash, err := i.resolveFlash()
	if err != nil {
		return err
	}

	page := &Page{
		Component:        component,
		Props:            validProps,
		URL:              i.resolveURL(),
		Version:          i.Version(),
		EncryptHistory:   i.encryptHistory,
		ClearHistory:     i.pullClearHistory(),
		PreserveFragment: i.pullPreserveFragment(),
		Flash:            flash,
		SharedProps:      metadata.sharedProps,
		DeferredProps:    emptyMapAsNil(metadata.deferredProps),
		MergeProps:       metadata.mergeProps,
		PrependProps:     metadata.prependProps,
		DeepMergeProps:   metadata.deepMergeProps,
		MatchPropsOn:     metadata.matchPropsOn,
		ScrollProps:      emptyScrollMapAsNil(metadata.scrollProps),
		OnceProps:        emptyOnceMapAsNil(metadata.onceProps),
		RescuedProps:     metadata.rescuedProps,
	}

	addVaryHeader(res.Header(), HeaderXInertia)
	if req.Header.Get(HeaderXInertia) != "" {
		res.Header().Set(HeaderXInertia, "true")
		return i.echoContext.JSON(status, page)
	}

	buf := new(bytes.Buffer)
	ctx := &RenderContext{Inertia: i, ViewName: i.rootView, Page: page, ViewData: viewData, Writer: buf}
	if err := i.renderer.Render(ctx); err != nil {
		return err
	}
	return i.echoContext.HTMLBlob(status, buf.Bytes())
}

func emptyMapAsNil(value map[string]any) map[string]any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func emptyScrollMapAsNil(value map[string]ScrollMetadata) map[string]ScrollMetadata {
	if len(value) == 0 {
		return nil
	}
	return value
}

func emptyOnceMapAsNil(value map[string]OnceMetadata) map[string]OnceMetadata {
	if len(value) == 0 {
		return nil
	}
	return value
}

func SetRootView(c *echo.Context, name string) {
	MustGet(c).SetRootView(name)
}

func RootView(c *echo.Context) string {
	return MustGet(c).RootView()
}

func Share(c *echo.Context, props map[string]any) {
	MustGet(c).Share(props)
}

func ShareOnce(c *echo.Context, key string, callback func() (any, error)) *OnceProp {
	return MustGet(c).ShareOnce(key, callback)
}

func Shared(c *echo.Context) map[string]any {
	return MustGet(c).Shared()
}

func FlushShared(c *echo.Context) {
	MustGet(c).FlushShared()
}

func SetVersion(c *echo.Context, version VersionFunc) {
	MustGet(c).SetVersion(version)
}

func Version(c *echo.Context) string {
	return MustGet(c).Version()
}

func SetURLResolver(c *echo.Context, resolver URLResolverFunc) {
	MustGet(c).SetURLResolver(resolver)
}

func WithoutSsr(c *echo.Context, paths ...string) {
	MustGet(c).WithoutSsr(paths...)
}

func DisableSsr(c *echo.Context) {
	MustGet(c).DisableSsr()
}

func DisableSsrWhen(c *echo.Context, fn func(c *echo.Context) bool) {
	MustGet(c).DisableSsrWhen(fn)
}

func SsrEnabled(c *echo.Context) bool {
	return MustGet(c).SsrEnabled()
}

func Location(c *echo.Context, url string) error {
	return MustGet(c).Location(url)
}

func EncryptHistory(c *echo.Context, encrypt bool) {
	MustGet(c).EncryptHistory(encrypt)
}

func ClearHistory(c *echo.Context) {
	MustGet(c).ClearHistory()
}

func PreserveFragment(c *echo.Context) {
	MustGet(c).PreserveFragment()
}

func Flash(c *echo.Context, data map[string]any) {
	MustGet(c).Flash(data)
}

func Render(c *echo.Context, component string, props map[string]any) error {
	return MustGet(c).Render(component, props)
}

func RenderWithViewData(c *echo.Context, component string, props map[string]any, viewData map[string]any) error {
	return MustGet(c).RenderWithViewData(component, props, viewData)
}

func RenderWithStatus(c *echo.Context, status int, component string, props map[string]any) error {
	return MustGet(c).RenderWithStatus(status, component, props)
}

func RenderWithStatusAndViewData(c *echo.Context, status int, component string, props map[string]any, viewData map[string]any) error {
	return MustGet(c).RenderWithStatusAndViewData(status, component, props, viewData)
}
