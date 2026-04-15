# Configuration Reference

Complete documentation for `gpy.yaml` — the configuration file that drives all package generation.

---

## Overview

gpy uses YAML configuration (schema version 1) to define your project, build settings, release platforms, and package targets.

**Config Discovery:**  
gpy searches for config in this order (first found wins):
1. Explicit `--config <path>` flag
2. `gpy.yaml` / `gpy.yml`
3. `.gpy.yaml` / `.gpy.yml`  
4. `go-package-yourself.yaml` / `go-package-yourself.yml`
5. `.go-package-yourself.yaml` / `.go-package-yourself.yml`

**Validation:**
- Required fields must be provided
- Invalid platform/format combinations are rejected
- Unknown placeholders in templates raise errors with field context
- All errors include actionable messages

---

## Schema Reference

### `schemaVersion` (Required)

**Type:** integer  
**Default:** None (must be explicit)  
**Value:** `1`

```yaml
schemaVersion: 1
```

Declares the configuration schema version. v1 is the current stable schema.

---

## `project` Section (Required)

Project metadata used in all package manifests.

### `project.name` (Required)

**Type:** string  
**Length:** 1-64 characters  
**Pattern:** Alphanumeric + hyphens (no spaces, no underscores)  
**Examples:** `mycli`, `build-tool`, `task-runner`

Binary/package name, used as the executable name and in filenames.

```yaml
project:
  name: mycli
```

**Validation:** Must match `^[a-zA-Z0-9\-]+$`

### `project.repo` (Required)

**Type:** string  
**Format:** `owner/repo`  
**Examples:** `myorg/mycli`, `brandon/buildtool`

GitHub repository path. Used to construct release URLs and SCM links.

```yaml
project:
  repo: myorg/mycli
```

**Validation:** Must contain exactly one `/`

### `project.description` (Optional)

**Type:** string  
**Default:** Empty string  
**Usage:** npm package.json, Homebrew formula, Chocolatey description

Human-readable project description.

```yaml
project:
  description: "A powerful CLI tool for building"
```

### `project.homepage` (Optional)

**Type:** string  
**Default:** `https://github.com/{repo}`  
**Usage:** npm package.json, Homebrew formula

Project homepage URL.

```yaml
project:
  homepage: "https://mycli.example.com"
```

### `project.license` (Optional)

**Type:** string  
**Default:** Empty string  
**Pattern:** SPDX license identifier preferred (e.g., `MIT`, `Apache-2.0`)  
**Usage:** npm package.json, Homebrew formula

License identifier for your project.

```yaml
project:
  license: "MIT"
```

---

## `go` Section (Required)

Go build configuration.

### `go.main` (Required)

**Type:** string  
**Format:** Path to package directory  
**Examples:** `./cmd/mycli`, `./cmd/mytool`, `.`

Path to the main package for `go build`.

```yaml
go:
  main: ./cmd/mycli
```

**Validation:** Path must exist and be a valid Go package.

### `go.cgo` (Optional)

**Type:** boolean  
**Default:** `false`

Enable/disable CGO. Set to `true` if your binary uses C bindings.

```yaml
go:
  cgo: false
```

When `cgo: true`, the GitHub Actions workflow will pass `CGO_ENABLED=1`.

### `go.ldflags` (Optional)

**Type:** string  
**Default:** Empty string  
**Examples:** `-X main.Version=1.2.3`, `-s -w`

Linker flags passed to `go build -ldflags`. Use for embedding version, build time, etc.

```yaml
go:
  ldflags: "-X main.Version={{version}} -X main.BuildTime=$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
```

**Supported Placeholders:** `{{version}}` (replaced with git tag)  
The GitHub Actions workflow will set `-X main.Version=${{ github.ref_name }}` at release time.

---

## `release` Section (Optional)

Release and archive configuration.

### `release.tagTemplate` (Optional)

**Type:** string  
**Default:** `"v{{version}}"`  
**Placeholders:** `{{version}}`

Template for GitHub release tags. Used by GitHub Actions on tag push.

```yaml
release:
  tagTemplate: "v{{version}}"
```

Examples:
- `v{{version}}` → `v1.2.3`
- `release-{{version}}` → `release-1.2.3`
- `{{version}}` → `1.2.3`

### `release.platforms` (Optional)

**Type:** array of objects  
**Default:** See below  

List of target OS/arch combinations for cross-compilation.

```yaml
release:
  platforms:
    - os: darwin
      arch: arm64
    - os: darwin
      arch: amd64
    - os: linux
      arch: amd64
    - os: linux
      arch: arm64
    - os: windows
      arch: amd64
```

**Default Platforms (if omitted):**
```yaml
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
```

**Valid Values:**
- `os`: `darwin` | `linux` | `windows`
- `arch`: `amd64` | `arm64`

**Platform-Specific Notes:**
- Windows: Archive format is always `.zip` (required by Chocolatey)
- macOS (darwin): Both amd64 and arm64 supported (Apple Silicon + Intel)
- Linux: amd64 and arm64 supported

**Validation:**
- At least one platform must be defined
- Duplicate platforms are allowed but unnecessary
- Invalid os/arch combinations are rejected with error message

### `release.archive` (Optional)

Archive naming and format configuration.

#### `release.archive.nameTemplate` (Optional)

**Type:** string  
**Default:** `"{{name}}_{{version}}_{{os}}_{{arch}}"`  
**Placeholders:** `{{name}}`, `{{version}}`, `{{os}}`, `{{arch}}`, `{{ext}}`

Template for archive filenames.

```yaml
release:
  archive:
    nameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}"
```

Examples:
- `"{{name}}_{{version}}_{{os}}_{{arch}}"` → `mycli_1.2.3_linux_amd64.tar.gz`
- `"{{name}}-{{version}}-{{os}}"` → `mycli-1.2.3-linux.tar.gz` (includes platform-specific {{ext}})

**Placeholders:**
- `{{name}}` → Project name (from `project.name`)
- `{{version}}` → Version (from git tag or command line)
- `{{os}}` → Operating system (darwin, linux, windows)
- `{{arch}}` → Architecture (amd64, arm64)
- `{{ext}}` → File extension (.tar.gz, .zip — auto-determined)

**Validation:** Unknown placeholders raise an error with field context.

#### `release.archive.format` (Optional)

**Type:** object  
**Defaults:** `default: "tar.gz"`, `windows: "zip"`

Archive format per OS. Windows is always `.zip` (required by Chocolatey).

```yaml
release:
  archive:
    format:
      default: "tar.gz"
      windows: "zip"
```

**Valid Values:**
- `tar.gz` — GZip-compressed tarball (macOS, Linux)
- `zip` — ZIP archive (Windows, or any OS if desired)

**Format Extension Map:**
- `tar.gz` → `.tar.gz` (or `.tgz`)
- `zip` → `.zip`

**Validation:** 
- Windows must use `.zip` (no exceptions)
- Unknown formats raise error

#### `release.archive.binPathInArchive` (Optional)

**Type:** string  
**Default:** `"{{name}}"`  
**Placeholders:** `{{name}}`

Path to the binary inside the archive.

```yaml
release:
  archive:
    binPathInArchive: "bin/{{name}}"
```

Examples:
- `"{{name}}"` → Binary at archive root: `./mycli`
- `"bin/{{name}}"` → Binary in subdirectory: `./bin/mycli`

Used by npm launcher and Homebrew formula to locate the binary when installed.

### `release.checksums` (Optional)

**Type:** object  
**Defaults:** Algorithm and format are FIXED (v1 stability)

Checksum configuration (SHA256 only in v1).

```yaml
release:
  checksums:
    file: "checksums.txt"
    algorithm: "sha256"
    format: "goreleaser"
```

**Fixed Fields (v1):**
- `algorithm`: Always `"sha256"` (not configurable)
- `format`: Always `"goreleaser"` (not configurable)

**Configurable Fields:**
- `file`: Filename for checksums (default: `"checksums.txt"`)

**Output Format (goreleaser):**
```
abc123... archive1.tar.gz
def456... archive2.zip
```

---

## `packages` Section (Optional)

Per-package manager configuration.

### `packages.npm` (Optional)

npm/npx launcher package configuration.

#### `packages.npm.enabled` (Optional)

**Type:** boolean  
**Default:** `false`

Enable npm package generation.

```yaml
packages:
  npm:
    enabled: true
```

#### `packages.npm.packageName` (Optional)

**Type:** string  
**Default:** `project.name`

The published npm package name.

```yaml
packages:
  npm:
    packageName: "mycli"
```

**Common Patterns:**
- Unscoped: `mycli`
- Scoped: `@myorg/mycli`

#### `packages.npm.binName` (Optional)

**Type:** string  
**Default:** `project.name`

The CLI command name when installed globally (`npm install -g`).

```yaml
packages:
  npm:
    binName: "mycli"
```

After `npm install -g mycli`, users can run `mycli` from any terminal.

#### `packages.npm.nodeEngines` (Optional)

**Type:** string  
**Default:** `">=18"`

Node.js version requirement in package.json `engines.node` field.

```yaml
packages:
  npm:
    nodeEngines: ">=18"
```

**Examples:**
- `">=18"` — Node.js 18 or later
- `">=16"` — Node.js 16 or later
- `"18.x"` — Node.js 18.x only

---

### `packages.homebrew` (Optional)

Homebrew formula configuration.

#### `packages.homebrew.enabled` (Optional)

**Type:** boolean  
**Default:** `false`

Enable Homebrew formula generation.

```yaml
packages:
  homebrew:
    enabled: true
```

#### `packages.homebrew.tap` (Optional)

**Type:** string  
**Default:** Empty (user installs from GitHub Releases directly)

Homebrew tap (repository) name. If provided, the formula is added to a tap for users to install from.

```yaml
packages:
  homebrew:
    tap: "myorg/homebrew-tools"
```

**Format:** `username/homebrew-name` (the `homebrew-` prefix is standard)

#### `packages.homebrew.formulaName` (Optional)

**Type:** string  
**Default:** `project.name`

The Homebrew formula name (how users install: `brew install formulaName`).

```yaml
packages:
  homebrew:
    formulaName: "mycli"
```

---

### `packages.chocolatey` (Optional)

Chocolatey package configuration.

#### `packages.chocolatey.enabled` (Optional)

**Type:** boolean  
**Default:** `false`

Enable Chocolatey package generation.

```yaml
packages:
  chocolatey:
    enabled: true
```

#### `packages.chocolatey.packageId` (Optional)

**Type:** string  
**Default:** `project.name`

The Chocolatey package ID (how users install: `choco install packageId`).

```yaml
packages:
  chocolatey:
    packageId: "mycli"
```

#### `packages.chocolatey.authors` (Optional)

**Type:** string  
**Default:** Empty string

Author name(s) for the `.nuspec` metadata.

```yaml
packages:
  chocolatey:
    authors: "Your Name"
```

---

## `github` Section (Optional)

GitHub Actions workflow configuration.

### `github.workflows` (Optional)

GitHub Actions workflow settings.

#### `github.workflows.enabled` (Optional)

**Type:** boolean  
**Default:** `false`

Enable GitHub Actions workflow generation.

```yaml
github:
  workflows:
    enabled: true
```

#### `github.workflows.workflowFile` (Optional)

**Type:** string  
**Default:** `".github/workflows/gpy-release.yaml"`

Output path for the GitHub Actions workflow file.

```yaml
github:
  workflows:
    workflowFile: ".github/workflows/gpy-release.yaml"
```

The file is written when you run `gpy workflow --write`.

#### `github.workflows.tagPatterns` (Optional)

**Type:** array of strings  
**Default:** `["v*"]`

Glob patterns that trigger the release workflow on tag push.

```yaml
github:
  workflows:
    tagPatterns: ["v*", "release-*"]
```

Examples:
- `["v*"]` — Tags like `v1.0.0`, `v1.2.3`
- `["release-*"]` — Tags like `release-1.0.0`
- `["*"]` — All tags trigger releases

---

## Examples

### Example 1: Minimal Configuration

```yaml
schemaVersion: 1

project:
  name: mycli
  repo: myorg/mycli
  description: "A simple CLI tool"

go:
  main: ./cmd/mycli

packages:
  npm:
    enabled: true
  homebrew:
    enabled: true
  chocolatey:
    enabled: true
```

**Results:**
- Binary name: `mycli`
- npm package name: `mycli`
- Homebrew formula name: `mycli`
- Chocolatey package ID: `mycli`
- Platforms: darwin/linux (amd64 + arm64) + windows (amd64)
- Archive format: `.tar.gz` (or `.zip` on windows)

---

### Example 2: Full Configuration with All Options

```yaml
schemaVersion: 1

project:
  name: buildtool
  repo: acme/buildtool
  description: "ACME's build orchestration tool"
  homepage: "https://buildtool.example.com"
  license: "Apache-2.0"

go:
  main: ./cmd/buildtool
  cgo: false
  ldflags: "-X main.Version={{version}} -X main.BuildTime=$(date -u +'%Y-%m-%dT%H:%M:%SZ')"

release:
  tagTemplate: "release-{{version}}"
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
    nameTemplate: "{{name}}-{{version}}-{{os}}-{{arch}}"
    format:
      default: "tar.gz"
      windows: "zip"
    binPathInArchive: "bin/{{name}}"
  checksums:
    file: "checksums.txt"
    algorithm: "sha256"
    format: "goreleaser"

packages:
  npm:
    enabled: true
    packageName: "@acme/buildtool"
    binName: "buildtool"
    nodeEngines: ">=18.0"
  homebrew:
    enabled: true
    tap: "acme/homebrew-tools"
    formulaName: "BuildTool"
  chocolatey:
    enabled: true
    packageId: "acme-buildtool"
    authors: "ACME Corp"

github:
  workflows:
    enabled: true
    workflowFile: ".github/workflows/gpy-release.yaml"
    tagPatterns: ["release-*"]
```

---

### Example 3: Minimal with Custom Platforms

Only build for specific platforms (useful for light, quick releases).

```yaml
schemaVersion: 1

project:
  name: devtool
  repo: myorg/devtool

go:
  main: ./cmd/devtool

release:
  platforms:
    - os: darwin
      arch: amd64
    - os: linux
      arch: amd64

packages:
  npm:
    enabled: true
  homebrew:
    enabled: true
```

---

### Example 4: Custom Archive Naming with Placeholders

```yaml
schemaVersion: 1

project:
  name: mytool
  repo: myorg/mytool

go:
  main: ./cmd/mytool

release:
  archive:
    nameTemplate: "{{name}}-{{version}}-{{os}}-{{arch}}-build"
    binPathInArchive: "tools/{{name}}/bin/{{name}}"

packages:
  npm:
    enabled: true
```

**Generated Archive Names:**
- `mytool-1.2.3-darwin-amd64-build.tar.gz`
- `mytool-1.2.3-linux-amd64-build.tar.gz`
- `mytool-1.2.3-windows-amd64-build.zip`

**Binary Location Inside Archive:**
- `tools/mytool/bin/mytool`

---

## Validation Rules

### Required Fields
- `schemaVersion` (must be 1)
- `project.name` (1-64 alphanumeric + hyphens)
- `project.repo` (`owner/repo` format)
- `go.main` (path to main package)

### Conditional Requirements
- If `packages.npm.enabled: true`, npm config is validated
- If `packages.homebrew.enabled: true`, homebrew config is validated
- If `packages.chocolatey.enabled: true`, chocolatey config is validated
- If `github.workflows.enabled: true`, GitHub config is validated

### Platform Validation
- `os` must be `darwin`, `linux`, or `windows`
- `arch` must be `amd64` or `arm64`
- Windows must use `.zip` format (not `.tar.gz`)

### Template Placeholder Validation
- Unknown placeholders in `nameTemplate`, `tagTemplate`, `ldflags`, `binPathInArchive` raise an error
- Error message includes: field path, invalid placeholder, and supported placeholders

### Archive Format Validation
- `default` and `windows` must be valid format strings (`tar.gz` or `zip`)
- Windows format must be `zip`

---

## Error Messages

### Config Not Found
```
error: config file not found in search paths:
  (1) gpy.yaml
  (2) gpy.yml
  (3) .gpy.yaml
  (4) .gpy.yml
  (5) go-package-yourself.yaml
  (6) go-package-yourself.yml
  (7) .go-package-yourself.yaml
  (8) .go-package-yourself.yml

use --config flag to specify explicit path
```

### Invalid Field
```
error: field Release.Archive.NameTemplate: unknown placeholder {{typo}}
(supported: {{name}}, {{version}}, {{os}}, {{arch}}, {{ext}})
```

### Invalid Platform
```
error: field Release.Platforms[0].OS: invalid value "macos" (must be "darwin", "linux", or "windows")
```

---

## Tips & Tricks

### Multi-platform with Custom Naming

Generate different archive names per platform:

```yaml
release:
  archive:
    nameTemplate: "{{name}}_v{{version}}_{{os}}_{{arch}}"
```

### Version Embedding in Binary

Use ldflags to embed the version at build time:

```yaml
go:
  ldflags: "-X main.Version={{version}}"
```

The GitHub Actions workflow automatically replaces `{{version}}` with the git tag.

### Custom Binary Path in Archive

For non-root binaries:

```yaml
release:
  archive:
    binPathInArchive: "dist/{{name}}/bin/{{name}}"
```

This tells generators (npm, homebrew, chocolatey) to extract the binary from this path.

### Scoped npm Package

For organization-scoped packages:

```yaml
packages:
  npm:
    packageName: "@myorg/mycli"
    binName: "mycli"
```

After `npm install -g @myorg/mycli`, users run `mycli` (not `@myorg/mycli`).

---

## Defaults Table

| Field | Default |
|-------|---------|
| `go.cgo` | `false` |
| `go.ldflags` | Empty string |
| `release.tagTemplate` | `"v{{version}}"` |
| `release.platforms` | 5 default platforms (see above) |
| `release.archive.nameTemplate` | `"{{name}}_{{version}}_{{os}}_{{arch}}"` |
| `release.archive.format.default` | `"tar.gz"` |
| `release.archive.format.windows` | `"zip"` |
| `release.archive.binPathInArchive` | `"{{name}}"` |
| `release.checksums.file` | `"checksums.txt"` |
| `release.checksums.algorithm` | `"sha256"` (fixed) |
| `release.checksums.format` | `"goreleaser"` (fixed) |
| `packages.npm.enabled` | `false` |
| `packages.npm.packageName` | `project.name` |
| `packages.npm.binName` | `project.name` |
| `packages.npm.nodeEngines` | `">=18"` |
| `packages.homebrew.enabled` | `false` |
| `packages.homebrew.tap` | Empty string |
| `packages.homebrew.formulaName` | `project.name` |
| `packages.chocolatey.enabled` | `false` |
| `packages.chocolatey.packageId` | `project.name` |
| `packages.chocolatey.authors` | Empty string |
| `github.workflows.enabled` | `false` |
| `github.workflows.workflowFile` | `".github/workflows/gpy-release.yaml"` |
| `github.workflows.tagPatterns` | `["v*"]` |
