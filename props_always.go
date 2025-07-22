package inertia

type AlwaysProp struct {
	value any
}

func Always(value any) *AlwaysProp {
	return &AlwaysProp{
		value: value,
	}
}
