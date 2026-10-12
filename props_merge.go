package inertia

import "time"

// Mergeable represents a prop that can be merged
type Mergeable interface {
	ShouldMerge() bool
	ShouldDeepMerge() bool
	MatchesOn() []string
}

type MergeProp struct {
	value     any
	deepMerge bool
	matchesOn []string
	mergeOpts mergeSettings
	onceOpts  onceSettings
}

func (p *MergeProp) ShouldMerge() bool {
	return true
}

func (p *MergeProp) DeepMerge() {
	p.deepMerge = true
	p.mergeOpts = mergeSettings{deepMerge: true, matchesOn: append([]string(nil), p.matchesOn...)}
}

func (p *MergeProp) ShouldDeepMerge() bool {
	return p.deepMerge
}

func (p *MergeProp) MatchesOn() []string {
	return p.matchesOn
}

func (p *MergeProp) MatchOn(fields ...string) *MergeProp {
	p.matchesOn = append(p.matchesOn, fields...)
	p.mergeOpts.matchesOn = append(p.mergeOpts.matchesOn, fields...)
	return p
}

func (p *MergeProp) Append(paths ...string) *MergeProp {
	p.deepMerge = false
	p.mergeOpts.setAppend(paths)
	return p
}

func (p *MergeProp) Prepend(paths ...string) *MergeProp {
	p.deepMerge = false
	p.mergeOpts.setPrepend(paths)
	return p
}

func (p *MergeProp) mergeSettings() mergeSettings {
	settings := p.mergeOpts
	if p.deepMerge {
		settings = mergeSettings{deepMerge: true, matchesOn: append([]string(nil), p.matchesOn...)}
	}
	if len(settings.matchesOn) == 0 {
		settings.matchesOn = append([]string(nil), p.matchesOn...)
	}
	return settings
}

func (p *MergeProp) Once() *MergeProp {
	p.onceOpts.enabled = true
	return p
}

func (p *MergeProp) As(key string) *MergeProp {
	p.onceOpts.setKey(key)
	return p
}

func (p *MergeProp) Fresh(fresh bool) *MergeProp {
	p.onceOpts.setFresh(fresh)
	return p
}

func (p *MergeProp) Until(expiresAt time.Time) *MergeProp {
	p.onceOpts.setUntil(expiresAt)
	return p
}

func (p *MergeProp) For(duration time.Duration) *MergeProp {
	p.onceOpts.setFor(duration)
	return p
}

func (p *MergeProp) onceSettings() onceSettings {
	return p.onceOpts
}

func Merge(value any) *MergeProp {
	return &MergeProp{
		value:     value,
		deepMerge: false,
		matchesOn: []string{},
		mergeOpts: rootAppendMergeSettings(),
	}
}

func DeepMerge(value any) *MergeProp {
	return &MergeProp{
		value:     value,
		deepMerge: true,
		matchesOn: []string{},
		mergeOpts: mergeSettings{deepMerge: true},
	}
}
