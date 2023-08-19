package inertia

import (
	"testing"
)

func TestInArray(t *testing.T) {
	tests := []struct {
		needle   string
		haystack []string
		expected bool
	}{
		{
			needle:   "c",
			haystack: []string{"a", "b", "c", "d"},
			expected: true,
		},
		{
			needle:   "e",
			haystack: []string{"a", "b", "c", "d"},
			expected: false,
		},
	}

	for _, tt := range tests {
		if ret := inArray(tt.needle, tt.haystack); ret != tt.expected {
			t.Errorf("expected %v but %v", tt.expected, ret)
		}
	}
}

func TestMergeProps(t *testing.T) {
	tests := []struct {
		a        map[string]interface{}
		b        map[string]interface{}
		expected map[string]interface{}
	}{
		{
			a: map[string]interface{}{
				"a": "a-aaa",
				"b": "a-bbb",
				"c": "a-ccc",
			},
			b: map[string]interface{}{
				"c": "b-ccc",
				"d": "b-ddd",
				"e": "b-eee",
			},
			expected: map[string]interface{}{
				"a": "a-aaa",
				"b": "a-bbb",
				"c": "b-ccc",
				"d": "b-ddd",
				"e": "b-eee",
			},
		},
	}

	for _, tt := range tests {
		ret := mergeProps(tt.a, tt.b)
		if len(ret) != len(tt.expected) {
			t.Errorf("expected %v but %v", len(tt.expected), len(ret))
		}
		for k, v := range tt.expected {
			if ret[k] != v {
				t.Errorf("expected %v but %v", v, ret[k])
			}
		}
	}
}

func TestSplitAndRemoveEmpty(t *testing.T) {
	tests := []struct {
		s        string
		sep      string
		expected []string
	}{
		{
			s:        "aaa",
			sep:      ",",
			expected: []string{"aaa"},
		},
		{
			s:        "aaa,bbb,ccc",
			sep:      ",",
			expected: []string{"aaa", "bbb", "ccc"},
		},
		{
			s:        ",,,",
			sep:      ",",
			expected: []string{},
		},
		{
			s:        "",
			sep:      ",",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		ret := splitAndRemoveEmpty(tt.s, tt.sep)
		if len(ret) != len(tt.expected) {
			t.Errorf("expected %v but %v", len(tt.expected), len(ret))
		}
		for i, v := range tt.expected {
			if ret[i] != v {
				t.Errorf("expected %v but %v", v, ret[i])
			}
		}
	}
}

func TestEvaluateProps(t *testing.T) {
	a := map[string]interface{}{
		"a": "aaa",
		"b": map[string]interface{}{
			"b-a": "b-aaa",
			"b-b": map[string]interface{}{
				"b-b-a": "b-b-aaa",
			},
		},
		"c": Lazy(func() (interface{}, error) {
			return "ccc", nil
		}),
		"d": func() interface{} {
			return "ddd"
		},
	}

	evaluateProps(a)
	if a["c"] != "ccc" {
		t.Errorf("expected 'ccc' but %v", a["c"])
	}
	if a["d"] != "ddd" {
		t.Errorf("expected 'ddd' but %v", a["d"])
	}
}
