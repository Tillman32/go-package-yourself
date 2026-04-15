# gpy: Multi-Channel Go Binary Packager

**Generate production-ready packages for npm, Homebrew, Chocolatey, and GitHub Actions — from a single YAML config.**

gpy is a command-line tool for Go developers who want to distribute binaries across multiple package managers with minimal effort. Write a config file once, run `gpy`, and get release artifacts for npm/npx, Homebrew, Chocolatey, and GitHub Actions workflows.

[![Go](https://img.shields.io/badge/Go-1.22%2B-blue)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-154%2B-brightgreen.svg)](#testing)
![Status](https://img.shields.io/badge/status-production%20ready-brightgreen.svg)

---

## Why gpy?

Packaging Go binaries for multiple package managers is **tedious and error-prone**:

- **npm**: Requires a wrapper script in Node, SHA256 verification, architecture detection
- **Homebrew**: Ruby DSL, platform-specific build blocks, SHA256 hashes  
- **Chocolatey**: PowerShell verification, XML `.nuspec` metadata, Windows-specific handling
- **GitHub Actions**: CI/CD workflow YAML to run `go build` on multiple platforms, archive binaries, compute checksums

gpy eliminates this busywork. Define your project once in YAML, and gpy generates:
- ✅ npm launcher package (with SHA256 verification)
- ✅ Homebrew formula (with platform-specific blocks)
- ✅ Chocolatey package (Windows-ready)
- ✅ GitHub Actions workflow (multi-platform CI/CD)
- ✅ Checksums file (SHA256, goreleaser format)

**One config. All platforms.**

---

## Quick Start

### 1. Initialize Config

```bash
go run github.com/brandon/go-package-yourself@latest init
```

Interactive prompts create a `gpy.yaml` config file in your project root.

### 2. Generate Artifacts

```bash
go run github.com/brandon/go-package-yourself@latest package
```

gpy generates four files:
- `package.json` (npm launcher)
- `homebrew.rb` (Homebrew formula)
- `chocolatey.ps1` (Chocolatey spec)
- (GitHub Actions workflow generated separately)

### 3. (Optional) Generate GitHub Workflow

```bash
go run github.com/brandon/go-package-yourself@latest workflow --write
```

Creates `.github/workflows/gpy-release.yaml` to auto-build and release on tag push. The workflow is reusable, so it can also be called from other workflows.

---

## Installation

### Global Install
```bash
go install github.com/brandon/go-package-yourself@latest
gpy init
```

### Ad-Hoc (No Install)
```bash
go run github.com/brandon/go-package-yourself@latest init
go run github.com/brandon/go-package-yourself@latest package
```

### From Source
```bash
git clone https://github.com/brandon/go-package-yourself
cd go-package-yourself
go build -o gpy ./cmd/gpy
./gpy init
```

---

## Workflow Overview

```
Project Root (gpy.yaml)
        |
        v
   [gpy package]
   /    |    \     \
  /     |     \     \
npm  Homebrew Chocolatey GitHub Workflow
```

1. **`gpy init`**: Interactive wizard → creates `gpy.yaml` config
2. **`gpy package`**: Reads config → generates npm/homebrew/chocolatey packages
3. **`gpy workflow --write`**: Generates GitHub Actions workflow for multi-platform builds

Each command is independent. Use them individually or in sequence.

---

## Configuration

### Minimal Config (`gpy.yaml`)

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

### Full Config with All Options

See [docs/config-reference.md](docs/config-reference.md) for detailed field documentation and examples.

### Configuration Details

- **Required fields**: `project.name`, `project.repo`, `go.main`
- **Platforms**: Default is darwin/linux (amd64 + arm64) + windows (amd64)
- **Placeholders**: Use `{{name}}`, `{{version}}`, `{{os}}`, `{{arch}}`, `{{ext}}` in templates
- **Defaults**: Applied automatically; override in YAML

See [docs/config-reference.md](docs/config-reference.md) for the complete configuration reference.

---

## Commands

### `gpy init` — Interactive Configuration

```bash
gpy init
```

Guided wizard to create/update your `gpy.yaml` config:
- Project metadata (name, repo, description, license)
- Go build settings (main package, ldflags, cgo)
- Release platforms (OS/arch targets)
- Package settings (npm, homebrew, chocolatey toggles)
- GitHub Actions workflow settings

Output: `gpy.yaml` in your project root.

### `gpy package` — Generate Package Artifacts

```bash
gpy package [--config <path>] [--project-root <path>] [--yes]
```

Generates npm, homebrew, and chocolatey package files:
- `package.json` (npm launcher with SHA256 verification)
- `homebrew.rb` (Homebrew formula)
- `chocolatey.ps1` (Chocolatey spec)
- `checksums.txt` (SHA256 goreleaser format)

**Flags:**
- `--config <path>`: Explicit config file (default: auto-discover)
- `--project-root <path>`: Project root directory (default: current dir)
- `--yes`: Skip confirmation prompts

### `gpy workflow` — Generate GitHub Actions Workflow

```bash
gpy workflow [--config <path>] [--project-root <path>] [--write]
```

Generates GitHub Actions workflow YAML (`.github/workflows/gpy-release.yaml`):
- Matrix strategy for all configured platforms
- Automatic binary builds and archiving
- Checksum generation (SHA256)
- GitHub Release upload
- Reusable workflow support (can be called from other workflows)

**Flags:**
- `--config <path>`: Explicit config file (default: auto-discover)
- `--project-root <path>`: Project root directory (default: current dir)
- `--write`: Write workflow file (default: print to stdout)

---

## Example Projects

Real projects using gpy:

- **[example-cli](https://github.com/brandon/example-cli)** — Simple CLI, minimal config
- **[buildtool](https://github.com/acme/buildtool)** — Advanced: custom platforms, arm64 support, ldflags

See [docs/examples/](docs/examples/) for annotated config files.

---

## Documentation

- **[CLI Reference](docs/cli-usage.md)** — All commands, flags, prompts, and examples
- **[Configuration Reference](docs/config-reference.md)** — Complete field documentation, defaults, validation rules, and examples
- **[Architecture Guide](docs/architecture.md)** — Internal design, generator interface, and extending gpy
- **[Troubleshooting](docs/troubleshooting.md)** — Common errors and solutions
- **[Contributing](CONTRIBUTING.md)** — Development setup, adding features, testing

---

## Features

### Generators (v1.0.0)

- ✅ **npm/npx** — Node.js launcher with SHA256 verification, auto-cache, install-on-first-run
- ✅ **Homebrew** — Ruby formula with platform-specific build blocks
- ✅ **Chocolatey** — PowerShell verification, Windows-native packaging
- ✅ **GitHub Actions** — Multi-platform CI/CD workflow with auto-release

### Quality

- 154+ comprehensive tests with race detection
- Zero external dependencies (stdlib + gopkg.in/yaml.v3)
- Deterministic output (reproducible builds)
- Cross-platform support (darwin, linux, windows)
- Error messages with field context for debugging

### Built-in Defaults

- Go: CGO disabled by default
- Release: default platforms = darwin/linux (amd64+arm64) + windows (amd64)
- Archive: `{{name}}_{{version}}_{{os}}_{{arch}}.tar.gz` (or `.zip` on windows)
- Checksums: SHA256 in goreleaser format (mandatory)
- GitHub: Triggered on tag push (`v*` pattern)

---

## Testing

Run the full test suite:

```bash
go test -race ./...
```

154+ tests covering:
- Config loading and validation
- Template rendering and placeholders
- Archive naming for all platforms
- npm launcher (SHA256 verification, caching)
- Homebrew formula generation
- Chocolatey package generation
- GitHub Actions workflow generation
- End-to-end integration tests

---

## Key Behaviors

### Config Discovery
gpy searches for config in this order:
1. Explicit `--config` flag
2. `gpy.yaml` / `gpy.yml`
3. `.gpy.yaml` / `.gpy.yml`
4. `go-package-yourself.yaml` / `go-package-yourself.yml`
5. `.go-package-yourself.yaml` / `.go-package-yourself.yml`

### Archives
- macOS: `.tar.gz` (gzip tarball)
- Linux: `.tar.gz` (gzip tarball)
- Windows: `.zip` (always; required by Chocolatey)

### Verification
- npm launcher: SHA256 verification before execution (mandatory)
- Chocolatey: PowerShell SHA256 check (mandatory)
- Homebrew: SHA256 in formula (standard practice)

### Binary Locations
- Inside archive: `{{name}}` (at root) or customize via `archive.binPathInArchive`
- npm cache: `$NPM_CONFIG_CACHE/.gpy-cache/<name>-<version>-<os>-<arch>`
- Homebrew: `/usr/local/bin/{{name}}` (Homebrew standard)

---

## Limitations & Guarantees

### v1.0.0 Limitations
- Single binary per project (no multi-binary support)
- Archive format: tar.gz or zip (no 7z, rar, etc.)
- Checksum algorithm: SHA256 fixed (not configurable)
- GitHub Actions: tag-triggered releases only (no manual trigger)

### Guarantees
- ✅ Output is deterministic (same config → same artifact content)
- ✅ All generated YAML/JSON is valid syntax
- ✅ No breaking changes in v1.x (semver)
- ✅ Zero dependencies in generated code (npm launcher, formulas, etc.)

---

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Development setup
- Project structure
- Adding new features
- Testing guidelines
- PR checklist

---

## License

MIT — See [LICENSE](LICENSE)

---

## Status

**v1.0.0** — Production ready. All generators tested and validated.

- Latest release: [v1.0.0](https://github.com/brandon/go-package-yourself/releases/tag/v1.0.0)
- Issue tracker: [GitHub Issues](https://github.com/brandon/go-package-yourself/issues)
- Changelog: [CHANGELOG.md](CHANGELOG.md)
