package inertia

import "time"

type OptionalProp struct {
	callback func() (any, error)
	onceOpts onceSettings
}

func (p *OptionalProp) IsIgnoreFirstLoad() {}

func (p *OptionalProp) Once() *OptionalProp {
	p.onceOpts.enabled = true
	return p
}

func (p *OptionalProp) As(key string) *OptionalProp {
	p.onceOpts.setKey(key)
	return p
}

func (p *OptionalProp) Fresh(fresh bool) *OptionalProp {
	p.onceOpts.setFresh(fresh)
	return p
}

func (p *OptionalProp) Until(expiresAt time.Time) *OptionalProp {
	p.onceOpts.setUntil(expiresAt)
	return p
}

func (p *OptionalProp) For(duration time.Duration) *OptionalProp {
	p.onceOpts.setFor(duration)
	return p
}

func (p *OptionalProp) onceSettings() onceSettings {
	return p.onceOpts
}

func Optional(callback func() (any, error)) *OptionalProp {
	return &OptionalProp{
		callback: callback,
	}
}
