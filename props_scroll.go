package inertia

// ScrollMetadata contains pagination state for an infinite-scroll prop.
// Page identifiers may be numbers, cursor strings, or nil.
type ScrollMetadata struct {
	PageName     string `json:"pageName"`
	PreviousPage any    `json:"previousPage"`
	NextPage     any    `json:"nextPage"`
	CurrentPage  any    `json:"currentPage"`
	Reset        bool   `json:"reset"`
}

type ScrollResult struct {
	Value    any
	Metadata ScrollMetadata
}

type ScrollProp struct {
	callback     func() (ScrollResult, error)
	dataPath     string
	deferEnabled bool
	deferGroup   string
}

func Scroll(callback func() (ScrollResult, error)) *ScrollProp {
	return &ScrollProp{
		callback:   callback,
		dataPath:   "data",
		deferGroup: "default",
	}
}

func (p *ScrollProp) WithDataPath(path string) *ScrollProp {
	p.dataPath = path
	return p
}

// Defer delays the initial evaluation. An optional non-empty group overrides
// the default deferred group.
func (p *ScrollProp) Defer(group ...string) *ScrollProp {
	p.deferEnabled = true
	if len(group) > 0 && group[0] != "" {
		p.deferGroup = group[0]
	}
	return p
}

func (p *ScrollProp) shouldDefer() bool {
	return p.deferEnabled
}

func (p *ScrollProp) group() string {
	return p.deferGroup
}
