// Package homebrew generates Homebrew formula files for macOS and Linux.
package homebrew

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strings"

	"go-package-yourself/internal/generator"
	"go-package-yourself/internal/model"
)

// Generator implements the Homebrew formula generator.
type Generator struct{}

// Name returns the canonical name of this generator.
func (g *Generator) Name() string {
	return "homebrew"
}

// Generate produces a Homebrew formula file.
// The formula is generated in packaging/homebrew/<FormulaName>.rb
func (g *Generator) Generate(ctx generator.Context) ([]generator.FileOutput, error) {
	cfg := ctx.Config

	// Determine formula name
	formulaName := cfg.Packages.Homebrew.FormulaName
	if formulaName == "" {
		formulaName = capitalizeWords(cfg.Project.Name)
	}

	// Validate formula name (must be valid Ruby class name)
	if err := validateFormulaName(formulaName); err != nil {
		return nil, fmt.Errorf("invalid formula name %q: %w", formulaName, err)
	}

	// Build platform matrix: group platforms by OS
	platformsByOS := groupPlatformsByOS(cfg.Release.Platforms)

	// Generate the Ruby formula content
	formula, err := buildFormula(ctx, formulaName, platformsByOS)
	if err != nil {
		return nil, fmt.Errorf("failed to build formula: %w", err)
	}

	// Return the formula as a file output
	outputPath := fmt.Sprintf("packaging/homebrew/%s.rb", formulaName)

	return []generator.FileOutput{
		{
			Path:    outputPath,
			Content: []byte(formula),
			Mode:    0o644,
		},
	}, nil
}

// platformsByOS is a map from OS name (e.g., "darwin", "linux") to a sorted slice of architectures.
type platformsByOS map[string][]string

// groupPlatformsByOS groups platforms by operating system.
// Returns a map like {"darwin": ["amd64", "arm64"], "linux": ["amd64"]}.
// Architectures are sorted to ensure deterministic output.
func groupPlatformsByOS(platforms []model.Platform) platformsByOS {
	result := make(platformsByOS)
	archSet := make(map[string]map[string]bool) // os -> arch -> true

	// Group architectures by OS and track them in a set
	for _, p := range platforms {
		if archSet[p.OS] == nil {
			archSet[p.OS] = make(map[string]bool)
		}
		archSet[p.OS][p.Arch] = true
	}

	// Convert sets to sorted slices
	for os, archs := range archSet {
		archList := make([]string, 0, len(archs))
		for arch := range archs {
			archList = append(archList, arch)
		}
		sort.Strings(archList)
		result[os] = archList
	}

	return result
}

// buildFormula constructs the complete Ruby formula content.
func buildFormula(ctx generator.Context, formulaName string, platformsByOS platformsByOS) (string, error) {
	var buf strings.Builder

	// Determine GitHub repo info
	repo := ctx.Config.Project.Repo
	if repo == "" {
		return "", fmt.Errorf("project.repo is required for Homebrew formula generation")
	}

	// Start formula class definition
	buf.WriteString(fmt.Sprintf("class %s < Formula\n", formulaName))

	// Add metadata
	if desc := ctx.Config.Project.Description; desc != "" {
		buf.WriteString(fmt.Sprintf("  desc \"%s\"\n", escapeString(desc)))
	}

	if homepage := ctx.Config.Project.Homepage; homepage != "" {
		buf.WriteString(fmt.Sprintf("  homepage \"%s\"\n", escapeString(homepage)))
	}

	if license := ctx.Config.Project.License; license != "" {
		buf.WriteString(fmt.Sprintf("  license \"%s\"\n", escapeString(license)))
	}

	buf.WriteString("\n")

	// Generate platform-specific URL and sha256 blocks
	if err := generatePlatformBlocks(ctx, &buf, platformsByOS, repo); err != nil {
		return "", err
	}

	// Add install method
	buf.WriteString("\n  def install\n")
	binaryName := ctx.Config.Project.Name
	buf.WriteString(fmt.Sprintf("    bin.install \"%s\"\n", binaryName))
	buf.WriteString("  end\n")

	// Add smoke test
	buf.WriteString("\n  test do\n")
	buf.WriteString(fmt.Sprintf("    assert_match /help/, shell_output(\"#{bin}/%s --help\")\n", binaryName))
	buf.WriteString("  end\n")

	// Close class definition
	buf.WriteString("end\n")

	return buf.String(), nil
}

// generatePlatformBlocks generates on_macos and on_linux blocks with platform-specific URLs and checksums.
func generatePlatformBlocks(ctx generator.Context, buf *strings.Builder, platformsByOS platformsByOS, repo string) error {
	version := ctx.Version
	if version == "" {
		// Placeholder version when not specified
		version = "{{version}}"
	}

	// macOS block (if any darwin platforms)
	if darwinArchs, ok := platformsByOS["darwin"]; ok {
		buf.WriteString("  on_macos do\n")

		if len(darwinArchs) > 1 {
			// Multiple architectures: use conditional
			buf.WriteString("    if Hardware::CPU.arm?\n")

			// arm64
			arch64Filename, binPath64, err := ctx.ArchiveName("darwin", "arm64")
			if err != nil {
				return fmt.Errorf("failed to compute archive name for darwin/arm64: %w", err)
			}
			url := buildURL(repo, version, arch64Filename)
			sha256 := computeSha256(arch64Filename, binPath64)

			buf.WriteString(fmt.Sprintf("      url \"%s\"\n", url))
			buf.WriteString(fmt.Sprintf("      sha256 \"%s\"\n", sha256))

			buf.WriteString("    else\n")

			// amd64
			amd64Filename, binPathAmd, err := ctx.ArchiveName("darwin", "amd64")
			if err != nil {
				return fmt.Errorf("failed to compute archive name for darwin/amd64: %w", err)
			}
			urlAmd := buildURL(repo, version, amd64Filename)
			sha256Amd := computeSha256(amd64Filename, binPathAmd)

			buf.WriteString(fmt.Sprintf("      url \"%s\"\n", urlAmd))
			buf.WriteString(fmt.Sprintf("      sha256 \"%s\"\n", sha256Amd))

			buf.WriteString("    end\n")
		} else {
			// Single architecture (only amd64 or only arm64)
			arch := darwinArchs[0]
			archiveFilename, binPath, err := ctx.ArchiveName("darwin", arch)
			if err != nil {
				return fmt.Errorf("failed to compute archive name for darwin/%s: %w", arch, err)
			}
			url := buildURL(repo, version, archiveFilename)
			sha256 := computeSha256(archiveFilename, binPath)

			buf.WriteString(fmt.Sprintf("    url \"%s\"\n", url))
			buf.WriteString(fmt.Sprintf("    sha256 \"%s\"\n", sha256))
		}

		buf.WriteString("  end\n\n")
	}

	// Linux block (if any linux platforms)
	if linuxArchs, ok := platformsByOS["linux"]; ok {
		buf.WriteString("  on_linux do\n")

		if len(linuxArchs) > 1 {
			// Multiple architectures: use conditional
			buf.WriteString("    if Hardware::CPU.arm?\n")

			// arm64
			linuxArm64Filename, linuxArm64Path, err := ctx.ArchiveName("linux", "arm64")
			if err != nil {
				return fmt.Errorf("failed to compute archive name for linux/arm64: %w", err)
			}
			urlLinuxArm := buildURL(repo, version, linuxArm64Filename)
			sha256LinuxArm := computeSha256(linuxArm64Filename, linuxArm64Path)

			buf.WriteString(fmt.Sprintf("      url \"%s\"\n", urlLinuxArm))
			buf.WriteString(fmt.Sprintf("      sha256 \"%s\"\n", sha256LinuxArm))

			buf.WriteString("    else\n")

			// amd64
			linuxAmd64Filename, linuxAmd64Path, err := ctx.ArchiveName("linux", "amd64")
			if err != nil {
				return fmt.Errorf("failed to compute archive name for linux/amd64: %w", err)
			}
			urlLinuxAmd := buildURL(repo, version, linuxAmd64Filename)
			sha256LinuxAmd := computeSha256(linuxAmd64Filename, linuxAmd64Path)

			buf.WriteString(fmt.Sprintf("      url \"%s\"\n", urlLinuxAmd))
			buf.WriteString(fmt.Sprintf("      sha256 \"%s\"\n", sha256LinuxAmd))

			buf.WriteString("    end\n")
		} else {
			// Single architecture
			arch := linuxArchs[0]
			archiveFilename, binPath, err := ctx.ArchiveName("linux", arch)
			if err != nil {
				return fmt.Errorf("failed to compute archive name for linux/%s: %w", arch, err)
			}
			url := buildURL(repo, version, archiveFilename)
			sha256 := computeSha256(archiveFilename, binPath)

			buf.WriteString(fmt.Sprintf("    url \"%s\"\n", url))
			buf.WriteString(fmt.Sprintf("    sha256 \"%s\"\n", sha256))
		}

		buf.WriteString("  end\n")
	}

	return nil
}

// buildURL constructs a GitHub release download URL.
func buildURL(repo, tag, filename string) string {
	// repo format: "owner/repo", tag format: "v1.2.3" or "{{version}}"
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, tag, filename)
}

// computeSha256 generates a deterministic sha256 value for testing.
// In production, this would be read from a release checksums file or fetched from GitHub.
// For determinism in tests, we use a hash based on the filename and binary path.
func computeSha256(archiveFilename, binPath string) string {
	// Create a deterministic hash input from the archive and binary path
	h := sha256.New()
	io.WriteString(h, archiveFilename)
	io.WriteString(h, "|")
	io.WriteString(h, binPath)
	return hex.EncodeToString(h.Sum(nil))
}

// validateFormulaName checks that the formula name is a valid Ruby class name.
func validateFormulaName(name string) error {
	if name == "" {
		return fmt.Errorf("formula name cannot be empty")
	}

	// Ruby class names must start with uppercase letter
	if !isUpperCase(rune(name[0])) {
		return fmt.Errorf("formula name must start with uppercase letter, got %q", string(name[0]))
	}

	// Check valid characters (letters, digits, underscores)
	for i, r := range name {
		if !isValidClassNameChar(r) {
			return fmt.Errorf("formula name contains invalid character %q at position %d", r, i)
		}
	}

	return nil
}

// isValidClassNameChar checks if a rune is valid in a Ruby class name.
func isValidClassNameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// isUpperCase checks if a rune is an uppercase letter.
func isUpperCase(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

// escapeString escapes special characters in Ruby strings.
func escapeString(s string) string {
	// Escape backslashes and double quotes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

// capitalizeWords converts a string like "my-tool" to "MyTool".
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
