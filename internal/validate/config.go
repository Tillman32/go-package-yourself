// Package validate provides configuration validation for gpy.
package validate

import (
	"errors"
	"fmt"
	"strings"

	"go-package-yourself/internal/model"
)

// Config validates a gpy configuration and returns an error with actionable messages.
// Returns nil if the config is valid.
//
// Validation checks:
//   - Required fields: project.name, project.repo, go.main
//   - Platform validation: os ∈ {darwin, linux, windows}, arch ∈ {amd64, arm64}
//   - Archive format validation: default ∈ {tar.gz, zip}, windows ∈ {tar.gz, zip}
//   - Windows archive convention: windows format must be "zip"
//
// All errors include a field path for context (e.g., "field project.name").
//
// Example:
//
//	err := validate.Config(cfg)
//	if err != nil {
//		log.Fatalf("invalid config: %v", err)
//	}
func Config(cfg *model.Config) error {
	var errs []string

	// Validate required fields
	if cfg.Project.Name == "" {
		errs = append(errs, "field project.name: required field missing")
	}
	if cfg.Project.Repo == "" {
		errs = append(errs, "field project.repo: required field missing")
	}
	if cfg.Go.Main == "" {
		errs = append(errs, "field go.main: required field missing")
	}

	// Validate platforms
	for i, p := range cfg.Release.Platforms {
		if err := validatePlatform(i, p); err != nil {
			errs = append(errs, err.Error())
		}
	}

	// Validate archive formats
	if err := validateArchiveFormat(cfg.Release.Archive.Format); err != nil {
		errs = append(errs, err.Error())
	}

	// Validate Docker configuration
	if cfg.Packages.Docker.Port < 0 {
		errs = append(errs, "field packages.docker.port: must not be negative")
	}

	if len(errs) > 0 {
		return errors.New("config validation errors:\n  " + strings.Join(errs, "\n  "))
	}

	return nil
}

// validatePlatform validates a single platform entry.
func validatePlatform(idx int, p model.Platform) error {
	var errs []string

	validOS := map[string]bool{"darwin": true, "linux": true, "windows": true}
	if p.OS == "" {
		errs = append(errs, fmt.Sprintf("field release.platforms[%d].os: required field missing", idx))
	} else if !validOS[p.OS] {
		errs = append(errs, fmt.Sprintf("field release.platforms[%d].os: invalid OS %q (expected darwin, linux, or windows)", idx, p.OS))
	}

	validArch := map[string]bool{"amd64": true, "arm64": true}
	if p.Arch == "" {
		errs = append(errs, fmt.Sprintf("field release.platforms[%d].arch: required field missing", idx))
	} else if !validArch[p.Arch] {
		errs = append(errs, fmt.Sprintf("field release.platforms[%d].arch: invalid arch %q (expected amd64 or arm64)", idx, p.Arch))
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// validateArchiveFormat validates archive format configuration.
func validateArchiveFormat(format model.ArchiveFormat) error {
	validFormats := map[string]bool{"tar.gz": true, "zip": true}

	var errs []string

	// Validate default format
	if format.Default == "" {
		errs = append(errs, "field release.archive.format.default: required field missing")
	} else if !validFormats[format.Default] {
		errs = append(errs, fmt.Sprintf("field release.archive.format.default: invalid format %q (expected tar.gz or zip)", format.Default))
	}

	// Validate Windows format
	if format.Windows == "" {
		errs = append(errs, "field release.archive.format.windows: required field missing")
	} else if !validFormats[format.Windows] {
		errs = append(errs, fmt.Sprintf("field release.archive.format.windows: invalid format %q (expected tar.gz or zip)", format.Windows))
	} else if format.Windows != "zip" {
		errs = append(errs, "field release.archive.format.windows: must be \"zip\" by Windows convention")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}
