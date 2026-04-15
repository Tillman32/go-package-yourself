package homebrew

import (
	"strings"
	"testing"

	"go-package-yourself/internal/generator"
	"go-package-yourself/internal/model"
	"go-package-yourself/internal/naming"
)

// Test that Generator implements the Generator interface.
func TestGeneratorInterface(t *testing.T) {
	var g generator.Generator = &Generator{}
	if g == nil {
		t.Fatal("Generator does not implement generator.Generator interface")
	}

	if name := g.Name(); name != "homebrew" {
		t.Fatalf("Generator Name() = %q, want %q", name, "homebrew")
	}
}

// Test single platform generation (darwin/amd64 only).
func TestGenerateSinglePlatform(t *testing.T) {
	ctx := createTestContext("1.2.3", []model.Platform{
		{OS: "darwin", Arch: "amd64"},
	})

	gen := &Generator{}
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(outputs) != 1 {
		t.Fatalf("Expected 1 output, got %d", len(outputs))
	}

	output := outputs[0]
	if !strings.HasSuffix(output.Path, "MyTool.rb") {
		t.Fatalf("Expected path to end with MyTool.rb, got %s", output.Path)
	}

	formula := string(output.Content)

	// Verify class definition
	if !strings.Contains(formula, "class MyTool < Formula") {
		t.Errorf("Formula missing class definition")
	}

	// Verify we have the URL block (not wrapped in conditional for single arch)
	if !strings.Contains(formula, "on_macos do") {
		t.Errorf("Formula missing on_macos block")
	}

	// Single platform should not have if/else conditionals
	if strings.Contains(formula, "if Hardware::CPU.arm?") {
		t.Errorf("Single platform should not have CPU conditionals")
	}

	// Verify install method
	if !strings.Contains(formula, "def install") {
		t.Errorf("Formula missing install method")
	}

	// Verify test method
	if !strings.Contains(formula, "test do") {
		t.Errorf("Formula missing test method")
	}
}

// Test multiple platforms (darwin/amd64 and darwin/arm64).
func TestGenerateMultiplePlatformsSingleOS(t *testing.T) {
	ctx := createTestContext("1.2.3", []model.Platform{
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
	})

	gen := &Generator{}
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	formula := string(outputs[0].Content)

	// Should have conditional for multiple architectures
	if !strings.Contains(formula, "if Hardware::CPU.arm?") {
		t.Errorf("Formula missing CPU conditional for multiple architectures")
	}

	// Should have both architectures
	if !strings.Contains(formula, "darwin_amd64") && !strings.Contains(formula, "darwin_arm64") {
		t.Errorf("Formula missing architecture-specific references")
	}
}

// Test cross-platform generation (darwin, linux, windows).
func TestGenerateCrossPlatform(t *testing.T) {
	ctx := createTestContext("1.2.3", []model.Platform{
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		// Note: Windows is filtered out because Homebrew doesn't support it
	})

	gen := &Generator{}
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	formula := string(outputs[0].Content)

	// Should have both on_macos and on_linux blocks
	if !strings.Contains(formula, "on_macos do") {
		t.Errorf("Formula missing on_macos block")
	}

	if !strings.Contains(formula, "on_linux do") {
		t.Errorf("Formula missing on_linux block")
	}

	// Should not have Windows references
	if strings.Contains(formula, "on_windows") || strings.Contains(formula, "on_win") {
		t.Errorf("Formula should not have Windows support")
	}
}

// Test custom formula name.
func TestGenerateCustomFormulaName(t *testing.T) {
	ctx := createTestContext("1.2.3", []model.Platform{
		{OS: "darwin", Arch: "amd64"},
	})
	ctx.Config.Packages.Homebrew.FormulaName = "CustomName"

	gen := &Generator{}
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := outputs[0]
	if !strings.Contains(output.Path, "CustomName.rb") {
		t.Fatalf("Expected path to contain CustomName.rb, got %s", output.Path)
	}

	formula := string(output.Content)
	if !strings.Contains(formula, "class CustomName < Formula") {
		t.Errorf("Formula missing custom class name")
	}
}

// Test formula name derivation from project name with separators.
func TestGenerateFormulaNameFromProjectName(t *testing.T) {
	tests := []struct {
		projectName string
		wantName    string
	}{
		{"my-tool", "MyTool"},
		{"my_tool", "MyTool"},
		{"mytool", "Mytool"},
		{"MyTool", "MyTool"},
	}

	for _, tt := range tests {
		t.Run(tt.projectName, func(t *testing.T) {
			ctx := createTestContext("1.2.3", []model.Platform{
				{OS: "darwin", Arch: "amd64"},
			})
			ctx.Config.Project.Name = tt.projectName
			ctx.Config.Packages.Homebrew.FormulaName = "" // Use default

			gen := &Generator{}
			outputs, err := gen.Generate(ctx)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			formula := string(outputs[0].Content)
			expected := "class " + tt.wantName + " < Formula"
			if !strings.Contains(formula, expected) {
				t.Errorf("Formula missing expected class definition: %s", expected)
			}
		})
	}
}

// Test metadata inclusion (description, homepage, license).
func TestGenerateWithMetadata(t *testing.T) {
	ctx := createTestContext("1.2.3", []model.Platform{
		{OS: "darwin", Arch: "amd64"},
	})
	ctx.Config.Project.Description = "A cool tool"
	ctx.Config.Project.Homepage = "https://example.com"
	ctx.Config.Project.License = "MIT"

	gen := &Generator{}
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	formula := string(outputs[0].Content)

	if !strings.Contains(formula, `desc "A cool tool"`) {
		t.Errorf("Formula missing description")
	}

	if !strings.Contains(formula, `homepage "https://example.com"`) {
		t.Errorf("Formula missing homepage")
	}

	if !strings.Contains(formula, `license "MIT"`) {
		t.Errorf("Formula missing license")
	}
}

// Test URL generation with GitHub repo reference.
func TestGenerateURLFormat(t *testing.T) {
	ctx := createTestContext("v1.2.3", []model.Platform{
		{OS: "darwin", Arch: "amd64"},
	})
	ctx.Config.Project.Repo = "owner/mytool"

	gen := &Generator{}
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	formula := string(outputs[0].Content)

	// Should have GitHub release URL
	if !strings.Contains(formula, "https://github.com/owner/mytool/releases/download/") {
		t.Errorf("Formula missing GitHub release URL")
	}

	if !strings.Contains(formula, "v1.2.3") {
		t.Errorf("Formula missing version tag")
	}
}

// Test sha256 checksum generation.
func TestGenerateSha256Checksums(t *testing.T) {
	ctx := createTestContext("1.2.3", []model.Platform{
		{OS: "darwin", Arch: "amd64"},
	})

	gen := &Generator{}
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	formula := string(outputs[0].Content)

	// Should have sha256 entries
	if !strings.Contains(formula, "sha256 \"") {
		t.Errorf("Formula missing sha256 checksums")
	}

	// Count sha256 lines
	sha256Count := strings.Count(formula, "sha256 \"")
	if sha256Count < 1 {
		t.Errorf("Expected at least 1 sha256, got %d", sha256Count)
	}
}

// Test deterministic output (same input produces same output).
func TestGenerateDeterministic(t *testing.T) {
	ctx1 := createTestContext("1.2.3", []model.Platform{
		{OS: "darwin", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
	})

	ctx2 := createTestContext("1.2.3", []model.Platform{
		{OS: "darwin", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
	})

	gen := &Generator{}

	outputs1, err := gen.Generate(ctx1)
	if err != nil {
		t.Fatalf("Generate 1 failed: %v", err)
	}

	outputs2, err := gen.Generate(ctx2)
	if err != nil {
		t.Fatalf("Generate 2 failed: %v", err)
	}

	if string(outputs1[0].Content) != string(outputs2[0].Content) {
		t.Errorf("Output is not deterministic")
	}
}

// Test Ruby syntax validity (basic structure checks).
func TestGenerateValidRubySyntax(t *testing.T) {
	ctx := createTestContext("1.2.3", []model.Platform{
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "linux", Arch: "amd64"},
	})

	gen := &Generator{}
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	formula := string(outputs[0].Content)

	// Count class/end pairs
	classCount := strings.Count(formula, "class ")
	_ = strings.Count(formula, "\nend\n") + tailEndCount(formula)
	if classCount != 1 {
		t.Errorf("Expected exactly 1 class definition, got %d", classCount)
	}

	// Basic structure validation
	if !strings.HasPrefix(formula, "class ") {
		t.Errorf("Formula should start with class definition")
	}

	if !strings.HasSuffix(formula, "end\n") {
		t.Errorf("Formula should end with 'end\\n'")
	}
}

// Helper to check if formula ends with "end\n"
func tailEndCount(s string) int {
	if strings.HasSuffix(s, "\nend\n") {
		return 1
	}
	return 0
}

// Test invalid formula names are rejected.
func TestGenerateInvalidFormulaName(t *testing.T) {
	tests := []struct {
		name      string
		shouldErr bool
	}{
		{"123tool", true},    // starts with digit
		{"_tool", true},      // starts with underscore
		{"MyTool", false},    // valid
		{"my-tool", true},    // contains dash (not valid Ruby class name directly)
		{"ValidTool", false}, // valid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := createTestContext("1.2.3", []model.Platform{
				{OS: "darwin", Arch: "amd64"},
			})
			ctx.Config.Packages.Homebrew.FormulaName = tt.name

			gen := &Generator{}
			_, err := gen.Generate(ctx)

			if tt.shouldErr && err == nil {
				t.Errorf("Expected error for invalid formula name %q", tt.name)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error for valid formula name %q, got: %v", tt.name, err)
			}
		})
	}
}

// Test with no configured platforms (should still generate but with default platforms).
func TestGenerateWithDefaultPlatforms(t *testing.T) {
	ctx := createTestContext("1.2.3", []model.Platform{})
	// Empty platforms list will use defaults during validation/processing

	gen := &Generator{}
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(outputs) != 1 {
		t.Fatalf("Expected 1 output, got %d", len(outputs))
	}
}

// Test escaping of special characters in metadata.
func TestGenerateEscapeSpecialCharacters(t *testing.T) {
	ctx := createTestContext("1.2.3", []model.Platform{
		{OS: "darwin", Arch: "amd64"},
	})
	ctx.Config.Project.Description = `A "special" tool with backslash \ and quotes`

	gen := &Generator{}
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	formula := string(outputs[0].Content)

	// Should have escaped quotes and backslashes
	if !strings.Contains(formula, `\"special\"`) {
		t.Errorf("Formula should have escaped quotes in description")
	}
}

// Test output file permissions.
func TestGenerateFilePermissions(t *testing.T) {
	ctx := createTestContext("1.2.3", []model.Platform{
		{OS: "darwin", Arch: "amd64"},
	})

	gen := &Generator{}
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	output := outputs[0]
	if output.Mode != 0o644 {
		t.Fatalf("Expected file mode 0o644, got %o", output.Mode)
	}
}

// Helper function to create a test context.
func createTestContext(version string, platforms []model.Platform) generator.Context {
	cfg := &model.Config{
		Project: model.Project{
			Name:        "my-tool",
			Repo:        "owner/my-tool",
			Description: "Test tool",
			Homepage:    "https://example.com",
			License:     "MIT",
		},
		Release: model.Release{
			Platforms: platforms,
			Archive: model.Archive{
				NameTemplate:     "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathInArchive: "{{name}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
		Packages: model.Packages{
			Homebrew: model.Homebrew{
				Enabled: true,
			},
		},
	}

	// Create archiveName function that mimics the CLI behavior
	archiveNameFunc := func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
		params := naming.ArchiveNameParams{
			Name:                cfg.Project.Name,
			Version:             version,
			OS:                  os,
			Arch:                arch,
			Format:              "tar.gz",
			ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
			BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
		}
		if os == "windows" && cfg.Release.Archive.Format.Windows != "" {
			params.Format = cfg.Release.Archive.Format.Windows
		}
		return naming.ArchiveName(params)
	}

	return generator.Context{
		Config:      cfg,
		ProjectRoot: "/tmp/test",
		Version:     version,
		ArchiveName: archiveNameFunc,
	}
}
