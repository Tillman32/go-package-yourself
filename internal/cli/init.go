package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"go-package-yourself/internal/generator"
	"go-package-yourself/internal/generator/workflow"
	"go-package-yourself/internal/model"
	"go-package-yourself/internal/naming"
	"go-package-yourself/internal/templatex"
)

// Init implements the `gpy init` command.
// It prompts for configuration (unless --yes or --no-tui is set) and writes gpy.yaml.
func Init(opts *GlobalOpts, args []string) error {
	// Parse command-specific flags (currently none, but support -h)
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		printInitUsage()
		return nil
	}

	// Determine config directory
	configDir := opts.ProjectRoot
	if err := os.Chdir(configDir); err != nil {
		return fmt.Errorf("failed to change to project root %q: %w", configDir, err)
	}

	// Get absolute path for consistency
	absConfigDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Build config with prompts
	cfg := defaultConfig()

	// Prompt for values (unless --yes or --no-tui)
	if !opts.Yes && !opts.NoTUI {
		if err := interactiveInit(cfg, absConfigDir); err != nil {
			return err
		}
	} else {
		// --yes or --no-tui: use defaults
		if err := applyDefaults(cfg, absConfigDir); err != nil {
			return err
		}
	}

	// Write config file
	configPath := filepath.Join(absConfigDir, "gpy.yaml")
	if err := writeConfigFile(configPath, cfg); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✓ Created config file: %s\n", configPath)

	// Generate GitHub Actions workflow if enabled
	if cfg.GitHub.Workflows.Enabled {
		if err := generateWorkflowFile(cfg, absConfigDir); err != nil {
			fmt.Printf("⚠ Warning: Failed to auto-generate workflow: %v\n", err)
		} else {
			fmt.Printf("✓ Created GitHub Actions workflow\n")
		}
	}

	fmt.Printf("✓ Edit the file to customize your configuration\n")
	fmt.Printf("\nNext step:\n")
	fmt.Printf("  gpy package          # Generate npm, homebrew, chocolatey, docker artifacts\n")
	fmt.Printf("\nSee docs at: https://github.com/your/repo\n")

	return nil
}

// interactiveInit prompts for configuration values interactively.
func interactiveInit(cfg *model.Config, projectRoot string) error {
	reader := bufio.NewReader(os.Stdin)

	// Project name
	defaultName := filepath.Base(projectRoot)
	fmt.Printf("Project name [%s]: ", defaultName)
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)
	if name == "" {
		name = defaultName
	}
	cfg.Project.Name = name

	// Project repo
	defaultRepo := getGitRemote()
	if defaultRepo == "" {
		defaultRepo = "owner/repo"
	}
	fmt.Printf("GitHub repository (owner/repo) [%s]: ", defaultRepo)
	repo, _ := reader.ReadString('\n')
	repo = strings.TrimSpace(repo)
	if repo == "" {
		repo = defaultRepo
	}
	if repo == "owner/repo" {
		fmt.Printf("⚠ Using placeholder 'owner/repo'. Update gpy.yaml before publishing.\n")
	}
	cfg.Project.Repo = repo
	cfg.Project.Homepage = defaultHomepage(repo)

	// Project metadata
	fmt.Printf("Project description (shown in package managers): ")
	description, _ := reader.ReadString('\n')
	cfg.Project.Description = strings.TrimSpace(description)

	if cfg.Project.Homepage == "" {
		fmt.Printf("Project homepage URL: ")
	} else {
		fmt.Printf("Project homepage URL [%s]: ", cfg.Project.Homepage)
	}
	homepage, _ := reader.ReadString('\n')
	homepage = strings.TrimSpace(homepage)
	if homepage != "" {
		cfg.Project.Homepage = homepage
	}

	detectedLicense, detectedLicenseFile := detectLicense(projectRoot)
	if detectedLicense == "" {
		fmt.Printf("Project license (SPDX identifier, e.g. MIT): ")
	} else {
		fmt.Printf("Project license (SPDX identifier) [%s from %s]: ", detectedLicense, detectedLicenseFile)
	}
	license, _ := reader.ReadString('\n')
	cfg.Project.License = strings.TrimSpace(license)
	if cfg.Project.License == "" {
		cfg.Project.License = detectedLicense
	}

	// Go main
	defaultMain := fmt.Sprintf("./cmd/%s", name)
	fmt.Printf("Go main package path for builds [%s]: ", defaultMain)
	main, _ := reader.ReadString('\n')
	main = strings.TrimSpace(main)
	if main == "" {
		main = defaultMain
	}
	cfg.Go.Main = main

	// Package options
	fmt.Printf("\nPackage distributions:\n")

	fmt.Printf("Enable npm package? (y/n) [n]: ")
	result, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(result)) == "y" {
		cfg.Packages.NPM.Enabled = true
		fmt.Printf("  npm package name [%s]: ", name)
		pkgName, _ := reader.ReadString('\n')
		pkgName = strings.TrimSpace(pkgName)
		if pkgName != "" {
			cfg.Packages.NPM.PackageName = pkgName
		} else {
			cfg.Packages.NPM.PackageName = name
		}
	}

	fmt.Printf("Enable homebrew package? (y/n) [n]: ")
	result, _ = reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(result)) == "y" {
		cfg.Packages.Homebrew.Enabled = true
		formulaName := capitalizeWords(name)
		fmt.Printf("  homebrew formula name [%s]: ", formulaName)
		fmName, _ := reader.ReadString('\n')
		fmName = strings.TrimSpace(fmName)
		if fmName != "" {
			cfg.Packages.Homebrew.FormulaName = fmName
		} else {
			cfg.Packages.Homebrew.FormulaName = formulaName
		}
	}

	fmt.Printf("Enable chocolatey package? (y/n) [n]: ")
	result, _ = reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(result)) == "y" {
		cfg.Packages.Chocolatey.Enabled = true
		fmt.Printf("  chocolatey package ID [%s]: ", name)
		pkgID, _ := reader.ReadString('\n')
		pkgID = strings.TrimSpace(pkgID)
		if pkgID != "" {
			cfg.Packages.Chocolatey.PackageID = pkgID
		} else {
			cfg.Packages.Chocolatey.PackageID = name
		}
	}

	fmt.Printf("Enable docker image? (y/n) [n]: ")
	result, _ = reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(result)) == "y" {
		cfg.Packages.Docker.Enabled = true
		fmt.Printf("  docker image name [%s]: ", name)
		imgName, _ := reader.ReadString('\n')
		imgName = strings.TrimSpace(imgName)
		if imgName != "" {
			cfg.Packages.Docker.ImageName = imgName
		} else {
			cfg.Packages.Docker.ImageName = name
		}
	}

	fmt.Printf("Enable GitHub Actions workflow? (y/n) [y]: ")
	result, _ = reader.ReadString('\n')
	result = strings.TrimSpace(strings.ToLower(result))
	cfg.GitHub.Workflows.Enabled = result != "n"

	return nil
}

// applyDefaults sets sensible defaults without prompting.
func applyDefaults(cfg *model.Config, projectRoot string) error {
	// Try to infer project name from cmd/<name>/main.go structure
	defaultName := inferProjectName(projectRoot)
	if defaultName == "" {
		// Fallback to directory base name
		defaultName = filepath.Base(projectRoot)
	}
	cfg.Project.Name = defaultName

	// Try to get git remote, fallback to placeholder
	repo := getGitRemote()
	if repo == "" {
		repo = "owner/repo"
	}
	cfg.Project.Repo = repo
	cfg.Project.Homepage = defaultHomepage(repo)

	cfg.Go.Main = fmt.Sprintf("./cmd/%s", defaultName)
	cfg.Packages.NPM.PackageName = defaultName

	cfg.Packages.Homebrew.FormulaName = capitalizeWords(defaultName)
	cfg.Packages.Chocolatey.PackageID = defaultName

	return nil
}

func defaultConfig() *model.Config {
	return &model.Config{
		SchemaVersion: 1,
		Project:       model.Project{},
		Go:            model.Go{},
		Release: model.Release{
			TagTemplate: "v{{version}}",
			Platforms: []model.Platform{
				{OS: "darwin", Arch: "amd64"},
				{OS: "darwin", Arch: "arm64"},
				{OS: "linux", Arch: "amd64"},
				{OS: "linux", Arch: "arm64"},
				{OS: "windows", Arch: "amd64"},
			},
			Archive: model.Archive{
				NameTemplate:     "{{name}}_{{version}}_{{os}}_{{arch}}",
				Format:           model.ArchiveFormat{Default: "tar.gz", Windows: "zip"},
				BinPathInArchive: "{{name}}",
			},
			Checksums: model.Checksums{
				File:      "checksums.txt",
				Algorithm: "sha256",
				Format:    "goreleaser",
			},
		},
		Packages: model.Packages{
			NPM: model.NPM{
				Enabled:     false,
				NodeEngines: ">=24",
			},
			Homebrew: model.Homebrew{
				Enabled: false,
			},
			Chocolatey: model.Chocolatey{
				Enabled: false,
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
}

func defaultHomepage(repo string) string {
	if repo == "" || repo == "owner/repo" {
		return ""
	}

	return fmt.Sprintf("https://github.com/%s", repo)
}

func detectLicense(projectRoot string) (spdx string, sourceFile string) {
	candidates := []string{
		"LICENSE",
		"LICENSE.md",
		"LICENSE.txt",
		"LICENCE",
		"LICENCE.md",
		"LICENCE.txt",
		"COPYING",
		"COPYING.md",
		"COPYING.txt",
	}

	for _, name := range candidates {
		data, err := os.ReadFile(filepath.Join(projectRoot, name))
		if err == nil {
			spdx := inferSPDXLicense(string(data))
			if spdx != "" {
				return spdx, name
			}
		}
	}

	return "", ""
}

func inferSPDXLicense(content string) string {
	lower := strings.ToLower(content)

	switch {
	case strings.Contains(lower, "mit license"):
		return "MIT"
	case strings.Contains(lower, "apache license") && strings.Contains(lower, "version 2.0"):
		return "Apache-2.0"
	case strings.Contains(lower, "mozilla public license") && strings.Contains(lower, "version 2.0"):
		return "MPL-2.0"
	case strings.Contains(lower, "isc license"):
		return "ISC"
	case strings.Contains(lower, "bsd 3-clause") || strings.Contains(lower, "redistribution and use in source and binary forms") && strings.Contains(lower, "neither the name of"):
		return "BSD-3-Clause"
	case strings.Contains(lower, "bsd 2-clause") || strings.Contains(lower, "redistribution and use in source and binary forms") && strings.Contains(lower, "this list of conditions and the following disclaimer") && !strings.Contains(lower, "neither the name of"):
		return "BSD-2-Clause"
	case strings.Contains(lower, "gnu affero general public license") && strings.Contains(lower, "version 3"):
		return "AGPL-3.0"
	case strings.Contains(lower, "gnu general public license") && strings.Contains(lower, "version 3"):
		return "GPL-3.0"
	case strings.Contains(lower, "gnu general public license") && strings.Contains(lower, "version 2"):
		return "GPL-2.0"
	case strings.Contains(lower, "gnu lesser general public license") && strings.Contains(lower, "version 3"):
		return "LGPL-3.0"
	case strings.Contains(lower, "gnu lesser general public license") && strings.Contains(lower, "version 2.1"):
		return "LGPL-2.1"
	default:
		return ""
	}
}

// getGitRemote attempts to retrieve the GitHub remote URL.
// Returns "owner/repo" format, or empty string if not found.
func getGitRemote() string {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	url := strings.TrimSpace(string(output))

	// Parse GitHub URLs
	// https://github.com/owner/repo.git
	// git@github.com:owner/repo.git
	if strings.Contains(url, "github.com") {
		parts := strings.FieldsFunc(url, func(r rune) bool {
			return r == ':' || r == '/' || r == '@'
		})

		for i, part := range parts {
			if part == "github.com" && i+2 < len(parts) {
				owner := parts[i+1]
				repo := strings.TrimSuffix(parts[i+2], ".git")
				return fmt.Sprintf("%s/%s", owner, repo)
			}
		}
	}

	return ""
}

// inferProjectName looks for cmd/<name>/main.go and returns the first valid name found.
// Returns empty string if no cmd structure is found.
func inferProjectName(projectRoot string) string {
	cmdDir := filepath.Join(projectRoot, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			mainPath := filepath.Join(cmdDir, entry.Name(), "main.go")
			if _, err := os.Stat(mainPath); err == nil {
				return entry.Name()
			}
		}
	}

	return ""
}

// capitalizeWords converts a string like "my-tool" to "MyTool" for formula names.
func capitalizeWords(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// writeConfigFile writes the configuration to a YAML file with clear comments.
func writeConfigFile(path string, cfg *model.Config) error {
	// Create a YAML encoder with nice formatting
	var yamlData strings.Builder

	yamlData.WriteString(`# go-package-yourself configuration
# See: https://github.com/your/repo for documentation
#
# This file configures artifact generation for:
# - npm/npx packages
# - Homebrew formulas
# - Chocolatey packages
# - GitHub Actions workflows for releases

schemaVersion: 1

project:
  name: `)
	yamlData.WriteString(cfg.Project.Name)
	yamlData.WriteString(`
  repo: `)
	yamlData.WriteString(cfg.Project.Repo)
	if cfg.Project.Description != "" {
		yamlData.WriteString(`
  description: `)
		yamlData.WriteString(strconv.Quote(cfg.Project.Description))
	} else {
		yamlData.WriteString(`
  # description: Short description of your tool`)
	}
	if cfg.Project.Homepage != "" {
		yamlData.WriteString(`
  homepage: `)
		yamlData.WriteString(strconv.Quote(cfg.Project.Homepage))
	} else {
		yamlData.WriteString(`
  # homepage: https://example.com`)
	}
	if cfg.Project.License != "" {
		yamlData.WriteString(`
  license: `)
		yamlData.WriteString(strconv.Quote(cfg.Project.License))
	} else {
		yamlData.WriteString(`
  # license: MIT`)
	}
	yamlData.WriteString(`

go:
  main: `)
	yamlData.WriteString(cfg.Go.Main)
	yamlData.WriteString(`
  # cgo: false
  # ldflags: -X main.version=...

release:
  tagTemplate: v{{version}}
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
  archive:
    nameTemplate: '{{name}}_{{version}}_{{os}}_{{arch}}'
    format:
      default: tar.gz
      windows: zip
    binPathInArchive: '{{name}}'
  checksums:
    file: checksums.txt
    algorithm: sha256
    format: goreleaser

packages:
  npm:
    enabled: `)
	yamlData.WriteString(fmt.Sprintf("%v", cfg.Packages.NPM.Enabled))
	if cfg.Packages.NPM.Enabled && cfg.Packages.NPM.PackageName != "" {
		yamlData.WriteString(`
    packageName: `)
		yamlData.WriteString(cfg.Packages.NPM.PackageName)
	}
	yamlData.WriteString(`
    nodeEngines: ">=24"

  homebrew:
    enabled: `)
	yamlData.WriteString(fmt.Sprintf("%v", cfg.Packages.Homebrew.Enabled))
	if cfg.Packages.Homebrew.Enabled && cfg.Packages.Homebrew.FormulaName != "" {
		yamlData.WriteString(`
    formulaName: `)
		yamlData.WriteString(cfg.Packages.Homebrew.FormulaName)
	}
	yamlData.WriteString(`

  chocolatey:
    enabled: `)
	yamlData.WriteString(fmt.Sprintf("%v", cfg.Packages.Chocolatey.Enabled))
	if cfg.Packages.Chocolatey.Enabled && cfg.Packages.Chocolatey.PackageID != "" {
		yamlData.WriteString(`
    packageId: `)
		yamlData.WriteString(cfg.Packages.Chocolatey.PackageID)
	}
	yamlData.WriteString(`

  docker:
    enabled: `)
	yamlData.WriteString(fmt.Sprintf("%v", cfg.Packages.Docker.Enabled))
	if cfg.Packages.Docker.Enabled && cfg.Packages.Docker.ImageName != "" {
		yamlData.WriteString(`
    imageName: `)
		yamlData.WriteString(cfg.Packages.Docker.ImageName)
	}
	yamlData.WriteString(`

github:
  workflows:
    enabled: `)
	yamlData.WriteString(fmt.Sprintf("%v", cfg.GitHub.Workflows.Enabled))
	yamlData.WriteString(`
    workflowFile: `)
	yamlData.WriteString(cfg.GitHub.Workflows.WorkflowFile)
	yamlData.WriteString(`
    tagPatterns: ["v*"]
`)

	if err := os.WriteFile(path, []byte(yamlData.String()), 0o644); err != nil {
		return err
	}

	return nil
}

// generateWorkflowFile generates and writes the GitHub Actions workflow file.
func generateWorkflowFile(cfg *model.Config, projectRoot string) error {
	// Create generator context
	ctx := generator.Context{
		Config:      cfg,
		ProjectRoot: projectRoot,
		Version:     "", // init doesn't use version
		ArchiveName: func(os, arch string) (archiveFilename, binPathInArchive string, err error) {
			params := naming.ArchiveNameParams{
				Name:                cfg.Project.Name,
				Version:             "",
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
					"version": "",
				},
			}
			return renderer.RenderWithFieldPath(template, fieldPath)
		},
	}

	// Generate workflow
	gen := workflow.New()
	outputs, err := gen.Generate(ctx)
	if err != nil {
		return fmt.Errorf("workflow generation failed: %w", err)
	}

	if len(outputs) == 0 {
		return fmt.Errorf("workflow generation produced no output")
	}

	// Write workflow file
	for _, output := range outputs {
		path := filepath.Join(projectRoot, output.Path)
		dir := filepath.Dir(path)

		// Create directory if needed
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create workflow directory %q: %w", dir, err)
		}

		// Write file
		if err := os.WriteFile(path, output.Content, output.Mode); err != nil {
			return fmt.Errorf("failed to write workflow file %q: %w", path, err)
		}
	}

	return nil
}

func printInitUsage() {
	fmt.Fprintf(os.Stderr, `gpy init - Create a new gpy.yaml configuration

Usage:
  gpy init [flags]

Flags:
  -h, --help             Show this help message

Global flags:
  --yes                  Accept all defaults without prompting
  --no-tui               Disable prompts; fail if required fields are missing
  --project-root <path>  Project root directory (default: current working directory)

Examples:
  gpy init
  gpy init --yes
  gpy --project-root /path/to/project init --yes

Description:
  Creates a new gpy.yaml configuration file in the project root.
  If --yes is specified, uses sensible defaults. Otherwise, prompts for configuration
  values interactively.

  Defaults:
  - Project name: current directory name
  - GitHub repo: auto-detected from git remote (owner/repo)
  - Go main: ./cmd/<project-name>
  - Packages: all disabled by default
  - GitHub workflows: enabled by default

`)
}
