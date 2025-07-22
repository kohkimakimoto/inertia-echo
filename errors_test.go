package inertia

import (
	"testing"
)

func TestErrorMessageMap_Set(t *testing.T) {
	emm := NewErrorMessageMap()
	emm.Set("key1", "error1")
	if emm.Len() != 1 {
		t.Errorf("Expected length 1, got %d", emm.Len())
	}

	value, exists := emm.Get("key1")
	if !exists {
		t.Error("Expected key1 to exist")
	}
	if value != "error1" {
		t.Errorf("Expected value 'error1', got %q", value)
	}
}

func TestErrorMessageMap_Get(t *testing.T) {
	emm := NewErrorMessageMap()
	emm.Set("key1", "error1")

	// Test existing key
	value, exists := emm.Get("key1")
	if !exists {
		t.Error("Expected key1 to exist")
	}
	if value != "error1" {
		t.Errorf("Expected value 'error1', got %q", value)
	}

	// Test non-existing key
	value, exists = emm.Get("nonexistent")
	if exists {
		t.Error("Expected nonexistent key to not exist")
	}
	if value != "" {
		t.Errorf("Expected empty value for nonexistent key, got %q", value)
	}
}

func TestErrorMessageMap_Update(t *testing.T) {
	emm := NewErrorMessageMap()
	emm.Set("key1", "error1")

	updates := map[string]string{
		"key1": "updated_error1",
		"key2": "error2",
		"key3": "error3",
	}

	result := emm.Update(updates)

	if result != emm {
		t.Error("Update should return the same ErrorMessageMap instance")
	}

	if emm.Len() != 3 {
		t.Errorf("Expected length 3, got %d", emm.Len())
	}

	// Test updated value
	value, exists := emm.Get("key1")
	if !exists || value != "updated_error1" {
		t.Errorf("Expected key1 to have value 'updated_error1', got %q (exists: %t)", value, exists)
	}

	// Test new values
	value, exists = emm.Get("key2")
	if !exists || value != "error2" {
		t.Errorf("Expected key2 to have value 'error2', got %q (exists: %t)", value, exists)
	}

	value, exists = emm.Get("key3")
	if !exists || value != "error3" {
		t.Errorf("Expected key3 to have value 'error3', got %q (exists: %t)", value, exists)
	}
}

func TestErrorMessageMap_Len(t *testing.T) {
	emm := NewErrorMessageMap()

	if emm.Len() != 0 {
		t.Errorf("Expected length 0, got %d", emm.Len())
	}

	emm.Set("key1", "error1")
	if emm.Len() != 1 {
		t.Errorf("Expected length 1, got %d", emm.Len())
	}

	emm.Set("key2", "error2")
	if emm.Len() != 2 {
		t.Errorf("Expected length 2, got %d", emm.Len())
	}

	// Setting same key should not increase length
	emm.Set("key1", "updated_error1")
	if emm.Len() != 2 {
		t.Errorf("Expected length 2 after updating existing key, got %d", emm.Len())
	}
}

func TestErrorMessageMap_ToMap(t *testing.T) {
	emm := NewErrorMessageMap()
	emm.Set("key1", "error1")
	emm.Set("key2", "error2")

	result := emm.ToMap()

	if len(result) != 2 {
		t.Errorf("Expected map length 2, got %d", len(result))
	}

	if result["key1"] != "error1" {
		t.Errorf("Expected result['key1'] to be 'error1', got %q", result["key1"])
	}

	if result["key2"] != "error2" {
		t.Errorf("Expected result['key2'] to be 'error2', got %q", result["key2"])
	}
}

func TestErrorMessageMap_Clear(t *testing.T) {
	emm := NewErrorMessageMap()
	emm.Set("key1", "error1")
	emm.Set("key2", "error2")
	emm.Clear()
	if emm.Len() != 0 {
		t.Errorf("Expected length 0 after clear, got %d", emm.Len())
	}

	_, exists := emm.Get("key1")
	if exists {
		t.Error("Expected key1 to not exist after clear")
	}
}
