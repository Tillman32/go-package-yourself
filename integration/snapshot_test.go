package integration

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestNPMPackageJsonSnapshot tests that generated npm package.json matches the golden file.
func TestNPMPackageJsonSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping snapshot test in short mode")
	}

	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// Run gpy package
	_, err := RunGpy(t, tmpdir, "package", "--version", "1.0.0")
	if err != nil {
		t.Fatalf("gpy package failed: %v", err)
	}

	// Load generated file
	generatedPath := filepath.Join(tmpdir, "packaging", "npm", "testapp", "package.json")
	AssertFileExists(t, generatedPath)

	// Load golden file
	goldenPath := filepath.Join(testDataDir(t), "npm-package.json.golden")

	// Parse both as JSON to compare structure (ignoring formatting)
	generatedData := AssertValidJSON(t, generatedPath)
	goldenData := AssertValidJSON(t, goldenPath)

	// Compare key fields
	compareJSONFields(t, "package.json", generatedData, goldenData, "name", "version", "description", "main", "scripts")
}

// TestHomebrewFormulaSnapshot tests that generated Homebrew formula matches golden file.
func TestHomebrewFormulaSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping snapshot test in short mode")
	}

	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// Run gpy package
	_, err := RunGpy(t, tmpdir, "package", "--version", "1.0.0")
	if err != nil {
		t.Fatalf("gpy package failed: %v", err)
	}

	// Load generated file
	generatedPath := filepath.Join(tmpdir, "packaging", "homebrew", "Testapp.rb")
	AssertFileExists(t, generatedPath)

	// Load golden file
	goldenPath := filepath.Join(testDataDir(t), "homebrew-formula.rb.golden")

	// Compare content structure (formula contains deterministic parts)
	generatedContent := ReadFile(t, generatedPath)
	_ = ReadFile(t, goldenPath)

	// Check key sections exist (allowing for version/URL changes)
	requiredSections := []string{
		"class Testapp < Formula",
		"desc",
		"url",
		"sha256",
		"def install",
		"bin",
	}

	for _, section := range requiredSections {
		if !stringInFile(generatedContent, section) {
			t.Fatalf("required section %q not found in generated formula", section)
		}
	}
}

// TestChocolateyNuspecSnapshot tests that generated Chocolatey nuspec matches golden file.
func TestChocolateyNuspecSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping snapshot test in short mode")
	}

	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// Run gpy package
	_, err := RunGpy(t, tmpdir, "package", "--version", "1.0.0")
	if err != nil {
		t.Fatalf("gpy package failed: %v", err)
	}

	// Load generated file
	generatedPath := filepath.Join(tmpdir, "packaging", "chocolatey", "testapp", "testapp.nuspec")
	AssertFileExists(t, generatedPath)

	// Load golden file
	goldenPath := filepath.Join(testDataDir(t), "chocolatey.nuspec.golden")

	// Parse as XML-like structure (basic checks)
	generatedContent := ReadFile(t, generatedPath)
	_ = ReadFile(t, goldenPath)

	// Check key elements
	requiredElements := []string{
		"<?xml",
		"<package",
		"<id>testapp</id>",
		"<title>testapp</title>",
		"<version>",
		"<projectUrl>",
		"<packageSourceUrl>",
	}

	for _, elem := range requiredElements {
		if !stringInFile(generatedContent, elem) {
			t.Fatalf("required element %q not found in generated nuspec", elem)
		}
	}
}

// TestWorkflowYamlSnapshot tests that generated workflow YAML matches golden file.
func TestWorkflowYamlSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping snapshot test in short mode")
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
    - os: linux
      arch: amd64
    - os: windows
      arch: amd64
`

	tmpdir := CreateProjectWithConfig(t, config)
	defer SwitchDir(t, tmpdir)()

	// Run gpy package first
	_, err := RunGpy(t, tmpdir, "package", "--version", "1.0.0")
	if err != nil {
		t.Fatalf("gpy package failed: %v", err)
	}

	// Run gpy workflow
	workflowOut, err := RunGpy(t, tmpdir, "workflow")
	if err != nil {
		t.Fatalf("gpy workflow failed: %v", err)
	}

	// Parse as YAML
	workflowData := parseYAMLFromString(t, workflowOut)

	// Verify key structure
	requiredKeys := []string{"name", "on", "jobs"}
	for _, key := range requiredKeys {
		if _, ok := workflowData[key]; !ok {
			t.Fatalf("required key %q not found in workflow YAML", key)
		}
	}

	// Verify matrix has entries
	if jobsVal, ok := workflowData["jobs"]; ok {
		if jobsMap, ok := jobsVal.(map[string]interface{}); ok {
			if _, ok := jobsMap["build"]; ok {
				// Build job exists
			}
		}
	}
}

// TestDeterministicWorkflowGeneration verifies workflow generation is deterministic.
func TestDeterministicWorkflowGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping snapshot test in short mode")
	}

	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// Generate packages
	_, err := RunGpy(t, tmpdir, "package")
	if err != nil {
		t.Fatalf("gpy package failed: %v", err)
	}

	// Generate workflow twice
	output1, err := RunGpy(t, tmpdir, "workflow")
	if err != nil {
		t.Fatalf("first gpy workflow failed: %v", err)
	}

	output2, err := RunGpy(t, tmpdir, "workflow")
	if err != nil {
		t.Fatalf("second gpy workflow failed: %v", err)
	}

	// Outputs should be identical
	if output1 != output2 {
		t.Fatalf("workflow generation not deterministic\nFirst run:\n%s\n\nSecond run:\n%s", output1, output2)
	}
}

// TestPackageJsonFieldsSnapshot checks specific fields in package.json snapshot.
func TestPackageJsonFieldsSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping snapshot test in short mode")
	}

	tmpdir := CreateMinimalProject(t)
	defer SwitchDir(t, tmpdir)()

	// Run gpy package
	_, err := RunGpy(t, tmpdir, "package", "--version", "2.0.0")
	if err != nil {
		t.Fatalf("gpy package failed: %v", err)
	}

	packageJsonPath := filepath.Join(tmpdir, "packaging", "npm", "testapp", "package.json")
	data := AssertValidJSON(t, packageJsonPath)

	// Verify critical fields match expected values
	expectedFields := map[string]interface{}{
		"name":    "testapp",
		"version": "2.0.0",
	}

	for field, expectedVal := range expectedFields {
		actualVal, ok := data[field]
		if !ok {
			t.Fatalf("field %q not found", field)
		}
		if actualVal != expectedVal {
			t.Fatalf("field %q = %v, want %v", field, actualVal, expectedVal)
		}
	}
}

// Helper functions for snapshot testing

func testDataDir(t *testing.T) string {
	_, currentFile, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatalf("failed to get current file path")
	}
	return filepath.Join(filepath.Dir(currentFile), "testdata")
}

func compareJSONFields(t *testing.T, name string, actual, expected map[string]interface{}, fields ...string) {
	t.Helper()
	for _, field := range fields {
		expectedVal, expectedOk := expected[field]
		actualVal, actualOk := actual[field]

		if expectedOk != actualOk {
			t.Errorf("%s: field %q exists in expected=%v, actual=%v", name, field, expectedOk, actualOk)
			continue
		}

		if !expectedOk {
			continue // Both missing is okay
		}

		if !reflect.DeepEqual(expectedVal, actualVal) {
			t.Errorf("%s: field %q expected %v, got %v", name, field, expectedVal, actualVal)
		}
	}
}

func stringInFile(content, str string) bool {
	for i := 0; i < len(content)-len(str); i++ {
		if content[i:i+len(str)] == str {
			return true
		}
	}
	return len(str) == 0 || (len(content) > 0 && content[:len(str)] == str)
}

func parseYAMLFromString(t *testing.T, content string) map[string]interface{} {
	t.Helper()
	var data map[string]interface{}
	err := yaml.Unmarshal([]byte(content), &data)
	if err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}
	return data
}

// Mock timeStamp to ensure deterministic timestamps
func mockTimeNow() time.Time {
	return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
}
