package inertia

import (
	"reflect"
	"testing"
)

func TestInArray(t *testing.T) {
	tests := []struct {
		name     string
		needle   string
		haystack []string
		expected bool
	}{
		{
			name:     "found in array",
			needle:   "apple",
			haystack: []string{"orange", "apple", "banana"},
			expected: true,
		},
		{
			name:     "not found in array",
			needle:   "grape",
			haystack: []string{"orange", "apple", "banana"},
			expected: false,
		},
		{
			name:     "empty array",
			needle:   "apple",
			haystack: []string{},
			expected: false,
		},
		{
			name:     "nil array",
			needle:   "apple",
			haystack: nil,
			expected: false,
		},
		{
			name:     "empty needle",
			needle:   "",
			haystack: []string{"orange", "", "banana"},
			expected: true,
		},
		{
			name:     "empty needle not in array",
			needle:   "",
			haystack: []string{"orange", "apple", "banana"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inArray(tt.needle, tt.haystack)
			if result != tt.expected {
				t.Errorf("inArray(%q, %v) = %v, expected %v", tt.needle, tt.haystack, result, tt.expected)
			}
		})
	}
}

func TestSplitAndRemoveEmpty(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		separator string
		expected  []string
	}{
		{
			name:      "normal split",
			input:     "apple,orange,banana",
			separator: ",",
			expected:  []string{"apple", "orange", "banana"},
		},
		{
			name:      "empty string",
			input:     "",
			separator: ",",
			expected:  nil,
		},
		{
			name:      "single element",
			input:     "apple",
			separator: ",",
			expected:  []string{"apple"},
		},
		{
			name:      "empty elements",
			input:     "apple,,orange,,banana",
			separator: ",",
			expected:  []string{"apple", "orange", "banana"},
		},
		{
			name:      "leading empty elements",
			input:     ",,apple,orange,banana",
			separator: ",",
			expected:  []string{"apple", "orange", "banana"},
		},
		{
			name:      "trailing empty elements",
			input:     "apple,orange,banana,,",
			separator: ",",
			expected:  []string{"apple", "orange", "banana"},
		},
		{
			name:      "only separators",
			input:     ",,,",
			separator: ",",
			expected:  nil,
		},
		{
			name:      "different separator",
			input:     "apple|orange|banana",
			separator: "|",
			expected:  []string{"apple", "orange", "banana"},
		},
		{
			name:      "space separator",
			input:     "apple orange banana",
			separator: " ",
			expected:  []string{"apple", "orange", "banana"},
		},
		{
			name:      "space separator with extra spaces",
			input:     "apple  orange  banana",
			separator: " ",
			expected:  []string{"apple", "orange", "banana"},
		},
		{
			name:      "multi-character separator",
			input:     "apple::orange::banana",
			separator: "::",
			expected:  []string{"apple", "orange", "banana"},
		},
		{
			name:      "multi-character separator with empty",
			input:     "apple::::orange::banana",
			separator: "::",
			expected:  []string{"apple", "orange", "banana"},
		},
		{
			name:      "no separator found",
			input:     "apple",
			separator: ",",
			expected:  []string{"apple"},
		},
		{
			name:      "whitespace elements",
			input:     " apple , orange , banana ",
			separator: ",",
			expected:  []string{"apple", "orange", "banana"},
		},
		{
			name:      "single separator at start",
			input:     ",apple",
			separator: ",",
			expected:  []string{"apple"},
		},
		{
			name:      "single separator at end",
			input:     "apple,",
			separator: ",",
			expected:  []string{"apple"},
		},
		{
			name:      "just separator",
			input:     ",",
			separator: ",",
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitAndRemoveEmpty(tt.input, tt.separator)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("splitAndRemoveEmpty(%q, %q) = %v, expected %v", tt.input, tt.separator, result, tt.expected)
			}
		})
	}
}
