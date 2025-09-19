package inertia

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/mitchellh/mapstructure"
)

const (
	HeaderXInertia                 = "X-Inertia"
	HeaderXInertiaErrorBag         = "X-Inertia-Error-Bag"
	HeaderXInertiaLocation         = "X-Inertia-Location"
	HeaderXInertiaVersion          = "X-Inertia-Version"
	HeaderXInertiaPartialComponent = "X-Inertia-Partial-Component"
	HeaderXInertiaPartialData      = "X-Inertia-Partial-Data"
	HeaderXInertiaPartialExcept    = "X-Inertia-Partial-Except"
	HeaderXInertiaReset            = "X-Inertia-Reset"
)

// Inertia is a echo.Context wrapper that handles Inertia.js protocol.
type Inertia struct {
	echoContext           echo.Context
	rootView              string
	sharedProps           map[string]any
	sharedPropsMutex      sync.RWMutex
	version               VersionFunc
	renderer              Renderer
	encryptHistory        bool
	clearHistoryCookieKey string
	clearHistory          bool
	isSsrDisabled         bool
	partialComponent      string
	onlyProps             []string
	exceptProps           []string
	resetProps            []string
}

func (i *Inertia) EchoContext() echo.Context {
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
	// Reset clearHistory after reading the current state or cookie value.
	defer func() {
		i.clearHistory = false
	}()

	// Check if the clear history cookie is set
	cookie, err := i.echoContext.Request().Cookie(i.clearHistoryCookieKey)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			// No cookie found, use the current state
			return i.clearHistory
		}
	}

	// You got the cookie value, therefore you should delete the cookie
	http.SetCookie(i.echoContext.Response(), &http.Cookie{
		Name:     i.clearHistoryCookieKey,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // Negative value tells browser to delete immediately
	})

	if cookie.Value == "true" {
		return true
	}

	return false
}

func (i *Inertia) sendClearHistoryCookieIfNeeded() {
	if i.clearHistory {
		// In this case, you called the ClearHistory method, but you still haven't called the pullClearHistory method.
		// Typically, this happens when you call ClearHistory and then redirect to another page.
		// To keep the clearHistory flag, you need to set the cookie.
		http.SetCookie(i.echoContext.Response(), &http.Cookie{
			Name:     i.clearHistoryCookieKey,
			Value:    "true",
			Path:     "/",
			HttpOnly: true,
		})
	}
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

	// merge shared props
	for k, v := range props {
		i.sharedProps[k] = v
	}
}

func (i *Inertia) Shared() map[string]any {
	i.sharedPropsMutex.RLock()
	defer i.sharedPropsMutex.RUnlock()

	return i.sharedProps
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
		res.Header().Set(HeaderXInertiaLocation, url)
		res.WriteHeader(409)
		return nil
	} else {
		return i.echoContext.Redirect(http.StatusFound, url)
	}
}

func (i *Inertia) isPartial(component string) bool {
	return i.partialComponent == component
}

type Page struct {
	Component      string         `json:"component"`
	Props          map[string]any `json:"props"`
	URL            string         `json:"url"`
	Version        string         `json:"version"`
	EncryptHistory bool           `json:"encryptHistory"`
	ClearHistory   bool           `json:"clearHistory"`
	DeferredProps  map[string]any `json:"deferredProps,omitempty"`
	MergeProps     []string       `json:"mergeProps,omitempty"`
	DeepMergeProps []string       `json:"deepMergeProps,omitempty"`
	MatchPropsOn   []string       `json:"matchPropsOn,omitempty"`
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

func (i *Inertia) Render(component string, propsData any) error {
	return i.RenderWithViewData(component, propsData, nil)
}

func (i *Inertia) RenderWithViewData(component string, propsData any, viewData any) error {
	if i.renderer == nil {
		return ErrRendererNotRegistered
	}

	req := i.echoContext.Request()
	res := i.echoContext.Response()

	props, ok := propsData.(map[string]any)
	if !ok {
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			TagName: "prop",
			Result:  &props,
		})
		if err != nil {
			return err
		}
		if err := decoder.Decode(propsData); err != nil {
			return fmt.Errorf("failed to decode propsData: %w", err)
		}
	}

	// merge shared props
	props = i.mergeProps(i.sharedProps, props)

	// Note:
	// The official `laravel-inertia` package executes the following methods:
	// * `resolveInertiaPropsProviders`
	// * `resolveArrayableProperties`
	// but this package does not implement them. This is by design.
	// I believe that they represent additional layers of data abstraction that don't align with the Go language philosophy.

	// Note:
	// The official `laravel-inertia` package implements the `resolveValidationErrors` method to extract validation errors from the session.
	// However, this package does not implement it because Echo does not have a built-in session implementation.
	// Echo’s official documentation provides information about sessions:
	// https://echo.labstack.com/docs/middleware/session
	// Nonetheless, in my opinion, you probably need to implement your own logic to handle sessions.
	// This means it is difficult to implement a common error-handling logic with sessions, or it may lead to a poor abstraction layer.
	// It is not match to the Go language philosophy.

	// process partial reloads
	// https://inertiajs.com/partial-reloads
	validProps := i.copyProps(props)
	validProps = i.resolvePartialProps(component, validProps)
	validProps = i.resolveAlwaysProps(props, validProps)

	if err := evaluateProps(validProps); err != nil {
		return err
	}

	page := &Page{
		Component:      component,
		Props:          validProps,
		URL:            req.URL.String(),
		Version:        i.Version(),
		EncryptHistory: i.encryptHistory,
		ClearHistory:   i.pullClearHistory(),
		DeferredProps:  i.resolveDeferredProps(component, props),
	}

	mergeProps, deepMergeProps, matchPropsOn := i.resolveMergeProps(props)
	page.MergeProps = mergeProps
	page.DeepMergeProps = deepMergeProps
	page.MatchPropsOn = matchPropsOn

	res.Header().Set("Vary", HeaderXInertia)

	if req.Header.Get(HeaderXInertia) != "" {
		// The request is an Inertia request, so we return JSON response
		res.Header().Set(HeaderXInertia, "true")
		return i.echoContext.JSON(http.StatusOK, page)
	}

	// The request is a normal request, so we render HTML content.
	buf := new(bytes.Buffer)
	renderContext := &RenderContext{
		Inertia:  i,
		ViewName: i.rootView,
		Page:     page,
		ViewData: viewData,
		Writer:   buf,
	}
	if err := i.renderer.Render(renderContext); err != nil {
		return err
	}
	return i.echoContext.HTMLBlob(http.StatusOK, buf.Bytes())
}

func (i *Inertia) mergeProps(props ...map[string]any) map[string]any {
	merged := map[string]any{}
	for _, a := range props {
		for k, v := range a {
			merged[k] = v
		}
	}
	return merged
}

func (i *Inertia) copyProps(props map[string]any) map[string]any {
	newProps := make(map[string]any, len(props))
	for k, v := range props {
		newProps[k] = v
	}
	return newProps
}

func (i *Inertia) resolvePartialProps(component string, validProps map[string]any) map[string]any {
	if !i.isPartial(component) {
		// Not a partial request, filter out IgnoreFirstLoad props
		newProps := make(map[string]any)
		for key, value := range validProps {
			if _, isIgnoreFirstLoad := value.(IgnoreFirstLoadProp); !isIgnoreFirstLoad {
				newProps[key] = value
			}
		}
		return newProps
	}

	if len(i.onlyProps) > 0 {
		newProps := make(map[string]any)
		for _, key := range i.onlyProps {
			if value, exists := validProps[key]; exists {
				newProps[key] = value
			}
		}
		validProps = newProps
	}

	if len(i.exceptProps) > 0 {
		for _, key := range i.exceptProps {
			if _, exists := validProps[key]; exists {
				delete(validProps, key)
			}
		}
	}

	return validProps
}

func (i *Inertia) resolveAlwaysProps(props, validProps map[string]any) map[string]any {
	for k, v := range props {
		if _, ok := v.(*AlwaysProp); ok {
			validProps[k] = v
		}
	}

	return validProps
}

func (i *Inertia) resolveDeferredProps(component string, props map[string]any) map[string]any {
	if i.isPartial(component) {
		return nil
	}

	groups := make(map[string][]string)
	for key, prop := range props {
		if deferProp, ok := prop.(*DeferProp); ok {
			group := deferProp.Group()
			groups[group] = append(groups[group], key)
		}
	}

	if len(groups) == 0 {
		return nil
	}

	// Convert to map[string]any
	result := make(map[string]any)
	for k, v := range groups {
		result[k] = v
	}
	return result
}

func (i *Inertia) resolveMergeProps(props map[string]any) ([]string, []string, []string) {
	var mergeProps []string
	var deepMergeProps []string
	var matchOnProps []string

	// Extract props for mergeProps
	for key, prop := range props {
		if mergeable, ok := prop.(Mergeable); ok && mergeable.ShouldMerge() {
			// reject the prop if it is in resetProps
			if inArray(key, i.resetProps) {
				continue
			}

			// if onlyProps is specified, skip the prop if it is not in onlyProps
			if len(i.onlyProps) > 0 && !inArray(key, i.onlyProps) {
				continue
			}

			// skip the prop if it is in exceptProps
			if inArray(key, i.exceptProps) {
				continue
			}

			if mergeable.ShouldDeepMerge() {
				deepMergeProps = append(deepMergeProps, key)
			} else {
				mergeProps = append(mergeProps, key)
			}

			matchesOn := mergeable.MatchesOn()
			for _, strategy := range matchesOn {
				matchOnProps = append(matchOnProps, key+"."+strategy)
			}
		}
	}

	return mergeProps, deepMergeProps, matchOnProps
}

func SetRootView(c echo.Context, name string) {
	MustGet(c).SetRootView(name)
}

func RootView(c echo.Context) string {
	return MustGet(c).RootView()
}

func Share(c echo.Context, props map[string]any) {
	MustGet(c).Share(props)
}

func Shared(c echo.Context) map[string]any {
	return MustGet(c).Shared()
}

func FlushShared(c echo.Context) {
	MustGet(c).FlushShared()
}

func SetVersion(c echo.Context, version VersionFunc) {
	MustGet(c).SetVersion(version)
}

func Version(c echo.Context) string {
	return MustGet(c).Version()
}

func Location(c echo.Context, url string) error {
	return MustGet(c).Location(url)
}

func EncryptHistory(c echo.Context, encrypt bool) {
	MustGet(c).EncryptHistory(encrypt)
}

func ClearHistory(c echo.Context) {
	MustGet(c).ClearHistory()
}

func Render(c echo.Context, component string, props any) error {
	return MustGet(c).Render(component, props)
}

func RenderWithViewData(c echo.Context, component string, props any, viewData any) error {
	return MustGet(c).RenderWithViewData(component, props, viewData)
}
