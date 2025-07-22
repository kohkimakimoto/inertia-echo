package inertia

import (
	"errors"
	"testing"
)

func TestEvaluatePropValue(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		expected    any
		expectError bool
	}{
		// Basic value types
		{
			name:     "string value",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "int value",
			input:    42,
			expected: 42,
		},
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
		{
			name:     "slice value",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "map value",
			input:    map[string]any{"key": "value"},
			expected: map[string]any{"key": "value"},
		},

		// LazyProp tests
		{
			name: "lazy prop success",
			input: Lazy(func() (any, error) {
				return "lazy value", nil
			}),
			expected: "lazy value",
		},
		{
			name: "lazy prop error",
			input: Lazy(func() (any, error) {
				return nil, errors.New("lazy error")
			}),
			expectError: true,
		},

		// OptionalProp tests
		{
			name: "optional prop success",
			input: Optional(func() (any, error) {
				return "optional value", nil
			}),
			expected: "optional value",
		},
		{
			name: "optional prop error",
			input: Optional(func() (any, error) {
				return nil, errors.New("optional error")
			}),
			expectError: true,
		},

		// DeferProp tests
		{
			name: "defer prop success",
			input: Defer(func() (any, error) {
				return "defer value", nil
			}),
			expected: "defer value",
		},
		{
			name: "defer prop error",
			input: Defer(func() (any, error) {
				return nil, errors.New("defer error")
			}),
			expectError: true,
		},

		// AlwaysProp tests
		{
			name:     "always prop simple value",
			input:    Always("always value"),
			expected: "always value",
		},
		{
			name:     "always prop complex value",
			input:    Always(map[string]any{"key": "always"}),
			expected: map[string]any{"key": "always"},
		},

		// MergeProp tests
		{
			name:     "merge prop simple value",
			input:    Merge("merge value"),
			expected: "merge value",
		},
		{
			name:     "merge prop complex value",
			input:    Merge([]int{1, 2, 3}),
			expected: []int{1, 2, 3},
		},

		// Function tests - func() (any, error)
		{
			name: "function with error return success",
			input: func() (any, error) {
				return "function value", nil
			},
			expected: "function value",
		},
		{
			name: "function with error return error",
			input: func() (any, error) {
				return nil, errors.New("function error")
			},
			expectError: true,
		},

		// Function tests - func() any
		{
			name: "simple function",
			input: func() any {
				return "simple function value"
			},
			expected: "simple function value",
		},
		{
			name: "simple function with nil",
			input: func() any {
				return nil
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluatePropValue(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Compare results
			if !testDeepEqual(t, result, tt.expected) {
				t.Errorf("Expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestEvaluateProps(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]any
		expected    map[string]any
		expectError bool
	}{
		{
			name: "simple props",
			input: map[string]any{
				"name":   "John",
				"age":    30,
				"active": true,
			},
			expected: map[string]any{
				"name":   "John",
				"age":    30,
				"active": true,
			},
		},
		{
			name: "props with lazy prop",
			input: map[string]any{
				"static": "value",
				"lazy": Lazy(func() (any, error) {
					return "lazy result", nil
				}),
			},
			expected: map[string]any{
				"static": "value",
				"lazy":   "lazy result",
			},
		},
		{
			name: "props with multiple prop types",
			input: map[string]any{
				"always": Always("always value"),
				"merge":  Merge("merge value"),
				"defer": Defer(func() (any, error) {
					return "defer value", nil
				}),
				"function": func() any {
					return "function value"
				},
			},
			expected: map[string]any{
				"always":   "always value",
				"merge":    "merge value",
				"defer":    "defer value",
				"function": "function value",
			},
		},
		{
			name: "props with error",
			input: map[string]any{
				"good": "value",
				"bad": Lazy(func() (any, error) {
					return nil, errors.New("evaluation error")
				}),
			},
			expectError: true,
		},
		{
			name:     "empty props",
			input:    map[string]any{},
			expected: map[string]any{},
		},
		{
			name: "nested evaluation",
			input: map[string]any{
				"nested": Optional(func() (any, error) {
					return Always(Merge("deeply nested")), nil
				}),
			},
			expected: map[string]any{
				"nested": "deeply nested",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy of input to avoid modifying the original
			inputCopy := make(map[string]any)
			for k, v := range tt.input {
				inputCopy[k] = v
			}

			err := evaluateProps(inputCopy)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Compare results
			if !testDeepEqual(t, inputCopy, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, inputCopy)
			}
		})
	}
}
