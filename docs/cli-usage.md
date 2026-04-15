# gpy CLI Reference

Complete reference for every `gpy` command, flag, and global option.

---

## Global Flags

These flags apply to **all** commands and must be placed before the subcommand:

```
gpy [global-flags] <command> [command-flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--config <path>` | auto-detect | Explicit path to `gpy.yaml`. Skips config discovery. |
| `--project-root <path>` | current dir | Root directory for config discovery and file output. |
| `--yes` | false | Accept all defaults; skip interactive prompts. |
| `--no-tui` | false | Disable all prompts; fail if required fields are missing. |

**Config discovery order** (when `--config` is not set):

1. `gpy.yaml` / `gpy.yml`
2. `.gpy.yaml` / `.gpy.yml`
3. `go-package-yourself.yaml` / `go-package-yourself.yml`
4. `.go-package-yourself.yaml` / `.go-package-yourself.yml`

---

## `gpy init`

Create a new `gpy.yaml` configuration file through an interactive wizard.

```bash
gpy init [flags]
```

### Flags

| Flag | Description |
|------|-------------|
| `-h`, `--help` | Show help and exit. |

### Interactive Prompts

When run without `--yes`, the wizard asks:

| Prompt | Default | Notes |
|--------|---------|-------|
| Project name | directory name | Used as the binary name. |
| GitHub repo (`owner/repo`) | auto-detect from git remote | Used to construct release download URLs. |
| Go main package path | `./cmd/<name>` | Build target passed to `go build`. |
| Enable npm package? | n | Generates `packaging/npm/` launcher. |
| → npm package name | project name | Name published to npmjs.org. |
| Enable Homebrew package? | n | Generates `packaging/homebrew/<name>.rb`. |
| → Homebrew formula name | CamelCase(name) | e.g. `my-tool` → `MyTool`. |
| Enable Chocolatey package? | n | Generates `packaging/chocolatey/<id>/`. |
| → Chocolatey package ID | project name | ID on chocolatey.org. |
| Enable Docker image? | n | Adds `docker` section to config. |
| → Docker image name | project name | Image name for `docker build`. |
| Enable GitHub Actions workflow? | y | Generates `.github/workflows/release.yml`. |

### Non-Interactive Mode

```bash
gpy init --yes
# or
gpy --project-root /path/to/project init --yes
```

Uses all defaults. Safe for CI or project scaffolding scripts.

### Output

Creates `gpy.yaml` in the project root. Existing file is **overwritten**.

---

## `gpy package`

Generate packaging artifacts from `gpy.yaml`.

```bash
gpy package [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--only <list>` | all enabled | Comma-separated generators to run: `npm`, `homebrew`, `chocolatey`, `docker`. |
| `--version <ver>` | from config | Override the version embedded in generated files. |
| `--config <path>` | auto-detect | Per-command config override (also available as global flag). |
| `--project-root <path>` | current dir | Per-command root override. |
| `-h`, `--help` | | Show help and exit. |

### Generators

| Generator | Enabled condition | Output location |
|-----------|------------------|-----------------|
| `npm` | `packages.npm.enabled: true` | `packaging/npm/<package-name>/` |
| `homebrew` | `packages.homebrew.enabled: true` | `packaging/homebrew/<formula>.rb` |
| `chocolatey` | `packages.chocolatey.enabled: true` | `packaging/chocolatey/<id>/` |
| `docker` | `packages.docker.enabled: true` | `packaging/docker/` |

When no generators are enabled and `--only` is not set, a helpful message is printed and the command exits cleanly.

### Examples

```bash
# Generate all enabled packages
gpy package

# Generate only npm and Homebrew
gpy package --only npm,homebrew

# Override version in generated files
gpy package --version 1.2.3

# Use explicit config and root
gpy --config /path/to/gpy.yaml --project-root /path/to/project package
```

---

## `gpy workflow`

Generate a GitHub Actions release workflow.

```bash
gpy workflow [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--write` | false | Write workflow to the configured file. Without this, prints to stdout. |
| `--config <path>` | auto-detect | Per-command config override. |
| `--project-root <path>` | current dir | Per-command root override. |
| `-h`, `--help` | | Show help and exit. |

### Workflow contents

The generated workflow (`release.yml`) includes:

| Job | Triggered by | Description |
|-----|-------------|-------------|
| `build` | tag push (`v*`) | Cross-compiles for all configured platforms, archives binaries, uploads to GitHub Release. |
| `publish-npm` | `build` complete | Publishes npm package to npmjs.org (requires `NPM_TOKEN` secret). |
| `publish-homebrew` | `build` complete | Creates PR to `homebrew/homebrew-core` (requires `HOMEBREW_CORE_PAT` secret). |
| `publish-chocolatey` | `build` complete | Pushes to chocolatey.org on Windows runner (requires `CHOCOLATEY_API_KEY` secret). |
| `publish-docker` | `build` complete | Builds and pushes Docker image to registry (requires `DOCKER_USERNAME`/`DOCKER_PASSWORD` secrets). |

Publishing jobs are **conditionally included** — they only appear in the workflow when the corresponding package is enabled in `gpy.yaml`.

### Examples

```bash
# Preview workflow YAML
gpy workflow

# Write to .github/workflows/release.yml
gpy workflow --write

# Write from a different directory
gpy --project-root /path/to/project workflow --write
```

> **Note:** `--write` will not overwrite an existing workflow file. Delete it first if you want to regenerate.

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Configuration error, validation failure, or generator error |

---

## Common Patterns

```bash
# First-time setup (interactive)
gpy init
gpy package
gpy workflow --write

# CI/CD setup (non-interactive)
gpy --project-root . init --yes
gpy package --only npm,homebrew,chocolatey

# Iterate on a single generator
gpy package --only docker

# Inspect workflow before writing
gpy workflow | less
```
