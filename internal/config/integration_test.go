package config

import (
	"os"
	"path/filepath"
	"testing"

	"go-package-yourself/internal/validate"
)

// TestIntegrationLoadAndValidate demonstrates the complete workflow:
// Load config from disk, apply defaults, and validate the result.
func TestIntegrationLoadAndValidate(t *testing.T) {
	tmpdir := t.TempDir()

	// Create a minimal valid config
	yml := `schemaVersion: 1
project:
  name: integration-test
  repo: owner/integration-test
go:
  main: ./cmd/test
`

	configPath := filepath.Join(tmpdir, "gpy.yaml")
	if err := os.WriteFile(configPath, []byte(yml), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load the config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify defaults were applied
	if cfg.Release.TagTemplate == "" {
		t.Fatalf("defaults were not applied")
	}
	if len(cfg.Release.Platforms) == 0 {
		t.Fatalf("platforms defaults not applied")
	}

	// Validate the loaded config
	if err := validate.Config(cfg); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// Verify config structure
	if cfg.Project.Name != "integration-test" {
		t.Errorf("Project.Name = %q, want %q", cfg.Project.Name, "integration-test")
	}
	if cfg.Release.TagTemplate != "v{{version}}" {
		t.Errorf("TagTemplate = %q, want %q", cfg.Release.TagTemplate, "v{{version}}")
	}
	if cfg.Release.Archive.Format.Default != "tar.gz" {
		t.Errorf("Format.Default = %q, want %q", cfg.Release.Archive.Format.Default, "tar.gz")
	}
	if cfg.Release.Archive.Format.Windows != "zip" {
		t.Errorf("Format.Windows = %q, want %q", cfg.Release.Archive.Format.Windows, "zip")
	}
}

// TestIntegrationLoadMinimalValidConfig tests loading and validating
// the absolute minimum valid configuration.
func TestIntegrationLoadMinimalValidConfig(t *testing.T) {
	tmpdir := t.TempDir()

	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpdir)

	// Create minimal config
	yml := `schemaVersion: 1
project:
  name: minimal
  repo: x/y
go:
  main: ./cmd
`

	if err := os.WriteFile(".gpy.yaml", []byte(yml), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load with auto-discovery
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Should have full defaults
	if len(cfg.Release.Platforms) != 5 {
		t.Errorf("should have 5 default platforms, got %d", len(cfg.Release.Platforms))
	}

	// Should validate successfully
	if err := validate.Config(cfg); err != nil {
		t.Fatalf("validation failed: %v", err)
	}
}

// TestIntegrationLoadFailsWithInvalidConfig tests that Load succeeds
// but validation catches config problems.
func TestIntegrationLoadFailsValidationWithInvalidOS(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "gpy.yaml")

	yml := `schemaVersion: 1
project:
  name: invalid
  repo: x/y
go:
  main: ./cmd
release:
  platforms:
    - os: bsd
      arch: amd64
`

	if err := os.WriteFile(configPath, []byte(yml), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load should succeed even with invalid platform, got: %v", err)
	}

	// But validation should fail
	err = validate.Config(cfg)
	if err == nil {
		t.Fatal("validation should have caught invalid OS")
	}
}
