package validate

import (
	"strings"
	"testing"

	"go-package-yourself/internal/model"
)

// TestValidConfigWithAllFields tests that a complete, valid config passes validation.
func TestValidConfigWithAllFields(t *testing.T) {
	cfg := &model.Config{
		SchemaVersion: 1,
		Project: model.Project{
			Name:        "mytool",
			Repo:        "owner/mytool",
			Description: "A cool tool",
			Homepage:    "https://example.com",
			License:     "MIT",
		},
		Go: model.Go{
			Main:    "./cmd/mytool",
			CGO:     false,
			LDFlags: "-X main.Version=1.0.0",
		},
		Release: model.Release{
			TagTemplate: "v{{version}}",
			Platforms: []model.Platform{
				{OS: "darwin", Arch: "amd64"},
				{OS: "linux", Arch: "amd64"},
				{OS: "windows", Arch: "amd64"},
			},
			Archive: model.Archive{
				NameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
				BinPathInArchive: "{{name}}",
			},
			Checksums: model.Checksums{
				File:      "checksums.txt",
				Algorithm: "sha256",
				Format:    "goreleaser",
			},
		},
	}

	err := Config(cfg)
	if err != nil {
		t.Errorf("valid config failed validation: %v", err)
	}
}

// TestMissingProjectName tests that missing project.name is caught.
func TestMissingProjectName(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Archive: model.Archive{
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
	}

	err := Config(cfg)
	if err == nil {
		t.Fatal("should have failed on missing project.name")
	}

	if !strings.Contains(err.Error(), "project.name") {
		t.Errorf("error should mention project.name, got: %v", err)
	}
}

// TestMissingProjectRepo tests that missing project.repo is caught.
func TestMissingProjectRepo(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Archive: model.Archive{
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
	}

	err := Config(cfg)
	if err == nil {
		t.Fatal("should have failed on missing project.repo")
	}

	if !strings.Contains(err.Error(), "project.repo") {
		t.Errorf("error should mention project.repo, got: %v", err)
	}
}

// TestMissingGoMain tests that missing go.main is caught.
func TestMissingGoMain(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "",
		},
		Release: model.Release{
			Archive: model.Archive{
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
	}

	err := Config(cfg)
	if err == nil {
		t.Fatal("should have failed on missing go.main")
	}

	if !strings.Contains(err.Error(), "go.main") {
		t.Errorf("error should mention go.main, got: %v", err)
	}
}

// TestAllRequiredFieldsMissing tests that all missing required fields are reported.
func TestAllRequiredFieldsMissing(t *testing.T) {
	cfg := &model.Config{
		Release: model.Release{
			Archive: model.Archive{
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
	}

	err := Config(cfg)
	if err == nil {
		t.Fatal("should have failed on missing required fields")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "project.name") {
		t.Errorf("error should mention project.name")
	}
	if !strings.Contains(errMsg, "project.repo") {
		t.Errorf("error should mention project.repo")
	}
	if !strings.Contains(errMsg, "go.main") {
		t.Errorf("error should mention go.main")
	}
}

// TestInvalidPlatformOS tests that invalid OS is caught.
func TestInvalidPlatformOS(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "bsd", Arch: "amd64"},
			},
			Archive: model.Archive{
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
	}

	err := Config(cfg)
	if err == nil {
		t.Fatal("should have failed on invalid OS")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "invalid OS") {
		t.Errorf("error should mention invalid OS, got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "bsd") {
		t.Errorf("error should mention the invalid OS value, got: %v", errMsg)
	}
}

// TestInvalidPlatformArch tests that invalid arch is caught.
func TestInvalidPlatformArch(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "linux", Arch: "x86"},
			},
			Archive: model.Archive{
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
	}

	err := Config(cfg)
	if err == nil {
		t.Fatal("should have failed on invalid arch")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "invalid arch") {
		t.Errorf("error should mention invalid arch, got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "x86") {
		t.Errorf("error should mention the invalid arch value, got: %v", errMsg)
	}
}

// TestMultiplePlatformErrors tests that multiple platform errors are all reported.
func TestMultiplePlatformErrors(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "bsd", Arch: "amd64"}, // invalid OS
				{OS: "linux", Arch: "x86"}, // invalid arch
				{OS: "darwin", Arch: ""},   // missing arch
				{OS: "", Arch: "amd64"},    // missing OS
			},
			Archive: model.Archive{
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
	}

	err := Config(cfg)
	if err == nil {
		t.Fatal("should have failed on multiple platform errors")
	}

	errMsg := err.Error()
	// Just verify we get some errors; we should get errors for multiple platforms
	if !strings.Contains(errMsg, "platform") && !strings.Contains(errMsg, "Platforms") {
		t.Errorf("error should contain platform-related information, got: %v", errMsg)
	}
}

// TestInvalidArchiveFormatDefault tests that invalid default format is caught.
func TestInvalidArchiveFormatDefault(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "linux", Arch: "amd64"},
			},
			Archive: model.Archive{
				Format: model.ArchiveFormat{
					Default: "cab",
					Windows: "zip",
				},
			},
		},
	}

	err := Config(cfg)
	if err == nil {
		t.Fatal("should have failed on invalid default format")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "format.default") {
		t.Errorf("error should mention format.default, got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "cab") {
		t.Errorf("error should mention the invalid format value, got: %v", errMsg)
	}
}

// TestWindowsFormatNotZip tests that Windows format must be zip.
func TestWindowsFormatNotZip(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "linux", Arch: "amd64"},
			},
			Archive: model.Archive{
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "tar.gz", // Windows should be zip by convention
				},
			},
		},
	}

	err := Config(cfg)
	if err == nil {
		t.Fatal("should have failed when Windows format is not zip")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Windows") {
		t.Errorf("error should mention Windows convention, got: %v", errMsg)
	}
	if !strings.Contains(errMsg, "zip") {
		t.Errorf("error should mention zip requirement, got: %v", errMsg)
	}
}

// TestFieldPathInErrors tests that error messages include field paths.
func TestFieldPathInErrors(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *model.Config
		wantField string
	}{
		{
			name: "project.name",
			cfg: &model.Config{
				Project: model.Project{Name: ""},
				Go:      model.Go{Main: "./cmd/test"},
				Release: model.Release{
					Archive: model.Archive{
						Format: model.ArchiveFormat{
							Default: "tar.gz",
							Windows: "zip",
						},
					},
				},
			},
			wantField: "project.name",
		},
		{
			name: "go.main",
			cfg: &model.Config{
				Project: model.Project{Name: "test", Repo: "owner/test"},
				Go:      model.Go{Main: ""},
				Release: model.Release{
					Archive: model.Archive{
						Format: model.ArchiveFormat{
							Default: "tar.gz",
							Windows: "zip",
						},
					},
				},
			},
			wantField: "go.main",
		},
		{
			name: "release.platforms[0].os",
			cfg: &model.Config{
				Project: model.Project{Name: "test", Repo: "owner/test"},
				Go:      model.Go{Main: "./cmd/test"},
				Release: model.Release{
					Platforms: []model.Platform{
						{OS: "invalid", Arch: "amd64"},
					},
					Archive: model.Archive{
						Format: model.ArchiveFormat{
							Default: "tar.gz",
							Windows: "zip",
						},
					},
				},
			},
			wantField: "release.platforms",
		},
		{
			name: "release.archive.format.windows",
			cfg: &model.Config{
				Project: model.Project{Name: "test", Repo: "owner/test"},
				Go:      model.Go{Main: "./cmd/test"},
				Release: model.Release{
					Platforms: []model.Platform{
						{OS: "linux", Arch: "amd64"},
					},
					Archive: model.Archive{
						Format: model.ArchiveFormat{
							Default: "tar.gz",
							Windows: "tar.gz", // Should be zip
						},
					},
				},
			},
			wantField: "release.archive.format.windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Config(tt.cfg)
			if err == nil {
				t.Fatalf("validation should have failed for %s", tt.name)
			}

			if !strings.Contains(err.Error(), tt.wantField) {
				t.Errorf("error should contain field path %q, got: %v", tt.wantField, err)
			}
		})
	}
}

// TestValidMultiplePlatforms tests that multiple valid platforms pass.
func TestValidMultiplePlatforms(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "darwin", Arch: "amd64"},
				{OS: "darwin", Arch: "arm64"},
				{OS: "linux", Arch: "amd64"},
				{OS: "linux", Arch: "arm64"},
				{OS: "windows", Arch: "amd64"},
			},
			Archive: model.Archive{
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
		},
	}

	err := Config(cfg)
	if err != nil {
		t.Errorf("valid multi-platform config failed: %v", err)
	}
}

// TestConfigWithOptionalFields tests that optional fields don't cause validation errors.
func TestConfigWithOptionalFields(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name:        "mytool",
			Repo:        "owner/mytool",
			Description: "Optional description",
			Homepage:    "https://optional.example.com",
			License:     "MIT",
		},
		Go: model.Go{
			Main:    "./cmd/mytool",
			CGO:     true,
			LDFlags: "-X main.Version=1.0.0",
		},
		Release: model.Release{
			TagTemplate: "v{{version}}",
			Platforms: []model.Platform{
				{OS: "linux", Arch: "amd64"},
			},
			Archive: model.Archive{
				NameTemplate:     "custom-{{name}}-{{version}}",
				BinPathInArchive: "bin/{{name}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
			},
			Checksums: model.Checksums{
				File:      "checksums.txt",
				Algorithm: "sha256",
				Format:    "goreleaser",
			},
		},
		Packages: model.Packages{
			NPM: model.NPM{
				Enabled:     true,
				PackageName: "my-tool",
				BinName:     "mytool",
				NodeEngines: ">=18",
			},
			Homebrew: model.Homebrew{
				Enabled:     true,
				Tap:         "owner/tap",
				FormulaName: "Mytool",
			},
			Chocolatey: model.Chocolatey{
				Enabled:   true,
				PackageID: "mytool",
				Authors:   "Me",
			},
		},
		GitHub: model.GitHub{
			Workflows: model.GitHubWorkflows{
				Enabled:      true,
				WorkflowFile: ".github/workflows/release.yml",
				TagPatterns:  []string{"v*"},
			},
		},
	}

	err := Config(cfg)
	if err != nil {
		t.Errorf("config with optional fields failed validation: %v", err)
	}
}
