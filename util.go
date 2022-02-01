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
		if a != nil {
			for k, v := range a {
				merged[k] = v
			}
		}
	}
	return merged
}

func splitOrNil(s string, sep string) []string {
	if s == "" {
		return nil
	}

	var ret []string
	for _, str := range strings.Split(s, sep) {
		if str != "" {
			ret = append(ret, str)
		}
	}

	if len(ret) == 0 {
		return nil
	}

	return ret
}

// evaluatePropsRecursive evaluates the given props and update it.
func evaluatePropsRecursive(values map[string]interface{}) {
	for k, v := range values {
		switch converted := v.(type) {
		case map[string]interface{}:
			evaluatePropsRecursive(converted)
		case *LazyProp:
			values[k] = converted.callback()
		case func() interface{}:
			values[k] = converted()
		}
	}
}
