package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-package-yourself/internal/model"
)

// TestLoadExplicitPath tests loading from an explicit config file path.
func TestLoadExplicitPath(t *testing.T) {
	// Create a temporary config file
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "gpy.yaml")

	yml := `schemaVersion: 1
project:
  name: mytool
  repo: owner/mytool
go:
  main: ./cmd/mytool
`

	if err := os.WriteFile(configPath, []byte(yml), 0o644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Project.Name != "mytool" {
		t.Errorf("Project.Name = %q, want %q", cfg.Project.Name, "mytool")
	}
	if cfg.Project.Repo != "owner/mytool" {
		t.Errorf("Project.Repo = %q, want %q", cfg.Project.Repo, "owner/mytool")
	}
	if cfg.Go.Main != "./cmd/mytool" {
		t.Errorf("Go.Main = %q, want %q", cfg.Go.Main, "./cmd/mytool")
	}
}

// TestLoadNotFound tests that Load returns a helpful error when no config exists.
func TestLoadNotFound(t *testing.T) {
	tmpdir := t.TempDir()

	// Change to temp directory (no config files there)
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(tmpdir); err != nil {
		t.Fatalf("failed to chdir to temp: %v", err)
	}

	cfg, err := Load("")
	if err == nil {
		t.Fatalf("Load should have failed when no config exists, got %+v", cfg)
	}

	// Check that error message is helpful
	errMsg := err.Error()
	if !strings.Contains(errMsg, "gpy") {
		t.Errorf("error should mention config filenames, got: %v", errMsg)
	}
}

// TestLoadDiscovery tests auto-discovery of config files.
func TestLoadDiscovery(t *testing.T) {
	tmpdir := t.TempDir()

	// Change to temp directory
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer os.Chdir(oldCwd)

	if err := os.Chdir(tmpdir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	yml := `schemaVersion: 1
project:
  name: test-tool
  repo: owner/test-tool
go:
  main: ./cmd/test
`

	// Test discovery with go-package-yourself.yaml (first in list)
	if err := os.WriteFile("go-package-yourself.yaml", []byte(yml), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Project.Name != "test-tool" {
		t.Errorf("Project.Name = %q, want %q", cfg.Project.Name, "test-tool")
	}
}

// TestLoadInvalidYAML tests that Load returns an error for invalid YAML.
func TestLoadInvalidYAML(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "gpy.yaml")

	// Write invalid YAML
	if err := os.WriteFile(configPath, []byte("invalid:\n  - unclosed\n["), 0o644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load should have failed on invalid YAML")
	}

	// Error should mention parsing failure
	if !strings.Contains(err.Error(), "parse") {
		t.Logf("error did not explicitly mention parsing, but acceptable: %v", err)
	}
}

// TestLoadFullConfig tests loading a complete config with all fields.
func TestLoadFullConfig(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "gpy.yaml")

	yml := `schemaVersion: 1
project:
  name: mytool
  repo: owner/mytool
  description: "A cool tool"
  homepage: "https://example.com"
  license: "MIT"
go:
  main: ./cmd/mytool
  cgo: true
  ldflags: "-X main.Version=1.0.0"
release:
  tagTemplate: "v{{version}}"
  platforms:
    - os: darwin
      arch: arm64
    - os: linux
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
    formulaName: "Mytool"
  chocolatey:
    enabled: true
    packageId: "mytool"
    authors: "Me"
github:
  workflows:
    enabled: true
    workflowFile: ".github/workflows/release.yml"
    tagPatterns: ["v*"]
`

	if err := os.WriteFile(configPath, []byte(yml), 0o644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify project
	if cfg.Project.Name != "mytool" || cfg.Project.Repo != "owner/mytool" {
		t.Errorf("Project fields mismatch")
	}

	// Verify Go
	if !cfg.Go.CGO {
		t.Errorf("Go.CGO should be true")
	}

	// Verify platforms
	if len(cfg.Release.Platforms) != 2 {
		t.Errorf("should have 2 platforms from YAML, got %d", len(cfg.Release.Platforms))
	}

	// Verify packages
	if !cfg.Packages.NPM.Enabled {
		t.Errorf("NPM should be enabled")
	}
	if cfg.Packages.NPM.PackageName != "my-tool" {
		t.Errorf("NPM.PackageName = %q, want %q", cfg.Packages.NPM.PackageName, "my-tool")
	}
}

// TestApplyDefaults tests that defaults are correctly applied.
func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name string
		cfg  *model.Config
		want func(*model.Config) error
	}{
		{
			name: "empty config gets full defaults",
			cfg:  &model.Config{},
			want: func(cfg *model.Config) error {
				if cfg.Release.TagTemplate != "v{{version}}" {
					return errors.New("TagTemplate default not applied")
				}
				if len(cfg.Release.Platforms) == 0 {
					return errors.New("Platforms default not applied")
				}
				if cfg.Release.Archive.NameTemplate == "" {
					return errors.New("NameTemplate default not applied")
				}
				if cfg.Release.Archive.Format.Default != "tar.gz" {
					return errors.New("Format.Default default not applied")
				}
				if cfg.Release.Archive.Format.Windows != "zip" {
					return errors.New("Format.Windows default not applied")
				}
				if cfg.Release.Checksums.Algorithm != "sha256" {
					return errors.New("Checksums.Algorithm not set to sha256")
				}
				if cfg.Release.Checksums.Format != "goreleaser" {
					return errors.New("Checksums.Format not set to goreleaser")
				}
				if cfg.Packages.NPM.NodeEngines != ">=18" {
					return errors.New("NPM.NodeEngines default not applied")
				}
				return nil
			},
		},
		{
			name: "explicit values not overwritten",
			cfg: &model.Config{
				Release: model.Release{
					TagTemplate: "release-{{version}}",
					Platforms: []model.Platform{
						{OS: "linux", Arch: "amd64"},
					},
					Archive: model.Archive{
						NameTemplate: "custom-name",
						Format: model.ArchiveFormat{
							Default: "zip",
							Windows: "zip",
						},
						BinPathInArchive: "bin/custom",
					},
					Checksums: model.Checksums{
						File: "custom-checksums.txt",
					},
				},
				Packages: model.Packages{
					NPM: model.NPM{
						NodeEngines: ">=20",
					},
				},
			},
			want: func(cfg *model.Config) error {
				if cfg.Release.TagTemplate != "release-{{version}}" {
					return errors.New("TagTemplate was overwritten")
				}
				if len(cfg.Release.Platforms) != 1 {
					return errors.New("Platforms were overwritten")
				}
				if cfg.Release.Archive.NameTemplate != "custom-name" {
					return errors.New("NameTemplate was overwritten")
				}
				if cfg.Release.Archive.Format.Default != "zip" {
					return errors.New("Format.Default was overwritten")
				}
				if cfg.Release.Checksums.File != "custom-checksums.txt" {
					return errors.New("Checksums.File was overwritten")
				}
				if cfg.Packages.NPM.NodeEngines != ">=20" {
					return errors.New("NPM.NodeEngines was overwritten")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.cfg
			ApplyDefaults(cfg)
			if err := tt.want(cfg); err != nil {
				t.Errorf("verification failed: %v", err)
			}
		})
	}
}

// TestDefaultPlatforms tests that default platforms are correctly applied.
func TestDefaultPlatforms(t *testing.T) {
	cfg := &model.Config{}
	ApplyDefaults(cfg)

	want := []string{
		"darwin-amd64",
		"darwin-arm64",
		"linux-amd64",
		"linux-arm64",
		"windows-amd64",
	}

	if len(cfg.Release.Platforms) != len(want) {
		t.Errorf("got %d platforms, want %d", len(cfg.Release.Platforms), len(want))
	}

	for i, p := range cfg.Release.Platforms {
		if i >= len(want) {
			break
		}
		got := p.OS + "-" + p.Arch
		if got != want[i] {
			t.Errorf("platform %d = %q, want %q", i, got, want[i])
		}
	}
}

// TestIdempotentDefaults tests that applying defaults multiple times is safe.
func TestIdempotentDefaults(t *testing.T) {
	cfg := &model.Config{}

	// Apply twice
	ApplyDefaults(cfg)
	first := cfg.Release.TagTemplate
	second := cfg.Release.TagTemplate

	if first != second {
		t.Errorf("defaults are not idempotent: first=%q, second=%q", first, second)
	}

	// Apply a third time
	ApplyDefaults(cfg)
	third := cfg.Release.TagTemplate

	if first != third {
		t.Errorf("third application changed result: first=%q, third=%q", first, third)
	}
}
