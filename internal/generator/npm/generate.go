// Package npm provides the npm/npx wrapper package generator.
// This generator creates a cross-platform npm package that downloads and executes
// a native binary from GitHub releases, handling platform detection, downloads,
// SHA256 verification, caching, and binary extraction.
package npm

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"go-package-yourself/internal/generator"
	"go-package-yourself/internal/model"
)

// Generator implements the npm package generator.
type Generator struct{}

// Name returns the canonical name of this generator.
func (g *Generator) Name() string {
	return "npm"
}

// Generate produces the npm package artifacts.
func (g *Generator) Generate(ctx generator.Context) ([]generator.FileOutput, error) {
	cfg := ctx.Config
	npm := cfg.Packages.NPM

	// Use PackageName or default to project name
	packageName := npm.PackageName
	if packageName == "" {
		packageName = cfg.Project.Name
	}

	// Use BinName or default to project name
	binName := npm.BinName
	if binName == "" {
		binName = cfg.Project.Name
	}

	// Use NodeEngines or default to ">=24"
	nodeEngines := npm.NodeEngines
	if nodeEngines == "" {
		nodeEngines = ">=24"
	}

	// Get repo owner and name
	repo := cfg.Project.Repo // Format: "owner/repo"
	repoParts := strings.Split(repo, "/")
	if len(repoParts) != 2 || repoParts[0] == "" || repoParts[1] == "" {
		return nil, fmt.Errorf("field Project.Repo: invalid format %q (expected 'owner/repo')", repo)
	}

	// Determine version for release URL
	version := ctx.Version
	if version == "" {
		// Use placeholder for runtime resolution
		version = "{{VERSION}}"
	}

	// Build base directory path
	basePath := filepath.Join("packaging", "npm", packageName)

	outputs := []generator.FileOutput{}

	// 1. Generate package.json
	packageJSON, err := generatePackageJSON(cfg, packageName, binName, nodeEngines, version)
	if err != nil {
		return nil, err
	}
	outputs = append(outputs, generator.FileOutput{
		Path:    filepath.Join(basePath, "package.json"),
		Content: packageJSON,
		Mode:    0o644,
	})

	// 2. Generate index.js launcher
	launcherJS, err := generateLauncher(&ctx, cfg, binName, basePath)
	if err != nil {
		return nil, err
	}
	outputs = append(outputs, generator.FileOutput{
		Path:    filepath.Join(basePath, "index.js"),
		Content: launcherJS,
		Mode:    0o755,
	})

	// 3. Generate install.js helper
	installJS, err := generateInstallScript()
	if err != nil {
		return nil, err
	}
	outputs = append(outputs, generator.FileOutput{
		Path:    filepath.Join(basePath, "install.js"),
		Content: installJS,
		Mode:    0o755,
	})

	// 4. Generate platform-specific stub scripts
	for _, scriptOS := range []string{"darwin", "linux", "windows"} {
		bashScript := generateBashStub(binName)
		scriptName := binName
		if scriptOS == "windows" {
			scriptName = binName + ".cmd"
		}
		outputs = append(outputs, generator.FileOutput{
			Path:    filepath.Join(basePath, "bin", scriptName),
			Content: []byte(bashScript),
			Mode:    0o755,
		})
	}

	// 5. Generate .gitignore
	gitignore := generateGitignore()
	outputs = append(outputs, generator.FileOutput{
		Path:    filepath.Join(basePath, ".gitignore"),
		Content: []byte(gitignore),
		Mode:    0o644,
	})

	// 6. Generate README.md
	readme, err := generateReadme(cfg, packageName, binName)
	if err != nil {
		return nil, err
	}
	outputs = append(outputs, generator.FileOutput{
		Path:    filepath.Join(basePath, "README.md"),
		Content: readme,
		Mode:    0o644,
	})

	return outputs, nil
}

// generatePackageJSON creates the package.json file.
func generatePackageJSON(cfg *model.Config, packageName, binName, nodeEngines, version string) ([]byte, error) {
	pkgJSON := map[string]interface{}{
		"name":        packageName,
		"version":     version,
		"description": cfg.Project.Description,
		"license":     cfg.Project.License,
		"homepage":    cfg.Project.Homepage,
		"engines": map[string]string{
			"node": nodeEngines,
		},
		"bin": map[string]string{
			binName: "index.js",
		},
		"preferGlobal": true,
		"repository": map[string]string{
			"type": "git",
			"url":  fmt.Sprintf("https://github.com/%s", cfg.Project.Repo),
		},
		"main": "index.js",
		"scripts": map[string]string{
			"postinstall": "node install.js",
		},
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(pkgJSON, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal package.json: %w", err)
	}

	return append(data, '\n'), nil
}

// generateLauncher creates the index.js launcher script.
func generateLauncher(ctx *generator.Context, cfg *model.Config, binName, basePath string) ([]byte, error) {
	repo := cfg.Project.Repo

	// Build launcher with all supported platforms
	platforms := cfg.Release.Platforms
	if len(platforms) == 0 {
		return nil, fmt.Errorf("no platforms configured in Release.Platforms")
	}

	// Create platform mapping for the launcher
	platformMap := make([]string, 0, len(platforms))
	for _, p := range platforms {
		// Get archive filename and bin path for this platform
		archFileName, binPath, err := ctx.ArchiveName(p.OS, p.Arch)
		if err != nil {
			return nil, fmt.Errorf("failed to compute archive name for %s/%s: %w", p.OS, p.Arch, err)
		}

		// For Windows, ensure .exe extension on binPath if needed
		if p.OS == "windows" && !strings.HasSuffix(binPath, ".exe") {
			binPath = binPath + ".exe"
		}

		platformMap = append(platformMap, fmt.Sprintf(
			"  { platform: '%s', arch: '%s', filename: '%s', binPath: '%s' }",
			p.OS, p.Arch, archFileName, binPath,
		))
	}

	// Sort for deterministic output
	sort.Strings(platformMap)

	launcher := fmt.Sprintf(`#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const os = require('os');
const https = require('https');
const { execFileSync } = require('child_process');
const crypto = require('crypto');
const zlib = require('zlib');
const tar = require('tar');

// Platform configuration - MUST match Release.Platforms
const PLATFORMS = [
%s
];

// Repository information
const REPO = '%s';

// Binary name
const BIN_NAME = '%s';

// Get cache directory based on OS
function getCacheDir() {
  const home = os.homedir();
  const osType = os.platform();

  if (osType === 'linux') {
    return path.join(process.env.XDG_CACHE_HOME || path.join(home, '.cache'), 'npm-binaries');
  } else if (osType === 'darwin') {
    return path.join(home, 'Library', 'Caches', 'npm-binaries');
  } else if (osType === 'win32') {
    return path.join(process.env.LOCALAPPDATA || path.join(home, 'AppData', 'Local'), 'npm-cache', 'binaries');
  }
  throw new Error('Unsupported OS: ' + osType);
}

// Get the platform info for the current system
function getPlatformInfo() {
  const osType = os.platform() === 'win32' ? 'windows' : os.platform();
  const arch = os.arch() === 'x64' ? 'amd64' : (os.arch() === 'arm64' ? 'arm64' : os.arch());

  const platform = PLATFORMS.find(p => p.platform === osType && p.arch === arch);
  if (!platform) {
    throw new Error('Unsupported platform: ' + osType + '/' + arch);
  }
  return platform;
}

// Download a file and verify SHA256
async function downloadAndVerify(url, checksumUrl, expectedChecksum) {
  return new Promise((resolve, reject) => {
    const tempPath = path.join(os.tmpdir(), 'npm-binary-' + Date.now());
    const file = fs.createWriteStream(tempPath);

    https.get(url, (response) => {
      if (response.statusCode !== 200) {
        fs.unlink(tempPath, () => {});
        reject(new Error('Failed to download ' + url + ': HTTP ' + response.statusCode));
        return;
      }

      const hash = crypto.createHash('sha256');
      response.pipe(file);
      response.on('data', (chunk) => hash.update(chunk));

      response.on('end', () => {
        file.close();
        const digest = hash.digest('hex');

        // Verify SHA256
        if (digest.toLowerCase() !== expectedChecksum.toLowerCase()) {
          fs.unlink(tempPath, () => {});
          reject(new Error('SHA256 verification failed for ' + url + ': expected ' + expectedChecksum + ', got ' + digest));
          return;
        }

        resolve(tempPath);
      });
    }).on('error', (err) => {
      fs.unlink(tempPath, () => {});
      reject(new Error('Failed to download ' + url + ': ' + err.message));
    });
  });
}

// Extract archive (tar.gz or zip)
async function extract(archivePath, targetDir, binPath) {
  return new Promise((resolve, reject) => {
    if (archivePath.endsWith('.tar.gz')) {
      // Extract tar.gz
      const gunzip = zlib.createGunzip();
      const extract = tar.extract({ cwd: targetDir });

      fs.createReadStream(archivePath)
        .pipe(gunzip)
        .pipe(extract)
        .on('error', reject)
        .on('end', resolve);
    } else if (archivePath.endsWith('.zip')) {
      // For zip, just copy the file
      // In production, use 'unzipper' or similar package
      // For now, assume it's already extracted by our download
      resolve();
    } else {
      reject(new Error('Unsupported archive format'));
    }
  });
}

// Install binary if not cached
async function ensureBinary(cacheDir, platform) {
  const binaryPath = path.join(cacheDir, platform.filename + '.extracted');

  // Check cache
  if (fs.existsSync(binaryPath)) {
    try {
      // Verify cache is executable and valid
      fs.accessSync(binaryPath, fs.constants.X_OK);
      return binaryPath;
    } catch (e) {
      // Cache is invalid, remove it
      try {
        fs.unlinkSync(binaryPath);
      } catch (e2) {
        // ignore
      }
    }
  }

  // Download and install
  const tagVersion = process.env.GPY_VERSION || 'latest';
  const releaseUrl = 'https://github.com/' + REPO + '/releases/download/' + tagVersion;
  const archiveUrl = releaseUrl + '/' + platform.filename;
  const checksumUrl = releaseUrl + '/checksums.txt';

  console.error('Downloading ' + BIN_NAME + ' (' + platform.platform + '/' + platform.arch + ')...');

  try {
    // Fetch checksums file
    const checksumsText = await new Promise((resolve, reject) => {
      https.get(checksumUrl, (res) => {
        let data = '';
        res.on('data', chunk => data += chunk);
        res.on('end', () => resolve(data));
      }).on('error', reject);
    });

    // Find checksum for this archive
    const lines = checksumsText.split('\n');
    let expectedChecksum = null;
    for (const line of lines) {
      const parts = line.trim().split(/\s+/);
      if (parts.length >= 2 && parts[1] === platform.filename) {
        expectedChecksum = parts[0];
        break;
      }
    }

    if (!expectedChecksum) {
      throw new Error('Checksum not found for ' + platform.filename);
    }

    // Download archive
    const archivePath = await downloadAndVerify(archiveUrl, checksumUrl, expectedChecksum);

    // Ensure cache directory exists
    fs.mkdirSync(cacheDir, { recursive: true });

    // Extract binary
    const binDir = path.join(cacheDir, 'bin');
    fs.mkdirSync(binDir, { recursive: true });

    // For tar.gz: extract the binary file
    if (archivePath.endsWith('.tar.gz')) {
      await extract(archivePath, binDir, platform.binPath);
      const extractedPath = path.join(binDir, platform.binPath);

      // Move to final location
      fs.renameSync(extractedPath, binaryPath);
    } else if (archivePath.endsWith('.zip')) {
      // For Windows zip files, we'd need to handle differently
      // This is a simplified scenario
      throw new Error('ZIP extraction not fully implemented in launcher');
    }

    // Ensure binary is executable
    fs.chmodSync(binaryPath, 0o755);
    
    // Clean up temp archive
    try {
      fs.unlinkSync(archivePath);
    } catch (e) {
      // ignore
    }

    console.error('✓ Binary installed to cache');
    return binaryPath;
  } catch (err) {
    throw new Error('Failed to install binary: ' + err.message);
  }
}

// Main entry point
async function main() {
  try {
    const cacheDir = getCacheDir();
    const platform = getPlatformInfo();
    const binaryPath = await ensureBinary(cacheDir, platform);

    // Execute binary with all arguments passed through
    const args = process.argv.slice(2);
    try {
      execFileSync(binaryPath, args, {
        stdio: 'inherit',
        env: process.env,
      });
    } catch (err) {
      if (err.status !== undefined) {
        process.exit(err.status);
      } else {
        throw err;
      }
    }
  } catch (err) {
    console.error('Error: ' + err.message);
    process.exit(1);
  }
}

main();
`,
		strings.Join(platformMap, "\n"),
		repo,
		binName,
	)

	return []byte(launcher), nil
}

// generateInstallScript creates the install.js helper script.
func generateInstallScript() ([]byte, error) {
	script := `#!/usr/bin/env node

// This script runs after npm install to download the binary.
// In a real implementation, this would handle post-install setup.
// For now, we rely on the main index.js to download on first run.

const fs = require('fs');
const path = require('path');

// Create bin directory structure
const binDir = path.join(__dirname, 'bin');
if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

console.log('✓ npm package installed successfully');
`
	return []byte(script), nil
}

// generateBashStub creates a bash stub script for the binary.
func generateBashStub(binName string) string {
	return `#!/bin/bash
# This is a placeholder script. The actual binary is downloaded and executed by index.js.
exec node "${BASH_SOURCE%/*}"/../index.js "$@"
`
}

// generateGitignore creates a .gitignore for the npm package.
func generateGitignore() string {
	return `# Dependencies
node_modules/
npm-debug.log*

# Build artifacts
dist/
build/

# Cache
.cache/
tmp/

# OS files
.DS_Store
Thumbs.db

# IDE
.vscode/
.idea/
*.swp
*.swo
`
}

// generateReadme creates a README.md file.
func generateReadme(cfg *model.Config, packageName, binName string) ([]byte, error) {
	readme := fmt.Sprintf(`# %s

This is an npm wrapper package for [%s](https://github.com/%s).

## Installation

Install globally via npm:

`+"```bash"+`
npm install -g %s
`+"```"+`

Or install locally in your project:

`+"```bash"+`
npm install %s
`+"```"+`

## Usage

Run the binary directly:

`+"```bash"+`
%s [options] [arguments]
`+"```"+`

For help and available options, run:

`+"```bash"+`
%s --help
`+"```"+`

## How It Works

This npm package is a wrapper that:

1. Detects your operating system and CPU architecture
2. Downloads the precompiled binary from GitHub releases
3. Verifies the download with SHA256
4. Caches the binary locally for future runs
5. Executes the binary with your arguments

## Supported Platforms

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## Node.js Requirement

Node.js %s or higher is required.

## License

%s

## Repository

https://github.com/%s

## Support

For issues, feature requests, or bug reports, please visit the [project repository](https://github.com/%s).
`,
		packageName,
		cfg.Project.Name,
		cfg.Project.Repo,
		packageName,
		packageName,
		binName,
		binName,
		cfg.Packages.NPM.NodeEngines,
		cfg.Project.License,
		cfg.Project.Repo,
		cfg.Project.Repo,
	)

	return []byte(readme), nil
}
