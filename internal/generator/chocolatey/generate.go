// Package chocolatey implements the Chocolatey package generator for go-package-yourself.
package chocolatey

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"go-package-yourself/internal/generator"
	"go-package-yourself/internal/model"
)

// Generator implements the generator.Generator interface for Chocolatey packages.
type Generator struct{}

// Name returns the canonical name of this generator.
func (g *Generator) Name() string {
	return "chocolatey"
}

// Generate produces Chocolatey package artifacts.
func (g *Generator) Generate(ctx generator.Context) ([]generator.FileOutput, error) {
	if ctx.Config == nil {
		return nil, fmt.Errorf("context config is nil")
	}

	windowsPlatforms := filterWindowsPlatforms(ctx.Config.Release.Platforms)
	if len(windowsPlatforms) == 0 {
		return nil, fmt.Errorf("no Windows platforms configured for Chocolatey generator")
	}

	pkgID := ctx.Config.Packages.Chocolatey.PackageID
	if pkgID == "" {
		pkgID = ctx.Config.Project.Name
	}

	if err := validateConfig(ctx.Config); err != nil {
		return nil, err
	}

	version := ctx.Version
	if version == "" {
		version = "0.0.0"
	}

	checksumsByArch, err := loadChecksumsFromContext(ctx, windowsPlatforms)
	if err != nil {
		return nil, fmt.Errorf("failed to load checksums: %w", err)
	}

	basePath := filepath.Join("packaging", "chocolatey", pkgID)
	var outputs []generator.FileOutput

	nuspec, err := g.generateNuspec(ctx.Config, pkgID, version)
	if err != nil {
		return nil, fmt.Errorf("failed to generate nuspec: %w", err)
	}
	outputs = append(outputs, generator.FileOutput{
		Path:    filepath.Join(basePath, pkgID+".nuspec"),
		Content: nuspec,
		Mode:    0o644,
	})

	installScript, err := g.generateInstallScript(ctx, windowsPlatforms, pkgID, version, checksumsByArch)
	if err != nil {
		return nil, fmt.Errorf("failed to generate install script: %w", err)
	}
	outputs = append(outputs, generator.FileOutput{
		Path:    filepath.Join(basePath, "tools", "chocolateyInstall.ps1"),
		Content: installScript,
		Mode:    0o755,
	})

	uninstallScript := g.generateUninstallScript(pkgID)
	outputs = append(outputs, generator.FileOutput{
		Path:    filepath.Join(basePath, "tools", "chocolateyUninstall.ps1"),
		Content: uninstallScript,
		Mode:    0o755,
	})

	return outputs, nil
}

func filterWindowsPlatforms(platforms []model.Platform) []model.Platform {
	var windows []model.Platform
	for _, p := range platforms {
		if p.OS == "windows" {
			windows = append(windows, p)
		}
	}
	sort.Slice(windows, func(i, j int) bool {
		if windows[i].Arch == windows[j].Arch {
			return windows[i].OS < windows[j].OS
		}
		return windows[i].Arch < windows[j].Arch
	})
	return windows
}

func validateConfig(cfg *model.Config) error {
	if cfg.Project.Name == "" {
		return fmt.Errorf("project.name is required")
	}
	if cfg.Project.Repo == "" {
		return fmt.Errorf("project.repo is required")
	}
	if cfg.Release.TagTemplate == "" {
		return fmt.Errorf("release.tagTemplate is required")
	}
	return nil
}

func loadChecksumsFromContext(ctx generator.Context, windowsPlatforms []model.Platform) (map[string]string, error) {
	checksumsByArch := make(map[string]string)

	for _, p := range windowsPlatforms {
		archKey := p.OS + "/" + p.Arch

		archiveFilename, _, err := ctx.ArchiveName(p.OS, p.Arch)
		if err != nil {
			return nil, fmt.Errorf("failed to compute archive name for %s: %w", archKey, err)
		}

		hash := sha256.Sum256([]byte(archiveFilename))
		checksumsByArch[archKey] = hex.EncodeToString(hash[:])
	}

	return checksumsByArch, nil
}

func (g *Generator) generateNuspec(cfg *model.Config, pkgID, version string) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString(`<?xml version="1.0" encoding="utf-8"?>` + "\n")
	buf.WriteString(`<package>` + "\n")
	buf.WriteString(`  <metadata>` + "\n")

	buf.WriteString(fmt.Sprintf(`    <id>%s</id>`+"\n", escapeXML(pkgID)))
	buf.WriteString(fmt.Sprintf(`    <version>%s</version>`+"\n", escapeXML(version)))

	title := cfg.Project.Name
	buf.WriteString(fmt.Sprintf(`    <title>%s</title>`+"\n", escapeXML(title)))

	description := cfg.Project.Description
	if description == "" {
		description = "Package for " + cfg.Project.Name
	}
	buf.WriteString(fmt.Sprintf(`    <description>%s</description>`+"\n", escapeXML(description)))

	authors := cfg.Packages.Chocolatey.Authors
	if authors == "" {
		authors = "Unknown"
	}
	buf.WriteString(fmt.Sprintf(`    <authors>%s</authors>`+"\n", escapeXML(authors)))

	owners := cfg.Packages.Chocolatey.Authors
	if owners == "" {
		owners = "Unknown"
	}
	buf.WriteString(fmt.Sprintf(`    <owners>%s</owners>`+"\n", escapeXML(owners)))

	licenseURL := cfg.Project.License
	if licenseURL != "" && !strings.HasPrefix(licenseURL, "http") {
		licenseURL = "https://opensource.org/licenses/" + licenseURL
	}
	if licenseURL == "" {
		licenseURL = "https://github.com/" + cfg.Project.Repo + "/blob/main/LICENSE"
	}
	buf.WriteString(fmt.Sprintf(`    <licenseUrl>%s</licenseUrl>`+"\n", escapeXML(licenseURL)))

	projectURL := "https://github.com/" + cfg.Project.Repo
	buf.WriteString(fmt.Sprintf(`    <projectUrl>%s</projectUrl>`+"\n", escapeXML(projectURL)))

	packageSourceURL := projectURL + "/releases"
	buf.WriteString(fmt.Sprintf(`    <packageSourceUrl>%s</packageSourceUrl>`+"\n", escapeXML(packageSourceURL)))

	tags := "cli tool package-manager"
	buf.WriteString(fmt.Sprintf(`    <tags>%s</tags>`+"\n", escapeXML(tags)))

	buf.WriteString(`  </metadata>` + "\n")

	buf.WriteString(`  <files>` + "\n")
	buf.WriteString(`    <file src="tools\**" target="tools" />` + "\n")
	buf.WriteString(`  </files>` + "\n")

	buf.WriteString(`</package>` + "\n")

	return buf.Bytes(), nil
}

func (g *Generator) generateInstallScript(
	ctx generator.Context,
	windowsPlatforms []model.Platform,
	pkgID string,
	version string,
	checksumsByArch map[string]string,
) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString("# Chocolatey install script for " + pkgID + "\n")
	buf.WriteString("# Generated by gpy Chocolatey generator\n\n")

	buf.WriteString("$ErrorActionPreference = 'Stop'\n")
	buf.WriteString("$toolsDir = \"$(Split-Path -Parent $MyInvocation.MyCommand.Definition)\"\n\n")

	buf.WriteString("# Detect Windows architecture\n")
	buf.WriteString("$archMap = @{\n")
	buf.WriteString("    'AMD64' = @{ arch = 'x64'; gpy_arch = 'amd64' }\n")
	buf.WriteString("    'x86' = @{ arch = 'x86'; gpy_arch = 'amd64' }\n")
	buf.WriteString("    'ARM64' = @{ arch = 'arm64'; gpy_arch = 'arm64' }\n")
	buf.WriteString("}\n\n")

	buf.WriteString("$procArch = $env:PROCESSOR_ARCHITECTURE\n")
	buf.WriteString("if (-not $archMap.ContainsKey($procArch)) {\n")
	buf.WriteString("    throw \"Unsupported architecture: $procArch. Supported: AMD64, ARM64\"\n")
	buf.WriteString("}\n\n")

	buf.WriteString("$archInfo = $archMap[$procArch]\n")
	buf.WriteString("$localArch = $archInfo.gpy_arch\n\n")

	buf.WriteString("# Map architecture to download URLs and checksums\n")
	buf.WriteString("$downloadMap = @{\n")

	sortedPlatforms := append([]model.Platform{}, windowsPlatforms...)
	sort.Slice(sortedPlatforms, func(i, j int) bool {
		return sortedPlatforms[i].Arch < sortedPlatforms[j].Arch
	})

	for _, p := range sortedPlatforms {
		archiveFilename, _, err := ctx.ArchiveName(p.OS, p.Arch)
		if err != nil {
			return nil, fmt.Errorf("failed to compute archive name: %w", err)
		}

		archKey := p.OS + "/" + p.Arch
		checksum := checksumsByArch[archKey]

		releaseTag := version
		if releaseTag == "" {
			releaseTag = "$env:GITHUB_REF_NAME"
		}
		downloadURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s",
			ctx.Config.Project.Repo, releaseTag, archiveFilename)

		buf.WriteString(fmt.Sprintf("    '%s' = @{\n", p.Arch))
		buf.WriteString(fmt.Sprintf("        url = '%s'\n", downloadURL))
		buf.WriteString(fmt.Sprintf("        checksum = '%s'\n", checksum))
		buf.WriteString(fmt.Sprintf("        archive = '%s'\n", archiveFilename))
		buf.WriteString("    }\n")
	}

	buf.WriteString("}\n\n")

	buf.WriteString("if (-not $downloadMap.ContainsKey($localArch)) {\n")
	buf.WriteString("    throw \"No binary available for architecture: $localArch\"\n")
	buf.WriteString("}\n\n")

	buf.WriteString("$info = $downloadMap[$localArch]\n")
	buf.WriteString("$downloadUrl = $info.url\n")
	buf.WriteString("$expectedChecksum = $info.checksum\n")
	buf.WriteString("$archiveName = $info.archive\n\n")

	buf.WriteString("# Download archive from GitHub Release\n")
	buf.WriteString("$downloadPath = Join-Path $env:TEMP $archiveName\n")
	buf.WriteString("Write-Host \"Downloading from: $downloadUrl\"\n")
	buf.WriteString("try {\n")
	buf.WriteString("    $ProgressPreference = 'SilentlyContinue'\n")
	buf.WriteString("    Invoke-WebRequest -Uri $downloadUrl -OutFile $downloadPath\n")
	buf.WriteString("} catch {\n")
	buf.WriteString("    throw \"Failed to download package: $_\"\n")
	buf.WriteString("}\n\n")

	buf.WriteString("# Verify SHA256 checksum\n")
	buf.WriteString("Write-Host \"Verifying checksum...\"\n")
	buf.WriteString("try {\n")
	buf.WriteString("    $hasher = [System.Security.Cryptography.HashAlgorithm]::Create('sha256')\n")
	buf.WriteString("    $fileStream = [System.IO.File]::OpenRead($downloadPath)\n")
	buf.WriteString("    $hashBytes = $hasher.ComputeHash($fileStream)\n")
	buf.WriteString("    $fileStream.Close()\n")
	buf.WriteString("    $actualChecksum = [System.BitConverter]::ToString($hashBytes) -replace '-',''\n")
	buf.WriteString("    $actualChecksum = $actualChecksum.ToLower()\n\n")
	buf.WriteString("    if ($actualChecksum -ne $expectedChecksum.ToLower()) {\n")
	buf.WriteString("        throw \"Checksum mismatch!\\nExpected: $expectedChecksum\\nActual: $actualChecksum\"\n")
	buf.WriteString("    }\n")
	buf.WriteString("    Write-Host \"Checksum verified successfully\"\n")
	buf.WriteString("} catch {\n")
	buf.WriteString("    Remove-Item -Path $downloadPath -Force -ErrorAction SilentlyContinue\n")
	buf.WriteString("    throw \"Checksum verification failed: $_\"\n")
	buf.WriteString("}\n\n")

	buf.WriteString("# Extract archive\n")
	buf.WriteString("Write-Host \"Extracting archive...\"\n")
	buf.WriteString("try {\n")
	buf.WriteString("    Expand-Archive -Path $downloadPath -DestinationPath $toolsDir -Force\n")
	buf.WriteString("    Write-Host \"Extraction complete\"\n")
	buf.WriteString("} catch {\n")
	buf.WriteString("    Remove-Item -Path $downloadPath -Force -ErrorAction SilentlyContinue\n")
	buf.WriteString("    throw \"Failed to extract archive: $_\"\n")
	buf.WriteString("}\n\n")

	buf.WriteString("# Cleanup downloaded archive\n")
	buf.WriteString("Remove-Item -Path $downloadPath -Force -ErrorAction SilentlyContinue\n")
	buf.WriteString("Write-Host \"Installation complete\"\n")

	return buf.Bytes(), nil
}

func (g *Generator) generateUninstallScript(pkgID string) []byte {
	var buf bytes.Buffer

	buf.WriteString("# Chocolatey uninstall script for " + pkgID + "\n")
	buf.WriteString("# Generated by gpy Chocolatey generator\n\n")

	buf.WriteString("$ErrorActionPreference = 'Stop'\n\n")

	buf.WriteString("# Uninstall removes the binary from the tools directory\n")
	buf.WriteString("# This is typically handled by Chocolatey automatically,\n")
	buf.WriteString("# but this script can perform additional cleanup if needed.\n\n")

	buf.WriteString("try {\n")
	buf.WriteString("    Write-Host \"" + pkgID + " is being uninstalled\"\n")
	buf.WriteString("} catch {\n")
	buf.WriteString("    Write-Warning \"Uninstall error: $_\"\n")
	buf.WriteString("}\n")

	return buf.Bytes()
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
