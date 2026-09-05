package inertia

import (
	"bytes"
	"errors"
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
	echoContext               *echo.Context
	rootView                  string
	sharedProps               map[string]any
	sharedPropsMutex          sync.RWMutex
	version                   VersionFunc
	renderer                  Renderer
	encryptHistory            bool
	clearHistoryCookieKey     string
	clearHistory              bool
	preserveFragmentCookieKey string
	preserveFragment          bool
	flash                     map[string]any
	flashLoaded               bool
	flashData                 FlashDataFunc
	reflash                   ReflashFunc
	rescueReporter            RescueReporter
	now                       func() time.Time
	isSsrDisabled             bool
	partialComponent          string
	onlyProps                 []string
	exceptProps               []string
	resetProps                []string
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
// see https://inertiajs.com/history-encryption
func (i *Inertia) ClearHistory() {
	i.clearHistory = true
}

// pullClearHistory pulls the clear history flag from the cookie or the current state.
// Note:
// The design of the inertia-echo package used a dedicated cookie to store the clear history flag.
// While the official inertia-laravel adapter uses a session for this purpose,
// the Echo framework lacks a built-in session store, so we use a cookie as an alternative.
func (i *Inertia) pullClearHistory() bool {
	return i.pullBooleanFlag(i.clearHistoryCookieKey, &i.clearHistory)
}

func (i *Inertia) sendClearHistoryCookieIfNeeded() {
	i.sendBooleanFlag(i.clearHistoryCookieKey, i.clearHistory)
}

// PreserveFragment asks the client to carry the source URL fragment through
// the next rendered page. The flag survives redirects in a boolean cookie.
func (i *Inertia) PreserveFragment() {
	i.preserveFragment = true
}

func (i *Inertia) pullPreserveFragment() bool {
	return i.pullBooleanFlag(i.preserveFragmentCookieKey, &i.preserveFragment)
}

func (i *Inertia) sendPreserveFragmentCookieIfNeeded() {
	i.sendBooleanFlag(i.preserveFragmentCookieKey, i.preserveFragment)
}

func (i *Inertia) pullBooleanFlag(cookieKey string, current *bool) bool {
	value := *current
	*current = false
	cookie, err := i.echoContext.Request().Cookie(cookieKey)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return value
		}
		return value
	}
	http.SetCookie(i.echoContext.Response(), &http.Cookie{
		Name:     cookieKey,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	return cookie.Value == "true" || value
}

func (i *Inertia) sendBooleanFlag(cookieKey string, value bool) {
	if !value {
		return
	}
	http.SetCookie(i.echoContext.Response(), &http.Cookie{
		Name:     cookieKey,
		Value:    "true",
		Path:     "/",
		HttpOnly: true,
	})
}

func (i *Inertia) IsSsrDisabled() bool {
	return i.isSsrDisabled
}

func (i *Inertia) IsSsrEnabled() bool {
	return !i.isSsrDisabled
}

func (i *Inertia) EnableSsr() {
	i.isSsrDisabled = false
}

func (i *Inertia) DisableSsr() {
	i.isSsrDisabled = true
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
		if i.flashData != nil {
			stored, err := i.flashData(i.echoContext)
			if err != nil {
				return nil, err
			}
			for key, value := range stored {
				i.flash[key] = value
			}
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

func (i *Inertia) FlushShared() {
	i.sharedPropsMutex.Lock()
	defer i.sharedPropsMutex.Unlock()

	i.sharedProps = map[string]any{}
}

type VersionFunc func() string

func (i *Inertia) SetVersion(version VersionFunc) {
	i.version = version
}

func (i *Inertia) Version() string {
	return i.version()
}

// Location generates 409 response for external redirects
// see https://inertiajs.com/redirects#external-redirects
func (i *Inertia) Location(url string) error {
	if i.echoContext.Request().Header.Get(HeaderXInertia) != "" {
		res := i.echoContext.Response()
		addVaryHeader(res.Header(), HeaderXInertia)
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
	// You can set any data you want to ViewData, but the renderer needs to be able to handle it.
	// For example, the official HTMLRenderer can only accept ViewData as a map[string]any.
	ViewData any
	Writer   io.Writer
}

func (i *Inertia) Render(component string, props map[string]any) error {
	return i.render(http.StatusOK, component, props, nil)
}

func (i *Inertia) RenderWithViewData(component string, props map[string]any, viewData any) error {
	return i.render(http.StatusOK, component, props, viewData)
}

func (i *Inertia) RenderWithStatus(status int, component string, props map[string]any) error {
	return i.render(status, component, props, nil)
}

func (i *Inertia) RenderWithStatusAndViewData(status int, component string, props map[string]any, viewData any) error {
	return i.render(status, component, props, viewData)
}

func (i *Inertia) render(status int, component string, props map[string]any, viewData any) error {
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
		Component: component, Props: validProps, URL: req.URL.String(), Version: i.Version(),
		EncryptHistory: i.encryptHistory, ClearHistory: i.pullClearHistory(),
		PreserveFragment: i.pullPreserveFragment(), Flash: flash,
		SharedProps: metadata.sharedProps, DeferredProps: emptyMapAsNil(metadata.deferredProps),
		MergeProps: metadata.mergeProps, PrependProps: metadata.prependProps,
		DeepMergeProps: metadata.deepMergeProps, MatchPropsOn: metadata.matchPropsOn,
		ScrollProps: emptyScrollMapAsNil(metadata.scrollProps), OnceProps: emptyOnceMapAsNil(metadata.onceProps),
		RescuedProps: metadata.rescuedProps,
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

func RenderWithViewData(c *echo.Context, component string, props map[string]any, viewData any) error {
	return MustGet(c).RenderWithViewData(component, props, viewData)
}

func RenderWithStatus(c *echo.Context, status int, component string, props map[string]any) error {
	return MustGet(c).RenderWithStatus(status, component, props)
}

func RenderWithStatusAndViewData(c *echo.Context, status int, component string, props map[string]any, viewData any) error {
	return MustGet(c).RenderWithStatusAndViewData(status, component, props, viewData)
}
