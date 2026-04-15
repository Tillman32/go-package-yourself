// Package templatex provides template rendering for gpy.
// Supports placeholders: {{name}}, {{version}}, {{os}}, {{arch}}, {{ext}}
package templatex

import (
	"fmt"
	"strings"
)

// Renderer handles template placeholder substitution.
//
// Performance: Uses strings.Builder for allocation-efficient string building.
// Typically does not allocate for simple templates.
type Renderer struct {
	// Data maps placeholder names to their values.
	// Supported keys: "name", "version", "os", "arch", "ext"
	Data map[string]string
}

// RenderWithFieldPath renders a template string and includes field context in errors.
//
// Parameters:
//   - template: the template string with {{placeholder}} syntax
//   - fieldPath: the path in the config for error context (e.g., "Release.Archive.NameTemplate")
//
// Returns the rendered string with all {{placeholder}} replaced by Data values.
// Returns error if any placeholder is unknown, with field context included.
//
// Supported placeholders: {{name}}, {{version}}, {{os}}, {{arch}}, {{ext}}
// Unknown placeholders cause an error like: "field Release.Archive.NameTemplate: unknown placeholder {{typo}}"
//
// Performance: O(n) in template length. Allocates only for output string.
func (r *Renderer) RenderWithFieldPath(template, fieldPath string) (string, error) {
	result, err := r.render(template)
	if err != nil {
		// Wrap error with field context
		return "", fmt.Errorf("field %s: %w", fieldPath, err)
	}
	return result, nil
}

// Render renders a template string without field context.
// Use RenderWithFieldPath for better error messages.
func (r *Renderer) Render(template string) (string, error) {
	return r.render(template)
}

// render performs the actual template substitution.
// Performance: Uses strings.Builder to minimize allocations.
func (r *Renderer) render(template string) (string, error) {
	// Fast path for templates with no placeholders
	if !strings.Contains(template, "{{") {
		return template, nil
	}

	var buf strings.Builder
	buf.Grow(len(template)) // Pre-allocate approximate space

	for i := 0; i < len(template); i++ {
		// Look for placeholder start
		if i < len(template)-3 && template[i:i+2] == "{{" {
			// Find placeholder end
			end := strings.Index(template[i+2:], "}}")
			if end == -1 {
				// Malformed placeholder, write it as-is
				buf.WriteString(template[i : i+2])
				i++
				continue
			}

			// Extract placeholder name
			placeholder := strings.TrimSpace(template[i+2 : i+2+end])

			// Look up value in data
			value, ok := r.Data[placeholder]
			if !ok {
				// Unknown placeholder
				return "", fmt.Errorf("unknown placeholder {{%s}}", placeholder)
			}

			// Write replacement value
			buf.WriteString(value)

			// Skip past the placeholder
			i += 1 + end + 2 // 1 for current position, end for placeholder content, 2 for "}}"

			continue
		}

		// Regular character
		buf.WriteByte(template[i])
	}

	return buf.String(), nil
}

// ValidatePlaceholders checks if a template contains any unknown placeholders.
// This is useful for early validation before template rendering.
//
// Returns a list of unknown placeholder names, or empty slice if all are valid.
func (r *Renderer) ValidatePlaceholders(template string) []string {
	if !strings.Contains(template, "{{") {
		return nil
	}

	var unknown []string
	seen := make(map[string]bool) // Avoid duplicate errors for same placeholder

	for i := 0; i < len(template); i++ {
		if i < len(template)-3 && template[i:i+2] == "{{" {
			end := strings.Index(template[i+2:], "}}")
			if end == -1 {
				continue // Malformed, will be caught at render time
			}

			placeholder := strings.TrimSpace(template[i+2 : i+2+end])

			// Check if unknown
			if _, ok := r.Data[placeholder]; !ok && !seen[placeholder] {
				unknown = append(unknown, placeholder)
				seen[placeholder] = true
			}

			i += 1 + end + 2
			continue
		}
	}

	return unknown
}

// RenderMultiple renders multiple template strings with shared field context prefix.
// Useful for validating multiple templates with the same placeholder definitions.
//
// fieldPathPrefix: prefix for error context (e.g., "Release.Archive")
// templates: map of field names to template strings
//
// Returns a map of field names to rendered strings, or error with full field path.
func (r *Renderer) RenderMultiple(fieldPathPrefix string, templates map[string]string) (map[string]string, error) {
	results := make(map[string]string, len(templates))

	for name, template := range templates {
		fieldPath := fieldPathPrefix + "." + name
		result, err := r.RenderWithFieldPath(template, fieldPath)
		if err != nil {
			return nil, err
		}
		results[name] = result
	}

	return results, nil
}

// SupportedPlaceholders returns the set of supported placeholder names.
func SupportedPlaceholders() []string {
	return []string{"name", "version", "os", "arch", "ext"}
}

// PlaceholderHelp returns help text describing supported placeholders and their meanings.
func PlaceholderHelp() string {
	const help = `
Supported template placeholders:
  {{name}}     - Project name (e.g., "mytool")
  {{version}}  - Release version (e.g., "1.2.3")
  {{os}}       - Operating system (darwin, linux, windows)
  {{arch}}     - Architecture (amd64, arm64)
  {{ext}}      - File extension (e.g., ".tar.gz", ".zip")

Example: "{{name}}_{{version}}_{{os}}_{{arch}}" -> "mytool_1.2.3_darwin_arm64"
`
	return strings.TrimSpace(help)
}

// CompileTemplate pre-compiles a template for validation without rendering.
// This can be used during config loading to catch errors early.
//
// Returns list of placeholders found in the template.
func CompileTemplate(template string) ([]string, error) {
	if !strings.Contains(template, "{{") {
		return nil, nil
	}

	var placeholders []string
	seen := make(map[string]bool)

	for i := 0; i < len(template); i++ {
		if i < len(template)-3 && template[i:i+2] == "{{" {
			end := strings.Index(template[i+2:], "}}")
			if end == -1 {
				return nil, fmt.Errorf("malformed placeholder at position %d", i)
			}

			placeholder := strings.TrimSpace(template[i+2 : i+2+end])

			// Validate placeholder name
			if !isValidPlaceholder(placeholder) {
				return nil, fmt.Errorf("invalid placeholder name: {{%s}}", placeholder)
			}

			if !seen[placeholder] {
				placeholders = append(placeholders, placeholder)
				seen[placeholder] = true
			}

			i += 1 + end + 2
			continue
		}
	}

	return placeholders, nil
}

// isValidPlaceholder checks if a placeholder name is valid.
func isValidPlaceholder(name string) bool {
	switch name {
	case "name", "version", "os", "arch", "ext":
		return true
	default:
		return false
	}
}

// SafeRender renders a template and returns a default value on error (for safe templating).
// Use Render() for strict error handling.
func (r *Renderer) SafeRender(template, defaultValue string) string {
	result, err := r.render(template)
	if err != nil {
		return defaultValue
	}
	return result
}

// DebugRenderResult holds diagnostic information from rendering.
type DebugRenderResult struct {
	Result           string
	Error            error
	PlaceholdersUsed []string
	MissingValues    map[string]bool
}

// RenderWithDiagnostics renders a template and returns diagnostic info.
func (r *Renderer) RenderWithDiagnostics(template string) *DebugRenderResult {
	result, err := r.render(template)

	diag := &DebugRenderResult{
		Result:        result,
		Error:         err,
		MissingValues: make(map[string]bool),
	}

	// Extract placeholders found in template
	if !strings.Contains(template, "{{") {
		return diag
	}

	for i := 0; i < len(template); i++ {
		if i < len(template)-3 && template[i:i+2] == "{{" {
			end := strings.Index(template[i+2:], "}}")
			if end == -1 {
				continue
			}

			placeholder := strings.TrimSpace(template[i+2 : i+2+end])
			diag.PlaceholdersUsed = append(diag.PlaceholdersUsed, placeholder)

			if _, ok := r.Data[placeholder]; !ok {
				diag.MissingValues[placeholder] = true
			}

			i += 1 + end + 2
			continue
		}
	}

	return diag
}
