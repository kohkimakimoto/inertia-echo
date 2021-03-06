package inertia

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
)

const (
	HeaderXInertia                 = "X-Inertia"
	HeaderXInertiaVersion          = "X-Inertia-Version"
	HeaderXInertiaLocation         = "X-Inertia-Location"
	HeaderXInertiaPartialData      = "X-Inertia-Partial-Data"
	HeaderXInertiaPartialComponent = "X-Inertia-Partial-Component"
)

type LazyProp struct {
	callback func() interface{}
}

type VersionFunc func() string

type Page struct {
	Component string                 `json:"component"`
	Props     map[string]interface{} `json:"props"`
	Url       string                 `json:"url"`
	Version   string                 `json:"version"`
}

type Inertia struct {
	c           echo.Context
	rootView    string
	sharedProps map[string]interface{}
	version     VersionFunc
	mu          sync.RWMutex
}

// New creates a new inertia instance.
func New(c echo.Context, rootView string, sharedProps map[string]interface{}, versionFunc VersionFunc) *Inertia {
	return &Inertia{
		c:           c,
		rootView:    rootView,
		sharedProps: sharedProps,
		version:     versionFunc,
	}
}

func (i *Inertia) SetRootView(name string) {
	i.rootView = name
}

func (i *Inertia) Share(props map[string]interface{}) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// merge shared props
	for k, v := range props {
		i.sharedProps[k] = v
	}
}

func (i *Inertia) Shared() map[string]interface{} {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.sharedProps
}

func (i *Inertia) FlushShared() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.sharedProps = map[string]interface{}{}
}

func (i *Inertia) SetVersion(version VersionFunc) {
	i.version = version
}

func (i *Inertia) Version() string {
	return i.version()
}

// Location generates 409 response for external redirects
// see https://inertiajs.com/redirects#external-redirects
func (i *Inertia) Location(url string) error {
	res := i.c.Response()
	res.Header().Set(HeaderXInertiaLocation, url)
	res.WriteHeader(409)
	return nil
}

func (i *Inertia) Render(code int, component string, props map[string]interface{}) error {
	return i.render(code, component, props, map[string]interface{}{})
}

func (i *Inertia) RenderWithViewData(code int, component string, props, viewData map[string]interface{}) error {
	return i.render(code, component, props, viewData)
}

func (i *Inertia) render(code int, component string, props, viewData map[string]interface{}) error {
	c := i.c
	req := c.Request()
	res := c.Response()

	props = mergeProps(i.sharedProps, props)

	only := splitOrNil(req.Header.Get(HeaderXInertiaPartialData), ",")
	if only != nil && req.Header.Get(HeaderXInertiaPartialComponent) == component {
		filteredProps := map[string]interface{}{}
		for _, key := range only {
			filteredProps[key] = props[key]
		}
		props = filteredProps
	} else {
		filteredProps := map[string]interface{}{}
		for key, prop := range props {
			// LazyProp is only used in partial reloads
			// see https://inertiajs.com/partial-reloads#lazy-data-evaluation
			if _, ok := prop.(*LazyProp); !ok {
				filteredProps[key] = prop
			}
		}
		props = filteredProps
	}

	evaluatePropsRecursive(props)

	page := &Page{
		Component: component,
		Props:     props,
		Url:       req.URL.String(),
		Version:   i.Version(),
	}

	res.Header().Set("Vary", HeaderXInertia)

	if req.Header.Get(HeaderXInertia) != "" {
		res.Header().Set(HeaderXInertia, "true")
		return c.JSON(http.StatusOK, page)
	}

	viewData["page"] = page
	return c.Render(code, i.rootView, viewData)
}

// Lazy defines a lazy evaluated data.
// see https://inertiajs.com/partial-reloads#lazy-data-evaluation
func Lazy(callback func() interface{}) *LazyProp {
	return &LazyProp{
		callback: callback,
	}
}
