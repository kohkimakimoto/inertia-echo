package inertia

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type propsMetadata struct {
	sharedProps    []string
	deferredProps  map[string]any
	mergeProps     []string
	prependProps   []string
	deepMergeProps []string
	matchPropsOn   []string
	scrollProps    map[string]ScrollMetadata
	onceProps      map[string]OnceMetadata
	rescuedProps   []string
}

type propsResolver struct {
	inertia     *Inertia
	component   string
	isPartial   bool
	isInertia   bool
	only        []string
	except      []string
	reset       []string
	loadedOnce  map[string]struct{}
	now         time.Time
	metadata    propsMetadata
	scrollValue map[string]ScrollMetadata
}

func newPropsResolver(i *Inertia, component string) *propsResolver {
	req := i.echoContext.Request()
	loadedOnce := make(map[string]struct{})
	for _, key := range splitAndRemoveEmpty(req.Header.Get(HeaderXInertiaExceptOnceProps), ",") {
		loadedOnce[key] = struct{}{}
	}

	now := time.Now()
	if i.now != nil {
		now = i.now()
	}

	return &propsResolver{
		inertia:    i,
		component:  component,
		isPartial:  i.isPartial(component),
		isInertia:  req.Header.Get(HeaderXInertia) != "",
		only:       append([]string(nil), i.onlyProps...),
		except:     append([]string(nil), i.exceptProps...),
		reset:      append([]string(nil), i.resetProps...),
		loadedOnce: loadedOnce,
		now:        now,
		metadata: propsMetadata{
			deferredProps: make(map[string]any),
			scrollProps:   make(map[string]ScrollMetadata),
			onceProps:     make(map[string]OnceMetadata),
		},
		scrollValue: make(map[string]ScrollMetadata),
	}
}

func (r *propsResolver) resolve(shared, page map[string]any) (map[string]any, propsMetadata, error) {
	if shared == nil {
		shared = map[string]any{}
	}
	if page == nil {
		page = map[string]any{}
	}

	for key := range shared {
		r.metadata.sharedProps = append(r.metadata.sharedProps, key)
	}

	definitions := make(map[string]any, len(shared)+len(page))
	for key, value := range shared {
		definitions[key] = value
	}
	for key, value := range page {
		definitions[key] = value
	}

	props, err := r.resolveMap(definitions, "", false)
	if err != nil {
		return nil, propsMetadata{}, err
	}
	r.sortMetadata()
	return props, r.metadata, nil
}

func (r *propsResolver) resolveMap(values map[string]any, prefix string, partialBypass bool) (map[string]any, error) {
	result := make(map[string]any)
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := values[key]
		path := joinPropPath(prefix, key)
		resolved, include, err := r.resolveEntry(value, path, partialBypass, false)
		if err != nil {
			return nil, err
		}
		if include {
			result[key] = resolved
		}
	}
	return result, nil
}

func (r *propsResolver) resolveEntry(value any, path string, partialBypass, sliceElement bool) (any, bool, error) {
	if !partialBypass && !r.shouldInclude(value, path) {
		return nil, false, nil
	}
	if sliceElement && canOmitProp(value) {
		return nil, false, fmt.Errorf("inertia-echo: prop %q uses an omittable wrapper directly as a slice element", path)
	}
	if !r.isPartial && r.excludeFromInitial(value, path) {
		return nil, false, nil
	}

	resolved, err := r.resolveValue(value, path, partialBypass)
	if err != nil {
		if prop, ok := value.(*DeferProp); ok && prop.rescue {
			if prop.reporter != nil {
				prop.reporter(path, err)
			}
			if r.inertia.rescueReporter != nil {
				r.inertia.rescueReporter(path, err)
			}
			r.metadata.rescuedProps = append(r.metadata.rescuedProps, path)
			return nil, false, nil
		}
		return nil, false, err
	}

	r.collectMetadata(value, path)
	return resolved, true, nil
}

func (r *propsResolver) resolveValue(value any, path string, partialBypass bool) (any, error) {
	switch prop := value.(type) {
	case *AlwaysProp:
		return r.resolveDefinitionValue(prop.value, path, true)
	case *MergeProp:
		return r.resolveDefinitionValue(prop.value, path, partialBypass)
	case *ScrollProp:
		result, err := prop.callback()
		if err != nil {
			return nil, err
		}
		result.Metadata.Reset = inArray(path, r.reset)
		r.scrollValue[path] = result.Metadata
		return result.Value, nil
	case map[string]any:
		return r.resolveMap(prop, path, partialBypass)
	case []any:
		return r.resolveAnySlice(prop, path, partialBypass)
	case []map[string]any:
		return r.resolveMapSlice(prop, path, partialBypass)
	default:
		return evaluatePropValue(value)
	}
}

func (r *propsResolver) resolveDefinitionValue(value any, path string, partialBypass bool) (any, error) {
	switch value := value.(type) {
	case map[string]any:
		return r.resolveMap(value, path, partialBypass)
	case []any:
		return r.resolveAnySlice(value, path, partialBypass)
	case []map[string]any:
		return r.resolveMapSlice(value, path, partialBypass)
	default:
		return evaluatePropValue(value)
	}
}

func (r *propsResolver) resolveAnySlice(values []any, prefix string, partialBypass bool) ([]any, error) {
	result := make([]any, len(values))
	for index, value := range values {
		path := joinPropPath(prefix, fmt.Sprintf("%d", index))
		switch typed := value.(type) {
		case map[string]any:
			resolved, err := r.resolveMap(typed, path, partialBypass)
			if err != nil {
				return nil, err
			}
			result[index] = resolved
		case []any:
			resolved, err := r.resolveAnySlice(typed, path, partialBypass)
			if err != nil {
				return nil, err
			}
			result[index] = resolved
		case []map[string]any:
			resolved, err := r.resolveMapSlice(typed, path, partialBypass)
			if err != nil {
				return nil, err
			}
			result[index] = resolved
		default:
			resolved, include, err := r.resolveEntry(value, path, partialBypass, true)
			if err != nil {
				return nil, err
			}
			if !include {
				return nil, fmt.Errorf("inertia-echo: prop %q cannot be omitted from a slice", path)
			}
			result[index] = resolved
		}
	}
	return result, nil
}

func (r *propsResolver) resolveMapSlice(values []map[string]any, prefix string, partialBypass bool) ([]map[string]any, error) {
	result := make([]map[string]any, len(values))
	for index, value := range values {
		resolved, err := r.resolveMap(value, joinPropPath(prefix, fmt.Sprintf("%d", index)), partialBypass)
		if err != nil {
			return nil, err
		}
		result[index] = resolved
	}
	return result, nil
}

func canOmitProp(value any) bool {
	if _, ok := value.(IgnoreFirstLoadProp); ok {
		return true
	}
	if settings, ok := getOnceSettings(value); ok && settings.enabled {
		return true
	}
	if prop, ok := value.(*ScrollProp); ok && prop.shouldDefer() {
		return true
	}
	return false
}

func (r *propsResolver) shouldInclude(value any, path string) bool {
	if !r.isPartial {
		return true
	}
	if _, ok := value.(*AlwaysProp); ok {
		return true
	}
	if len(r.only) > 0 && !matchesOnly(path, r.only) && !leadsToOnly(path, r.only) {
		return false
	}
	if len(r.except) > 0 && matchesExcept(path, r.except) {
		return false
	}
	return true
}

func (r *propsResolver) excludeFromInitial(value any, path string) bool {
	if _, ignored := value.(IgnoreFirstLoadProp); ignored {
		if prop, ok := value.(*DeferProp); ok && !r.wasLoadedOnce(value, path) {
			r.collectDeferred(path, prop.Group())
		}
		r.collectMerge(value, path)
		r.collectOnce(value, path)
		return true
	}

	if prop, ok := value.(*ScrollProp); ok && prop.shouldDefer() {
		r.collectDeferred(path, prop.group())
		r.collectScrollMerge(prop, path)
		return true
	}

	if r.isInertia && r.wasLoadedOnce(value, path) {
		r.collectOnce(value, path)
		return true
	}
	return false
}

func (r *propsResolver) wasLoadedOnce(value any, path string) bool {
	settings, ok := getOnceSettings(value)
	if !ok || !settings.enabled || settings.fresh {
		return false
	}
	key := settings.key
	if key == "" {
		key = path
	}
	_, loaded := r.loadedOnce[key]
	return loaded
}

func getOnceSettings(value any) (onceSettings, bool) {
	provider, ok := value.(onceSettingsProvider)
	if !ok {
		return onceSettings{}, false
	}
	return provider.onceSettings(), true
}

func (r *propsResolver) collectMetadata(value any, path string) {
	r.collectMerge(value, path)
	r.collectOnce(value, path)
	if prop, ok := value.(*ScrollProp); ok {
		if metadata, exists := r.scrollValue[path]; exists {
			r.metadata.scrollProps[path] = metadata
		}
		r.collectScrollMerge(prop, path)
	}
}

func (r *propsResolver) collectDeferred(path, group string) {
	if group == "" {
		group = "default"
	}
	paths, _ := r.metadata.deferredProps[group].([]string)
	r.metadata.deferredProps[group] = append(paths, path)
}

func (r *propsResolver) collectOnce(value any, path string) {
	settings, ok := getOnceSettings(value)
	if !ok || !settings.enabled || !r.includePartialMetadata(path) {
		return
	}
	key := settings.key
	if key == "" {
		key = path
	}
	r.metadata.onceProps[key] = OnceMetadata{
		Prop:      path,
		ExpiresAt: settings.expiresAt(r.now),
	}
}

func (r *propsResolver) collectMerge(value any, path string) {
	mergeable, ok := value.(Mergeable)
	if !ok || !mergeable.ShouldMerge() || inArray(path, r.reset) || !r.includePartialMetadata(path) {
		return
	}

	settings := rootAppendMergeSettings()
	if provider, ok := value.(mergeSettingsProvider); ok {
		settings = provider.mergeSettings()
	} else {
		settings.deepMerge = mergeable.ShouldDeepMerge()
		if settings.deepMerge {
			settings.appendRoot = false
		}
		settings.matchesOn = append([]string(nil), mergeable.MatchesOn()...)
	}
	r.collectMergeSettings(path, settings)
}

func (r *propsResolver) collectScrollMerge(prop *ScrollProp, path string) {
	if inArray(path, r.reset) || !r.includePartialMetadata(path) {
		return
	}
	settings := mergeSettings{}
	dataPath := prop.dataPath
	if dataPath == "" {
		if r.inertia.echoContext.Request().Header.Get(HeaderXInertiaInfiniteScrollMergeIntent) == "prepend" {
			settings.prependRoot = true
		} else {
			settings.appendRoot = true
		}
	} else if r.inertia.echoContext.Request().Header.Get(HeaderXInertiaInfiniteScrollMergeIntent) == "prepend" {
		settings.prependPaths = []string{dataPath}
	} else {
		settings.appendPaths = []string{dataPath}
	}
	r.collectMergeSettings(path, settings)
}

func (r *propsResolver) collectMergeSettings(path string, settings mergeSettings) {
	if settings.deepMerge {
		r.metadata.deepMergeProps = append(r.metadata.deepMergeProps, path)
	} else if settings.appendRoot {
		r.metadata.mergeProps = append(r.metadata.mergeProps, path)
	} else if settings.prependRoot {
		r.metadata.prependProps = append(r.metadata.prependProps, path)
	} else {
		for _, child := range settings.appendPaths {
			r.metadata.mergeProps = append(r.metadata.mergeProps, joinPropPath(path, child))
		}
		for _, child := range settings.prependPaths {
			r.metadata.prependProps = append(r.metadata.prependProps, joinPropPath(path, child))
		}
	}
	for _, strategy := range settings.matchesOn {
		r.metadata.matchPropsOn = append(r.metadata.matchPropsOn, joinPropPath(path, strategy))
	}
}

func (r *propsResolver) includePartialMetadata(path string) bool {
	if !r.isPartial {
		return true
	}
	if len(r.only) > 0 && !matchesOnly(path, r.only) {
		return false
	}
	return len(r.except) == 0 || !matchesExcept(path, r.except)
}

func matchesOnly(path string, filters []string) bool {
	for _, filter := range filters {
		if path == filter || strings.HasPrefix(path, filter+".") {
			return true
		}
	}
	return false
}

func leadsToOnly(path string, filters []string) bool {
	for _, filter := range filters {
		if strings.HasPrefix(filter, path+".") {
			return true
		}
	}
	return false
}

func matchesExcept(path string, filters []string) bool {
	for _, filter := range filters {
		if path == filter || strings.HasPrefix(path, filter+".") {
			return true
		}
	}
	return false
}

func joinPropPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	if key == "" {
		return prefix
	}
	return prefix + "." + key
}

func (r *propsResolver) sortMetadata() {
	r.metadata.sharedProps = sortedUnique(r.metadata.sharedProps)
	r.metadata.mergeProps = sortedUnique(r.metadata.mergeProps)
	r.metadata.prependProps = sortedUnique(r.metadata.prependProps)
	r.metadata.deepMergeProps = sortedUnique(r.metadata.deepMergeProps)
	r.metadata.matchPropsOn = sortedUnique(r.metadata.matchPropsOn)
	r.metadata.rescuedProps = sortedUnique(r.metadata.rescuedProps)
	for group, value := range r.metadata.deferredProps {
		paths, ok := value.([]string)
		if !ok {
			continue
		}
		r.metadata.deferredProps[group] = sortedUnique(paths)
	}
}

func sortedUnique(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	values = append([]string(nil), values...)
	sort.Strings(values)
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || result[len(result)-1] != value {
			result = append(result, value)
		}
	}
	return result
}
