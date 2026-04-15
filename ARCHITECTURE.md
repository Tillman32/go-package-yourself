# gpy Architecture

## Package Structure

### `internal/model/`
**Domain Types for YAML Configuration**

Defines all types for the gpy configuration format (v1 schema). These types bind directly to the YAML structure.

- `Config`: Top-level configuration
- `Project`, `Go`, `Release`, `Platform`, `Archive`, `Checksums`: Nested config sections
- `Packages`, `NPM`, `Homebrew`, `Chocolatey`, `GitHub`, `GitHubWorkflows`: Per-channel config

**Key Principle**: Struct tags and field names are frozen. `internal/config` applies defaults after parsing.

### `internal/generator/`
**Generator Interface and Context**

Defines the contract all artifact generators must implement:

- `Generator` interface: `Name()` and `Generate(Context) ([]FileOutput, error)`
- `FileOutput`: Represents a file to write (Path, Content, Mode)
- `Context`: Generator execution context with config, helpers, and bindings
- `Registry`: Generator discovery and execution

**Key Principle**: The interface is frozen. All generators (npm, homebrew, chocolatey, workflow) implement this identically. The CLI binds helper functions into Context before invoking generators.

### `internal/naming/`
**Canonical Archive Naming**

Single source of truth for archive filenames and binary paths inside archives. ALL generators must use this module.

- `ArchiveName(params)`: Compute archive filename and binary path
- `ExtensionFor(format, os)`: Map format to file extension
- Platform validation (OS ∈ {darwin, linux, windows}, Arch ∈ {amd64, arm64})

**Key Principle**: No per-generator naming logic. All naming goes through here to ensure consistency.

**Example**:

```go
params := ArchiveNameParams{
    Name:                "mytool",
    Version:             "1.2.3",
    OS:                  "linux",
    Arch:                "amd64",
    Format:              "tar.gz",
    ArchiveNameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
    BinPathTemplate:     "bin/{{name}}",
}
archive, binPath, err := ArchiveName(params)
// archive: "mytool_1.2.3_linux_amd64.tar.gz"
// binPath: "bin/mytool"
```

### `internal/templatex/`
**Deterministic Template Rendering**

Fast, allocation-efficient template rendering for config placeholders.

- `Renderer`: Handles `{{placeholder}}` substitution
- Supported placeholders: `name`, `version`, `os`, `arch`, `ext`
- Unknown placeholders → hard error with field context
- Pre-compile validation available

**Key Principle**: Field context in error messages helps users find mistakes in their YAML.

**Example Error**:

```
field Release.Archive.NameTemplate: unknown placeholder {{typo}}
(supported: {{name}}, {{version}}, {{os}}, {{arch}}, {{ext}})
```

---

## Design Principles

### 1. Performance-First
- Minimal allocations in hot paths (template rendering, archive naming)
- Use `strings.Builder` for string concatenation
- Pre-allocate maps and slices with known capacity
- No reflection; direct struct field access

### 2. Contract Frozen
- These types are immutable during v1
- Changes require a version bump and deprecation period

### 3. Error Context First
- All errors include field paths (e.g., "field Release.Archive.NameTemplate")
- Actionable messages; quote actual values
- Never generic "config error"

### 4. Determinism
- Same inputs → identical outputs, always
- No filesystem access from generators
- No environment variable dependencies

### 5. Accessibility
- Types are exported with comprehensive godoc
- Examples in comments
- Constants for valid values (OS, Arch, Formats)

---

## Using These Packages

### For Config Loading

```go
import "go-package-yourself/internal/model"

// Parse YAML
var config model.Config
err := yaml.Unmarshal(bytes, &config)

// Apply defaults
config.Go.CGO = config.Go.CGO // false by default
if config.Release.TagTemplate == "" {
    config.Release.TagTemplate = "v{{version}}"
}
```

### For Generator Implementation

```go
import (
    "go-package-yourself/internal/generator"
    "go-package-yourself/internal/naming"
    "go-package-yourself/internal/templatex"
)

// Implement Generator interface
type MyGenerator struct{}

func (g *MyGenerator) Name() string { return "mygenerator" }

func (g *MyGenerator) Generate(ctx generator.Context) ([]generator.FileOutput, error) {
    // Use context helpers (do NOT reimplement naming or templating)
    
    for _, platform := range ctx.Config.Release.Platforms {
        // Get archive name via context helper
        archive, binPath, err := ctx.ArchiveName(platform.OS, platform.Arch)
        if err != nil {
            return nil, err
        }
        
        // Render templates via context helper
        filename, err := ctx.RenderTemplate(
            ctx.Config.Release.Archive.NameTemplate,
            "Release.Archive.NameTemplate",
        )
        if err != nil {
            return nil, err
        }
        
        // Generate files
        outputs = append(outputs, generator.FileOutput{
            Path:    fmt.Sprintf("packaging/mygen/%s", filename),
            Content: []byte("file content"),
            Mode:    0o644,
        })
    }
    
    return outputs, nil
}
```

### For CLI Integration

```go
import (
    "go-package-yourself/internal/generator"
    "go-package-yourself/internal/naming"
    "go-package-yourself/internal/templatex"
)

// Create and populate context
ctx := &generator.Context{
    Config:      config,
    ProjectRoot: absProject,
    Version:     version, // "" if not specified
}

// Bind helper functions
ctx.ArchiveName = func(os, arch string) (string, string, error) {
    params := naming.ArchiveNameParams{
        Name:                config.Project.Name,
        Version:             ctx.Version,
        OS:                  os,
        Arch:                arch,
        Format:              determineFormat(config, os),
        ArchiveNameTemplate: config.Release.Archive.NameTemplate,
        BinPathTemplate:     config.Release.Archive.BinPathInArchive,
    }
    return naming.ArchiveName(params)
}

ctx.RenderTemplate = func(template, fieldPath string) (string, error) {
    r := &templatex.Renderer{
        Data: map[string]string{
            "name":    config.Project.Name,
            "version": ctx.Version,
            // Note: os, arch, ext filled in by caller context
        },
    }
    return r.RenderWithFieldPath(template, fieldPath)
}

// Invoke generators
registry := generator.NewRegistry()
registry.Register(&npm.Generator{})
registry.Register(&homebrew.Generator{})
registry.Register(&chocolatey.Generator{})
registry.Register(&workflow.Generator{})

outputs, err := registry.Get("npm").Generate(*ctx)
```

---

## Testing

### Run Tests

```bash
# All tests
go test ./internal/...

# Specific package
go test ./internal/model/
go test ./internal/naming/ -v
go test ./internal/templatex/

# With coverage
go test ./internal/... -cover

# Benchmarks
go test ./internal/naming/ -bench=. -benchmem
go test ./internal/templatex/ -bench=. -benchmem
```

### Test Coverage

All critical paths have unit tests:
- Model struct serialization/deserialization
- Template rendering (edge cases, errors, diagnostics)
- Archive naming (all platform combinations, Windows `.exe` handling)
- Validation (required fields, platform constraints)

See individual `*_test.go` files for examples.

---

## YAML Schema Example

See [PLAN.md](../PLAN.md) for the full schema. Quick example:

```yaml
schemaVersion: 1

project:
  name: mytool
  repo: owner/mytool
  license: MIT

go:
  main: ./cmd/mytool
  ldflags: "-X main.Version={{version}}"

release:
  tagTemplate: "v{{version}}"
  platforms:
    - os: darwin
      arch: arm64
    - os: linux
      arch: amd64
  archive:
    nameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}"
    format:
      default: tar.gz
      windows: zip
    binPathInArchive: bin/{{name}}

packages:
  npm:
    enabled: true
  homebrew:
    enabled: true
  chocolatey:
    enabled: false

github:
  workflows:
    enabled: true
    workflowFile: .github/workflows/release.yml
    tagPatterns: [v*]
```

---

## Frozen v1 Contracts

The following are stable for all of v1.x — do not change without a version bump:

1. **Config Schema**: `internal/model` struct tags and field names
2. **Generator Interface**: `Name()`, `Generate(Context) ([]FileOutput, error)`
3. **Naming Module**: `ArchiveName()` signature and behavior
4. **Template Rendering**: `{{name}}`, `{{version}}`, `{{os}}`, `{{arch}}`, `{{ext}}` placeholder syntax and error format
5. **Platform Matrix**: valid OS (`darwin`, `linux`, `windows`) and Arch (`amd64`, `arm64`) values
6. **File Paths**: generated artifacts go under `packaging/` (npm, homebrew, chocolatey) and `.github/workflows/` (workflow)

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────┐
│                   user YAML config                   │
└────────────────────┬────────────────────────────────┘
                     │
                     ↓
         ┌───────────────────────┐
         │    Config Loading      │
         │   (parse + validate)  │
         └────────┬──────────────┘
                  │
         parsed config: model.Config
                  │
                  ↓
    ┌──────────────── CLI Layer ───────────┐
    │      • Create Context                 │
    │      • Bind helpers (ArchiveName,     │
    │        RenderTemplate)                │
    │      • Registry of generators         │
    └───────┬──────────────────────┬────────┘
            │                      │
            │ Generate(Context)    │
            │                      │
    ┌───────┴────────┬────────────┴────────┐
    ↓                ↓                      ↓
  ┌─────────┐   ┌──────────┐        ┌──────────────┐
  │   NPM   │   │Homebrew  │        │  Chocolatey  │
  │Generator│   │Generator │        │  Generator   │
  └────┬────┘   └────┬─────┘        └──────┬───────┘
       │             │                      │
       └─────────────┴──────┬───────────────┘
              (uses)       │
                          ↓
           ┌──────────────────────────┐
           │  Shared Naming Service   │
           │  (naming.ArchiveName)    │  ← Single source of truth
           └──────────────────────────┘
           
           ┌──────────────────────────┐
           │ Template Rendering (tx)  │
           │  (templatex.Renderer)    │  ← Deterministic, errors with context
           └──────────────────────────┘
                  │
                  ↓
         []generator.FileOutput
                  │
                  ↓ (written by CLI)
           packaging/
           ├── npm/
           ├── homebrew/
           └── chocolatey/
```
