package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// TestConfigYAMLParsing tests that the Config struct correctly parses YAML.
func TestConfigYAMLParsing(t *testing.T) {
	yml := `
schemaVersion: 1
project:
  name: mytool
  repo: owner/mytool
  description: "A cool tool"
  homepage: "https://example.com"
  license: "MIT"
go:
  main: ./cmd/mytool
  cgo: false
  ldflags: "-X main.Version=1.0.0"
release:
  tagTemplate: "v{{version}}"
  platforms:
    - os: darwin
      arch: arm64
    - os: linux
      arch: amd64
    - os: windows
      arch: amd64
  archive:
    nameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}"
    format:
      default: "tar.gz"
      windows: "zip"
    binPathInArchive: "bin/{{name}}"
  checksums:
    file: "checksums.txt"
    algorithm: "sha256"
    format: "goreleaser"
packages:
  npm:
    enabled: true
    packageName: "my-tool"
    binName: "mytool"
    nodeEngines: ">=18"
  homebrew:
    enabled: true
    tap: "owner/homebrew-tap"
    formulaName: "MyTool"
  chocolatey:
    enabled: true
    packageId: "mytool"
    authors: "John Doe"
github:
  workflows:
    enabled: true
    workflowFile: ".github/workflows/release.yml"
    tagPatterns:
      - "v*"
`

	var config Config
	err := yaml.Unmarshal([]byte(yml), &config)
	if err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	// Verify parsed values
	if config.SchemaVersion != 1 {
		t.Errorf("expected schemaVersion 1, got %d", config.SchemaVersion)
	}
	if config.Project.Name != "mytool" {
		t.Errorf("expected project name 'mytool', got %q", config.Project.Name)
	}
	if config.Project.Repo != "owner/mytool" {
		t.Errorf("expected repo 'owner/mytool', got %q", config.Project.Repo)
	}
	if config.Project.Description != "A cool tool" {
		t.Errorf("expected description 'A cool tool', got %q", config.Project.Description)
	}
	if config.Go.Main != "./cmd/mytool" {
		t.Errorf("expected go.main './cmd/mytool', got %q", config.Go.Main)
	}
	if config.Go.CGO != false {
		t.Errorf("expected cgo false, got %v", config.Go.CGO)
	}

	// Verify platforms
	if len(config.Release.Platforms) != 3 {
		t.Errorf("expected 3 platforms, got %d", len(config.Release.Platforms))
	}

	// Verify first platform
	if config.Release.Platforms[0].OS != "darwin" || config.Release.Platforms[0].Arch != "arm64" {
		t.Errorf("unexpected first platform: %v", config.Release.Platforms[0])
	}

	// Verify archive config
	if config.Release.Archive.NameTemplate != "{{name}}_{{version}}_{{os}}_{{arch}}" {
		t.Errorf("unexpected archive name template")
	}
	if config.Release.Archive.Format.Default != "tar.gz" {
		t.Errorf("expected default format 'tar.gz', got %q", config.Release.Archive.Format.Default)
	}
	if config.Release.Archive.Format.Windows != "zip" {
		t.Errorf("expected windows format 'zip', got %q", config.Release.Archive.Format.Windows)
	}

	// Verify packages
	if !config.Packages.NPM.Enabled {
		t.Errorf("expected npm enabled")
	}
	if config.Packages.NPM.PackageName != "my-tool" {
		t.Errorf("expected npm package name 'my-tool', got %q", config.Packages.NPM.PackageName)
	}

	if !config.Packages.Homebrew.Enabled {
		t.Errorf("expected homebrew enabled")
	}
	if config.Packages.Homebrew.FormulaName != "MyTool" {
		t.Errorf("expected homebrew formula name 'MyTool', got %q", config.Packages.Homebrew.FormulaName)
	}

	if !config.Packages.Chocolatey.Enabled {
		t.Errorf("expected chocolatey enabled")
	}

	// Verify github workflows
	if !config.GitHub.Workflows.Enabled {
		t.Errorf("expected workflows enabled")
	}
	if len(config.GitHub.Workflows.TagPatterns) != 1 || config.GitHub.Workflows.TagPatterns[0] != "v*" {
		t.Errorf("unexpected tag patterns: %v", config.GitHub.Workflows.TagPatterns)
	}
}

// TestConfigMinimal tests parsing with minimal required fields.
func TestConfigMinimal(t *testing.T) {
	yml := `
schemaVersion: 1
project:
  name: tool
  repo: owner/repo
go:
  main: ./cmd/tool
`

	var config Config
	err := yaml.Unmarshal([]byte(yml), &config)
	if err != nil {
		t.Fatalf("failed to parse minimal config: %v", err)
	}

	if config.Project.Name != "tool" {
		t.Errorf("expected name 'tool', got %q", config.Project.Name)
	}
	if config.Project.Repo != "owner/repo" {
		t.Errorf("expected repo 'owner/repo', got %q", config.Project.Repo)
	}
	if config.Go.Main != "./cmd/tool" {
		t.Errorf("expected main './cmd/tool', got %q", config.Go.Main)
	}
}

// TestConfigMarshalling tests that Config can be marshalled back to YAML.
func TestConfigMarshalling(t *testing.T) {
	config := Config{
		SchemaVersion: 1,
		Project: Project{
			Name:    "mytool",
			Repo:    "owner/mytool",
			License: "MIT",
		},
		Go: Go{
			Main:    "./cmd/mytool",
			CGO:     false,
			LDFlags: "-X main.Version=1.0.0",
		},
	}

	// Marshal to YAML
	bytes, err := yaml.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	// Unmarshal back
	var restored Config
	err = yaml.Unmarshal(bytes, &restored)
	if err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	// Verify round-trip
	if restored.Project.Name != config.Project.Name {
		t.Errorf("round-trip failed: name mismatch")
	}
	if restored.Go.LDFlags != config.Go.LDFlags {
		t.Errorf("round-trip failed: ldflags mismatch")
	}
}

// TestPlatformValidation tests that Platform struct holds valid data.
func TestPlatformValidation(t *testing.T) {
	tests := []struct {
		name string
		os   string
		arch string
	}{
		{"darwin amd64", "darwin", "amd64"},
		{"darwin arm64", "darwin", "arm64"},
		{"linux amd64", "linux", "amd64"},
		{"linux arm64", "linux", "arm64"},
		{"windows amd64", "windows", "amd64"},
		{"windows arm64", "windows", "arm64"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Platform{OS: tt.os, Arch: tt.arch}
			if p.OS != tt.os || p.Arch != tt.arch {
				t.Errorf("Platform mismatch")
			}
		})
	}
}

// TestArchiveFormatDefaults tests ArchiveFormat struct.
func TestArchiveFormatDefaults(t *testing.T) {
	af := ArchiveFormat{
		Default: "tar.gz",
		Windows: "zip",
	}

	if af.Default != "tar.gz" {
		t.Errorf("expected default tar.gz")
	}
	if af.Windows != "zip" {
		t.Errorf("expected windows zip")
	}
}

// TestConfigWithMultiplePlatforms tests Config with many platforms.
func TestConfigWithMultiplePlatforms(t *testing.T) {
	yml := `
schemaVersion: 1
project:
  name: tool
  repo: owner/repo
go:
  main: ./cmd/tool
release:
  platforms:
    - os: darwin
      arch: amd64
    - os: darwin
      arch: arm64
    - os: linux
      arch: amd64
    - os: linux
      arch: arm64
    - os: windows
      arch: amd64
    - os: windows
      arch: arm64
`

	var config Config
	err := yaml.Unmarshal([]byte(yml), &config)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	if len(config.Release.Platforms) != 6 {
		t.Errorf("expected 6 platforms, got %d", len(config.Release.Platforms))
	}

	// Verify all combinations are present
	found := make(map[string]bool)
	for _, p := range config.Release.Platforms {
		key := p.OS + "_" + p.Arch
		found[key] = true
	}

	expected := []string{
		"darwin_amd64", "darwin_arm64",
		"linux_amd64", "linux_arm64",
		"windows_amd64", "windows_arm64",
	}

	for _, exp := range expected {
		if !found[exp] {
			t.Errorf("expected platform %q not found", exp)
		}
	}
}
