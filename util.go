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
func evaluateProps(values map[string]interface{}) {
	for k, v := range values {
		switch converted := v.(type) {
		case map[string]interface{}:
			evaluateProps(converted)
		case *LazyProp:
			values[k] = converted.callback()
		case func() interface{}:
			values[k] = converted()
		}
	}
}
