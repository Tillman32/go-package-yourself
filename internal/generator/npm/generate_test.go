package npm

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"go-package-yourself/internal/generator"
	"go-package-yourself/internal/model"
	"go-package-yourself/internal/naming"
	"go-package-yourself/internal/templatex"
)

// TestGeneratorName verifies the generator name is correct.
func TestGeneratorName(t *testing.T) {
	gen := &Generator{}
	if got := gen.Name(); got != "npm" {
		t.Fatalf("Name() = %q, want %q", got, "npm")
	}
}

// TestGenerateOutputs verifies that all expected files are generated.
func TestGenerateOutputs(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name:        "mytool",
			Repo:        "owner/mytool",
			Description: "A test tool",
			License:     "MIT",
			Homepage:    "https://example.com",
		},
		Packages: model.Packages{
			NPM: model.NPM{
				Enabled:     true,
				PackageName: "my-tool",
				BinName:     "mytool",
				NodeEngines: ">=18",
			},
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "linux", Arch: "amd64"},
				{OS: "darwin", Arch: "arm64"},
			},
			Archive: model.Archive{
				NameTemplate:     "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathInArchive: "bin/{{name}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
	}

	ctx := makeContext(cfg, "1.2.3")
	gen := &Generator{}
	outputs, err := gen.Generate(ctx)

	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	expectedFiles := map[string]bool{
		"packaging/npm/my-tool/package.json":   false,
		"packaging/npm/my-tool/index.js":       false,
		"packaging/npm/my-tool/install.js":     false,
		"packaging/npm/my-tool/.gitignore":     false,
		"packaging/npm/my-tool/README.md":      false,
		"packaging/npm/my-tool/bin/mytool":     false,
		"packaging/npm/my-tool/bin/mytool.cmd": false,
	}

	if len(outputs) < len(expectedFiles) {
		t.Errorf("Generate() returned %d outputs, want at least %d", len(outputs), len(expectedFiles))
	}

	foundFiles := make(map[string]bool)
	for _, output := range outputs {
		if _, expect := expectedFiles[output.Path]; expect {
			foundFiles[output.Path] = true
		}
	}

	for expected := range expectedFiles {
		if !foundFiles[expected] {
			t.Errorf("Missing output file: %s", expected)
		}
	}
}

// TestPackageJSON verifies the generated package.json is valid.
func TestPackageJSON(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name:        "mytool",
			Repo:        "owner/mytool",
			Description: "A test tool",
			License:     "MIT",
			Homepage:    "https://example.com",
		},
		Packages: model.Packages{
			NPM: model.NPM{
				Enabled:     true,
				PackageName: "my-tool-pkg",
				BinName:     "mytool",
				NodeEngines: ">=16",
			},
		},
	}

	pkgJSON, err := generatePackageJSON(cfg, "my-tool-pkg", "mytool", ">=16", "1.0.0")
	if err != nil {
		t.Fatalf("generatePackageJSON() failed: %v", err)
	}

	// Verify it's valid JSON
	var pkg map[string]interface{}
	if err := json.Unmarshal(pkgJSON, &pkg); err != nil {
		t.Fatalf("Generated package.json is invalid: %v", err)
	}

	// Check key fields
	if got := pkg["name"]; got != "my-tool-pkg" {
		t.Errorf("package.json name = %q, want %q", got, "my-tool-pkg")
	}

	if got := pkg["description"]; got != "A test tool" {
		t.Errorf("package.json description = %q, want %q", got, "A test tool")
	}

	if got := pkg["license"]; got != "MIT" {
		t.Errorf("package.json license = %q, want %q", got, "MIT")
	}

	if got := pkg["homepage"]; got != "https://example.com" {
		t.Errorf("package.json homepage = %q, want %q", got, "https://example.com")
	}

	// Check bin field
	if bin, ok := pkg["bin"].(map[string]interface{}); ok {
		if got := bin["mytool"]; got != "index.js" {
			t.Errorf("package.json bin.mytool = %q, want %q", got, "index.js")
		}
	} else {
		t.Errorf("package.json bin field is not a map")
	}

	// Check engines field
	if engines, ok := pkg["engines"].(map[string]interface{}); ok {
		if got := engines["node"]; got != ">=16" {
			t.Errorf("package.json engines.node = %q, want %q", got, ">=16")
		}
	} else {
		t.Errorf("package.json engines field is not a map")
	}

	// Check postinstall field
	if got := pkg["postinstall"]; got != "node install.js" {
		t.Errorf("package.json postinstall = %q, want %q", got, "node install.js")
	}
}

// TestLauncherScriptGeneration verifies the launcher script is generated correctly.
func TestLauncherScriptGeneration(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Packages: model.Packages{
			NPM: model.NPM{
				Enabled: true,
				BinName: "mytool",
			},
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "linux", Arch: "amd64"},
				{OS: "darwin", Arch: "arm64"},
			},
			Archive: model.Archive{
				NameTemplate:     "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathInArchive: "bin/{{name}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
	}

	ctx := makeContext(cfg, "1.0.0")
	launcher, err := generateLauncher(&ctx, cfg, "mytool", "packaging/npm/mytool")
	if err != nil {
		t.Fatalf("generateLauncher() failed: %v", err)
	}

	launcherStr := string(launcher)

	// Verify key content
	if !strings.Contains(launcherStr, "#!/usr/bin/env node") {
		t.Error("Launcher is missing shebang")
	}

	if !strings.Contains(launcherStr, "REPO = 'owner/mytool'") {
		t.Error("Launcher is missing repo information")
	}

	if !strings.Contains(launcherStr, "BIN_NAME = 'mytool'") {
		t.Error("Launcher is missing binary name")
	}

	if !strings.Contains(launcherStr, "crypto.createHash('sha256')") {
		t.Error("Launcher is missing SHA256 verification")
	}

	if !strings.Contains(launcherStr, "checksumUrl") {
		t.Error("Launcher is missing checksum verification")
	}

	if !strings.Contains(launcherStr, "getCacheDir()") {
		t.Error("Launcher is missing cache directory logic")
	}

	if !strings.Contains(launcherStr, "execFileSync") {
		t.Error("Launcher is missing binary execution")
	}
}

// TestLauncherIncludesPlatforms verifies all platforms are included in launcher.
func TestLauncherIncludesPlatforms(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Packages: model.Packages{
			NPM: model.NPM{Enabled: true, BinName: "mytool"},
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "linux", Arch: "amd64"},
				{OS: "linux", Arch: "arm64"},
				{OS: "darwin", Arch: "amd64"},
				{OS: "darwin", Arch: "arm64"},
				{OS: "windows", Arch: "amd64"},
			},
			Archive: model.Archive{
				NameTemplate:     "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathInArchive: "{{name}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
	}

	ctx := makeContext(cfg, "1.0.0")
	launcher, err := generateLauncher(&ctx, cfg, "mytool", "packaging/npm/mytool")
	if err != nil {
		t.Fatalf("generateLauncher() failed: %v", err)
	}

	launcherStr := string(launcher)

	platformChecks := []struct {
		os   string
		arch string
	}{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
	}

	for _, check := range platformChecks {
		platformStr := "platform: '" + check.os + "', arch: '" + check.arch + "'"
		if !strings.Contains(launcherStr, platformStr) {
			t.Errorf("Launcher missing platform %s/%s", check.os, check.arch)
		}
	}
}

// TestReadmeGeneration verifies the README is generated with correct content.
func TestReadmeGeneration(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name:    "mytool",
			Repo:    "owner/mytool",
			License: "MIT",
		},
		Packages: model.Packages{
			NPM: model.NPM{
				Enabled:     true,
				NodeEngines: ">=18",
			},
		},
	}

	readme, err := generateReadme(cfg, "my-tool", "mytool")
	if err != nil {
		t.Fatalf("generateReadme() failed: %v", err)
	}

	readmeStr := string(readme)

	// Check key sections
	if !strings.Contains(readmeStr, "# my-tool") {
		t.Error("README missing title")
	}

	if !strings.Contains(readmeStr, "npm install -g my-tool") {
		t.Error("README missing install command")
	}

	if !strings.Contains(readmeStr, "mytool [options] [arguments]") {
		t.Error("README missing usage example")
	}

	if !strings.Contains(readmeStr, "mytool --help") {
		t.Error("README missing help command")
	}

	if !strings.Contains(readmeStr, ">=18") {
		t.Error("README missing Node.js requirement")
	}
}

// TestGitignoreGeneration verifies .gitignore content.
func TestGitignoreGeneration(t *testing.T) {
	gitignore := generateGitignore()

	if !strings.Contains(gitignore, "node_modules/") {
		t.Error(".gitignore missing node_modules")
	}

	if !strings.Contains(gitignore, ".DS_Store") {
		t.Error(".gitignore missing .DS_Store")
	}
}

// TestFileModes verifies correct file permissions are set.
func TestFileModes(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name:        "mytool",
			Repo:        "owner/mytool",
			Description: "Test",
		},
		Packages: model.Packages{
			NPM: model.NPM{Enabled: true, BinName: "mytool"},
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "linux", Arch: "amd64"},
			},
			Archive: model.Archive{
				NameTemplate:     "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathInArchive: "{{name}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
	}

	ctx := makeContext(cfg, "1.0.0")
	gen := &Generator{}
	outputs, err := gen.Generate(ctx)

	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Verify executable scripts have correct mode
	modeChecks := map[string]os.FileMode{
		"packaging/npm/mytool/index.js":     0o755,
		"packaging/npm/mytool/install.js":   0o755,
		"packaging/npm/mytool/package.json": 0o644,
		"packaging/npm/mytool/README.md":    0o644,
		"packaging/npm/mytool/.gitignore":   0o644,
	}

	for _, output := range outputs {
		if expected, ok := modeChecks[output.Path]; ok {
			if output.Mode != expected {
				t.Errorf("File %s mode = 0o%o, want 0o%o", output.Path, output.Mode, expected)
			}
		}
	}
}

// TestDeterministicOutput verifies output is deterministic.
func TestDeterministicOutput(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name:        "mytool",
			Repo:        "owner/mytool",
			Description: "Test",
			License:     "MIT",
		},
		Packages: model.Packages{
			NPM: model.NPM{Enabled: true, PackageName: "my-tool", BinName: "mytool"},
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "windows", Arch: "amd64"},
				{OS: "linux", Arch: "amd64"},
				{OS: "darwin", Arch: "arm64"},
			},
			Archive: model.Archive{
				NameTemplate:     "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathInArchive: "{{name}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
	}

	ctx := makeContext(cfg, "1.0.0")

	// Generate twice
	gen := &Generator{}
	outputs1, err1 := gen.Generate(ctx)
	if err1 != nil {
		t.Fatalf("First Generate() failed: %v", err1)
	}

	outputs2, err2 := gen.Generate(ctx)
	if err2 != nil {
		t.Fatalf("Second Generate() failed: %v", err2)
	}

	// Outputs should match exactly
	if len(outputs1) != len(outputs2) {
		t.Errorf("Second run produced %d outputs, first produced %d", len(outputs2), len(outputs1))
	}

	for i := range outputs1 {
		if outputs1[i].Path != outputs2[i].Path {
			t.Errorf("Output[%d] path mismatch: %q vs %q", i, outputs1[i].Path, outputs2[i].Path)
		}

		if !bytes.Equal(outputs1[i].Content, outputs2[i].Content) {
			t.Errorf("Output[%d] content mismatch", i)
		}

		if outputs1[i].Mode != outputs2[i].Mode {
			t.Errorf("Output[%d] mode mismatch: 0o%o vs 0o%o", i, outputs1[i].Mode, outputs2[i].Mode)
		}
	}
}

// TestErrorHandling verifies error cases are handled correctly.
func TestErrorHandling(t *testing.T) {
	t.Run("InvalidRepo", func(t *testing.T) {
		cfg := &model.Config{
			Project: model.Project{
				Name: "mytool",
				Repo: "invalid-repo",
			},
			Packages: model.Packages{
				NPM: model.NPM{Enabled: true},
			},
		}

		ctx := makeContext(cfg, "1.0.0")
		gen := &Generator{}
		_, err := gen.Generate(ctx)

		if err == nil {
			t.Error("Expected error for invalid repo format, got nil")
		}
	})

	t.Run("NoPlatforms", func(t *testing.T) {
		cfg := &model.Config{
			Project: model.Project{
				Name: "mytool",
				Repo: "owner/mytool",
			},
			Packages: model.Packages{
				NPM: model.NPM{Enabled: true, BinName: "mytool"},
			},
			Release: model.Release{
				Platforms: []model.Platform{},
				Archive: model.Archive{
					NameTemplate:     "{{name}}",
					BinPathInArchive: "{{name}}",
					Format: model.ArchiveFormat{
						Default: "tar.gz",
					},
				},
			},
		}

		ctx := makeContext(cfg, "1.0.0")
		gen := &Generator{}
		_, err := gen.Generate(ctx)

		if err == nil {
			t.Error("Expected error for empty platforms, got nil")
		}
	})
}

// TestBinPathWindowsExe verifies Windows executables get .exe extension.
func TestBinPathWindowsExe(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Packages: model.Packages{
			NPM: model.NPM{Enabled: true, BinName: "mytool"},
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "windows", Arch: "amd64"},
			},
			Archive: model.Archive{
				NameTemplate:     "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathInArchive: "{{name}}",
				Format: model.ArchiveFormat{
					Windows: "zip",
				},
			},
		},
	}

	ctx := makeContext(cfg, "1.0.0")
	launcher, err := generateLauncher(&ctx, cfg, "mytool", "packaging/npm/mytool")
	if err != nil {
		t.Fatalf("generateLauncher() failed: %v", err)
	}

	launcherStr := string(launcher)

	// For windows platform, expect .exe extension
	if !strings.Contains(launcherStr, ".exe") {
		t.Error("Launcher missing .exe extension for Windows binary")
	}
}

// TestDefaultPackageName verifies default package name uses project name.
func TestDefaultPackageName(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Packages: model.Packages{
			NPM: model.NPM{
				Enabled: true,
				// PackageName not set, should default to project name
			},
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "linux", Arch: "amd64"},
			},
			Archive: model.Archive{
				NameTemplate:     "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathInArchive: "{{name}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
				},
			},
		},
	}

	ctx := makeContext(cfg, "1.0.0")
	gen := &Generator{}
	outputs, err := gen.Generate(ctx)

	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Should create package in packaging/npm/mytool (default name)
	foundDefault := false
	for _, output := range outputs {
		if strings.HasPrefix(output.Path, "packaging/npm/mytool/") {
			foundDefault = true
			break
		}
	}

	if !foundDefault {
		t.Error("Generator did not use default package name (project name)")
	}
}

// Helper function to create a Context for testing.
func makeContext(cfg *model.Config, version string) generator.Context {
	return generator.Context{
		Config:      cfg,
		ProjectRoot: "/test/project",
		Version:     version,
		ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
			params := naming.ArchiveNameParams{
				Name:                cfg.Project.Name,
				Version:             version,
				OS:                  os,
				Arch:                arch,
				Format:              cfg.Release.Archive.Format.Default,
				ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
				BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
			}
			if os == "windows" && cfg.Release.Archive.Format.Windows != "" {
				params.Format = cfg.Release.Archive.Format.Windows
			}
			return naming.ArchiveName(params)
		},
		RenderTemplate: func(template, fieldPath string) (string, error) {
			renderer := &templatex.Renderer{
				Data: map[string]string{
					"name":    cfg.Project.Name,
					"version": version,
				},
			}
			return renderer.RenderWithFieldPath(template, fieldPath)
		},
	}
}
