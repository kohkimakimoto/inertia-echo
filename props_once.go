package inertia

import "time"

type onceSettings struct {
	enabled  bool
	key      string
	fresh    bool
	until    *time.Time
	duration *time.Duration
}

func (s *onceSettings) setKey(key string) {
	s.enabled = true
	s.key = key
}

func (s *onceSettings) setFresh(fresh bool) {
	s.enabled = true
	s.fresh = fresh
}

func (s *onceSettings) setUntil(expiresAt time.Time) {
	s.enabled = true
	s.until = &expiresAt
	s.duration = nil
}

func (s *onceSettings) setFor(duration time.Duration) {
	s.enabled = true
	s.duration = &duration
	s.until = nil
}

func (s onceSettings) expiresAt(now time.Time) *int64 {
	if s.until != nil {
		value := s.until.UnixMilli()
		return &value
	}
	if s.duration != nil {
		value := now.Add(*s.duration).UnixMilli()
		return &value
	}
	return nil
}

type onceSettingsProvider interface {
	onceSettings() onceSettings
}

// OnceProp is a prop whose value can be reused by the Inertia client across
// visits. The server remains stateless; the client reports loaded keys.
type OnceProp struct {
	callback func() (any, error)
	settings onceSettings
}

func Once(callback func() (any, error)) *OnceProp {
	return &OnceProp{
		callback: callback,
		settings: onceSettings{enabled: true},
	}
}

func (p *OnceProp) As(key string) *OnceProp {
	p.settings.setKey(key)
	return p
}

func (p *OnceProp) Fresh(fresh bool) *OnceProp {
	p.settings.setFresh(fresh)
	return p
}

func (p *OnceProp) Until(expiresAt time.Time) *OnceProp {
	p.settings.setUntil(expiresAt)
	return p
}

func (p *OnceProp) For(duration time.Duration) *OnceProp {
	p.settings.setFor(duration)
	return p
}

func (p *OnceProp) onceSettings() onceSettings {
	return p.settings
}

// OnceMetadata describes a client-side once-prop cache entry. A nil
// ExpiresAt is encoded as JSON null and means that the entry does not expire.
type OnceMetadata struct {
	Prop      string `json:"prop"`
	ExpiresAt *int64 `json:"expiresAt"`
}
