package inertia

import (
	"testing"
)

func TestInArray(t *testing.T) {
	if !inArray("c", []string{"a", "b", "c", "d"}) {
		t.Error("expected true but false")
	}

	if inArray("e", []string{"a", "b", "c", "d"}) {
		t.Error("expected false but true")
	}
}

func TestMergeProps(t *testing.T) {
	a := map[string]interface{}{
		"a": "a-aaa",
		"b": "a-bbb",
		"c": "a-ccc",
	}
	b := map[string]interface{}{
		"c": "b-ccc",
		"d": "b-ddd",
		"e": "b-eee",
	}

	ret := mergeProps(a, b)
	if len(ret) != 5 {
		t.Errorf("expected 5 but %v", len(ret))
	}
	if ret["a"] != "a-aaa" {
		t.Errorf("expected 'a-aaa' but %v", ret["a"])
	}
	if ret["b"] != "a-bbb" {
		t.Errorf("expected 'a-aaa' but %v", ret["b"])
	}
	if ret["c"] != "b-ccc" {
		t.Errorf("expected 'a-aaa' but %v", ret["c"])
	}
	if ret["d"] != "b-ddd" {
		t.Errorf("expected 'a-aaa' but %v", ret["d"])
	}
	if ret["e"] != "b-eee" {
		t.Errorf("expected 'a-aaa' but %v", ret["e"])
	}
}

func TestSplitOrNil(t *testing.T) {
	ret := splitOrNil("aaa", ",")
	if ret[0] != "aaa" {
		t.Errorf("expected 'aaa' but %v", ret[0])
	}

	ret = splitOrNil("aaa,bbb,ccc", ",")
	if ret[0] != "aaa" {
		t.Errorf("expected 'aaa' but %v", ret[0])
	}
	if ret[1] != "bbb" {
		t.Errorf("expected 'bbb' but %v", ret[1])
	}
	if ret[2] != "ccc" {
		t.Errorf("expected 'ccc' but %v", ret[2])
	}

	ret = splitOrNil(",,,", ",")
	if ret != nil {
		t.Errorf("expected nil but %v", ret)
	}

	ret = splitOrNil("", ",")
	if ret != nil {
		t.Errorf("expected nil but %v", ret)
	}
}

func TestEvaluatePropsRecursive(t *testing.T) {
	a := map[string]interface{}{
		"a": "aaa",
		"b": map[string]interface{}{
			"b-a": "b-aaa",
			"b-b": map[string]interface{}{
				"b-b-a": "b-b-aaa",
			},
		},
		"c": Lazy(func() interface{} {
			return "ccc"
		}),
		"d": func() interface{} {
			return "ddd"
		},
	}

	evaluatePropsRecursive(a)
	if a["c"] != "ccc" {
		t.Errorf("expected 'ccc' but %v", a["c"])
	}
	if a["d"] != "ddd" {
		t.Errorf("expected 'ddd' but %v", a["d"])
	}
	t.Log(a)
}
