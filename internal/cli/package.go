package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-package-yourself/internal/config"
	"go-package-yourself/internal/generator"
	"go-package-yourself/internal/generator/chocolatey"
	"go-package-yourself/internal/generator/docker"
	"go-package-yourself/internal/generator/homebrew"
	"go-package-yourself/internal/generator/npm"
	"go-package-yourself/internal/model"
	"go-package-yourself/internal/naming"
	"go-package-yourself/internal/templatex"
	"go-package-yourself/internal/validate"
)

// Package implements the `gpy package` command.
// It generates packaging artifacts (npm, homebrew, chocolatey) to <project-root>/packaging/.
func Package(opts *GlobalOpts, args []string) error {
	// Parse command-specific flags
	fs := flag.NewFlagSet("package", flag.ContinueOnError)
	onlyStr := fs.String("only", "", "comma-separated list of generators to run (npm,homebrew,chocolatey,docker)")
	version := fs.String("version", "", "version to use in generated files (optional)")
	sync := fs.Bool("sync", false, "regenerate and overwrite existing artifacts (for config updates)")
	configPath := fs.String("config", "", "path to config file (overrides global --config)")
	projectRoot := fs.String("project-root", "", "project root directory (overrides global --project-root)")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			printPackageUsage()
			return nil
		}
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	// Per-command flags override global opts
	if *configPath != "" {
		opts.ConfigPath = *configPath
	}
	if *projectRoot != "" {
		opts.ProjectRoot = *projectRoot
	}

	// Change to project root
	if err := os.Chdir(opts.ProjectRoot); err != nil {
		return fmt.Errorf("failed to change to project root %q: %w", opts.ProjectRoot, err)
	}

	// Get absolute project root
	absProjectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Load configuration
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w\n\nNext step: Run 'gpy init --yes' to create a starter config", err)
	}

	// Validate configuration
	if err := validate.Config(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w\n\nNext step: Edit gpy.yaml to fix the issues", err)
	}

	// Parse --only flag
	var enabled map[string]bool
	if *onlyStr != "" {
		var err error
		enabled, err = parseOnlyFlag(*onlyStr)
		if err != nil {
			return err
		}
	} else {
		// Use config defaults
		enabled = map[string]bool{
			"npm":        cfg.Packages.NPM.Enabled,
			"homebrew":   cfg.Packages.Homebrew.Enabled,
			"chocolatey": cfg.Packages.Chocolatey.Enabled,
			"docker":     cfg.Packages.Docker.Enabled,
		}
		// If no generators are enabled in config, default to npm+homebrew+chocolatey
		if countEnabled(enabled) == 0 {
			enabled = map[string]bool{
				"npm":        true,
				"homebrew":   true,
				"chocolatey": true,
			}
		}
	}

	// Ensure packaging directory exists
	packagingDir := filepath.Join(absProjectRoot, "packaging")
	if err := os.MkdirAll(packagingDir, 0o755); err != nil {
		return fmt.Errorf("failed to create packaging directory: %w", err)
	}

	// Create generator context
	ctx := createGeneratorContext(cfg, absProjectRoot, *version)

	// Generate artifacts
	generatorCount := countEnabled(enabled)
	if generatorCount == 0 {
		fmt.Println("No generators enabled. Check your configuration or use --only flag.")
		fmt.Println("Run 'gpy init' to enable generators or 'gpy package --only npm,homebrew,chocolatey,docker' to generate all.")
		return nil
	}

	// Run enabled generators (stub implementations for now)
	if enabled["npm"] {
		outputs, err := generatorNPM(ctx)
		if err != nil {
			return fmt.Errorf("npm generation failed: %w", err)
		}
		if err := writeOutputs(absProjectRoot, outputs, *sync); err != nil {
			return fmt.Errorf("failed to write npm artifacts: %w", err)
		}
		fmt.Printf("✓ Generated packaging/npm/%s/\n", cfg.Packages.NPM.PackageName)
	}

	if enabled["homebrew"] {
		outputs, err := generatorHomebrew(ctx)
		if err != nil {
			return fmt.Errorf("homebrew generation failed: %w", err)
		}
		if err := writeOutputs(absProjectRoot, outputs, *sync); err != nil {
			return fmt.Errorf("failed to write homebrew artifacts: %w", err)
		}
		formulaName := cfg.Packages.Homebrew.FormulaName
		if formulaName == "" {
			formulaName = capitalizeWords(cfg.Project.Name)
		}
		fmt.Printf("✓ Generated packaging/homebrew/%s.rb\n", formulaName)
	}

	if enabled["chocolatey"] {
		outputs, err := generatorChocolatey(ctx)
		if err != nil {
			return fmt.Errorf("chocolatey generation failed: %w", err)
		}
		if err := writeOutputs(absProjectRoot, outputs, *sync); err != nil {
			return fmt.Errorf("failed to write chocolatey artifacts: %w", err)
		}
		pkgID := cfg.Packages.Chocolatey.PackageID
		if pkgID == "" {
			pkgID = cfg.Project.Name
		}
		fmt.Printf("✓ Generated packaging/chocolatey/%s/\n", pkgID)
	}

	if enabled["docker"] {
		outputs, err := generatorDocker(ctx)
		if err != nil {
			return fmt.Errorf("docker generation failed: %w", err)
		}
		if err := writeOutputs(absProjectRoot, outputs, *sync); err != nil {
			return fmt.Errorf("failed to write Docker artifacts: %w", err)
		}
		fmt.Printf("✓ Generated Dockerfile and .dockerignore\n")
	}

	if *sync {
		fmt.Printf("\n✓ All artifacts synchronized successfully\n")
	} else {
		fmt.Printf("\n✓ All artifacts generated successfully\n")
	}
	fmt.Printf("Review generated files in: %s/packaging/\n", absProjectRoot)

	return nil
}

// parseOnlyFlag parses the --only flag into a map of enabled generators.
// Returns an error if any invalid generators are specified.
func parseOnlyFlag(onlyStr string) (map[string]bool, error) {
	enabled := make(map[string]bool)
	valid := map[string]bool{"npm": true, "homebrew": true, "chocolatey": true, "docker": true}

	parts := strings.Split(onlyStr, ",")
	var invalid []string
	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		if valid[part] {
			enabled[part] = true
		} else {
			invalid = append(invalid, part)
		}
	}

	if len(invalid) > 0 {
		return nil, fmt.Errorf("invalid generator(s): %s (valid: npm, homebrew, chocolatey, docker)", strings.Join(invalid, ", "))
	}

	if len(enabled) == 0 {
		return nil, fmt.Errorf("--only flag requires at least one generator (npm, homebrew, chocolatey, docker)")
	}

	return enabled, nil
}

// countEnabled counts how many generators are enabled.
func countEnabled(enabled map[string]bool) int {
	count := 0
	for _, v := range enabled {
		if v {
			count++
		}
	}
	return count
}

// createGeneratorContext creates a generator.Context with all necessary helpers.
func createGeneratorContext(cfg *model.Config, projectRoot, version string) generator.Context {
	return generator.Context{
		Config:      cfg,
		ProjectRoot: projectRoot,
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
					// os, arch, and ext are set by the caller
				},
			}
			return renderer.RenderWithFieldPath(template, fieldPath)
		},
	}
}

// generatorNPM delegates to the npm generator implementation.
func generatorNPM(ctx generator.Context) ([]generator.FileOutput, error) {
	gen := &npm.Generator{}
	return gen.Generate(ctx)
}

// generatorHomebrew delegates to the homebrew generator implementation.
func generatorHomebrew(ctx generator.Context) ([]generator.FileOutput, error) {
	gen := &homebrew.Generator{}
	return gen.Generate(ctx)
}

// generatorChocolatey delegates to the chocolatey generator implementation.
func generatorChocolatey(ctx generator.Context) ([]generator.FileOutput, error) {
	gen := &chocolatey.Generator{}
	return gen.Generate(ctx)
}

// generatorDocker delegates to the docker generator implementation.
func generatorDocker(ctx generator.Context) ([]generator.FileOutput, error) {
	gen := &docker.Generator{}
	return gen.Generate(ctx)
}

// writeOutputs writes all generator outputs to disk.
// If sync is true, it will silently overwrite (for config updates).
// Otherwise, it will still overwrite but could warn the user.
func writeOutputs(projectRoot string, outputs []generator.FileOutput, sync bool) error {
	for _, output := range outputs {
		path := filepath.Join(projectRoot, output.Path)

		// Create directory if needed
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %q: %w", dir, err)
		}

		// Write file (always allow overwrites)
		if err := os.WriteFile(path, output.Content, output.Mode); err != nil {
			return fmt.Errorf("failed to write file %q: %w", path, err)
		}
	}

	return nil
}

func printPackageUsage() {
	fmt.Fprintf(os.Stderr, `gpy package - Generate packaging artifacts

Usage:
  gpy package [flags]

Flags:
  --only <generators>    Comma-separated list of generators to run
                         Valid values: npm, homebrew, chocolatey, docker
                         Example: --only npm,homebrew
  --version <version>    Version to use in generated files
                         Example: --version 1.2.3
  --sync                 Regenerate and overwrite existing artifacts
                         Use this after updating gpy.yaml
  -h, --help             Show this help message

Global flags:
  --config <path>        Path to config file (auto-detected if omitted)
  --project-root <path>  Project root directory (default: current working directory)

Examples:
  gpy package
  gpy package --only npm,homebrew
  gpy package --version 1.2.3
  gpy package --sync     # Regenerate after config changes
  gpy --project-root /path/to/project package

Description:
  Generates packaging artifacts based on the gpy.yaml configuration.
  Outputs are written to <project-root>/packaging/.

  Enabled generators are determined by:
  1. The --only flag if specified
  2. The packages configuration in gpy.yaml

  Use --sync to regenerate and overwrite existing artifacts after updating gpy.yaml.

  Exit codes:
    0  Success
    1  Error (invalid config, file I/O error, etc.)

`)
}
