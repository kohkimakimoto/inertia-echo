package inertia

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
)

type resolverTestNamedMap map[string]any
type resolverTestNamedSlice []any

type resolverTestStruct struct {
	Value any `json:"value"`
}

type resolverTestMarshaler struct {
	calls *int
}

func (v resolverTestMarshaler) MarshalJSON() ([]byte, error) {
	(*v.calls)++
	return []byte(`{"value":"marshaled"}`), nil
}

func TestPropsResolverNestedDefinitionContainers(t *testing.T) {
	callCount := 0
	value := func(result string) func() any {
		return func() any {
			callCount++
			return result
		}
	}
	props := map[string]any{
		"profile": map[string]any{
			"name": value("Ada"),
		},
		"items": []any{
			map[string]any{"name": value("first")},
			[]any{map[string]any{"name": value("nested")}},
		},
		"records": []map[string]any{
			{"name": value("one")},
			{"name": value("two")},
		},
	}

	resolver, _ := newTestPropsResolver(t, "Dashboard", nil, nil)
	got, _, err := resolver.resolve(nil, props)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"profile": map[string]any{"name": "Ada"},
		"items": []any{
			map[string]any{"name": "first"},
			[]any{map[string]any{"name": "nested"}},
		},
		"records": []map[string]any{
			{"name": "one"},
			{"name": "two"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected resolved props:\n got: %#v\nwant: %#v", got, want)
	}
	if callCount != 5 {
		t.Fatalf("expected each nested callback to run once, got %d calls", callCount)
	}
}

func TestPropsResolverDotPathOnlyAndExcept(t *testing.T) {
	selectedCalls := 0
	excludedCalls := 0
	props := map[string]any{
		"auth": map[string]any{
			"user": map[string]any{
				"name": func() any {
					selectedCalls++
					return "Ada"
				},
				"token": func() any {
					excludedCalls++
					return "secret"
				},
			},
			"roles": func() any {
				excludedCalls++
				return []string{"admin"}
			},
		},
		"items": []any{
			map[string]any{
				"name": func() any {
					selectedCalls++
					return "first"
				},
				"secret": func() any {
					excludedCalls++
					return "hidden"
				},
			},
			map[string]any{
				"name": func() any {
					excludedCalls++
					return "second"
				},
			},
		},
		"records": []map[string]any{
			{
				"name": func() any {
					excludedCalls++
					return "one"
				},
			},
			{
				"name": func() any {
					selectedCalls++
					return "two"
				},
				"secret": func() any {
					excludedCalls++
					return "hidden"
				},
			},
		},
	}
	headers := map[string]string{
		HeaderXInertia:                 "true",
		HeaderXInertiaPartialComponent: "Dashboard",
		HeaderXInertiaPartialData:      "auth.user,items.0.name,records.1.name",
		HeaderXInertiaPartialExcept:    "auth.user.token",
	}

	resolver, _ := newTestPropsResolver(t, "Dashboard", headers, nil)
	got, _, err := resolver.resolve(nil, props)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"auth": map[string]any{
			"user": map[string]any{"name": "Ada"},
		},
		"items": []any{
			map[string]any{"name": "first"},
			map[string]any{},
		},
		"records": []map[string]any{
			{},
			{"name": "two"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected partial props:\n got: %#v\nwant: %#v", got, want)
	}
	if selectedCalls != 3 {
		t.Fatalf("expected 3 selected callback calls, got %d", selectedCalls)
	}
	if excludedCalls != 0 {
		t.Fatalf("expected excluded callbacks not to run, got %d calls", excludedCalls)
	}
}

func TestPropsResolverPartialComponentMismatchUsesFullResponseRules(t *testing.T) {
	normalCalls := 0
	deferredCalls := 0
	props := map[string]any{
		"name": func() any {
			normalCalls++
			return "Ada"
		},
		"other": "included",
		"deferred": Defer(func() (any, error) {
			deferredCalls++
			return "later", nil
		}),
		"merged": Merge([]int{1, 2}),
	}
	headers := map[string]string{
		HeaderXInertia:                 "true",
		HeaderXInertiaPartialComponent: "OtherComponent",
		HeaderXInertiaPartialData:      "name",
	}

	resolver, _ := newTestPropsResolver(t, "Dashboard", headers, nil)
	got, metadata, err := resolver.resolve(nil, props)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"name":   "Ada",
		"other":  "included",
		"merged": []int{1, 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("component mismatch must use full-response rules:\n got: %#v\nwant: %#v", got, want)
	}
	if normalCalls != 1 || deferredCalls != 0 {
		t.Fatalf("unexpected callback counts: normal=%d deferred=%d", normalCalls, deferredCalls)
	}
	if !reflect.DeepEqual(metadata.deferredProps, map[string]any{"default": []string{"deferred"}}) {
		t.Fatalf("unexpected deferred metadata: %#v", metadata.deferredProps)
	}
	if !reflect.DeepEqual(metadata.mergeProps, []string{"merged"}) {
		t.Fatalf("partial headers must not filter merge metadata on component mismatch: %#v", metadata.mergeProps)
	}
}

func TestPropsResolverAlwaysBypassesPartialFiltering(t *testing.T) {
	alwaysCalls := 0
	excludedCalls := 0
	props := map[string]any{
		"errors": Always(map[string]any{
			"email": func() any {
				alwaysCalls++
				return "invalid"
			},
		}),
		"selected": "yes",
		"excluded": func() any {
			excludedCalls++
			return "no"
		},
	}
	headers := map[string]string{
		HeaderXInertia:                 "true",
		HeaderXInertiaPartialComponent: "Dashboard",
		HeaderXInertiaPartialData:      "selected",
		HeaderXInertiaPartialExcept:    "errors.email",
	}

	resolver, _ := newTestPropsResolver(t, "Dashboard", headers, nil)
	got, _, err := resolver.resolve(nil, props)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"errors":   map[string]any{"email": "invalid"},
		"selected": "yes",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected Always result:\n got: %#v\nwant: %#v", got, want)
	}
	if alwaysCalls != 1 || excludedCalls != 0 {
		t.Fatalf("unexpected callback counts: always=%d excluded=%d", alwaysCalls, excludedCalls)
	}
}

func TestPropsResolverDoesNotExploreCallbackReturnedMap(t *testing.T) {
	outerCalls := 0
	innerCalls := 0
	deferred := Defer(func() (any, error) {
		innerCalls++
		return "deferred", nil
	})
	optional := Optional(func() (any, error) {
		innerCalls++
		return "optional", nil
	})
	innerCallback := func() any {
		innerCalls++
		return "callback"
	}
	props := map[string]any{
		"payload": func() any {
			outerCalls++
			return map[string]any{
				"deferred": deferred,
				"optional": optional,
				"callback": innerCallback,
			}
		},
	}

	resolver, _ := newTestPropsResolver(t, "Dashboard", nil, nil)
	got, metadata, err := resolver.resolve(nil, props)
	if err != nil {
		t.Fatal(err)
	}
	payload, ok := got["payload"].(map[string]any)
	if !ok {
		t.Fatalf("expected callback map as one JSON value, got %T", got["payload"])
	}
	if payload["deferred"] != deferred || payload["optional"] != optional {
		t.Fatalf("callback-returned wrappers were replaced: %#v", payload)
	}
	if _, ok := payload["callback"].(func() any); !ok {
		t.Fatalf("callback-returned nested callback was replaced: %T", payload["callback"])
	}
	if outerCalls != 1 || innerCalls != 0 {
		t.Fatalf("unexpected callback counts: outer=%d inner=%d", outerCalls, innerCalls)
	}
	if len(metadata.deferredProps) != 0 || len(metadata.rescuedProps) != 0 {
		t.Fatalf("callback-returned wrappers must not produce metadata: %#v", metadata)
	}
}

func TestPropsResolverDoesNotMutateDefinitionContainers(t *testing.T) {
	props := map[string]any{
		"auth": map[string]any{
			"user":  "Ada",
			"token": "secret",
		},
		"items": []any{
			map[string]any{"name": "first", "token": "hidden"},
		},
		"records": []map[string]any{
			{"name": "one", "token": "hidden"},
		},
	}
	wantInput := map[string]any{
		"auth": map[string]any{
			"user":  "Ada",
			"token": "secret",
		},
		"items": []any{
			map[string]any{"name": "first", "token": "hidden"},
		},
		"records": []map[string]any{
			{"name": "one", "token": "hidden"},
		},
	}
	headers := map[string]string{
		HeaderXInertia:                 "true",
		HeaderXInertiaPartialComponent: "Dashboard",
		HeaderXInertiaPartialData:      "auth.user,items.0.name,records.0.name",
	}

	resolver, _ := newTestPropsResolver(t, "Dashboard", headers, nil)
	got, _, err := resolver.resolve(nil, props)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(props, wantInput) {
		t.Fatalf("resolver mutated input definitions:\n got: %#v\nwant: %#v", props, wantInput)
	}

	got["auth"].(map[string]any)["user"] = "changed"
	got["items"].([]any)[0].(map[string]any)["name"] = "changed"
	got["records"].([]map[string]any)[0]["name"] = "changed"
	if !reflect.DeepEqual(props, wantInput) {
		t.Fatalf("resolved containers alias input definitions:\n got: %#v\nwant: %#v", props, wantInput)
	}
}

func TestPropsResolverOnceCustomKeyFreshAndTTL(t *testing.T) {
	fixedNow := time.Date(2026, time.September, 5, 12, 0, 0, 123_000_000, time.UTC)

	t.Run("loaded custom key is omitted without evaluation", func(t *testing.T) {
		calls := 0
		props := map[string]any{
			"settings": Once(func() (any, error) {
				calls++
				return "value", nil
			}).As("user-settings"),
		}
		headers := map[string]string{
			HeaderXInertia:                "true",
			HeaderXInertiaExceptOnceProps: "user-settings",
		}

		resolver, _ := newTestPropsResolver(t, "Dashboard", headers, func() time.Time { return fixedNow })
		got, metadata, err := resolver.resolve(nil, props)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 || calls != 0 {
			t.Fatalf("loaded once prop must be omitted without evaluation: props=%#v calls=%d", got, calls)
		}
		want := OnceMetadata{Prop: "settings", ExpiresAt: nil}
		if !reflect.DeepEqual(metadata.onceProps["user-settings"], want) {
			t.Fatalf("unexpected once metadata: %#v", metadata.onceProps)
		}
	})

	t.Run("fresh custom key is evaluated", func(t *testing.T) {
		calls := 0
		props := map[string]any{
			"settings": Once(func() (any, error) {
				calls++
				return "fresh", nil
			}).As("user-settings").Fresh(true),
		}
		headers := map[string]string{
			HeaderXInertia:                "true",
			HeaderXInertiaExceptOnceProps: "user-settings",
		}

		resolver, _ := newTestPropsResolver(t, "Dashboard", headers, func() time.Time { return fixedNow })
		got, metadata, err := resolver.resolve(nil, props)
		if err != nil {
			t.Fatal(err)
		}
		if got["settings"] != "fresh" || calls != 1 {
			t.Fatalf("fresh once prop was not evaluated exactly once: props=%#v calls=%d", got, calls)
		}
		if metadata.onceProps["user-settings"].Prop != "settings" {
			t.Fatalf("unexpected once metadata: %#v", metadata.onceProps)
		}
	})

	t.Run("duration uses request clock in milliseconds", func(t *testing.T) {
		calls := 0
		props := map[string]any{
			"settings": Once(func() (any, error) {
				calls++
				return "value", nil
			}).For(90 * time.Second),
		}

		resolver, _ := newTestPropsResolver(t, "Dashboard", map[string]string{HeaderXInertia: "true"}, func() time.Time { return fixedNow })
		got, metadata, err := resolver.resolve(nil, props)
		if err != nil {
			t.Fatal(err)
		}
		if got["settings"] != "value" || calls != 1 {
			t.Fatalf("once prop was not evaluated exactly once: props=%#v calls=%d", got, calls)
		}
		entry := metadata.onceProps["settings"]
		wantExpiry := fixedNow.Add(90 * time.Second).UnixMilli()
		if entry.ExpiresAt == nil || *entry.ExpiresAt != wantExpiry {
			t.Fatalf("unexpected expiry: got %#v, want %d", entry.ExpiresAt, wantExpiry)
		}
	})
}

func TestPropsResolverMergeMetadata(t *testing.T) {
	appendProp := Merge(map[string]any{"data": []int{1}}).Append("data").MatchOn("data.id")
	prependProp := Merge([]int{1}).Prepend()
	deepProp := DeepMerge(map[string]any{"data": []int{1}})
	resetProp := Merge([]int{1})
	props := map[string]any{
		"append":  appendProp,
		"prepend": prependProp,
		"deep":    deepProp,
		"reset":   resetProp,
	}
	headers := map[string]string{
		HeaderXInertia:      "true",
		HeaderXInertiaReset: "reset",
	}

	resolver, _ := newTestPropsResolver(t, "Dashboard", headers, nil)
	got, metadata, err := resolver.resolve(nil, props)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(props) {
		t.Fatalf("reset must suppress metadata, not the prop value: %#v", got)
	}
	if !reflect.DeepEqual(metadata.mergeProps, []string{"append.data"}) {
		t.Fatalf("unexpected append metadata: %#v", metadata.mergeProps)
	}
	if !reflect.DeepEqual(metadata.prependProps, []string{"prepend"}) {
		t.Fatalf("unexpected prepend metadata: %#v", metadata.prependProps)
	}
	if !reflect.DeepEqual(metadata.deepMergeProps, []string{"deep"}) {
		t.Fatalf("unexpected deep merge metadata: %#v", metadata.deepMergeProps)
	}
	if !reflect.DeepEqual(metadata.matchPropsOn, []string{"append.data.id"}) {
		t.Fatalf("unexpected match metadata: %#v", metadata.matchPropsOn)
	}
}

func TestPropsResolverScrollMetadataAndMergeIntent(t *testing.T) {
	tests := []struct {
		name        string
		headers     map[string]string
		dataPath    string
		metadata    ScrollMetadata
		wantMerge   []string
		wantPrepend []string
		wantReset   bool
	}{
		{
			name: "append by default",
			metadata: ScrollMetadata{
				PageName: "page", PreviousPage: nil, NextPage: 2, CurrentPage: 1,
			},
			wantMerge: []string{"feed.data"},
		},
		{
			name: "prepend cursor page",
			headers: map[string]string{
				HeaderXInertiaInfiniteScrollMergeIntent: "prepend",
			},
			metadata: ScrollMetadata{
				PageName: "cursor", PreviousPage: "previous", NextPage: "next", CurrentPage: "current",
			},
			dataPath:    "items",
			wantPrepend: []string{"feed.items"},
		},
		{
			name: "reset suppresses merge metadata",
			headers: map[string]string{
				HeaderXInertiaReset: "feed",
			},
			metadata: ScrollMetadata{
				PageName: "page", PreviousPage: nil, NextPage: nil, CurrentPage: 1,
			},
			wantReset: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			prop := Scroll(func() (ScrollResult, error) {
				calls++
				return ScrollResult{
					Value:    map[string]any{"data": []int{1, 2}, "items": []int{1, 2}},
					Metadata: tt.metadata,
				}, nil
			})
			if tt.dataPath != "" {
				prop.WithDataPath(tt.dataPath)
			}
			resolver, _ := newTestPropsResolver(t, "Dashboard", tt.headers, nil)
			got, metadata, err := resolver.resolve(nil, map[string]any{"feed": prop})
			if err != nil {
				t.Fatal(err)
			}
			if calls != 1 {
				t.Fatalf("expected scroll callback once, got %d", calls)
			}
			if !reflect.DeepEqual(got["feed"], map[string]any{"data": []int{1, 2}, "items": []int{1, 2}}) {
				t.Fatalf("unexpected scroll value: %#v", got["feed"])
			}
			wantScroll := tt.metadata
			wantScroll.Reset = tt.wantReset
			if !reflect.DeepEqual(metadata.scrollProps["feed"], wantScroll) {
				t.Fatalf("unexpected scroll metadata: got %#v, want %#v", metadata.scrollProps["feed"], wantScroll)
			}
			if !reflect.DeepEqual(metadata.mergeProps, tt.wantMerge) {
				t.Fatalf("unexpected append metadata: got %#v, want %#v", metadata.mergeProps, tt.wantMerge)
			}
			if !reflect.DeepEqual(metadata.prependProps, tt.wantPrepend) {
				t.Fatalf("unexpected prepend metadata: got %#v, want %#v", metadata.prependProps, tt.wantPrepend)
			}
		})
	}
}

func TestPropsResolverDeferredRescue(t *testing.T) {
	wantErr := errors.New("load failed")
	deferredCalls := 0
	okCalls := 0
	reporterCalls := 0
	var reportedPath string
	var reportedErr error
	props := map[string]any{
		"reports": map[string]any{
			"monthly": Defer(func() (any, error) {
				deferredCalls++
				return nil, wantErr
			}).Rescue(func(path string, err error) {
				reporterCalls++
				reportedPath = path
				reportedErr = err
			}),
		},
		"ok": func() any {
			okCalls++
			return "ready"
		},
	}
	headers := map[string]string{
		HeaderXInertia:                 "true",
		HeaderXInertiaPartialComponent: "Dashboard",
		HeaderXInertiaPartialData:      "reports.monthly,ok",
	}

	resolver, _ := newTestPropsResolver(t, "Dashboard", headers, nil)
	got, metadata, err := resolver.resolve(nil, props)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, map[string]any{"ok": "ready", "reports": map[string]any{}}) {
		t.Fatalf("rescued prop must be omitted while other props succeed: %#v", got)
	}
	if deferredCalls != 1 || okCalls != 1 || reporterCalls != 1 {
		t.Fatalf("unexpected callback counts: deferred=%d ok=%d reporter=%d", deferredCalls, okCalls, reporterCalls)
	}
	if reportedPath != "reports.monthly" || !errors.Is(reportedErr, wantErr) {
		t.Fatalf("unexpected rescue report: path=%q err=%v", reportedPath, reportedErr)
	}
	if !reflect.DeepEqual(metadata.rescuedProps, []string{"reports.monthly"}) {
		t.Fatalf("unexpected rescued metadata: %#v", metadata.rescuedProps)
	}
}

func TestPropsResolverPreservesLiteralDotKeysAndSharedTopLevelMetadata(t *testing.T) {
	shared := map[string]any{
		"auth": map[string]any{
			"user": "nested",
		},
		"auth.user": "shared literal",
	}
	page := map[string]any{
		"auth.user": "page literal",
	}

	resolver, _ := newTestPropsResolver(t, "Dashboard", nil, nil)
	got, metadata, err := resolver.resolve(shared, page)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"auth": map[string]any{
			"user": "nested",
		},
		"auth.user": "page literal",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("literal dot key was expanded or merged:\n got: %#v\nwant: %#v", got, want)
	}
	if !reflect.DeepEqual(metadata.sharedProps, []string{"auth", "auth.user"}) {
		t.Fatalf("sharedProps must contain the actual top-level keys: %#v", metadata.sharedProps)
	}
}

func TestPropsResolverDoesNotExploreNormalJSONTypes(t *testing.T) {
	innerCalls := 0
	marshalCalls := 0
	inner := func() any {
		innerCalls++
		return "resolved"
	}
	namedMap := resolverTestNamedMap{"callback": inner}
	namedSlice := resolverTestNamedSlice{inner}
	structValue := resolverTestStruct{Value: inner}
	raw := json.RawMessage(`{"value":"raw"}`)
	bytesValue := []byte{1, 2, 3}
	marshaler := resolverTestMarshaler{calls: &marshalCalls}
	props := map[string]any{
		"namedMap":   namedMap,
		"namedSlice": namedSlice,
		"struct":     structValue,
		"raw":        raw,
		"bytes":      bytesValue,
		"marshaler":  marshaler,
	}

	resolver, _ := newTestPropsResolver(t, "Dashboard", nil, nil)
	got, _, err := resolver.resolve(nil, props)
	if err != nil {
		t.Fatal(err)
	}
	if innerCalls != 0 || marshalCalls != 0 {
		t.Fatalf("normal JSON values were explored: callbacks=%d marshal=%d", innerCalls, marshalCalls)
	}
	if value, ok := got["namedMap"].(resolverTestNamedMap); !ok {
		t.Fatalf("named map type changed to %T", got["namedMap"])
	} else if _, ok := value["callback"].(func() any); !ok {
		t.Fatalf("named map callback was replaced with %T", value["callback"])
	}
	if value, ok := got["namedSlice"].(resolverTestNamedSlice); !ok {
		t.Fatalf("named slice type changed to %T", got["namedSlice"])
	} else if _, ok := value[0].(func() any); !ok {
		t.Fatalf("named slice callback was replaced with %T", value[0])
	}
	if value, ok := got["struct"].(resolverTestStruct); !ok {
		t.Fatalf("struct type changed to %T", got["struct"])
	} else if _, ok := value.Value.(func() any); !ok {
		t.Fatalf("struct callback was replaced with %T", value.Value)
	}
	if value, ok := got["raw"].(json.RawMessage); !ok || !reflect.DeepEqual(value, raw) {
		t.Fatalf("json.RawMessage changed: %#v (%T)", got["raw"], got["raw"])
	}
	if value, ok := got["bytes"].([]byte); !ok || !reflect.DeepEqual(value, bytesValue) {
		t.Fatalf("[]byte changed: %#v (%T)", got["bytes"], got["bytes"])
	}
	if _, ok := got["marshaler"].(resolverTestMarshaler); !ok {
		t.Fatalf("json.Marshaler type changed to %T", got["marshaler"])
	}
	encoded, err := json.Marshal(got["marshaler"])
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"value":"marshaled"}` || marshalCalls != 1 {
		t.Fatalf("json.Marshaler was not deferred to final JSON encoding: json=%s calls=%d", encoded, marshalCalls)
	}
}

func TestPropsResolverRejectsOmittableWrappersAsSliceElements(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{
			name: "Optional",
			value: Optional(func() (any, error) {
				return "optional", nil
			}),
		},
		{
			name: "Defer",
			value: Defer(func() (any, error) {
				return "deferred", nil
			}),
		},
		{
			name: "Once",
			value: Once(func() (any, error) {
				return "once", nil
			}),
		},
		{
			name: "deferred Scroll",
			value: Scroll(func() (ScrollResult, error) {
				return ScrollResult{Value: []int{1}}, nil
			}).Defer(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver, _ := newTestPropsResolver(t, "Dashboard", nil, nil)
			_, _, err := resolver.resolve(nil, map[string]any{"items": []any{tt.value}})
			if err == nil {
				t.Fatalf("expected omittable slice element error, got %v", err)
			}
			if want := `prop "items.0" uses an omittable wrapper directly as a slice element`; !strings.Contains(err.Error(), want) {
				t.Fatalf("expected error containing %q, got %v", want, err)
			}
		})
	}
}

func TestPropsResolverLoadedDeferredOnceOmitsDeferredMetadata(t *testing.T) {
	calls := 0
	prop := Defer(func() (any, error) {
		calls++
		return "value", nil
	}).Once().As("slow-report")
	headers := map[string]string{
		HeaderXInertia:                "true",
		HeaderXInertiaExceptOnceProps: "slow-report",
	}

	resolver, _ := newTestPropsResolver(t, "Dashboard", headers, nil)
	got, metadata, err := resolver.resolve(nil, map[string]any{"report": prop})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 || calls != 0 {
		t.Fatalf("loaded Defer+Once must be omitted without evaluation: props=%#v calls=%d", got, calls)
	}
	if len(metadata.deferredProps) != 0 {
		t.Fatalf("loaded Defer+Once must not schedule another deferred request: %#v", metadata.deferredProps)
	}
	if !reflect.DeepEqual(metadata.onceProps["slow-report"], OnceMetadata{Prop: "report", ExpiresAt: nil}) {
		t.Fatalf("once metadata must remain available: %#v", metadata.onceProps)
	}
}

func TestPropsResolverOptionalAndMergeOnceModifiersEncodeNoExpiryAsNull(t *testing.T) {
	optionalCalls := 0
	mergeCalls := 0
	props := map[string]any{
		"optional": Optional(func() (any, error) {
			optionalCalls++
			return "optional", nil
		}).Once().As("optional-key"),
		"merged": Merge(func() any {
			mergeCalls++
			return []int{1, 2}
		}).Once().As("merged-key"),
	}

	resolver, _ := newTestPropsResolver(t, "Dashboard", map[string]string{HeaderXInertia: "true"}, nil)
	got, metadata, err := resolver.resolve(nil, props)
	if err != nil {
		t.Fatal(err)
	}
	if optionalCalls != 0 || mergeCalls != 1 {
		t.Fatalf("unexpected callback counts: optional=%d merge=%d", optionalCalls, mergeCalls)
	}
	if _, exists := got["optional"]; exists {
		t.Fatalf("Optional must remain excluded from an initial response: %#v", got)
	}
	if !reflect.DeepEqual(got["merged"], []int{1, 2}) {
		t.Fatalf("unexpected merged value: %#v", got["merged"])
	}
	wantMetadata := map[string]OnceMetadata{
		"optional-key": {Prop: "optional", ExpiresAt: nil},
		"merged-key":   {Prop: "merged", ExpiresAt: nil},
	}
	if !reflect.DeepEqual(metadata.onceProps, wantMetadata) {
		t.Fatalf("unexpected once metadata: %#v", metadata.onceProps)
	}
	encoded, err := json.Marshal(metadata.onceProps)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"optional-key", "merged-key"} {
		if string(fields[key]["expiresAt"]) != "null" {
			t.Fatalf("%s expiresAt must encode as JSON null: %s", key, encoded)
		}
	}
}

func TestPropsResolverDeferredErrorsAndGlobalRescueReporter(t *testing.T) {
	wantErr := errors.New("load failed")
	headers := map[string]string{
		HeaderXInertia:                 "true",
		HeaderXInertiaPartialComponent: "Dashboard",
		HeaderXInertiaPartialData:      "report",
	}

	t.Run("error propagates without Rescue", func(t *testing.T) {
		calls := 0
		reporterCalls := 0
		resolver, inertia := newTestPropsResolver(t, "Dashboard", headers, nil)
		inertia.rescueReporter = func(string, error) {
			reporterCalls++
		}
		_, _, err := resolver.resolve(nil, map[string]any{
			"report": Defer(func() (any, error) {
				calls++
				return nil, wantErr
			}),
		})
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected original deferred error, got %v", err)
		}
		if calls != 1 || reporterCalls != 0 {
			t.Fatalf("unexpected callback counts: deferred=%d reporter=%d", calls, reporterCalls)
		}
	})

	t.Run("global reporter receives rescued error", func(t *testing.T) {
		calls := 0
		reporterCalls := 0
		var reportedPath string
		var reportedErr error
		resolver, inertia := newTestPropsResolver(t, "Dashboard", headers, nil)
		inertia.rescueReporter = func(path string, err error) {
			reporterCalls++
			reportedPath = path
			reportedErr = err
		}
		got, metadata, err := resolver.resolve(nil, map[string]any{
			"report": Defer(func() (any, error) {
				calls++
				return nil, wantErr
			}).Rescue(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 || calls != 1 || reporterCalls != 1 {
			t.Fatalf("unexpected rescue result: props=%#v deferred=%d reporter=%d", got, calls, reporterCalls)
		}
		if reportedPath != "report" || !errors.Is(reportedErr, wantErr) {
			t.Fatalf("unexpected global report: path=%q err=%v", reportedPath, reportedErr)
		}
		if !reflect.DeepEqual(metadata.rescuedProps, []string{"report"}) {
			t.Fatalf("unexpected rescued metadata: %#v", metadata.rescuedProps)
		}
	})
}

func newTestPropsResolver(
	t *testing.T,
	component string,
	headers map[string]string,
	now func() time.Time,
) (*propsResolver, *Inertia) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	e := echo.New()
	c := e.NewContext(req, httptest.NewRecorder())
	i := &Inertia{
		echoContext:      c,
		partialComponent: req.Header.Get(HeaderXInertiaPartialComponent),
		onlyProps:        splitAndRemoveEmpty(req.Header.Get(HeaderXInertiaPartialData), ","),
		exceptProps:      splitAndRemoveEmpty(req.Header.Get(HeaderXInertiaPartialExcept), ","),
		resetProps:       splitAndRemoveEmpty(req.Header.Get(HeaderXInertiaReset), ","),
		now:              now,
	}
	return newPropsResolver(i, component), i
}
