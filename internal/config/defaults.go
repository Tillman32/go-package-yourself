package config

import "go-package-yourself/internal/model"

// ApplyDefaults applies sensible defaults to a Config struct.
// This is idempotent and safe to call multiple times.
//
// Defaults applied:
//   - Go.CGO: false (if not set)
//   - Release.TagTemplate: "v{{version}}" (if empty)
//   - Release.Platforms: darwin/linux/windows with amd64/arm64 (if empty)
//   - Release.Archive.NameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}" (if empty)
//   - Release.Archive.Format.Default: "tar.gz" (if empty)
//   - Release.Archive.Format.Windows: "zip" (if empty)
//   - Release.Archive.BinPathInArchive: "{{name}}" (if empty)
//   - Release.Checksums.File: "checksums.txt" (if empty)
//   - Release.Checksums.Algorithm: "sha256" (fixed v1)
//   - Release.Checksums.Format: "goreleaser" (fixed v1)
//   - Packages.NPM.NodeEngines: ">=18" (if empty)
//   - All package fields (enabled, etc.): false by default (if not set)
//   - Packages.Docker.ImageName: project.name (if empty)
//   - Packages.Docker.Cmd: "./{{name}}" (if empty)
func ApplyDefaults(cfg *model.Config) {
	// Go defaults
	// Note: Go.CGO defaults to false and bool zero value is false, so no action needed.
	// But we document it for clarity.

	// Release defaults
	if cfg.Release.TagTemplate == "" {
		cfg.Release.TagTemplate = "v{{version}}"
	}

	// Platform defaults: If empty, provide sensible defaults
	if len(cfg.Release.Platforms) == 0 {
		cfg.Release.Platforms = []model.Platform{
			{OS: "darwin", Arch: "amd64"},
			{OS: "darwin", Arch: "arm64"},
			{OS: "linux", Arch: "amd64"},
			{OS: "linux", Arch: "arm64"},
			{OS: "windows", Arch: "amd64"},
		}
	}

	// Archive defaults
	if cfg.Release.Archive.NameTemplate == "" {
		cfg.Release.Archive.NameTemplate = "{{name}}_{{version}}_{{os}}_{{arch}}"
	}

	if cfg.Release.Archive.Format.Default == "" {
		cfg.Release.Archive.Format.Default = "tar.gz"
	}

	if cfg.Release.Archive.Format.Windows == "" {
		cfg.Release.Archive.Format.Windows = "zip"
	}

	if cfg.Release.Archive.BinPathInArchive == "" {
		cfg.Release.Archive.BinPathInArchive = "{{name}}"
	}

	// Checksums defaults (algorithm and format are fixed v1)
	if cfg.Release.Checksums.File == "" {
		cfg.Release.Checksums.File = "checksums.txt"
	}
	cfg.Release.Checksums.Algorithm = "sha256"  // v1: fixed
	cfg.Release.Checksums.Format = "goreleaser" // v1: fixed

	// Packages defaults
	// NPM
	if cfg.Packages.NPM.NodeEngines == "" {
		cfg.Packages.NPM.NodeEngines = ">=18"
	}

	// Homebrew has no additional defaults beyond zero values (enabled=false by default)

	// Chocolatey has no additional defaults beyond zero values (enabled=false by default)

	// Docker defaults
	if cfg.Packages.Docker.ImageName == "" {
		cfg.Packages.Docker.ImageName = cfg.Project.Name
	}
	if cfg.Packages.Docker.Cmd == "" {
		cfg.Packages.Docker.Cmd = "./{{name}}"
	}

	// GitHub defaults
	cfg.GitHub.Workflows.Enabled = true // Enable workflows by default
	if cfg.GitHub.Workflows.WorkflowFile == "" {
		// Use gpy- prefix to avoid collision with existing user release workflows
		cfg.GitHub.Workflows.WorkflowFile = ".github/workflows/gpy-release.yaml"
	}
}
