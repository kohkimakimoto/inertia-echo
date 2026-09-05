package inertia

import "time"

type DeferProp struct {
	callback  func() (any, error)
	group     string
	merge     bool
	deepMerge bool
	matchesOn []string
	mergeOpts mergeSettings
	onceOpts  onceSettings
	rescue    bool
	reporter  RescueReporter
}

func (p *DeferProp) IsIgnoreFirstLoad() {}

func (p *DeferProp) Group() string {
	return p.group
}

func (p *DeferProp) Merge() {
	p.merge = true
	p.mergeOpts = rootAppendMergeSettings()
}

func (p *DeferProp) ShouldMerge() bool {
	return p.merge
}

func (p *DeferProp) DeepMerge() {
	p.merge = true
	p.deepMerge = true
	p.mergeOpts = mergeSettings{deepMerge: true}
}

func (p *DeferProp) ShouldDeepMerge() bool {
	return p.deepMerge
}

func (p *DeferProp) MatchesOn() []string {
	return p.matchesOn
}

func (p *DeferProp) MatchOn(fields ...string) *DeferProp {
	p.matchesOn = append(p.matchesOn, fields...)
	p.mergeOpts.matchesOn = append(p.mergeOpts.matchesOn, fields...)
	return p
}

// Append declares client-side append merging. With paths, each path is
// relative to the prop; without paths, the prop itself is appended.
func (p *DeferProp) Append(paths ...string) *DeferProp {
	p.merge = true
	p.deepMerge = false
	p.mergeOpts.setAppend(paths)
	return p
}

// Prepend declares client-side prepend merging.
func (p *DeferProp) Prepend(paths ...string) *DeferProp {
	p.merge = true
	p.deepMerge = false
	p.mergeOpts.setPrepend(paths)
	return p
}

func (p *DeferProp) mergeSettings() mergeSettings {
	settings := p.mergeOpts
	if p.deepMerge {
		settings = mergeSettings{deepMerge: true, matchesOn: append([]string(nil), p.matchesOn...)}
	} else if p.merge && !settings.enabled() {
		settings = rootAppendMergeSettings()
	}
	if len(settings.matchesOn) == 0 {
		settings.matchesOn = append([]string(nil), p.matchesOn...)
	}
	return settings
}

// Once resolves this deferred prop once per client-side cache key.
func (p *DeferProp) Once() *DeferProp {
	p.onceOpts.enabled = true
	return p
}

func (p *DeferProp) As(key string) *DeferProp {
	p.onceOpts.setKey(key)
	return p
}

func (p *DeferProp) Fresh(fresh bool) *DeferProp {
	p.onceOpts.setFresh(fresh)
	return p
}

func (p *DeferProp) Until(expiresAt time.Time) *DeferProp {
	p.onceOpts.setUntil(expiresAt)
	return p
}

func (p *DeferProp) For(duration time.Duration) *DeferProp {
	p.onceOpts.setFor(duration)
	return p
}

func (p *DeferProp) onceSettings() onceSettings {
	return p.onceOpts
}

// Rescue omits this prop when its callback returns an error. The optional
// reporter is invoked in addition to the middleware-level reporter.
func (p *DeferProp) Rescue(reporters ...RescueReporter) *DeferProp {
	p.rescue = true
	if len(reporters) > 0 {
		p.reporter = reporters[0]
	}
	return p
}

func Defer(callback func() (any, error)) *DeferProp {
	return &DeferProp{
		callback: callback,
		group:    "default",
	}
}

func DeferWithGroup(callback func() (any, error), group string) *DeferProp {
	return &DeferProp{
		callback: callback,
		group:    group,
	}
}
