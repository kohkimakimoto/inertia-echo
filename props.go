package inertia

// IgnoreFirstLoadProp represents a prop that should be ignored on first load
type IgnoreFirstLoadProp interface {
	IsIgnoreFirstLoad()
}

// evaluateProps evaluates the given props and update it.
// It is the same purpose as resolvePropertyInstances that is used in official inertia-laravel package.
func evaluateProps(values map[string]any) error {
	for k, v := range values {
		vv, err := evaluatePropValue(v)
		if err != nil {
			return err
		}
		values[k] = vv
	}
	return nil
}

func evaluatePropValue(value any) (any, error) {
	switch v := value.(type) {
	case *LazyProp:
		vv, err := v.callback()
		if err != nil {
			return nil, err
		}
		return evaluatePropValue(vv)
	case *OptionalProp:
		vv, err := v.callback()
		if err != nil {
			return nil, err
		}
		return evaluatePropValue(vv)
	case *DeferProp:
		vv, err := v.callback()
		if err != nil {
			return nil, err
		}
		return evaluatePropValue(vv)
	case *AlwaysProp:
		return evaluatePropValue(v.value)
	case *MergeProp:
		return evaluatePropValue(v.value)
	case func() (any, error):
		vv, err := v()
		if err != nil {
			return nil, err
		}
		return evaluatePropValue(vv)
	case func() any:
		return evaluatePropValue(v())
	default:
		return value, nil
	}
}
