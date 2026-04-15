# Changelog

All notable changes to gpy will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Added

- **`--sync` flag** — Automatically sync generated artifacts to `gpy.yaml` (available on `package` and `workflow` commands)
- **Auto-generated workflows in init** — `gpy init` now generates a GitHub Actions workflow file automatically during setup
- **Reusable workflow support** — GitHub Actions workflows use `workflow_call` trigger for better integration with CI/CD pipelines
- **Configurable workflow filename** — Workflow generation respects `workflowFilename` config value (defaults to `gpy-release.yaml`)

### Changed

- **Workflow command improvements** — Better defaults and clearer output when generating workflows

---

## [1.0.0] - 2026-04-13

### Initial Release ✨

gpy v1.0.0 is the first stable release, featuring complete implementation of all package managers and GitHub Actions workflow generation.

#### Added

##### Core Features
- **Multi-channel packaging** — Generate packages for npm, Homebrew, Chocolatey, and GitHub Actions
- **YAML configuration** — Single config file (`gpy.yaml`) drives all package generation
- **Cross-platform support** — Build binaries for darwin (amd64/arm64), linux (amd64/arm64), and windows (amd64)
- **Interactive wizard** — `gpy init` command guides users through configuration setup

##### Generators (WS4-7)

- **npm/npx Generator**
  - JavaScript launcher package with SHA256 verification
  - Automatic binary download from GitHub Releases
  - Cross-platform cache management
  - Install-on-first-run behavior
  - Works with scoped packages (`@org/package`)
  - 14 comprehensive tests, fully validated

- **Homebrew Generator**
  - Ruby formula with platform-specific build blocks
  - Support for arm64 (Apple Silicon) and amd64 (Intel)
  - SHA256 hash verification
  - Optional custom tap support
  - 11 comprehensive tests

- **Chocolatey Generator**
  - XML `.nuspec` package manifesto
  - PowerShell SHA256 hash verification
  - Windows architecture detection
  - Custom package ID support
  - 10 comprehensive tests

- **GitHub Actions Workflow Generator**
  - Multi-platform CI/CD matrix strategy
  - Automatic binary building and archiving
  - SHA256 checksum generation (goreleaser format)
  - GitHub Release uploads
  - Configurable tag patterns
  - 11 comprehensive tests

##### Configuration System (WS1)
- Config file auto-discovery (8 filename patterns)
- YAML v1 schema with frozen contract
- Default values applied for all optional fields
- Comprehensive validation with field path errors
- Support for template placeholders (`{{name}}`, `{{version}}`, `{{os}}`, `{{arch}}`, `{{ext}}`)

##### Template & Naming Engine (WS2)
- Deterministic template rendering for configuration placeholders
- Canonical archive naming (consistent across all generators)
- Platform-specific file extensions (`.tar.gz`, `.zip`)
- Binary path templates for custom archive structures

##### CLI Commands (WS3)
- `gpy init` — Interactive configuration wizard
- `gpy package` — Generate npm/homebrew/chocolatey artifacts
- `gpy workflow` — Generate GitHub Actions release workflow
- Global flags: `--config`, `--project-root`, `--yes`, `--no-tui`

##### Quality & Testing
- 154+ comprehensive tests across all packages
- Race condition detection (`go test -race ./...`) — 0 failures
- 90%+ code coverage per package
- Deterministic output (reproducible builds)
- Zero external dependencies (except `gopkg.in/yaml.v3`)

##### Documentation (WS9)
- User-friendly README.md with quick start
- Comprehensive configuration reference (docs/config-reference.md)
- Architecture guide for contributors (docs/architecture.md)
- Annotated example configurations
- Troubleshooting guide (docs/troubleshooting.md)
- Contributing guide (CONTRIBUTING.md)

#### Key Behaviors

- **Archive Formats**
  - macOS/Linux: `.tar.gz` (gzip compressed tarball)
  - Windows: `.zip` (required by Chocolatey)
  - Can be customized per platform in config

- **Verification**
  - npm launcher: SHA256 verification before binary execution (mandatory)
  - Chocolatey: PowerShell SHA256 check (mandatory)
  - Homebrew: SHA256 in formula metadata

- **Defaults**
  - Platforms: darwin/linux (amd64 + arm64) + windows (amd64)
  - Tag template: `v{{version}}`
  - Archive name: `{{name}}_{{version}}_{{os}}_{{arch}}`
  - Checksums: SHA256 in goreleaser format

#### Limitations

- Single binary per project (no multi-binary support)
- Archive formats limited to tar.gz and zip
- Checksum algorithm fixed to SHA256 (not configurable in v1)
- GitHub Actions: tag-triggered releases only (no manual trigger)

#### Known Considerations

- Config schema (v1) is frozen for backward compatibility
- Generator interface is frozen for v1 (no API changes)
- Deterministic output guaranteed (same config → same artifacts)
- All generated files follow platform conventions (no machine paths in output)

---

## Design & Build Philosophy

### v1 is Stable

- Frozen interfaces documented in ARCHITECTURE.md
- No breaking changes throughout v1.x
- All generators share common interface + context
- Naming module centralized for consistency

### Zero Dependencies in Generated Code

- npm launcher: pure JavaScript (no external npm dependencies)
- Homebrew formula: pure Ruby (no gems required)
- Chocolatey spec: pure PowerShell (no external modules)
- GitHub Actions workflow: pure YAML (uses standard actions)

### Testing & Quality

- 154+ tests covering:
  - Config loading (all 8 file patterns)
  - Config validation (all rules and edge cases)
  - Template rendering (all placeholders)
  - Archive naming (all platform/format combinations)
  - Each generator (functionality and output)
  - End-to-end workflows
- All tests pass with race detection
- Deterministic output validated
- CI/CD integration ready

---

## Roadmap for v1.x

### Potential Future Enhancements (Non-Breaking)
- Additional package managers (Scoop, Linux distros, Docker)
- Binary signing support (gpg, codesign)
- SLSA provenance generation
- More template functions
- Pre/post-build hooks (if schema-compatible)

### v2 Plans (Breaking Changes OK)
- Multi-binary support
- Pluggable generator system
- Advanced template language
- Environment variable expansion
- Custom naming strategies

---

## Credits

Developed as a comprehensive Go CLI packaging solution to solve the tedious problem of distributing Go binaries across multiple package managers.

---

## Links

- [Homepage](https://github.com/brandon/go-package-yourself)
- [Issues](https://github.com/brandon/go-package-yourself/issues)
- [Releases](https://github.com/brandon/go-package-yourself/releases)
- [Documentation](docs/)
- [Contributing](CONTRIBUTING.md)

---

**v1.0.0** — Ready for production. All generators tested and battle-ready. 🚀
