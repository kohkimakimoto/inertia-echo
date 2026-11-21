package slowjs_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/kohkimakimoto/inertia-echo/v5/ssr/slowjs"
)

const stubBundle = `
globalThis.renderPage = function (page) {
  return {
    head: ["<title data-inertia>" + (page.props && page.props.title ? page.props.title : "") + "</title>"],
    body: '<div id="app" data-page="' + page.component + '">ok</div>'
  };
};
`

// Busy-wait ~25ms so pool concurrency is observable in wall time.
const slowStubBundle = `
globalThis.renderPage = function (page) {
  var end = Date.now() + 25;
  while (Date.now() < end) {}
  return { head: [], body: "<div>ok</div>" };
};
`

func TestGatewayDispatch(t *testing.T) {
	g, err := slowjs.NewFromSource(stubBundle, slowjs.Config{PoolSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()
	if g.PoolSize() != 1 {
		t.Fatalf("PoolSize=%d", g.PoolSize())
	}

	res, err := g.Dispatch(&inertia.RenderContext{
		Page: &inertia.Page{
			Component: "Index",
			Props:     map[string]any{"title": "hello"},
			URL:       "/",
			Version:   "1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Head) != 1 || !strings.Contains(res.Head[0], "hello") {
		t.Fatalf("head: %#v", res.Head)
	}
	if !strings.Contains(res.Body, `data-page="Index"`) {
		t.Fatalf("body: %q", res.Body)
	}
}

func TestGatewaySkipWhenSsrDisabled(t *testing.T) {
	g, err := slowjs.NewFromSource(stubBundle, slowjs.Config{PoolSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer g.Close()

	in := &inertia.Inertia{}
	in.DisableSsr()
	res, err := g.Dispatch(&inertia.RenderContext{
		Inertia: in,
		Page:    &inertia.Page{Component: "Index", URL: "/"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res != nil {
		t.Fatalf("want nil skip, got %#v", res)
	}
}

func TestGatewayPoolConcurrentFasterThanSerial(t *testing.T) {
	page := &inertia.RenderContext{
		Page: &inertia.Page{Component: "Index", Props: map[string]any{}, URL: "/"},
	}
	runParallel := func(poolSize, n int) time.Duration {
		t.Helper()
		g, err := slowjs.NewFromSource(slowStubBundle, slowjs.Config{PoolSize: poolSize})
		if err != nil {
			t.Fatal(err)
		}
		defer g.Close()
		var wg sync.WaitGroup
		wg.Add(n)
		start := time.Now()
		for i := 0; i < n; i++ {
			go func() {
				defer wg.Done()
				if _, err := g.Dispatch(page); err != nil {
					t.Errorf("dispatch: %v", err)
				}
			}()
		}
		wg.Wait()
		return time.Since(start)
	}

	const n = 4
	serial := runParallel(1, n)
	parallel := runParallel(n, n)
	// 4×25ms serial ≈ 100ms; parallel ≈ 25ms. Allow slack for CI noise.
	if parallel*2 >= serial {
		t.Fatalf("expected PoolSize=%d to be much faster than PoolSize=1; serial=%s parallel=%s", n, serial, parallel)
	}
}
