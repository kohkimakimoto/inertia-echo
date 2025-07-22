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

func splitAndRemoveEmpty(s string, sep string) []string {
	var ret []string
	if s == "" {
		return ret
	}

	for _, str := range strings.Split(s, sep) {
		str = strings.TrimSpace(str)
		if str != "" {
			ret = append(ret, str)
		}
	}

	return ret
}
