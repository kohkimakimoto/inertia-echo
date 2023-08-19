package inertia

import (
	"strings"
)

func inArray(needle string, heyStack []string) bool {
	for _, v := range heyStack {
		if needle == v {
			return true
		}
	}
	return false
}

func mergeProps(props ...map[string]interface{}) map[string]interface{} {
	merged := map[string]interface{}{}
	for _, a := range props {
		for k, v := range a {
			merged[k] = v
		}
	}
	return merged
}

func splitAndRemoveEmpty(s string, sep string) []string {
	var ret []string
	if s == "" {
		return ret
	}

	for _, str := range strings.Split(s, sep) {
		if str != "" {
			ret = append(ret, str)
		}
	}

	return ret
}

// evaluateProps evaluates the given props and update it.
func evaluateProps(values map[string]interface{}) error {
	for k, v := range values {
		switch converted := v.(type) {
		case map[string]interface{}:
			if err := evaluateProps(converted); err != nil {
				return err
			}
		case *LazyProp:
			vv, err := converted.callback()
			if err != nil {
				return err
			}
			values[k] = vv
		case func() (interface{}, error):
			vv, err := converted()
			if err != nil {
				return err
			}
			values[k] = vv
		case func() interface{}:
			values[k] = converted()
		}
	}
	return nil
}
