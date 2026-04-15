package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGpyInitAndPackage tests the full workflow: gpy init → gpy package.
func TestGpyInitAndPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	tmpdir := t.TempDir()
	defer SwitchDir(t, tmpdir)()

	// Create basic project structure for init to work
	cmdDir := filepath.Join(tmpdir, "cmd", "testapp")
	if err := createDirsIfNotExist(cmdDir); err != nil {
		t.Fatalf("failed to create cmd directory: %v", err)
	}

	mainGo := `package main
func main() { println("testapp") }
`
	if err := writeFile(filepath.Join(cmdDir, "main.go"), mainGo); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Run gpy init --yes
	_, err := RunGpy(t, tmpdir, "--yes", "init")
	if err != nil {
		t.Fatalf("gpy init --yes failed: %v", err)
	}

	// Verify config file was created
	AssertFileExists(t, filepath.Join(tmpdir, "gpy.yaml"))

	// Run gpy package (without explicit config, should auto-discover)
	_, err = RunGpy(t, tmpdir, "package")
	if err != nil {
		t.Fatalf("gpy package failed: %v", err)
	}

	// Verify packaging artifacts were generated
	AssertDirExists(t, filepath.Join(tmpdir, "packaging"))
	AssertFileExists(t, filepath.Join(tmpdir, "packaging", "npm", "testapp", "package.json"))
}

// TestGpyWorkflowWrite tests gpy workflow with --write flag.
func TestGpyWorkflowWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// Generate packages first (workflow generator needs them)
	_, err := RunGpy(t, tmpdir, "package")
	if err != nil {
		t.Fatalf("gpy package failed: %v", err)
	}

	// Run gpy workflow --write
	_, err = RunGpy(t, tmpdir, "workflow", "--write")
	if err != nil {
		t.Fatalf("gpy workflow --write failed: %v", err)
	}

	// Verify workflow file was created at the new path (gpy-release.yaml)
	workflowPath := filepath.Join(tmpdir, ".github", "workflows", "gpy-release.yaml")
	AssertFileExists(t, workflowPath)

	// Verify the directory structure
	AssertDirExists(t, filepath.Join(tmpdir, ".github"))
	AssertDirExists(t, filepath.Join(tmpdir, ".github", "workflows"))

	// Verify YAML is valid
	AssertValidYAML(t, workflowPath)
}

// TestGpyPackageOnlyFlags tests --only flag filtering.
func TestGpyPackageOnlyFlags(t *testing.T) {
	tests := []struct {
		name           string
		onlyArg        string
		shouldExist    []string
		shouldNotExist []string
	}{
		{
			name:    "only npm",
			onlyArg: "npm",
			shouldExist: []string{
				"packaging/npm/testapp/package.json",
			},
			shouldNotExist: []string{
				"packaging/homebrew/testapp.rb",
				"packaging/chocolatey/testapp/testapp.nuspec",
			},
		},
		{
			name:    "only homebrew and chocolatey",
			onlyArg: "homebrew,chocolatey",
			shouldExist: []string{
				"packaging/homebrew/testapp.rb",
				"packaging/chocolatey/testapp/testapp.nuspec",
			},
			shouldNotExist: []string{
				"packaging/npm/testapp/package.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := CreateMinimalProject(t)
			defer SwitchDir(t, tmpdir)()

			_, err := RunGpy(t, tmpdir, "package", "--only", tt.onlyArg)
			if err != nil {
				t.Fatalf("gpy package --only %s failed: %v", tt.onlyArg, err)
			}

			// Check what should exist
			for _, file := range tt.shouldExist {
				path := filepath.Join(tmpdir, file)
				AssertFileExists(t, path)
			}

			// Check what should not exist
			for _, file := range tt.shouldNotExist {
				path := filepath.Join(tmpdir, file)
				// Directory might exist but individual files shouldn't be generated
				if strings.Contains(file, "package.json") || strings.Contains(file, ".rb") || strings.Contains(file, ".nuspec") {
					AssertFileNotExists(t, path)
				}
			}
		})
	}
}

// TestGpyPackageOnlyInvalidFlag tests error handling for invalid --only flag.
func TestGpyPackageOnlyInvalidFlag(t *testing.T) {
	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// This should fail with an error about invalid generator
	_, err := RunGpy(t, tmpdir, "package", "--only", "invalid_generator")
	if err == nil {
		t.Fatalf("expected error with invalid --only flag, but succeeded")
	}

	if !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("error message should mention invalid generator: %v", err)
	}
}

// TestGpyErrorMissingConfig verifies helpful error when config is missing.
func TestGpyErrorMissingConfig(t *testing.T) {
	tmpdir := t.TempDir()
	defer SwitchDir(t, tmpdir)()

	// Try to run gpy package without a config file
	_, err := RunGpy(t, tmpdir, "package")
	if err == nil {
		t.Fatalf("expected error when config file is missing")
	}

	// Error should be helpful
	if !strings.Contains(err.Error(), "gpy init") && !strings.Contains(err.Error(), "config") {
		t.Fatalf("error message not helpful: %v", err)
	}
}

// TestGpyErrorInvalidConfig verifies error handling for invalid configuration.
func TestGpyErrorInvalidConfig(t *testing.T) {
	tmpdir := t.TempDir()
	defer SwitchDir(t, tmpdir)()

	// Create invalid config (missing required fields)
	invalidConfig := `schemaVersion: 1
project:
  name: ""
`
	configPath := filepath.Join(tmpdir, "gpy.yaml")
	if err := writeFile(configPath, invalidConfig); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	// Try to run gpy package
	_, err := RunGpy(t, tmpdir, "package")
	if err == nil {
		t.Fatalf("expected error with invalid config")
	}

	// Error should be descriptive
	if !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("error message should mention validation: %v", err)
	}
}

// TestGpyErrorMissingRequiredField verifies actionable error for missing required fields.
func TestGpyErrorMissingRequiredField(t *testing.T) {
	tmpdir := t.TempDir()
	defer SwitchDir(t, tmpdir)()

	// Create config missing required 'repo' field
	incompleteConfig := `schemaVersion: 1
project:
  name: testapp
go:
  main: ./cmd/testapp
`
	configPath := filepath.Join(tmpdir, "gpy.yaml")
	if err := writeFile(configPath, incompleteConfig); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Try to run gpy package
	_, err := RunGpy(t, tmpdir, "package")
	if err == nil {
		t.Fatalf("expected error with incomplete config")
	}

	// Error should mention the specific field
	if !strings.Contains(err.Error(), "repo") && !strings.Contains(err.Error(), "required") {
		t.Fatalf("error message should mention missing field: %v", err)
	}
}

// TestGpyConfigAutoDiscovery tests that gpy auto-discovers config file.
func TestGpyConfigAutoDiscovery(t *testing.T) {
	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// Run without explicit --config flag
	_, err := RunGpy(t, tmpdir, "package")
	if err != nil {
		t.Fatalf("gpy package with auto-discovery failed: %v", err)
	}

	// Verify it found and used the config
	AssertFileExists(t, filepath.Join(tmpdir, "packaging", "npm", "testapp", "package.json"))
}

// TestGpyExplicitConfigPath tests using explicit --config flag.
func TestGpyExplicitConfigPath(t *testing.T) {
	tmpdir := t.TempDir()
	defer SwitchDir(t, tmpdir)()

	// Create config in non-standard location
	customConfigDir := filepath.Join(tmpdir, "config")
	if err := createDirsIfNotExist(customConfigDir); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}

	config := `schemaVersion: 1
project:
  name: customapp
  repo: user/customapp
go:
  main: ./cmd/app
`
	configPath := filepath.Join(customConfigDir, "custom.yaml")
	if err := writeFile(configPath, config); err != nil {
		t.Fatalf("failed to write custom config: %v", err)
	}

	// Create cmd structure
	cmdDir := filepath.Join(tmpdir, "cmd", "app")
	if err := createDirsIfNotExist(cmdDir); err != nil {
		t.Fatalf("failed to create cmd directory: %v", err)
	}
	if err := writeFile(filepath.Join(cmdDir, "main.go"), "package main\nfunc main() {}"); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	// Run with explicit config
	_, err := RunGpy(t, tmpdir, "package", "--config", configPath)
	if err != nil {
		t.Fatalf("gpy package with explicit --config failed: %v", err)
	}

	// Verify it used the correct config
	AssertFileExists(t, filepath.Join(tmpdir, "packaging", "npm", "customapp", "package.json"))
}

// Helper functions

func createDirsIfNotExist(path string) error {
	return mkdir(path)
}

func writeFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func mkdir(path string) error {
	return os.MkdirAll(path, 0o755)
}
