package inertia

type DeferProp struct {
	callback  func() (any, error)
	group     string
	merge     bool
	deepMerge bool
	matchesOn []string
}

func (p *DeferProp) IsIgnoreFirstLoad() {}

func (p *DeferProp) Group() string {
	return p.group
}

func (p *DeferProp) Merge() {
	p.merge = true
}

func (p *DeferProp) ShouldMerge() bool {
	return p.merge
}

func (p *DeferProp) DeepMerge() {
	p.deepMerge = true
}

func (p *DeferProp) ShouldDeepMerge() bool {
	return p.deepMerge
}

func (p *DeferProp) MatchesOn() []string {
	return p.matchesOn
}

func (p *DeferProp) MatchOn(fields ...string) *DeferProp {
	p.matchesOn = append(p.matchesOn, fields...)
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
