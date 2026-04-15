package chocolatey

import (
	"encoding/xml"
	"strings"
	"testing"

	"go-package-yourself/internal/generator"
	"go-package-yourself/internal/model"
)

// TestName tests that the generator returns the correct name.
func TestName(t *testing.T) {
	gen := &Generator{}
	if gen.Name() != "chocolatey" {
		t.Errorf("expected name 'chocolatey', got '%s'", gen.Name())
	}
}

// TestGenerateNoWindowsPlatforms tests that generation fails when no Windows platforms are configured.
func TestGenerateNoWindowsPlatforms(t *testing.T) {
	gen := &Generator{}
	ctx := generator.Context{
		Config: &model.Config{
			Project: model.Project{
				Name: "test-tool",
				Repo: "owner/test-tool",
			},
			Release: model.Release{
				TagTemplate: "v{{version}}",
				Platforms: []model.Platform{
					{OS: "linux", Arch: "amd64"},
					{OS: "darwin", Arch: "amd64"},
				},
			},
			Packages: model.Packages{
				Chocolatey: model.Chocolatey{
					PackageID: "test-tool",
				},
			},
		},
		ProjectRoot: "/test",
		Version:     "1.0.0",
		ArchiveName: func(os, arch string) (string, string, error) {
			return "test_1.0.0_" + os + "_" + arch + ".tar.gz", "test", nil
		},
	}

	_, err := gen.Generate(ctx)
	if err == nil {
		t.Errorf("expected error for no Windows platforms, got nil")
	}
	if !strings.Contains(err.Error(), "no Windows platforms") {
		t.Errorf("expected error about no Windows platforms, got: %v", err)
	}
}

// TestGenerateIncludesNuspec tests that generation includes a .nuspec file.
func TestGenerateIncludesNuspec(t *testing.T) {
	gen := &Generator{}
	ctx := createTestContext()

	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var nuspecOutput *generator.FileOutput
	for i := range outputs {
		if strings.HasSuffix(outputs[i].Path, ".nuspec") {
			nuspecOutput = &outputs[i]
			break
		}
	}

	if nuspecOutput == nil {
		t.Fatalf("no .nuspec file found in outputs")
	}

	// Verify it's valid XML
	if err := xml.Unmarshal(nuspecOutput.Content, &struct{}{}); err != nil {
		t.Errorf("generated .nuspec is not valid XML: %v", err)
	}
}

// TestGenerateNuspecContent tests that the .nuspec file contains expected metadata.
func TestGenerateNuspecContent(t *testing.T) {
	gen := &Generator{}
	ctx := createTestContext()

	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var nuspecOutput *generator.FileOutput
	for i := range outputs {
		if strings.HasSuffix(outputs[i].Path, ".nuspec") {
			nuspecOutput = &outputs[i]
			break
		}
	}

	content := string(nuspecOutput.Content)

	// Check for required elements
	tests := []struct {
		name   string
		substr string
	}{
		{"package ID", "<id>test-tool</id>"},
		{"version", "<version>1.0.0</version>"},
		{"title", "<title>test-tool</title>"},
		{"description", "<description>"},
		{"authors", "<authors>"},
		{"projectUrl", "<projectUrl>https://github.com/owner/test-tool</projectUrl>"},
	}

	for _, test := range tests {
		if !strings.Contains(content, test.substr) {
			t.Errorf("%s: expected '%s' in nuspec", test.name, test.substr)
		}
	}
}

// TestGenerateIncludesInstallScript tests that generation includes the install script.
func TestGenerateIncludesInstallScript(t *testing.T) {
	gen := &Generator{}
	ctx := createTestContext()

	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var installOutput *generator.FileOutput
	for i := range outputs {
		if strings.HasSuffix(outputs[i].Path, "chocolateyInstall.ps1") {
			installOutput = &outputs[i]
			break
		}
	}

	if installOutput == nil {
		t.Fatalf("no chocolateyInstall.ps1 file found in outputs")
	}

	content := string(installOutput.Content)
	if !strings.Contains(content, "Verify SHA256 checksum") {
		t.Errorf("install script missing SHA256 verification comment")
	}
}

// TestGenerateInstallScriptArchDetection tests that the install script includes arch detection.
func TestGenerateInstallScriptArchDetection(t *testing.T) {
	gen := &Generator{}
	ctx := createTestContext()

	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var installOutput *generator.FileOutput
	for i := range outputs {
		if strings.HasSuffix(outputs[i].Path, "chocolateyInstall.ps1") {
			installOutput = &outputs[i]
			break
		}
	}

	content := string(installOutput.Content)

	// Check for architecture mapping
	tests := []struct {
		name string
		text string
	}{
		{"AMD64 mapping", "AMD64"},
		{"ARM64 mapping", "ARM64"},
		{"processor architecture", "PROCESSOR_ARCHITECTURE"},
		{"SHA256 creation", "HashAlgorithm"},
		{"checksum verification", "ComputeHash"},
	}

	for _, test := range tests {
		if !strings.Contains(content, test.text) {
			t.Errorf("install script missing: %s", test.name)
		}
	}
}

// TestGenerateInstallScriptSHA256Verification tests that SHA256 verification is present and correct.
func TestGenerateInstallScriptSHA256Verification(t *testing.T) {
	gen := &Generator{}
	ctx := createTestContext()

	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var installOutput *generator.FileOutput
	for i := range outputs {
		if strings.HasSuffix(outputs[i].Path, "chocolateyInstall.ps1") {
			installOutput = &outputs[i]
			break
		}
	}

	content := string(installOutput.Content)

	// Check for SHA256 verification code
	sha256Checks := []string{
		"$hasher = [System.Security.Cryptography.HashAlgorithm]::Create('sha256')",
		"$hashBytes = $hasher.ComputeHash($fileStream)",
		"[System.BitConverter]::ToString($hashBytes)",
		"Checksum mismatch",
	}

	for _, check := range sha256Checks {
		if !strings.Contains(content, check) {
			t.Errorf("SHA256 verification missing: %s", check)
		}
	}
}

// TestGenerateIncludesUninstallScript tests that generation includes the uninstall script.
func TestGenerateIncludesUninstallScript(t *testing.T) {
	gen := &Generator{}
	ctx := createTestContext()

	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var uninstallOutput *generator.FileOutput
	for i := range outputs {
		if strings.HasSuffix(outputs[i].Path, "chocolateyUninstall.ps1") {
			uninstallOutput = &outputs[i]
			break
		}
	}

	if uninstallOutput == nil {
		t.Fatalf("no chocolateyUninstall.ps1 file found in outputs")
	}

	content := string(uninstallOutput.Content)
	if !strings.Contains(content, "uninstall") {
		t.Errorf("uninstall script missing uninstall reference")
	}
}

// TestGenerateFilePaths tests that output files are in the correct directory structure.
func TestGenerateFilePaths(t *testing.T) {
	gen := &Generator{}
	ctx := createTestContext()

	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	expectedPaths := []string{
		"packaging/chocolatey/test-tool/test-tool.nuspec",
		"packaging/chocolatey/test-tool/tools/chocolateyInstall.ps1",
		"packaging/chocolatey/test-tool/tools/chocolateyUninstall.ps1",
	}

	actualPaths := make(map[string]bool)
	for _, output := range outputs {
		actualPaths[output.Path] = true
	}

	for _, expected := range expectedPaths {
		if !actualPaths[expected] {
			t.Errorf("expected path not found: %s", expected)
		}
	}
}

// TestGenerateFileModes tests that output files have correct permissions.
func TestGenerateFileModes(t *testing.T) {
	gen := &Generator{}
	ctx := createTestContext()

	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	for _, output := range outputs {
		// PowerShell scripts should be executable
		if strings.HasSuffix(output.Path, ".ps1") {
			if output.Mode != 0o755 {
				t.Errorf("PowerShell script %s should be executable (0o755), got %o", output.Path, output.Mode)
			}
		}
		// XML files should not be executable
		if strings.HasSuffix(output.Path, ".nuspec") {
			if output.Mode != 0o644 {
				t.Errorf("nuspec file %s should not be executable (0o644), got %o", output.Path, output.Mode)
			}
		}
	}
}

// TestGenerateDeterministic tests that generation is deterministic (same input -> same output).
func TestGenerateDeterministic(t *testing.T) {
	gen1 := &Generator{}
	gen2 := &Generator{}

	ctx1 := createTestContext()
	ctx2 := createTestContext()

	outputs1, err1 := gen1.Generate(ctx1)
	outputs2, err2 := gen2.Generate(ctx2)

	if err1 != nil || err2 != nil {
		t.Fatalf("generate failed: %v, %v", err1, err2)
	}

	if len(outputs1) != len(outputs2) {
		t.Errorf("output count mismatch: %d vs %d", len(outputs1), len(outputs2))
		return
	}

	for i, out1 := range outputs1 {
		out2 := outputs2[i]
		if out1.Path != out2.Path {
			t.Errorf("output path mismatch at index %d: %s vs %s", i, out1.Path, out2.Path)
		}
		if string(out1.Content) != string(out2.Content) {
			t.Errorf("output content mismatch at index %d (%s)", i, out1.Path)
		}
		if out1.Mode != out2.Mode {
			t.Errorf("output mode mismatch at index %d: %o vs %o", i, out1.Mode, out2.Mode)
		}
	}
}

// TestGenerateWindowsOnlyPlatforms tests that only Windows platforms are included.
func TestGenerateWindowsOnlyPlatforms(t *testing.T) {
	gen := &Generator{}
	ctx := generator.Context{
		Config: &model.Config{
			Project: model.Project{
				Name: "test-tool",
				Repo: "owner/test-tool",
			},
			Release: model.Release{
				TagTemplate: "v{{version}}",
				Platforms: []model.Platform{
					{OS: "linux", Arch: "amd64"},
					{OS: "windows", Arch: "amd64"},
					{OS: "darwin", Arch: "arm64"},
					{OS: "windows", Arch: "arm64"},
				},
			},
			Packages: model.Packages{
				Chocolatey: model.Chocolatey{},
			},
		},
		ProjectRoot: "/test",
		Version:     "1.0.0",
		ArchiveName: func(os, arch string) (string, string, error) {
			return os + "_" + arch + ".zip", "test.exe", nil
		},
	}

	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	// Find install script and check it only references Windows architectures
	var installContent string
	for _, out := range outputs {
		if strings.HasSuffix(out.Path, "chocolateyInstall.ps1") {
			installContent = string(out.Content)
			break
		}
	}

	if !strings.Contains(installContent, "windows/amd64") && !strings.Contains(installContent, "amd64") {
		t.Errorf("install script should handle x64 (windows/amd64)")
	}

	if !strings.Contains(installContent, "windows/arm64") && !strings.Contains(installContent, "arm64") {
		t.Errorf("install script should handle ARM64 (windows/arm64)")
	}

	// Check that linux and darwin are NOT referenced
	if strings.Contains(installContent, "linux") {
		t.Errorf("install script should not reference linux")
	}
}

// TestGenerateCustomPackageID tests that custom package IDs are respected.
func TestGenerateCustomPackageID(t *testing.T) {
	gen := &Generator{}
	ctx := createTestContext()
	ctx.Config.Packages.Chocolatey.PackageID = "custom-pkg-id"

	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	for _, output := range outputs {
		if !strings.Contains(output.Path, "custom-pkg-id") {
			t.Errorf("output path should contain custom-pkg-id: %s", output.Path)
		}
	}
}

// TestGenerateGitHubReleaseTags tests that GitHub release URLs are properly formatted.
func TestGenerateGitHubReleaseTags(t *testing.T) {
	gen := &Generator{}
	ctx := createTestContext()
	ctx.Version = "v2.1.0"

	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var installContent string
	for _, out := range outputs {
		if strings.HasSuffix(out.Path, "chocolateyInstall.ps1") {
			installContent = string(out.Content)
			break
		}
	}

	// Check GitHub URLs are present
	if !strings.Contains(installContent, "https://github.com/owner/test-tool/releases/download/") {
		t.Errorf("install script should contain GitHub release download URL")
	}

	if !strings.Contains(installContent, "v2.1.0") {
		t.Errorf("install script should contain version v2.1.0")
	}
}

// TestEscapeXML tests XML entity escaping.
func TestEscapeXML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"text with & ampersand", "text with &amp; ampersand"},
		{"<tag>", "&lt;tag&gt;"},
		{"quote\"here", "quote&quot;here"},
		{"apostrophe'here", "apostrophe&apos;here"},
		{"multiple & < > cases", "multiple &amp; &lt; &gt; cases"},
	}

	for _, test := range tests {
		result := escapeXML(test.input)
		if result != test.expected {
			t.Errorf("escapeXML(%q) = %q, want %q", test.input, result, test.expected)
		}
	}
}

// TestFilterWindowsPlatforms tests platform filtering logic.
func TestFilterWindowsPlatforms(t *testing.T) {
	platforms := []model.Platform{
		{OS: "linux", Arch: "amd64"},
		{OS: "windows", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "windows", Arch: "arm64"},
	}

	windows := filterWindowsPlatforms(platforms)

	if len(windows) != 2 {
		t.Errorf("expected 2 Windows platforms, got %d", len(windows))
	}

	for _, p := range windows {
		if p.OS != "windows" {
			t.Errorf("filtered platform has OS=%s, expected windows", p.OS)
		}
	}
}

// TestGeneratePowerShellSyntax tests basic PowerShell syntax validity.
func TestGeneratePowerShellSyntax(t *testing.T) {
	gen := &Generator{}
	ctx := createTestContext()

	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	for _, out := range outputs {
		if strings.HasSuffix(out.Path, ".ps1") {
			content := string(out.Content)

			// Check for balanced braces
			openBraces := strings.Count(content, "{")
			closeBraces := strings.Count(content, "}")
			if openBraces != closeBraces {
				t.Errorf("unbalanced braces in %s: {%d != }%d", out.Path, openBraces, closeBraces)
			}

			// Check for balanced parentheses
			openParens := strings.Count(content, "(")
			closeParens := strings.Count(content, ")")
			if openParens != closeParens {
				t.Errorf("unbalanced parentheses in %s: (%d != )%d", out.Path, openParens, closeParens)
			}

			// Valid PowerShell should have proper error handling
			if !strings.Contains(content, "try") || !strings.Contains(content, "catch") {
				t.Errorf("PowerShell script %s should have try/catch error handling", out.Path)
			}
		}
	}
}

// TestGenerateWithoutVersion tests generation when version is empty.
func TestGenerateWithoutVersion(t *testing.T) {
	gen := &Generator{}
	ctx := createTestContext()
	ctx.Version = ""

	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	// Find nuspec and check version is placeholder
	var nuspecContent string
	for _, out := range outputs {
		if strings.HasSuffix(out.Path, ".nuspec") {
			nuspecContent = string(out.Content)
			break
		}
	}

	if !strings.Contains(nuspecContent, "<version>0.0.0</version>") {
		t.Errorf("nuspec should use placeholder version 0.0.0 when version is empty")
	}
}

// createTestContext creates a test generator context with reasonable defaults.
func createTestContext() generator.Context {
	return generator.Context{
		Config: &model.Config{
			Project: model.Project{
				Name:        "test-tool",
				Repo:        "owner/test-tool",
				Description: "Test tool description",
			},
			Release: model.Release{
				TagTemplate: "v{{version}}",
				Platforms: []model.Platform{
					{OS: "windows", Arch: "amd64"},
					{OS: "windows", Arch: "arm64"},
				},
			},
			Packages: model.Packages{
				Chocolatey: model.Chocolatey{
					Authors: "Test Author",
				},
			},
		},
		ProjectRoot: "/test",
		Version:     "1.0.0",
		ArchiveName: func(os, arch string) (string, string, error) {
			ext := ".tar.gz"
			if os == "windows" {
				ext = ".zip"
			}
			binPath := "test"
			if os == "windows" {
				binPath = "test.exe"
			}
			return "test_1.0.0_" + os + "_" + arch + ext, binPath, nil
		},
		RenderTemplate: func(template, fieldPath string) (string, error) {
			// Simple mock - just return the template as-is
			return template, nil
		},
	}
}
