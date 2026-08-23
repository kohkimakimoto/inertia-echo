package inertia

import (
	"net/http"
	"strings"
)

func addVaryHeader(header http.Header, value string) {
	for _, existingValue := range header.Values("Vary") {
		for _, existingToken := range splitAndRemoveEmpty(existingValue, ",") {
			if existingToken == "*" || strings.EqualFold(existingToken, value) {
				return
			}
		}
	}

	header.Add("Vary", value)
}

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
