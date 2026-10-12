package inertia

import (
	"net/http"
	"slices"
	"testing"
)

func TestAddVaryHeader(t *testing.T) {
	tests := []struct {
		name    string
		initial []string
		value   string
		want    []string
	}{
		{
			name:  "empty header",
			value: HeaderXInertia,
			want:  []string{HeaderXInertia},
		},
		{
			name:    "appends to existing tokens",
			initial: []string{"Accept-Encoding"},
			value:   HeaderXInertia,
			want:    []string{"Accept-Encoding", HeaderXInertia},
		},
		{
			name:    "does not duplicate exact match",
			initial: []string{HeaderXInertia},
			value:   HeaderXInertia,
			want:    []string{HeaderXInertia},
		},
		{
			name:    "does not duplicate case-insensitive match",
			initial: []string{"x-inertia"},
			value:   HeaderXInertia,
			want:    []string{"x-inertia"},
		},
		{
			name:    "match inside comma-separated Vary value",
			initial: []string{"Cookie, X-Inertia, Accept-Encoding"},
			value:   HeaderXInertia,
			want:    []string{"Cookie, X-Inertia, Accept-Encoding"},
		},
		{
			name:    "wildcard prevents adding",
			initial: []string{"*"},
			value:   HeaderXInertia,
			want:    []string{"*"},
		},
		{
			name:    "multiple Vary header values",
			initial: []string{"Cookie", "Accept-Encoding"},
			value:   HeaderXInertia,
			want:    []string{"Cookie", "Accept-Encoding", HeaderXInertia},
		},
		{
			name:    "already present across multiple Vary values",
			initial: []string{"Cookie", "X-Inertia"},
			value:   HeaderXInertia,
			want:    []string{"Cookie", "X-Inertia"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := make(http.Header)
			for _, v := range tt.initial {
				header.Add("Vary", v)
			}
			addVaryHeader(header, tt.value)
			got := header.Values("Vary")
			if !slices.Equal(got, tt.want) {
				t.Fatalf("Vary=%v, want %v", got, tt.want)
			}
		})
	}
}

func TestInArray(t *testing.T) {
	tests := []struct {
		name     string
		needle   string
		haystack []string
		want     bool
	}{
		{name: "found", needle: "apple", haystack: []string{"orange", "apple", "banana"}, want: true},
		{name: "not found", needle: "grape", haystack: []string{"orange", "apple", "banana"}, want: false},
		{name: "empty haystack", needle: "apple", haystack: []string{}, want: false},
		{name: "nil haystack", needle: "apple", haystack: nil, want: false},
		{name: "empty needle found", needle: "", haystack: []string{"orange", "", "banana"}, want: true},
		{name: "empty needle missing", needle: "", haystack: []string{"orange", "apple"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inArray(tt.needle, tt.haystack); got != tt.want {
				t.Fatalf("inArray(%q, %v) = %v, want %v", tt.needle, tt.haystack, got, tt.want)
			}
		})
	}
}

func TestSplitAndRemoveEmpty(t *testing.T) {
	tests := []struct {
		name string
		s    string
		sep  string
		want []string
	}{
		{name: "normal", s: "apple,orange,banana", sep: ",", want: []string{"apple", "orange", "banana"}},
		{name: "empty input", s: "", sep: ",", want: nil},
		{name: "single token", s: "apple", sep: ",", want: []string{"apple"}},
		{name: "skips empty segments", s: "apple,,orange,,banana", sep: ",", want: []string{"apple", "orange", "banana"}},
		{name: "trims whitespace", s: " apple , orange , banana ", sep: ",", want: []string{"apple", "orange", "banana"}},
		{name: "only separators", s: ",,,", sep: ",", want: nil},
		{name: "leading and trailing separators", s: ",apple,banana,", sep: ",", want: []string{"apple", "banana"}},
		{name: "custom separator", s: "apple|orange|banana", sep: "|", want: []string{"apple", "orange", "banana"}},
		{name: "multi-char separator", s: "apple::orange::::banana", sep: "::", want: []string{"apple", "orange", "banana"}},
		{name: "space separator collapses extras", s: "apple  orange  banana", sep: " ", want: []string{"apple", "orange", "banana"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAndRemoveEmpty(tt.s, tt.sep)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("splitAndRemoveEmpty(%q, %q) = %#v, want %#v", tt.s, tt.sep, got, tt.want)
			}
		})
	}
}
