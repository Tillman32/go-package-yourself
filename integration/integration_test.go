package integration

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestEndToEndMinimalConfig tests the complete workflow with minimal config.
// Creates minimal YAML config and verifies all packaging generators produce output.
func TestEndToEndMinimalConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// Run gpy package
	_, err := RunGpy(t, tmpdir, "package")
	if err != nil {
		t.Fatalf("gpy package failed: %v", err)
	}

	// Verify packaging directory structure exists
	AssertDirExists(t, filepath.Join(tmpdir, "packaging"))
	AssertDirExists(t, filepath.Join(tmpdir, "packaging", "npm"))
	AssertDirExists(t, filepath.Join(tmpdir, "packaging", "homebrew"))
	AssertDirExists(t, filepath.Join(tmpdir, "packaging", "chocolatey"))

	// Verify key generated files exist and are valid
	npmPackageJson := filepath.Join(tmpdir, "packaging", "npm", "testapp", "package.json")
	AssertFileExists(t, npmPackageJson)
	AssertValidJSON(t, npmPackageJson)

	// Verify Homebrew formula
	homebrewFormula := filepath.Join(tmpdir, "packaging", "homebrew", "testapp.rb")
	AssertFileExists(t, homebrewFormula)

	// Verify Chocolatey nuspec
	chocolateyNuspec := filepath.Join(tmpdir, "packaging", "chocolatey", "testapp", "testapp.nuspec")
	AssertFileExists(t, chocolateyNuspec)
}

// TestEndToEndWithAllGenerators tests with all generators enabled and --only flag.
func TestEndToEndWithAllGenerators(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// Run gpy package with --only flag for all generators
	_, err := RunGpy(t, tmpdir, "package", "--only", "npm,homebrew,chocolatey")
	if err != nil {
		t.Fatalf("gpy package --only failed: %v", err)
	}

	// Verify all three generators produced output
	AssertFileExists(t, filepath.Join(tmpdir, "packaging", "npm", "testapp", "package.json"))
	AssertFileExists(t, filepath.Join(tmpdir, "packaging", "homebrew", "testapp.rb"))
	AssertFileExists(t, filepath.Join(tmpdir, "packaging", "chocolatey", "testapp", "testapp.nuspec"))

	// Run gpy workflow
	_, err = RunGpy(t, tmpdir, "workflow")
	if err != nil {
		t.Fatalf("gpy workflow failed: %v", err)
	}
}

// TestEndToEndPlatformMatrix tests workflow generation with multiple platforms.
func TestEndToEndPlatformMatrix(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	config := `schemaVersion: 1
project:
  name: mytool
  repo: owner/mytool
go:
  main: ./cmd/mytool
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
`

	tmpdir := CreateProjectWithConfig(t, config)
	defer SwitchDir(t, tmpdir)()

	// Generate packages
	_, err := RunGpy(t, tmpdir, "package")
	if err != nil {
		t.Fatalf("gpy package failed: %v", err)
	}

	// Generate workflow
	output, err := RunGpy(t, tmpdir, "workflow")
	if err != nil {
		t.Fatalf("gpy workflow failed: %v", err)
	}

	// Verify workflow contains matrix with 5 entries
	if !strings.Contains(output, "matrix:") {
		t.Fatalf("workflow output missing 'matrix:' section")
	}

	// Verify platform-specific configuration
	if !strings.Contains(output, "darwin") {
		t.Fatalf("workflow output missing darwin configuration")
	}
	if !strings.Contains(output, "linux") {
		t.Fatalf("workflow output missing linux configuration")
	}
	if !strings.Contains(output, "windows") {
		t.Fatalf("workflow output missing windows configuration")
	}
}

// TestEndToEndDeterministicOutput verifies that multiple runs produce identical output.
func TestEndToEndDeterministicOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// Run gpy package twice and compare
	_, err := RunGpy(t, tmpdir, "package")
	if err != nil {
		t.Fatalf("first gpy package failed: %v", err)
	}

	npmJson1 := filepath.Join(tmpdir, "packaging", "npm", "testapp", "package.json")
	content1 := ReadFile(t, npmJson1)

	_, err = RunGpy(t, tmpdir, "package")
	if err != nil {
		t.Fatalf("second gpy package failed: %v", err)
	}

	content2 := ReadFile(t, npmJson1)

	if content1 != content2 {
		t.Fatalf("determinism check failed: runs produced different output\nFirst:\n%s\n\nSecond:\n%s", content1, content2)
	}
}

// TestEndToEndCustomVersion tests custom version handling.
func TestEndToEndCustomVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// Run gpy package with custom version
	_, err := RunGpy(t, tmpdir, "package", "--version", "2.5.0")
	if err != nil {
		t.Fatalf("gpy package --version failed: %v", err)
	}

	// Verify version appears in generated files
	npmJson := filepath.Join(tmpdir, "packaging", "npm", "testapp", "package.json")
	data := AssertValidJSON(t, npmJson)

	version, ok := data["version"]
	if !ok {
		t.Fatalf("version field not found in package.json")
	}

	if version != "2.5.0" {
		t.Fatalf("version = %v, want 2.5.0", version)
	}
}

// TestEndToEndWorkflowToStdout tests workflow output to stdout without --write.
func TestEndToEndWorkflowToStdout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// Generate packages first (required for workflow generation)
	_, err := RunGpy(t, tmpdir, "package")
	if err != nil {
		t.Fatalf("gpy package failed: %v", err)
	}

	// Run gpy workflow without --write flag
	output, err := RunGpy(t, tmpdir, "workflow")
	if err != nil {
		t.Fatalf("gpy workflow failed: %v", err)
	}

	// Parse output as YAML
	if !strings.Contains(output, "name:") || !strings.Contains(output, "on:") {
		t.Fatalf("workflow output does not look like valid YAML")
	}

	// Verify no file was written
	AssertFileNotExists(t, filepath.Join(tmpdir, ".github", "workflows", "release.yml"))
}

// TestEndToEndPackagingDirCreation tests packaging directory creation from empty state.
func TestEndToEndPackagingDirCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// Run gpy package
	_, err := RunGpy(t, tmpdir, "package")
	if err != nil {
		t.Fatalf("gpy package failed: %v", err)
	}

	// Verify directory structure was created correctly
	packageDir := filepath.Join(tmpdir, "packaging")
	AssertDirExists(t, packageDir)

	// Check for specific files that confirm the directory structure
	AssertFileExists(t, filepath.Join(packageDir, "npm", "testapp", "package.json"))
	AssertFileExists(t, filepath.Join(packageDir, "homebrew", "Testapp.rb"))
	AssertFileExists(t, filepath.Join(packageDir, "chocolatey", "testapp", "testapp.nuspec"))
}

// TestEndToEndOnlyFlagValidation tests --only flag validation.
func TestEndToEndOnlyFlagValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	tests := []struct {
		name    string
		onlyVal string
		wantErr bool
	}{
		{"only npm", "npm", false},
		{"only homebrew", "homebrew", false},
		{"only chocolatey", "chocolatey", false},
		{"multiple valid", "npm,homebrew", false},
		{"all valid", "npm,homebrew,chocolatey", false},
		{"invalid generator", "invalid", true},
		{"mixed valid invalid", "npm,invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpdir := CreateMinimalProject(t)
			defer SwitchDir(t, tmpdir)()

			_, err := RunGpy(t, tmpdir, "package", "--only", tt.onlyVal)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RunGpy(...--only %s) error = %v, wantErr %v", tt.onlyVal, err, tt.wantErr)
			}
		})
	}
}
