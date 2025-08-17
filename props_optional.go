package inertia

type OptionalProp struct {
	callback func() (any, error)
}

func (p *OptionalProp) IsIgnoreFirstLoad() {}

func Optional(callback func() (any, error)) *OptionalProp {
	return &OptionalProp{
		callback: callback,
	}
}
