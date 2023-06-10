package inertia

import (
	"html/template"
	"testing"
)

func TestSsrResponse_HeadHTML(t *testing.T) {
	tests := []struct {
		response *SsrResponse
		expected template.HTML
	}{
		{
			response: &SsrResponse{
				Head: []string{
					"<title>Test</title>",
					"<meta name=\"description\" content=\"Test\">",
				},
			},
			expected: "<title>Test</title>\n<meta name=\"description\" content=\"Test\">",
		},
	}

	for _, tt := range tests {
		if ret := tt.response.HeadHTML(); ret != tt.expected {
			t.Errorf("expected %v but %v", tt.expected, ret)
		}
	}
}

func TestSsrResponse_BodyHTML(t *testing.T) {
	tests := []struct {
		response *SsrResponse
		expected template.HTML
	}{
		{
			response: &SsrResponse{
				Body: "<div>Test</div>",
			},
			expected: "<div>Test</div>",
		},
	}

	for _, tt := range tests {
		if ret := tt.response.BodyHTML(); ret != tt.expected {
			t.Errorf("expected %v but %v", tt.expected, ret)
		}
	}
}
