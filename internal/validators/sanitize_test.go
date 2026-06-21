package validators

import (
	"testing"
)

func TestSanitizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "escapes less than",
			input:    "<script>alert('xss')</script>",
			expected: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:     "escapes ampersand",
			input:    "Tom & Jerry",
			expected: "Tom &amp; Jerry",
		},
		{
			name:     "escapes double quotes",
			input:    `He said "hello"`,
			expected: "He said &#34;hello&#34;",
		},
		{
			name:     "escapes single quotes",
			input:    "it's fine",
			expected: "it&#39;s fine",
		},
		{
			name:     "plain text unchanged",
			input:    "John Doe",
			expected: "John Doe",
		},
		{
			name:     "empty string unchanged",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeText(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeText(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTrimAndSanitize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "trims and sanitizes",
			input:    "  <b>bold</b>  ",
			expected: "&lt;b&gt;bold&lt;/b&gt;",
		},
		{
			name:     "trims leading whitespace",
			input:    "   hello",
			expected: "hello",
		},
		{
			name:     "trims trailing whitespace",
			input:    "hello   ",
			expected: "hello",
		},
		{
			name:     "trims tabs and newlines",
			input:    "\t\nhello\n\t",
			expected: "hello",
		},
		{
			name:     "empty after trim",
			input:    "   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TrimAndSanitize(tt.input)
			if result != tt.expected {
				t.Errorf("TrimAndSanitize(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
