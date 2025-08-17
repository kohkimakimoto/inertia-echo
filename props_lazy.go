package inertia

// LazyProp is the same as Optional
// It is for backward compatibility.
type LazyProp struct {
	callback func() (any, error)
}

func (p *LazyProp) IsIgnoreFirstLoad() {}

func Lazy(callback func() (any, error)) *LazyProp {
	return &LazyProp{
		callback: callback,
	}
}
