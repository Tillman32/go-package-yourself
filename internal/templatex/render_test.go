package templatex

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		data        map[string]string
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name:     "simple_replacement",
			template: "mytool_{{version}}_{{os}}_{{arch}}",
			data: map[string]string{
				"version": "1.2.3",
				"os":      "darwin",
				"arch":    "arm64",
			},
			expected:    "mytool_1.2.3_darwin_arm64",
			expectError: false,
		},
		{
			name:        "no_placeholders",
			template:    "just_a_string",
			data:        map[string]string{},
			expected:    "just_a_string",
			expectError: false,
		},
		{
			name:     "empty_template",
			template: "",
			data:     map[string]string{},
			expected: "",
		},
		{
			name:     "single_placeholder",
			template: "{{name}}",
			data: map[string]string{
				"name": "myexe",
			},
			expected:    "myexe",
			expectError: false,
		},
		{
			name:     "multiple_same_placeholder",
			template: "{{name}}-{{name}}.{{arch}}",
			data: map[string]string{
				"name": "tool",
				"arch": "amd64",
			},
			expected:    "tool-tool.amd64",
			expectError: false,
		},
		{
			name:        "unknown_placeholder",
			template:    "{{name}}_{{version}}_{{unknown}}",
			data:        map[string]string{"name": "tool", "version": "1.0"},
			expected:    "",
			expectError: true,
			errorMsg:    "unknown placeholder {{unknown}}",
		},
		{
			name:     "whitespace_in_placeholder",
			template: "{{ name }}_{{ version }}",
			data: map[string]string{
				"name":    "tool",
				"version": "1.0",
			},
			expected:    "tool_1.0",
			expectError: false,
		},
		{
			name:     "unclosed_placeholder",
			template: "tool_{{version",
			data: map[string]string{
				"version": "1.0",
			},
			expected:    "tool_{{version",
			expectError: false,
		},
		{
			name:     "ext_placeholder",
			template: "file{{ext}}",
			data: map[string]string{
				"ext": ".tar.gz",
			},
			expected:    "file.tar.gz",
			expectError: false,
		},
		{
			name:        "zero_value_placeholder",
			template:    "name:{{name}},version:{{version}}",
			data:        map[string]string{"version": "1.0"}, // name missing
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Renderer{Data: tt.data}
			result, err := r.Render(tt.template)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestRenderWithFieldPath(t *testing.T) {
	r := &Renderer{Data: map[string]string{"name": "tool"}}

	_, err := r.RenderWithFieldPath("{{unknown}}", "Release.Archive.NameTemplate")
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}

	if !strings.Contains(err.Error(), "field Release.Archive.NameTemplate") {
		t.Errorf("expected field path in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "unknown placeholder {{unknown}}") {
		t.Errorf("expected placeholder name in error, got: %v", err)
	}
}

func TestValidatePlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     map[string]string
		expected []string
	}{
		{
			name:     "all_valid",
			template: "{{name}}_{{version}}_{{os}}_{{arch}}_{{ext}}",
			data: map[string]string{
				"name":    "tool",
				"version": "1.0",
				"os":      "linux",
				"arch":    "amd64",
				"ext":     ".tar.gz",
			},
			expected: nil,
		},
		{
			name:     "some_unknown",
			template: "{{name}}_{{version}}_{{typo}}_{{unknown}}",
			data: map[string]string{
				"name":    "tool",
				"version": "1.0",
			},
			expected: []string{"typo", "unknown"},
		},
		{
			name:     "duplicate_placeholders",
			template: "{{name}}_{{name}}_{{name}}",
			data: map[string]string{
				"name": "tool",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Renderer{Data: tt.data}
			unknown := r.ValidatePlaceholders(tt.template)

			if len(unknown) != len(tt.expected) {
				t.Errorf("expected %d unknowns, got %d: %v", len(tt.expected), len(unknown), unknown)
				return
			}

			// Check all expected are present (order might differ)
			seen := make(map[string]bool)
			for _, p := range unknown {
				seen[p] = true
			}
			for _, exp := range tt.expected {
				if !seen[exp] {
					t.Errorf("expected %q in unknowns, got: %v", exp, unknown)
				}
			}
		})
	}
}

func TestCompileTemplate(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		expected    []string
		expectError bool
	}{
		{
			name:     "valid_placeholders",
			template: "{{name}}_{{version}}",
			expected: []string{"name", "version"},
		},
		{
			name:        "invalid_placeholder_name",
			template:    "{{invalid_name}}",
			expectError: true,
		},
		{
			name:     "no_placeholders",
			template: "plain_string",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompileTemplate(tt.template)

			if tt.expectError && err == nil {
				t.Errorf("expected error, got nil")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d placeholders, got %d: %v", len(tt.expected), len(result), result)
				return
			}

			seen := make(map[string]bool)
			for _, p := range result {
				seen[p] = true
			}
			for _, exp := range tt.expected {
				if !seen[exp] {
					t.Errorf("expected %q in result, got: %v", exp, result)
				}
			}
		})
	}
}

func TestSafeRender(t *testing.T) {
	r := &Renderer{Data: map[string]string{"name": "tool"}}

	// Valid render
	result := r.SafeRender("{{name}}", "default")
	if result != "tool" {
		t.Errorf("expected 'tool', got %q", result)
	}

	// Invalid render returns default
	result = r.SafeRender("{{unknown}}", "default")
	if result != "default" {
		t.Errorf("expected 'default', got %q", result)
	}
}

func TestRenderMultiple(t *testing.T) {
	r := &Renderer{Data: map[string]string{
		"name":    "tool",
		"version": "1.0",
	}}

	templates := map[string]string{
		"template1": "{{name}}_v{{version}}",
		"template2": "{{name}}-{{version}}",
	}

	results, err := r.RenderMultiple("Archive", templates)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
		return
	}

	if results["template1"] != "tool_v1.0" {
		t.Errorf("expected 'tool_v1.0', got %q", results["template1"])
	}
	if results["template2"] != "tool-1.0" {
		t.Errorf("expected 'tool-1.0', got %q", results["template2"])
	}
}

func TestRenderMultipleWithError(t *testing.T) {
	r := &Renderer{Data: map[string]string{"name": "tool"}}

	templates := map[string]string{
		"template1": "{{name}}",
		"template2": "{{unknown}}", // This will error
	}

	_, err := r.RenderMultiple("Archive", templates)
	if err == nil {
		t.Errorf("expected error, got nil")
		return
	}

	// Error should include field path prefix
	if !strings.Contains(err.Error(), "field Archive") {
		t.Errorf("expected 'field Archive' in error, got: %v", err)
	}
}

func BenchmarkRender(b *testing.B) {
	r := &Renderer{Data: map[string]string{
		"name":    "mytool",
		"version": "1.2.3",
		"os":      "darwin",
		"arch":    "arm64",
	}}

	template := "{{name}}_{{version}}_{{os}}_{{arch}}.tar.gz"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Render(template)
	}
}

func BenchmarkValidatePlaceholders(b *testing.B) {
	r := &Renderer{Data: map[string]string{
		"name":    "mytool",
		"version": "1.2.3",
	}}

	template := "{{name}}_{{version}}_{{os}}_{{arch}}_{{ext}}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.ValidatePlaceholders(template)
	}
}
