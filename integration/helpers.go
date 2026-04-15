package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"gopkg.in/yaml.v3"
)

// RunGpy executes the gpy binary with the given arguments in the specified directory.
// Returns stdout output and any execution error.
func RunGpy(t *testing.T, dir string, args ...string) (string, error) {
	t.Helper()

	// Prefer local build first - find it relative to this test file
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get current file path")
	}
	projectRoot := filepath.Dir(filepath.Dir(currentFile))
	localGpyPath := filepath.Join(projectRoot, "gpy")

	var gpyPath string
	if _, err := os.Stat(localGpyPath); err == nil {
		gpyPath = localGpyPath
	} else {
		// Fall back to PATH
		var err error
		gpyPath, err = exec.LookPath("gpy")
		if err != nil {
			t.Fatalf("gpy binary not found locally at %s or in PATH: %v", localGpyPath, err)
		}
	}

	cmd := exec.Command(gpyPath, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil && stderr.Len() > 0 {
		return stdout.String(), fmt.Errorf("%w: %s", err, stderr.String())
	}

	return stdout.String(), err
}

// CreateMinimalProject creates a temporary project directory with a minimal config.
// Returns the project directory path.
func CreateMinimalProject(t *testing.T) string {
	t.Helper()

	tmpdir := t.TempDir()

	// Create minimal gpy.yaml
	yml := `schemaVersion: 1
project:
  name: testapp
  repo: testuser/testapp
  description: "Test application"
go:
  main: ./cmd/testapp
`

	configPath := filepath.Join(tmpdir, "gpy.yaml")
	if err := os.WriteFile(configPath, []byte(yml), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Create cmd/testapp directory structure
	cmdDir := filepath.Join(tmpdir, "cmd", "testapp")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatalf("failed to create cmd directory: %v", err)
	}

	// Create a minimal main.go
	mainGo := `package main

func main() {
	println("testapp")
}
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	return tmpdir
}

// CreateProjectWithConfig creates a temporary project with a custom config.
func CreateProjectWithConfig(t *testing.T, configYAML string) string {
	t.Helper()

	tmpdir := t.TempDir()

	configPath := filepath.Join(tmpdir, "gpy.yaml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Create cmd/testapp directory structure
	cmdDir := filepath.Join(tmpdir, "cmd", "testapp")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatalf("failed to create cmd directory: %v", err)
	}

	mainGo := `package main
func main() { println("testapp") }
`
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	return tmpdir
}

// AssertFileExists checks that a file exists and is non-empty.
func AssertFileExists(t *testing.T, path string) {
	t.Helper()

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file %s does not exist: %v", path, err)
	}

	if fi.Size() == 0 {
		t.Fatalf("file %s is empty", path)
	}
}

// AssertFileNotExists checks that a file does not exist.
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()

	_, err := os.Stat(path)
	if err == nil {
		t.Fatalf("file %s exists but should not", path)
	}

	if !os.IsNotExist(err) {
		t.Fatalf("unexpected error checking if file exists: %v", err)
	}
}

// AssertDirExists checks that a directory exists.
func AssertDirExists(t *testing.T, path string) {
	t.Helper()

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("directory %s does not exist: %v", path, err)
	}

	if !fi.IsDir() {
		t.Fatalf("%s is not a directory", path)
	}
}

// ReadFile reads a file and returns its content as a string.
func ReadFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}

	return string(content)
}

// AssertValidJSON parses a file as JSON and returns the parsed data.
// Fails the test if JSON is invalid.
func AssertValidJSON(t *testing.T, path string) map[string]interface{} {
	t.Helper()

	content := ReadFile(t, path)

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		t.Fatalf("file %s is not valid JSON: %v", path, err)
	}

	return data
}

// AssertValidYAML parses a file as YAML and returns the parsed data.
// Fails the test if YAML is invalid.
func AssertValidYAML(t *testing.T, path string) map[string]interface{} {
	t.Helper()

	content := ReadFile(t, path)

	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		t.Fatalf("file %s is not valid YAML: %v", path, err)
	}

	return data
}

// AssertJSONField checks that a JSON file contains a specific field with an expected value.
func AssertJSONField(t *testing.T, path string, field string, expected interface{}) {
	t.Helper()

	data := AssertValidJSON(t, path)

	val, ok := data[field]
	if !ok {
		t.Fatalf("field %q not found in JSON file %s", field, path)
	}

	if val != expected {
		t.Fatalf("field %q = %v, want %v", field, val, expected)
	}
}

// AssertYAMLContains checks that a YAML file contains all specified keys.
func AssertYAMLContains(t *testing.T, path string, keys ...string) {
	t.Helper()

	data := AssertValidYAML(t, path)

	for _, key := range keys {
		if _, ok := data[key]; !ok {
			t.Fatalf("key %q not found in YAML file %s", key, path)
		}
	}
}

// FileContentsEqual checks if two files have identical contents.
func FileContentsEqual(t *testing.T, path1, path2 string) bool {
	t.Helper()

	content1 := ReadFile(t, path1)
	content2 := ReadFile(t, path2)

	return content1 == content2
}

// GetFilesList returns a list of files in a directory (relative paths).
func GetFilesList(t *testing.T, dir string) []string {
	t.Helper()

	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files = append(files, rel)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("failed to walk directory %s: %v", dir, err)
	}

	return files
}

// AssertDirectoryStructure checks that all expected files exist in a directory.
// expectedFiles is a list of relative paths.
func AssertDirectoryStructure(t *testing.T, dir string, expectedFiles ...string) {
	t.Helper()

	files := GetFilesList(t, dir)
	found := make(map[string]bool)

	for _, f := range files {
		found[f] = true
	}

	for _, expected := range expectedFiles {
		if !found[expected] {
			t.Fatalf("expected file %s not found in directory %s", expected, dir)
		}
	}
}

// DiffFiles returns a string describing differences between two files.
// Returns empty string if files are identical.
func DiffFiles(t *testing.T, path1, path2 string) string {
	t.Helper()

	content1 := ReadFile(t, path1)
	content2 := ReadFile(t, path2)

	if content1 == content2 {
		return ""
	}

	// Simple diff output
	lines1 := bytes.Split([]byte(content1), []byte("\n"))
	lines2 := bytes.Split([]byte(content2), []byte("\n"))

	var diff bytes.Buffer
	diff.WriteString(fmt.Sprintf("--- %s\n", path1))
	diff.WriteString(fmt.Sprintf("+++ %s\n", path2))

	maxLines := len(lines1)
	if len(lines2) > maxLines {
		maxLines = len(lines2)
	}

	for i := 0; i < maxLines; i++ {
		if i >= len(lines1) {
			diff.WriteString(fmt.Sprintf("+ %s\n", lines2[i]))
		} else if i >= len(lines2) {
			diff.WriteString(fmt.Sprintf("- %s\n", lines1[i]))
		} else if !bytes.Equal(lines1[i], lines2[i]) {
			diff.WriteString(fmt.Sprintf("- %s\n", lines1[i]))
			diff.WriteString(fmt.Sprintf("+ %s\n", lines2[i]))
		}
	}

	return diff.String()
}

// LoadFixture loads a test fixture file from the fixtures directory.
func LoadFixture(t *testing.T, filename string) string {
	t.Helper()

	// Find the fixtures directory relative to this file
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to get current file path")
	}

	fixtureDir := filepath.Join(filepath.Dir(currentFile), "fixtures")
	fixturePath := filepath.Join(fixtureDir, filename)

	content := ReadFile(t, fixturePath)
	return content
}

// MustGetwd is a test helper to get the current working directory.
func MustGetwd(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	return wd
}

// SwitchDir temporarily changes to a directory and returns a function to switch back.
func SwitchDir(t *testing.T, dir string) func() {
	t.Helper()

	oldDir := MustGetwd(t)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change directory to %s: %v", dir, err)
	}

	return func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Fatalf("failed to change directory back to %s: %v", oldDir, err)
		}
	}
}
