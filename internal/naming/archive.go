// Package naming provides canonical naming functions for artifacts.
// All generators must use these functions; no duplicated per-generator naming logic.
package naming

import (
	"fmt"
	"strings"

	"go-package-yourself/internal/templatex"
)

// ExtensionFor returns the file extension for a given archive format and OS.
//
// Supported formats: "tar.gz", "zip"
// Returns extension including the leading dot (e.g., ".tar.gz", ".zip")
//
// Performance: O(1), no allocations for common cases.
// Contract: This function is frozen. All generators must use it.
func ExtensionFor(format, os string) (string, error) {
	// Normalize format
	format = strings.TrimSpace(strings.ToLower(format))
	if format == "" {
		format = "tar.gz" // default
	}

	switch format {
	case "tar.gz", "targz":
		return ".tar.gz", nil
	case "zip":
		return ".zip", nil
	default:
		return "", fmt.Errorf("unsupported archive format: %s", format)
	}
}

// ArchiveNameParams holds parameters for archive naming computation.
// This struct allows future extension without changing function signature.
type ArchiveNameParams struct {
	Name                string // project name (required)
	Version             string // version (required for concrete naming; may be empty for placeholder resolution)
	OS                  string // "darwin", "linux", or "windows" (required)
	Arch                string // "amd64" or "arm64" (required)
	Format              string // "tar.gz" or "zip" (required)
	ArchiveNameTemplate string // template string (required; e.g., "{{name}}_{{version}}_{{os}}_{{arch}}")
	BinPathTemplate     string // template for binary path (required; e.g., "{{name}}")
}

// ArchiveNameResult holds the output of ArchiveName computation.
type ArchiveNameResult struct {
	ArchiveFilename  string // e.g., "mytool_1.2.3_darwin_arm64.tar.gz"
	BinPathInArchive string // e.g., "mytool" or "bin/mytool"
}

// ArchiveName computes the canonical archive filename and binary path inside the archive.
//
// Parameters:
//   - name: project name (e.g., "mytool")
//   - version: version string (e.g., "1.2.3"). If empty, {{version}} placeholders are preserved.
//   - os: "darwin", "linux", or "windows"
//   - arch: "amd64" or "arm64"
//   - format: "tar.gz" or "zip"
//   - archiveNameTemplate: template for filename (e.g., "{{name}}_{{version}}_{{os}}_{{arch}}")
//   - binPathTemplate: template for binary path in archive (e.g., "{{name}}")
//
// Returns:
//   - archiveFilename: the full filename including extension (e.g., "mytool_1.2.3_darwin_arm64.tar.gz")
//   - binPathInArchive: path to the binary inside the archive (e.g., "mytool" for tar.gz, "mytool.exe" for windows)
//   - error if format is invalid or templates contain errors
//
// Performance: Allocates only when necessary for template rendering. Common case uses stack allocation.
//
// Contract: This function is frozen. All generators must call this for naming.
func ArchiveName(params ArchiveNameParams) (archiveFilename, binPathInArchive string, err error) {
	// Validate inputs
	if err := validateArchiveParams(params); err != nil {
		return "", "", err
	}

	// Get extension
	ext, err := ExtensionFor(params.Format, params.OS)
	if err != nil {
		return "", "", err
	}

	// Create template context for rendering
	templateData := map[string]string{
		"name":    params.Name,
		"version": params.Version,
		"os":      params.OS,
		"arch":    params.Arch,
		"ext":     ext,
	}

	// Render archive name template
	renderer := &templatex.Renderer{
		Data: templateData,
	}

	baseArchiveName, err := renderer.Render(params.ArchiveNameTemplate)
	if err != nil {
		return "", "", fmt.Errorf("render archive name template: %w", err)
	}

	// Render binary path template
	binPath, err := renderer.Render(params.BinPathTemplate)
	if err != nil {
		return "", "", fmt.Errorf("render binary path template: %w", err)
	}

	// Append .exe for Windows binaries if not already present
	if params.OS == "windows" && !strings.HasSuffix(binPath, ".exe") {
		binPath += ".exe"
	}

	// Combine base name with extension to get full archive filename
	archiveFilename = baseArchiveName + ext

	return archiveFilename, binPath, nil
}

// validateArchiveParams validates archive naming parameters.
func validateArchiveParams(params ArchiveNameParams) error {
	if params.Name == "" {
		return fmt.Errorf("name is required")
	}
	if params.OS == "" {
		return fmt.Errorf("os is required")
	}
	if !isValidOS(params.OS) {
		return fmt.Errorf("invalid os: %s (expected darwin, linux, or windows)", params.OS)
	}
	if params.Arch == "" {
		return fmt.Errorf("arch is required")
	}
	if !isValidArch(params.Arch) {
		return fmt.Errorf("invalid arch: %s (expected amd64 or arm64)", params.Arch)
	}
	if params.Format == "" {
		return fmt.Errorf("format is required")
	}
	if params.ArchiveNameTemplate == "" {
		return fmt.Errorf("archiveNameTemplate is required")
	}
	if params.BinPathTemplate == "" {
		return fmt.Errorf("binPathTemplate is required")
	}
	return nil
}

// isValidOS checks if os is a supported platform.
func isValidOS(os string) bool {
	switch os {
	case "darwin", "linux", "windows":
		return true
	default:
		return false
	}
}

// isValidArch checks if arch is a supported architecture.
func isValidArch(arch string) bool {
	switch arch {
	case "amd64", "arm64":
		return true
	default:
		return false
	}
}
