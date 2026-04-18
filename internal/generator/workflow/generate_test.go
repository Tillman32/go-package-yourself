package workflow

import (
	"strings"
	"testing"

	"go-package-yourself/internal/generator"
	"go-package-yourself/internal/model"
	"go-package-yourself/internal/naming"

	"gopkg.in/yaml.v3"
)

// TestWorkflowGeneratorName tests the Name method.
func TestWorkflowGeneratorName(t *testing.T) {
	gen := New()
	if gen.Name() != "workflow" {
		t.Errorf("Name() = %q, want %q", gen.Name(), "workflow")
	}
}

// TestMatrixGeneration tests that matrix entries are correctly generated for all platforms.
func TestMatrixGeneration(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
		},
		Release: model.Release{
			Archive: model.Archive{
				NameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
				BinPathInArchive: "{{name}}",
			},
		},
	}

	ctx := &generator.Context{
		Config: cfg,
		ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
			params := naming.ArchiveNameParams{
				Name:                cfg.Project.Name,
				Version:             "1.0.0",
				OS:                  os,
				Arch:                arch,
				Format:              "tar.gz",
				ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
				BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
			}
			if os == "windows" {
				params.Format = "zip"
			}
			return naming.ArchiveName(params)
		},
	}

	// Test with specific platform
	cfg.Release.Platforms = []model.Platform{
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "linux", Arch: "amd64"},
		{OS: "windows", Arch: "amd64"},
	}

	entries, err := buildMatrixEntries(*ctx)
	if err != nil {
		t.Fatalf("buildMatrixEntries failed: %v", err)
	}

	if len(entries) != 4 {
		t.Errorf("expected 4 entries, got %d", len(entries))
	}

	// Verify platform mapping
	tests := []struct {
		index   int
		os      string
		arch    string
		runsOn  string
		extWant string
	}{
		{0, "darwin", "amd64", "macos-latest", ".tar.gz"},
		{1, "darwin", "arm64", "macos-latest", ".tar.gz"},
		{2, "linux", "amd64", "ubuntu-latest", ".tar.gz"},
		{3, "windows", "amd64", "windows-latest", ".zip"},
	}

	for _, test := range tests {
		entry := entries[test.index]

		if entry.OS != test.os {
			t.Errorf("entries[%d].OS = %q, want %q", test.index, entry.OS, test.os)
		}
		if entry.Arch != test.arch {
			t.Errorf("entries[%d].Arch = %q, want %q", test.index, entry.Arch, test.arch)
		}
		if entry.RunsOn != test.runsOn {
			t.Errorf("entries[%d].RunsOn = %q, want %q", test.index, entry.RunsOn, test.runsOn)
		}
		if entry.Ext != test.extWant {
			t.Errorf("entries[%d].Ext = %q, want %q", test.index, entry.Ext, test.extWant)
		}

		// Verify archive names match expected format
		expectedPrefix := "mytool_1.0.0_" + test.os + "_" + test.arch
		if !strings.HasPrefix(entry.Archive, expectedPrefix) {
			t.Errorf("entries[%d].Archive = %q, want prefix %q", test.index, entry.Archive, expectedPrefix)
		}
	}
}

// TestMatrixArchiveNames tests that archive filenames are computed correctly.
func TestMatrixArchiveNames(t *testing.T) {
	tests := []struct {
		name string
		os   string
		arch string
		want string
	}{
		{"darwin-amd64", "darwin", "amd64", "tool_1.0.0_darwin_amd64.tar.gz"},
		{"darwin-arm64", "darwin", "arm64", "tool_1.0.0_darwin_arm64.tar.gz"},
		{"linux-amd64", "linux", "amd64", "tool_1.0.0_linux_amd64.tar.gz"},
		{"windows-amd64", "windows", "amd64", "tool_1.0.0_windows_amd64.zip"},
	}

	for _, test := range tests {
		cfg := &model.Config{
			Project: model.Project{Name: "tool"},
			Release: model.Release{
				Archive: model.Archive{
					NameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
					Format: model.ArchiveFormat{
						Default: "tar.gz",
						Windows: "zip",
					},
					BinPathInArchive: "{{name}}",
				},
			},
		}

		ctx := &generator.Context{
			Config:  cfg,
			Version: "1.0.0",
			ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
				params := naming.ArchiveNameParams{
					Name:                cfg.Project.Name,
					Version:             "1.0.0",
					OS:                  os,
					Arch:                arch,
					Format:              "tar.gz",
					ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
					BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
				}
				if os == "windows" {
					params.Format = "zip"
				}
				return naming.ArchiveName(params)
			},
		}

		archFile, _, err := ctx.ArchiveName(test.os, test.arch)
		if err != nil {
			t.Errorf("%s: ArchiveName failed: %v", test.name, err)
			continue
		}

		if archFile != test.want {
			t.Errorf("%s: got %q, want %q", test.name, archFile, test.want)
		}
	}
}

// TestYAMLSyntaxValid tests that generated YAML is syntactically valid.
func TestYAMLSyntaxValid(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main:    "./cmd/mytool",
			LDFlags: "-s -w",
		},
		Release: model.Release{
			TagTemplate: "v{{version}}",
			Platforms: []model.Platform{
				{OS: "darwin", Arch: "amd64"},
				{OS: "linux", Arch: "amd64"},
			},
			Archive: model.Archive{
				NameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
				},
				BinPathInArchive: "{{name}}",
			},
			Checksums: model.Checksums{
				File:      "checksums.txt",
				Algorithm: "sha256",
				Format:    "goreleaser",
			},
		},
		GitHub: model.GitHub{
			Workflows: model.GitHubWorkflows{
				WorkflowFile: ".github/workflows/gpy-release.yaml",
				Enabled:      true,
				TagPatterns:  []string{"v*"},
			},
		},
	}

	ctx := generator.Context{
		Config:  cfg,
		Version: "",
		ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
			params := naming.ArchiveNameParams{
				Name:                cfg.Project.Name,
				Version:             "1.0.0",
				OS:                  os,
				Arch:                arch,
				Format:              "tar.gz",
				ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
				BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
			}
			return naming.ArchiveName(params)
		},
	}

	gen := New()
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(outputs) != 1 {
		t.Fatalf("expected 1 FileOutput, got %d", len(outputs))
	}

	// Parse YAML to verify syntax
	var doc interface{}
	if err := yaml.Unmarshal(outputs[0].Content, &doc); err != nil {
		t.Fatalf("YAML syntax error: %v\nContent:\n%s", err, string(outputs[0].Content))
	}

	// Verify it's a map with required keys
	m, ok := doc.(map[string]interface{})
	if !ok {
		t.Fatalf("YAML root is not a map")
	}

	if _, ok := m["name"]; !ok {
		t.Errorf("YAML missing 'name' key")
	}
	if _, ok := m["on"]; !ok {
		t.Errorf("YAML missing 'on' key")
	}
	if _, ok := m["jobs"]; !ok {
		t.Errorf("YAML missing 'jobs' key")
	}
}

// TestWorkflowHasAllPlatforms tests that workflow includes all configured platforms.
func TestWorkflowHasAllPlatforms(t *testing.T) {
	platforms := []model.Platform{
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "windows", Arch: "amd64"},
	}

	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Platforms: platforms,
			Archive: model.Archive{
				NameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
				BinPathInArchive: "{{name}}",
			},
			Checksums: model.Checksums{
				File: "checksums.txt",
			},
		},
		GitHub: model.GitHub{
			Workflows: model.GitHubWorkflows{
				WorkflowFile: ".github/workflows/gpy-release.yaml",
				Enabled:      true,
				TagPatterns:  []string{"v*"},
			},
		},
	}

	ctx := generator.Context{
		Config: cfg,
		ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
			params := naming.ArchiveNameParams{
				Name:                cfg.Project.Name,
				Version:             "",
				OS:                  os,
				Arch:                arch,
				Format:              "tar.gz",
				ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
				BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
			}
			if os == "windows" {
				params.Format = "zip"
			}
			return naming.ArchiveName(params)
		},
	}

	gen := New()
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc WorkflowDoc
	if err := yaml.Unmarshal(outputs[0].Content[len("# GitHub Actions workflow for releasing binaries\n# Generated by gpy (go-package-yourself)\n# DO NOT EDIT - regenerate with: gpy workflow --write\n\n"):], &doc); err == nil || true {
		// Try to unmarshal the full content
		if err := yaml.Unmarshal(outputs[0].Content, &doc); err != nil {
			t.Fatalf("Failed to unmarshal YAML: %v", err)
		}
	}

	// Verify matrix has entries for all platforms
	buildJob, ok := doc.Jobs["build"]
	if !ok {
		t.Fatalf("'build' job not found in workflow")
	}

	if buildJob.Strategy == nil {
		t.Fatalf("'build' job has no strategy")
	}

	if len(buildJob.Strategy.Matrix.Include) != len(platforms) {
		t.Errorf("expected %d matrix entries, got %d", len(platforms), len(buildJob.Strategy.Matrix.Include))
	}
}

// TestErrorHandling_NoPlatforms tests error when no platforms configured.
func TestErrorHandling_NoPlatforms(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{Name: "mytool"},
		Go:      model.Go{Main: "./cmd/mytool"},
		Release: model.Release{
			Platforms: []model.Platform{}, // Empty
		},
	}

	ctx := generator.Context{Config: cfg}

	gen := New()
	_, err := gen.Generate(ctx)

	if err == nil {
		t.Fatalf("expected error for no platforms, got nil")
	}

	if !strings.Contains(err.Error(), "no platforms") {
		t.Errorf("error message should mention no platforms: %v", err)
	}
}

// TestDeterministicOutput tests that output is stable across runs.
func TestDeterministicOutput(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
		},
		Go: model.Go{
			Main:    "./cmd/mytool",
			LDFlags: "-s -w",
		},
		Release: model.Release{
			Platforms: []model.Platform{
				{OS: "linux", Arch: "amd64"},
				{OS: "darwin", Arch: "arm64"},
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
				File: "checksums.txt",
			},
		},
		GitHub: model.GitHub{
			Workflows: model.GitHubWorkflows{
				WorkflowFile: ".github/workflows/gpy-release.yaml",
				Enabled:      true,
				TagPatterns:  []string{"v*"},
			},
		},
	}

	newContext := func() generator.Context {
		return generator.Context{
			Config: cfg,
			ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
				params := naming.ArchiveNameParams{
					Name:                cfg.Project.Name,
					Version:             "",
					OS:                  os,
					Arch:                arch,
					Format:              "tar.gz",
					ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
					BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
				}
				if os == "windows" {
					params.Format = "zip"
				}
				return naming.ArchiveName(params)
			},
		}
	}

	gen := New()

	// Generate multiple times and compare
	var outputs1, outputs2, outputs3 []generator.FileOutput
	var err error

	outputs1, err = gen.Generate(newContext())
	if err != nil {
		t.Fatalf("First generation failed: %v", err)
	}

	outputs2, err = gen.Generate(newContext())
	if err != nil {
		t.Fatalf("Second generation failed: %v", err)
	}

	outputs3, err = gen.Generate(newContext())
	if err != nil {
		t.Fatalf("Third generation failed: %v", err)
	}

	// Compare content
	if string(outputs1[0].Content) != string(outputs2[0].Content) {
		t.Errorf("outputs differ between first and second generation")
	}

	if string(outputs2[0].Content) != string(outputs3[0].Content) {
		t.Errorf("outputs differ between second and third generation")
	}
}

// TestRunnerMapping tests runner selection for each OS.
func TestRunnerMapping(t *testing.T) {
	tests := []struct {
		os         string
		wantRunner string
	}{
		{"darwin", "macos-latest"},
		{"linux", "ubuntu-latest"},
		{"windows", "windows-latest"},
	}

	for _, tt := range tests {
		got := runnerFor(tt.os)
		if got != tt.wantRunner {
			t.Errorf("runnerFor(%q) = %q, want %q", tt.os, got, tt.wantRunner)
		}
	}
}

// TestTagPatterns tests that tag patterns are correctly used in workflow trigger.
func TestTagPatterns(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		want     []string
	}{
		{"default", nil, []string{"v*"}},
		{"custom", []string{"release-*", "v*"}, []string{"release-*", "v*"}},
		{"empty-uses-default", []string{}, []string{"v*"}},
	}

	for _, test := range tests {
		cfg := &model.Config{
			Project: model.Project{Name: "mytool"},
			Go:      model.Go{Main: "./cmd/mytool"},
			Release: model.Release{
				Platforms: []model.Platform{{OS: "linux", Arch: "amd64"}},
				Archive: model.Archive{
					NameTemplate:     "{{name}}_{{version}}_{{os}}_{{arch}}",
					Format:           model.ArchiveFormat{Default: "tar.gz"},
					BinPathInArchive: "{{name}}",
				},
			},
			GitHub: model.GitHub{
				Workflows: model.GitHubWorkflows{
					WorkflowFile: ".github/workflows/gpy-release.yaml",
					TagPatterns:  test.patterns,
				},
			},
		}

		ctx := generator.Context{
			Config: cfg,
			ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
				params := naming.ArchiveNameParams{
					Name:                cfg.Project.Name,
					Version:             "",
					OS:                  os,
					Arch:                arch,
					Format:              "tar.gz",
					ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
					BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
				}
				return naming.ArchiveName(params)
			},
		}

		gen := New()
		outputs, err := gen.Generate(ctx)
		if err != nil {
			t.Fatalf("%s: Generate failed: %v", test.name, err)
		}

		var doc WorkflowDoc
		if err := yaml.Unmarshal(outputs[0].Content, &doc); err != nil {
			t.Fatalf("%s: Unmarshal failed: %v", test.name, err)
		}

		if len(doc.On.Push.Tags) != len(test.want) {
			t.Errorf("%s: got %d tags, want %d", test.name, len(doc.On.Push.Tags), len(test.want))
		}

		for i, tag := range doc.On.Push.Tags {
			if i < len(test.want) && tag != test.want[i] {
				t.Errorf("%s: tags[%d] = %q, want %q", test.name, i, tag, test.want[i])
			}
		}
	}
}

func TestReleaseJobOnlyRunsOnTagPushes(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{Name: "mytool"},
		Go:      model.Go{Main: "./cmd/mytool"},
		Release: model.Release{
			Platforms: []model.Platform{{OS: "linux", Arch: "amd64"}},
			Archive: model.Archive{
				NameTemplate:     "{{name}}_{{version}}_{{os}}_{{arch}}",
				Format:           model.ArchiveFormat{Default: "tar.gz"},
				BinPathInArchive: "{{name}}",
			},
		},
		GitHub: model.GitHub{
			Workflows: model.GitHubWorkflows{
				WorkflowFile: ".github/workflows/gpy-release.yaml",
				TagPatterns:  []string{"v*"},
			},
		},
	}

	ctx := generator.Context{
		Config: cfg,
		ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
			params := naming.ArchiveNameParams{
				Name:                cfg.Project.Name,
				Version:             "",
				OS:                  os,
				Arch:                arch,
				Format:              "tar.gz",
				ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
				BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
			}
			return naming.ArchiveName(params)
		},
	}

	outputs, err := New().Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var doc WorkflowDoc
	if err := yaml.Unmarshal(outputs[0].Content, &doc); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	releaseJob, ok := doc.Jobs["release"]
	if !ok {
		t.Fatalf("release job not found")
	}

	if releaseJob.If != "github.event_name == 'push' && startsWith(github.ref, 'refs/tags/')" {
		t.Fatalf("release job if = %q", releaseJob.If)
	}

	if releaseJob.Permissions != nil {
		t.Fatalf("release job permissions = %#v, want nil", releaseJob.Permissions)
	}
}

// TestLDFlagsHandling tests that ldflags are correctly included in build command.
func TestLDFlagsHandling(t *testing.T) {
	tests := []struct {
		name      string
		ldflags   string
		wantInRun bool
	}{
		{"with-ldflags", "-s -w", true},
		{"without-ldflags", "", true}, // Version is always added
		{"complex-ldflags", "-s -w -X main.BuildTime=$(date)", true},
	}

	for _, test := range tests {
		cfg := &model.Config{
			Project: model.Project{Name: "mytool"},
			Go: model.Go{
				Main:    "./cmd/mytool",
				LDFlags: test.ldflags,
			},
			Release: model.Release{
				Platforms: []model.Platform{{OS: "linux", Arch: "amd64"}},
				Archive: model.Archive{
					NameTemplate:     "{{name}}_{{version}}_{{os}}_{{arch}}",
					Format:           model.ArchiveFormat{Default: "tar.gz"},
					BinPathInArchive: "{{name}}",
				},
			},
			GitHub: model.GitHub{
				Workflows: model.GitHubWorkflows{
					WorkflowFile: ".github/workflows/gpy-release.yaml",
					TagPatterns:  []string{"v*"},
				},
			},
		}

		ctx := generator.Context{
			Config: cfg,
			ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
				params := naming.ArchiveNameParams{
					Name:                cfg.Project.Name,
					Version:             "",
					OS:                  os,
					Arch:                arch,
					Format:              "tar.gz",
					ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
					BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
				}
				return naming.ArchiveName(params)
			},
		}

		gen := New()
		outputs, err := gen.Generate(ctx)
		if err != nil {
			t.Fatalf("%s: Generate failed: %v", test.name, err)
		}

		var doc WorkflowDoc
		if err := yaml.Unmarshal(outputs[0].Content, &doc); err != nil {
			t.Fatalf("%s: Unmarshal failed: %v", test.name, err)
		}

		// Find build step
		var buildStep *WorkflowStep
		for i := range doc.Jobs["build"].Steps {
			if doc.Jobs["build"].Steps[i].Name == "Build binary" {
				buildStep = &doc.Jobs["build"].Steps[i]
				break
			}
		}

		if buildStep == nil {
			t.Fatalf("%s: Build step not found", test.name)
		}

		if test.wantInRun {
			if !strings.Contains(buildStep.Run, "-ldflags") {
				t.Errorf("%s: missing -ldflags in build step: %s", test.name, buildStep.Run)
			}
			if !strings.Contains(buildStep.Run, "main.Version") {
				t.Errorf("%s: missing main.Version in ldflags: %s", test.name, buildStep.Run)
			}
		}
	}
}

// TestFileOutput tests that FileOutput is correctly structured.
func TestFileOutput(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{Name: "mytool"},
		Go:      model.Go{Main: "./cmd/mytool"},
		Release: model.Release{
			Platforms: []model.Platform{{OS: "linux", Arch: "amd64"}},
			Archive: model.Archive{
				NameTemplate:     "{{name}}_{{version}}_{{os}}_{{arch}}",
				Format:           model.ArchiveFormat{Default: "tar.gz"},
				BinPathInArchive: "{{name}}",
			},
		},
		GitHub: model.GitHub{
			Workflows: model.GitHubWorkflows{
				WorkflowFile: ".github/workflows/gpy-release.yaml",
			},
		},
	}

	ctx := generator.Context{
		Config: cfg,
		ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
			params := naming.ArchiveNameParams{
				Name:                cfg.Project.Name,
				Version:             "",
				OS:                  os,
				Arch:                arch,
				Format:              "tar.gz",
				ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
				BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
			}
			return naming.ArchiveName(params)
		},
	}

	gen := New()
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(outputs) != 1 {
		t.Errorf("expected 1 FileOutput, got %d", len(outputs))
	}

	output := outputs[0]

	if output.Path != ".github/workflows/gpy-release.yaml" {
		t.Errorf("Path = %q, want %q", output.Path, ".github/workflows/package-release.yaml")
	}

	if output.Mode != 0o644 {
		t.Errorf("Mode = %o, want %o", output.Mode, 0o644)
	}

	if len(output.Content) == 0 {
		t.Errorf("Content is empty")
	}
}

// TestWorkflow_DockerJobPresent tests that the Docker job is included when docker.enabled is true.
func TestWorkflow_DockerJobPresent(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Archive: model.Archive{
				NameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
				BinPathInArchive: "{{name}}",
			},
			Platforms: []model.Platform{
				{OS: "linux", Arch: "amd64"},
			},
		},
		Packages: model.Packages{
			Docker: model.Docker{
				Enabled:   true,
				ImageName: "mytool",
			},
		},
		GitHub: model.GitHub{
			Workflows: model.GitHubWorkflows{
				WorkflowFile: ".github/workflows/gpy-release.yaml",
				TagPatterns:  []string{"v*"},
			},
		},
	}

	ctx := &generator.Context{
		Config: cfg,
		ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
			params := naming.ArchiveNameParams{
				Name:                cfg.Project.Name,
				Version:             "1.0.0",
				OS:                  os,
				Arch:                arch,
				Format:              "tar.gz",
				ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
				BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
			}
			return naming.ArchiveName(params)
		},
	}

	gen := New()
	outputs, err := gen.Generate(*ctx)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if len(outputs) != 1 {
		t.Errorf("expected 1 FileOutput, got %d", len(outputs))
	}

	content := outputs[0].Content
	var workflow WorkflowDoc
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("Failed to unmarshal workflow YAML: %v", err)
	}

	// Check that both build and publish-docker jobs exist
	if _, hasPublishDocker := workflow.Jobs["publish-docker"]; !hasPublishDocker {
		t.Errorf("Jobs map is missing %q key", "publish-docker")
	}

	if _, hasBuild := workflow.Jobs["build"]; !hasBuild {
		t.Errorf("Jobs map is missing %q key", "build")
	}

	// Check that publish-docker job has "build" in needs
	publishDockerJob := workflow.Jobs["publish-docker"]
	if len(publishDockerJob.Needs) != 1 || publishDockerJob.Needs[0] != "build" {
		t.Errorf("publish-docker job Needs = %v, want [\"build\"]", publishDockerJob.Needs)
	}
}

// TestWorkflow_DockerJobAbsent tests that the Docker job is excluded when docker.enabled is false.
func TestWorkflow_DockerJobAbsent(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Archive: model.Archive{
				NameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
				BinPathInArchive: "{{name}}",
			},
			Platforms: []model.Platform{
				{OS: "linux", Arch: "amd64"},
			},
		},
		Packages: model.Packages{
			Docker: model.Docker{
				Enabled:   false,
				ImageName: "mytool",
			},
		},
		GitHub: model.GitHub{
			Workflows: model.GitHubWorkflows{
				WorkflowFile: ".github/workflows/gpy-release.yaml",
				TagPatterns:  []string{"v*"},
			},
		},
	}

	ctx := &generator.Context{
		Config: cfg,
		ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
			params := naming.ArchiveNameParams{
				Name:                cfg.Project.Name,
				Version:             "1.0.0",
				OS:                  os,
				Arch:                arch,
				Format:              "tar.gz",
				ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
				BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
			}
			return naming.ArchiveName(params)
		},
	}

	gen := New()
	outputs, err := gen.Generate(*ctx)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if len(outputs) != 1 {
		t.Errorf("expected 1 FileOutput, got %d", len(outputs))
	}

	content := outputs[0].Content
	var workflow WorkflowDoc
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("Failed to unmarshal workflow YAML: %v", err)
	}

	// Check that publish-docker job does NOT exist when disabled
	if _, hasPublishDocker := workflow.Jobs["publish-docker"]; hasPublishDocker {
		t.Errorf("workflow.Jobs should not contain %q key when docker is disabled", "publish-docker")
	}

	// Check that build and release jobs exist (2 jobs when no packaging is enabled)
	if len(workflow.Jobs) != 2 {
		t.Errorf("expected 2 jobs in workflow (build + release), got %d", len(workflow.Jobs))
	}

	if _, hasBuild := workflow.Jobs["build"]; !hasBuild {
		t.Errorf("Jobs map is missing %q key", "build")
	}

	if _, hasRelease := workflow.Jobs["release"]; !hasRelease {
		t.Errorf("Jobs map is missing %q key", "release")
	}
}

// TestWorkflow_NpmJobPresent tests that the npm job is included when enabled.
func TestWorkflow_NpmJobPresent(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Archive: model.Archive{
				NameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
				BinPathInArchive: "{{name}}",
			},
			Platforms: []model.Platform{
				{OS: "linux", Arch: "amd64"},
			},
		},
		Packages: model.Packages{
			NPM: model.NPM{
				Enabled:     true,
				PackageName: "mytool-launcher",
			},
		},
		GitHub: model.GitHub{
			Workflows: model.GitHubWorkflows{
				WorkflowFile: ".github/workflows/gpy-release.yaml",
				TagPatterns:  []string{"v*"},
			},
		},
	}

	ctx := &generator.Context{
		Config: cfg,
		ArchiveName: func(os, arch string) (string, string, error) {
			params := naming.ArchiveNameParams{
				Name:                cfg.Project.Name,
				Version:             "1.0.0",
				OS:                  os,
				Arch:                arch,
				Format:              "tar.gz",
				ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
				BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
			}
			return naming.ArchiveName(params)
		},
	}

	gen := New()
	outputs, err := gen.Generate(*ctx)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	content := outputs[0].Content
	var workflow WorkflowDoc
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("Failed to unmarshal workflow YAML: %v", err)
	}

	if _, hasPublishNpm := workflow.Jobs["publish-npm"]; !hasPublishNpm {
		t.Errorf("Jobs map is missing %q key", "publish-npm")
	}

	publishNpmJob := workflow.Jobs["publish-npm"]
	if len(publishNpmJob.Needs) != 1 || publishNpmJob.Needs[0] != "build" {
		t.Errorf("publish-npm job Needs = %v, want [\"build\"]", publishNpmJob.Needs)
	}

	if publishNpmJob.RunsOn != "ubuntu-latest" {
		t.Errorf("publish-npm job RunsOn = %q, want %q", publishNpmJob.RunsOn, "ubuntu-latest")
	}

	if len(publishNpmJob.Steps) == 0 {
		t.Fatalf("publish-npm job has no steps")
	}

	run := publishNpmJob.Steps[len(publishNpmJob.Steps)-1].Run
	if !strings.Contains(run, "cd \"packaging/npm/mytool-launcher\"") {
		t.Errorf("publish-npm run step = %q, want committed packaging path", run)
	}
	if strings.Contains(run, "go run ./cmd/gpy") {
		t.Errorf("publish-npm run step should not regenerate packaging in CI: %q", run)
	}
}

// TestWorkflow_HomebrewJobPresent tests that the homebrew job is included when enabled.
func TestWorkflow_HomebrewJobPresent(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Archive: model.Archive{
				NameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
				BinPathInArchive: "{{name}}",
			},
			Platforms: []model.Platform{
				{OS: "darwin", Arch: "amd64"},
				{OS: "linux", Arch: "amd64"},
			},
		},
		Packages: model.Packages{
			Homebrew: model.Homebrew{
				Enabled:     true,
				FormulaName: "Mytool",
			},
		},
		GitHub: model.GitHub{
			Workflows: model.GitHubWorkflows{
				WorkflowFile: ".github/workflows/gpy-release.yaml",
				TagPatterns:  []string{"v*"},
			},
		},
	}

	ctx := &generator.Context{
		Config: cfg,
		ArchiveName: func(os, arch string) (string, string, error) {
			params := naming.ArchiveNameParams{
				Name:                cfg.Project.Name,
				Version:             "1.0.0",
				OS:                  os,
				Arch:                arch,
				Format:              "tar.gz",
				ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
				BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
			}
			return naming.ArchiveName(params)
		},
	}

	gen := New()
	outputs, err := gen.Generate(*ctx)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	content := outputs[0].Content
	var workflow WorkflowDoc
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("Failed to unmarshal workflow YAML: %v", err)
	}

	if _, hasPublishHomebrew := workflow.Jobs["publish-homebrew"]; !hasPublishHomebrew {
		t.Errorf("Jobs map is missing %q key", "publish-homebrew")
	}

	publishHomebrewJob := workflow.Jobs["publish-homebrew"]
	if len(publishHomebrewJob.Needs) != 1 || publishHomebrewJob.Needs[0] != "build" {
		t.Errorf("publish-homebrew job Needs = %v, want [\"build\"]", publishHomebrewJob.Needs)
	}

	if publishHomebrewJob.RunsOn != "ubuntu-latest" {
		t.Errorf("publish-homebrew job RunsOn = %q, want %q", publishHomebrewJob.RunsOn, "ubuntu-latest")
	}

	if len(publishHomebrewJob.Steps) == 0 {
		t.Fatalf("publish-homebrew job has no steps")
	}

	run := publishHomebrewJob.Steps[len(publishHomebrewJob.Steps)-1].Run
	if !strings.Contains(run, "FORMULA_FILE=\"../packaging/homebrew/Mytool.rb\"") {
		t.Errorf("publish-homebrew run step = %q, want committed formula path", run)
	}
	if strings.Contains(run, "go run ./cmd/gpy") {
		t.Errorf("publish-homebrew run step should not regenerate packaging in CI: %q", run)
	}
}

// TestWorkflow_ChocolateyJobPresent tests that the chocolatey job is included when enabled.
func TestWorkflow_ChocolateyJobPresent(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Release: model.Release{
			Archive: model.Archive{
				NameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
					Windows: "zip",
				},
				BinPathInArchive: "{{name}}",
			},
			Platforms: []model.Platform{
				{OS: "windows", Arch: "amd64"},
			},
		},
		Packages: model.Packages{
			Chocolatey: model.Chocolatey{
				Enabled:   true,
				PackageID: "mytool",
			},
		},
		GitHub: model.GitHub{
			Workflows: model.GitHubWorkflows{
				WorkflowFile: ".github/workflows/gpy-release.yaml",
				TagPatterns:  []string{"v*"},
			},
		},
	}

	ctx := &generator.Context{
		Config: cfg,
		ArchiveName: func(os, arch string) (string, string, error) {
			params := naming.ArchiveNameParams{
				Name:                cfg.Project.Name,
				Version:             "1.0.0",
				OS:                  os,
				Arch:                arch,
				Format:              "zip",
				ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
				BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
			}
			return naming.ArchiveName(params)
		},
	}

	gen := New()
	outputs, err := gen.Generate(*ctx)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	content := outputs[0].Content
	var workflow WorkflowDoc
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		t.Fatalf("Failed to unmarshal workflow YAML: %v", err)
	}

	if _, hasPublishChocolatey := workflow.Jobs["publish-chocolatey"]; !hasPublishChocolatey {
		t.Errorf("Jobs map is missing %q key", "publish-chocolatey")
	}

	publishChocolateyJob := workflow.Jobs["publish-chocolatey"]
	if len(publishChocolateyJob.Needs) != 1 || publishChocolateyJob.Needs[0] != "build" {
		t.Errorf("publish-chocolatey job Needs = %v, want [\"build\"]", publishChocolateyJob.Needs)
	}

	if publishChocolateyJob.RunsOn != "windows-latest" {
		t.Errorf("publish-chocolatey job RunsOn = %q, want %q", publishChocolateyJob.RunsOn, "windows-latest")
	}

	if len(publishChocolateyJob.Steps) < 2 {
		t.Fatalf("publish-chocolatey job has too few steps: %d", len(publishChocolateyJob.Steps))
	}

	packRun := ""
	for _, step := range publishChocolateyJob.Steps {
		if step.Name == "Pack Chocolatey package" {
			packRun = step.Run
			break
		}
	}
	if packRun == "" {
		t.Fatalf("publish-chocolatey job missing pack step")
	}

	if !strings.Contains(packRun, "cd \"packaging\\chocolatey\\mytool\"") {
		t.Errorf("publish-chocolatey pack step = %q, want committed packaging path", packRun)
	}
	if strings.Contains(packRun, "go run ./cmd/gpy") {
		t.Errorf("publish-chocolatey pack step should not regenerate packaging in CI: %q", packRun)
	}
}

// TestWorkflowCallTrigger tests that the generated workflow includes workflow_call trigger for reusability.
func TestWorkflowCallTrigger(t *testing.T) {
	cfg := &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main:    "./cmd/mytool",
			LDFlags: "-s -w",
		},
		Release: model.Release{
			TagTemplate: "v{{version}}",
			Platforms: []model.Platform{
				{OS: "linux", Arch: "amd64"},
			},
			Archive: model.Archive{
				NameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				Format: model.ArchiveFormat{
					Default: "tar.gz",
				},
				BinPathInArchive: "{{name}}",
			},
			Checksums: model.Checksums{
				File:      "checksums.txt",
				Algorithm: "sha256",
				Format:    "goreleaser",
			},
		},
		GitHub: model.GitHub{
			Workflows: model.GitHubWorkflows{
				Enabled:      true,
				WorkflowFile: ".github/workflows/gpy-release.yaml",
				TagPatterns:  []string{"v*"},
			},
		},
	}

	ctx := generator.Context{
		Config:  cfg,
		Version: "",
		ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
			params := naming.ArchiveNameParams{
				Name:                cfg.Project.Name,
				Version:             "1.0.0",
				OS:                  os,
				Arch:                arch,
				Format:              "tar.gz",
				ArchiveNameTemplate: cfg.Release.Archive.NameTemplate,
				BinPathTemplate:     cfg.Release.Archive.BinPathInArchive,
			}
			return naming.ArchiveName(params)
		},
	}

	gen := New()
	outputs, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(outputs) != 1 {
		t.Fatalf("expected 1 FileOutput, got %d", len(outputs))
	}

	// Parse YAML and verify workflow_call trigger is present
	var doc map[string]interface{}
	if err := yaml.Unmarshal(outputs[0].Content, &doc); err != nil {
		t.Fatalf("YAML syntax error: %v", err)
	}

	onTriggers, ok := doc["on"].(map[string]interface{})
	if !ok {
		t.Fatalf("'on' field is not a map")
	}

	// Verify push trigger exists
	if _, hasPush := onTriggers["push"]; !hasPush {
		t.Errorf("'on' is missing 'push' trigger")
	}

	// Verify workflow_call trigger exists (makes it reusable)
	if _, hasWorkflowCall := onTriggers["workflow_call"]; !hasWorkflowCall {
		t.Errorf("'on' is missing 'workflow_call' trigger - workflow is not reusable")
	}
}
