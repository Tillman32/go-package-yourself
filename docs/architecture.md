# Architecture Guide

This guide explains how gpy works internally, how to extend it, and how the codebase is organized.

---

## High-Level Overview

gpy follows a simple pipeline:

```
YAML Config
    |
    v
Config Loading + Validation
    |
    v
Registry of Generators
    |
    v
Generator Execution (npm, homebrew, chocolatey, workflow)
    |
    v
File Output (write to disk or stdout)
```

Each step is independent and testable. The core design principle: **generators are pluggable components that implement a common interface**.

---

## Project Structure

```
go-package-yourself/
├── cmd/gpy/
│   └── main.go                    # Entry point, CLI setup
├── internal/
│   ├── cli/
│   │   ├── cli.go                 # Root command, flag parsing
│   │   ├── init.go                # `gpy init` command
│   │   ├── package.go             # `gpy package` command
│   │   └── workflow.go            # `gpy workflow` command
│   ├── config/
│   │   ├── load.go                # Config file discovery + parsing
│   │   ├── defaults.go            # Default values for all fields
│   │   ├── config_test.go         # Config loading tests
│   │   └── integration_test.go    # End-to-end config tests
│   ├── model/
│   │   ├── config.go              # Config structs (YAML schema binding)
│   │   └── config_test.go         # Model struct tests
│   ├── generator/
│   │   ├── generator.go           # Generator interface + registry
│   │   ├── npm/
│   │   │   ├── generate.go        # npm launcher package generator
│   │   │   └── generate_test.go   # npm generator tests
│   │   ├── homebrew/
│   │   │   ├── generate.go        # Homebrew formula generator
│   │   │   └── generate_test.go   # Homebrew tests
│   │   ├── chocolatey/
│   │   │   ├── generate.go        # Chocolatey package generator
│   │   │   └── generate_test.go   # Chocolatey tests
│   │   └── workflow/
│   │       ├── generate.go        # GitHub Actions workflow generator
│   │       └── generate_test.go   # Workflow tests
│   ├── naming/
│   │   ├── archive.go             # Archive naming + binary paths
│   │   └── archive_test.go        # Naming tests (all platform combos)
│   ├── templatex/
│   │   ├── render.go              # Template rendering ({{placeholders}})
│   │   └── render_test.go         # Template rendering tests
│   └── validate/
│       ├── config.go              # Config validation (field rules)
│       └── validate_test.go       # Validation tests
├── ARCHITECTURE.md                # Design documentation + frozen v1 contracts
└── README.md                       # User-facing documentation
```

---

## Key Modules

### `internal/model/` — Configuration Schema

Defines all types for the YAML v1 schema. These **MUST NOT CHANGE** during v1 (frozen contract).

**Key Types:**
- `Config` — Top-level config
- `Project`, `Go`, `Release`, `Platform`, `Archive`, `Checksums` — Config sections
- `Packages`, `NPM`, `Homebrew`, `Chocolatey` — Per-channel settings
- `GitHub`, `GitHubWorkflows` — Workflow settings

**Design Principle:**
- Struct tags bind directly to YAML field names
- No post-parsing transformation; mapping is 1:1
- Defaults applied later (see `internal/config/defaults.go`)

**Example:**
```go
type Config struct {
    SchemaVersion int         `yaml:"schemaVersion"`
    Project       Project     `yaml:"project"`
    Go            Go          `yaml:"go"`
    Release       Release     `yaml:"release"`
    Packages      Packages    `yaml:"packages"`
    GitHub        GitHub      `yaml:"github"`
}
```

### `internal/config/` — Config Loading & Defaults

**`load.go`:**
- Discovers config files (searches default names)
- Parses YAML into `Config` struct
- Applies defaults after parsing (see next section)
- Returns helpful error messages if config not found

**`defaults.go`:**
- Applies all default values after YAML parsing
- Called automatically by `Load()`
- Ensures all optional fields have sensible defaults

**Why separate loading and defaults?**
- Clean separation of concerns
- Easier to test
- Makes it obvious which fields have defaults
- Allows users to override defaults in YAML

**Usage:**
```go
cfg, err := config.Load("")  // empty string = auto-discover
if err != nil {
    return err  // helpful error message
}
// cfg now has all defaults applied
```

### `internal/validate/` — Config Validation

**`config.go`:**
- Validates config against rules
- Checks required fields presence
- Validates platform combinations (os/arch)
- Validates archive formats for platform compatibility
- Returns errors with field path context

**Design Principle:** Validation is **separate from loading**.
- Load succeeds even if YAML is malformed structure (parse error)
- Validation then checks semantic rules
- Allows detailed error context ("field Release.Platforms[0].OS: invalid value...")

**Usage:**
```go
if err := validate.Config(cfg); err != nil {
    return err  // actionable field path + problem
}
```

### `internal/generator/` — Generator Interface & Registry

**`generator.go` — Core Interface (FROZEN):**

```go
type Generator interface {
    Name() string
    Generate(ctx Context) ([]FileOutput, error)
}

type FileOutput struct {
    Path    string      // e.g., "package.json"
    Content string      // file content
    Mode    os.FileMode // permissions (0644, 0755, etc.)
}

type Context struct {
    Config      *model.Config
    ProjectRoot string
    // Helper functions for generators:
    ArchiveName func(...) (string, error)
    Render      func(...) (string, error)
}
```

**`registry.go` — Generator Discovery:**

```go
var generators = []Generator{
    npm.NewGenerator(),
    homebrew.NewGenerator(),
    chocolatey.NewGenerator(),
    workflow.NewGenerator(),
}

func Execute(ctx Context) ([]FileOutput, error) {
    var outputs []FileOutput
    for _, gen := range generators {
        out, err := gen.Generate(ctx)
        if err != nil {
            return nil, err
        }
        outputs = append(outputs, out...)
    }
    return outputs, nil
}
```

**Key Principles:**
- Each generator implements the same interface
- Generators are stateless (all data passed via `Context`)
- No generator knows about other generators
- Registry is defined in `generator.go` (central point for adding/removing generators)

### `internal/naming/` — Archive Naming & Binary Paths

**`archive.go` — Canonical Archive Naming:**

Single source of truth for archive filenames and binary paths inside archives.

**Key Functions:**

```go
func ArchiveName(params ArchiveNameParams) (archiveName, binPath string, err error)
func ExtensionFor(format string, os string) (ext string, err error)
```

**Design Principle:**
- All generators call this function for consistency
- No per-generator naming logic
- Platform validation happens here

**Example:**
```go
archive, binPath, _ := naming.ArchiveName(naming.ArchiveNameParams{
    Name:                "mycli",
    Version:             "1.2.3",
    OS:                  "linux",
    Arch:                "amd64",
    Format:              "tar.gz",
    ArchiveNameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
    BinPathTemplate:     "{{name}}",
})
// archive: "mycli_1.2.3_linux_amd64.tar.gz"
// binPath: "mycli"
```

### `internal/templatex/` — Template Rendering

**`render.go` — Deterministic Placeholder Substitution:**

Fast, allocation-efficient template rendering for config placeholders.

**Key Functions:**

```go
func Render(template string, bindings map[string]string) (string, error)
func RenderWithContext(template string, bindings map[string]string, fieldPath string) (string, error)
```

**Supported Placeholders:**
- `{{name}}` — Project name
- `{{version}}` — Release version
- `{{os}}` — Operating system
- `{{arch}}` — Architecture
- `{{ext}}` — File extension

**Error Context:**
```
Render("release-{{typo}}", ...) 
→ error: field Release.TagTemplate: unknown placeholder {{typo}}
  (supported: {{name}}, {{version}}, {{os}}, {{arch}}, {{ext}})
```

**Design Principle:**
- Unknown placeholders → hard error with field context
- No silent failures
- Error message helps users debug YAML

### `internal/cli/` — CLI Commands

**`cli.go` — Root Command Setup:**
- Global flags: `--config`, `--project-root`, `--no-tui`, `--yes`
- Command routing (init, package, workflow)
- Help text and usage information

**`init.go` — `gpy init` Command:**
- Interactive prompts for project metadata
- Writes config to `gpy.yaml`
- Uses terminal UI if `--no-tui` not set

**`package.go` — `gpy package` Command:**
- Loads config
- Validates config
- Executes npm/homebrew/chocolatey generators
- Writes output files or displays (depending on flags)

**`workflow.go` — `gpy workflow` Command:**
- Loads config
- Validates config
- Executes workflow generator
- Writes `.github/workflows/release.yml` if `--write` flag

---

## Data Flow: From Config to Files

### Example: `gpy package`

```
1. User runs: gpy package

2. CLI performs:
   - Parse flags (--config, --project-root, --yes)
   - Load config: config.Load(configPath)
   - Validate config: validate.Config(cfg)

3. Registry executes generators:
   - Create context (Context{Config: cfg, ...})
   - For each generator (npm, homebrew, chocolatey):
     - generator.Generate(ctx) → []FileOutput
     - []FileOutput contains {Path, Content, Mode}

4. Write outputs:
   - For each FileOutput:
     - Write to file: os.WriteFile(path, content, mode)

5. Display summary:
   - List generated files
   - Show sizes, paths
```

---

## Testing Strategy

### Unit Tests

**By Package:**
- `internal/model/config_test.go` — Config struct tests
- `internal/config/config_test.go` — Config loading tests
- `internal/config/integration_test.go` — End-to-end config loading
- `internal/validate/validate_test.go` — Validation rule tests
- `internal/naming/archive_test.go` — Naming tests (all platform combos)
- `internal/templatex/render_test.go` — Template rendering tests
- `internal/generator/npm/generate_test.go` — npm generator tests
- `internal/generator/homebrew/generate_test.go` — Homebrew tests
- `internal/generator/chocolatey/generate_test.go` — Chocolatey tests
- `internal/generator/workflow/generate_test.go` — Workflow tests

**Coverage Target:** 90%+ per package

**Race Detector:** All tests pass with `go test -race ./...`

### Integration Tests

- Config loading from different filenames
- Full pipeline: config → generators → files
- Deterministic output validation
- Cross-platform archive naming validation

### Golden Files (Snapshot Tests)

Some generators use golden files to verify output:
```go
// Generate once, check against golden file
generated := generator.Generate(ctx)
expected := readGoldenFile("testdata/expected.yml")
if generated != expected {
    t.Errorf("output mismatch: %s", diff(generated, expected))
}
```

**Advantages:**
- Catches subtle changes in template output
- Documents expected output format
- Regression detection

---

## Adding a New Generator

### Step 1: Create the Package

```
internal/generator/newtool/
├── generate.go
└── generate_test.go
```

### Step 2: Implement the Interface

```go
package newtool

import (
    "github.com/Tillman32/go-package-yourself/internal/generator"
    "github.com/Tillman32/go-package-yourself/internal/model"
)

type NewToolGenerator struct{}

func NewGenerator() generator.Generator {
    return &NewToolGenerator{}
}

func (g *NewToolGenerator) Name() string {
    return "newtool"
}

func (g *NewToolGenerator) Generate(ctx generator.Context) ([]generator.FileOutput, error) {
    // Check if enabled
    if !ctx.Config.Packages.NewTool.Enabled {
        return nil, nil  // Not enabled, skip
    }

    // Generate artifacts
    // Use ctx.ArchiveName() and ctx.Render() for consistency
    
    // Return FileOutput slice
    return []generator.FileOutput{
        {
            Path:    "newtool.conf",
            Content: "...",
            Mode:    0644,
        },
    }, nil
}
```

### Step 3: Register in Generator Registry

In `internal/generator/generator.go`:

```go
var generators = []Generator{
    npm.NewGenerator(),
    homebrew.NewGenerator(),
    chocolatey.NewGenerator(),
    workflow.NewGenerator(),
    newtool.NewGenerator(),  // Add here
}
```

### Step 4: Update Config Model

In `internal/model/config.go`, add new section:

```go
type Package struct {
    NPM       NPM       `yaml:"npm"`
    Homebrew  Homebrew  `yaml:"homebrew"`
    Chocolatey Chocolatey `yaml:"chocolatey"`
    NewTool   NewTool   `yaml:"newtool"`  // Add here
}

type NewTool struct {
    Enabled bool   `yaml:"enabled"`
    // ... other fields
}
```

### Step 5: Add Defaults

In `internal/config/defaults.go`:

```go
if cfg.Packages.NewTool.Enabled {
    // Set defaults if needed
}
```

### Step 6: Add Validation

In `internal/validate/config.go`:

```go
if cfg.Packages.NewTool.Enabled {
    if err := validateNewTool(cfg.Packages.NewTool); err != nil {
        return err
    }
}
```

### Step 7: Write Tests

In `internal/generator/newtool/generate_test.go`:

```go
func TestNewToolGeneratorName(t *testing.T) {
    gen := newtool.NewGenerator()
    if gen.Name() != "newtool" {
        t.Errorf("expected 'newtool', got %q", gen.Name())
    }
}

func TestGenerate(t *testing.T) {
    cfg := &model.Config{
        Project: model.Project{Name: "mytool"},
        Packages: model.Packages{
            NewTool: model.NewTool{Enabled: true},
        },
    }
    ctx := generator.Context{Config: cfg}
    gen := newtool.NewGenerator()
    
    outputs, err := gen.Generate(ctx)
    if err != nil {
        t.Fatal(err)
    }
    
    if len(outputs) != 1 {
        t.Errorf("expected 1 output, got %d", len(outputs))
    }
}
```

---

## Dependency Graph

```
WS0: Architecture + Contracts (FROZEN)
│   └── internal/model/ (Config structs)
│   └── internal/generator/ (Generator interface)
│   └── internal/naming/ (ArchiveName API)
│   └── internal/templatex/ (Render API)
│
├── WS1: Config Loading + Validation
│   └── internal/config/load.go, defaults.go
│   └── internal/validate/config.go
│
├── WS2: Template + Naming (Used by WS0)
│   └── internal/templatex/render.go
│   └── internal/naming/archive.go
│
├── WS3: CLI Commands
│   └── internal/cli/
│   └── Depends on: WS0, WS1, WS2
│
└── WS4-7: Generators (All depend on WS0-2)
    ├── WS4: npm/npx Generator
    ├── WS5: Homebrew Generator
    ├── WS6: Chocolatey Generator
    └── WS7: GitHub Workflow Generator
    └── Depend on: internal/generator/, internal/naming/, internal/templatex/
```

---

## Key Design Decisions

### 1. Generators Are Stateless

Each generator is a pure function given a `Context`. No mutable state.

**Why?** Makes them thread-safe, testable, and composable.

### 2. Naming Is Centralized

All archive naming goes through `internal/naming/archive.go`.

**Why?** Ensures consistency across npm, homebrew, chocolatey, workflow generators.

### 3. Validation Separate from Loading

Config loading and validation are two distinct phases.

**Why?** Allows detailed error context during validation phase.

### 4. Contracts Are Frozen (v1)

`internal/model/` types, `Generator` interface, and `Naming` API cannot change in v1.

**Why?** Ensures other workstreams can build reliably without churn.

### 5. Templates Are Deterministic

No random components in template rendering. Same input → same output.

**Why?** Makes artifacts reproducible and cacheable.

---

## Performance Considerations

### Hot Paths

1. **Template rendering** (`internal/templatex/render.go`)
   - Used by every generator
   - Optimized: uses `strings.Builder`, no reflection
   - Pre-compiled error messages

2. **Archive naming** (`internal/naming/archive.go`)
   - Used by every platform/archive combo
   - Optimized: direct string operations, no syscalls

### Allocation Efficiency

- Errors allocated with full field path context
- FileOutput slices pre-allocated when possible
- No temporary maps or goroutines in generators

### Build Speed

- No build optimization needed (executable < 10MB)
- CI builds in < 5 seconds
- Tests complete in < 1 second

---

## Error Handling

### Error Propagation

Errors flow upward with context:

```go
// naming.go
return fmt.Errorf("field Release.Archive.NameTemplate: invalid os %q", os)

// In CLI
err := generator.Generate(ctx)
if err != nil {
    log.Fatal(err)
}
// Output: field Release.Archive.NameTemplate: invalid os "windows"
```

### Error Messages

All errors include:
- Field path (e.g., "field Release.Platforms[0].OS")
- Problem description
- Actionable suggestion (when relevant)

---

## Extending for v2+

Future versions may add:

### New Generators
1. Package managers (Scoop, Linux distros)
2. Binary signing
3. Artifact signing + SLSA provenance

### Config Extensions
1. Pre/post-build hooks
2. Custom template functions
3. Environment variable expansion

### CLI Enhancements
1. Interactive config wizard (TUI)
2. Configuration preview
3. Dry-run mode

**Important:** All v1 backward compatibility must be maintained.

---

## References

- [ARCHITECTURE.md](../ARCHITECTURE.md) — Design + frozen v1 contracts
- [README.md](../README.md) — User documentation
- [Configuration Reference](./config-reference.md) — Config schema
