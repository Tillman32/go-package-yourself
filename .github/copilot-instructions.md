# GitHub Copilot Instructions for gpy

You are an AI pair programmer helping develop **gpy** — a Go CLI tool that generates multi-channel packaging artifacts (npm, Homebrew, Chocolatey, GitHub Actions workflows) from a single YAML config.

## Core Context

- **Language**: Go 1.21+
- **Main Dependency**: `gopkg.in/yaml.v3` only
- **Project Root**: Look for `gpy.yaml` or `go.mod`
- **Testing**: `go test -race ./...` must pass before any commit (race detection is mandatory)

## Key Architectural Principles

### Frozen v1 Contracts (MUST NOT CHANGE without v2 bump)

These are immutable and form the public API:

1. **Model Structs** (`internal/model/config.go`)
   - Field names and struct tags are frozen
   - Changes require v2 version bump
   - Any schema extensions must be additive (new fields only)

2. **Generator Interface** (`internal/generator/generator.go`)
   ```go
   type Generator interface {
       Name() string
       Generate(ctx Context) ([]FileOutput, error)
   }
   ```
   - Signature is frozen; method names must not change
   - All generators must implement this interface

3. **Archive Naming** (`internal/naming/naming.go`)
   - `ArchiveName()` is the single source of truth for filenames
   - All generators use it; no per-generator naming logic
   - Output is deterministic: same input → same bytes, always

4. **Template Placeholders** (`internal/templatex/renderer.go`)
   - Supported: `{{name}}`, `{{version}}`, `{{os}}`, `{{arch}}`, `{{ext}}`
   - Unknown placeholders are hard errors
   - No custom string interpolation; use the renderer

5. **Platform Matrix**
   - Valid OS: `darwin`, `linux`, `windows`
   - Valid Arch: `amd64`, `arm64`

### Generator Principles

- **Generators are pure functions**: no filesystem access during `Generate()` (only in CLI layer)
- **Output is deterministic**: identical config → identical bytes
- **All use shared naming**: `internal/naming.ArchiveName()` — zero duplication
- **All use shared templates**: `internal/templatex.Renderer` — consistent error handling

## Planning & Execution Workflow

**Important: Use two-session pattern for non-trivial changes**

### Session 1: Planning
- Research the problem; gather context
- Create a plan file: `.tmp/plans/<feature-name>.md`
- Document scope, tasks, testing strategy, success criteria
- Do NOT execute in the same session

**Why**: Plan files (2-5KB) + implementation (5-10 changes) exceed context budget in one session.

### Session 2: Execution
- Load the plan file from `.tmp/plans/`
- Follow tasks sequentially with the TodoWrite tool
- Mark each task `in_progress` then `completed` as you work
- Run tests, commit, update docs

**Small changes** (<50 lines, single file): implement immediately; no plan needed

**Example**: Typo fix, simple constant rename

**Complex changes** (>3 files, architecture decisions): always create a plan file

### Existing Plans
Check `.tmp/plans/` before starting work:
- `workflow-reusability.md` — Make GitHub Actions workflow callable from other workflows

## Development Standards

- **TDD**: Write tests first; target 90%+ coverage per package
- **Race detection**: `go test -race ./...` must pass before every commit
- **Error context**: Always include field path: `fmt.Errorf("field Release.Archive.NameTemplate: %w", err)`
- **No reflection** in hot paths (naming, template rendering)
- **No unnecessary allocations**: pre-allocate slices/maps when size is known
- **Commit format**: [Conventional Commits](https://www.conventionalcommits.org/)
  - `feat:` new feature
  - `fix:` bug fix
  - `docs:` documentation
  - `test:` test additions/updates
  - `refactor:` non-behavioral changes

## Config Discovery Order

The CLI searches for config in this order:

1. `--config <path>` flag (explicit wins)
2. `gpy.yaml` / `gpy.yml`
3. `.gpy.yaml` / `.gpy.yml`
4. `go-package-yourself.yaml` / `go-package-yourself.yml`
5. `.go-package-yourself.yaml` / `.go-package-yourself.yml`

## Directory Structure

```
cmd/gpy/main.go              CLI entry point
internal/
  cli/                       Commands: init, package, workflow
  config/                    Config loading (Load) + defaults (ApplyDefaults)
  model/                     YAML schema structs — FROZEN v1
  validate/                  Config validation with field-path errors
  generator/                 Generator interface + Registry + all implementations
    npm/                     npm/npx launcher generator
    homebrew/                Homebrew formula generator
    chocolatey/              Chocolatey package generator
    workflow/                GitHub Actions workflow generator
  naming/                    Canonical archive naming — single source of truth
  templatex/                 Template rendering
integration/                 End-to-end integration tests + golden files
docs/                        User documentation
.tmp/plans/                  Implementation plans (gitignored, local only)
```

## Common Tasks

### Adding a New Generator

1. Create `internal/generator/<name>/generate.go` implementing `Generator`
2. Add tests in `generate_test.go`
3. Register in `internal/generator/generator.go` (call it in `Registry()`)
4. Update model: `internal/model/config.go`, `internal/config/defaults.go`, `internal/validate/config.go`
5. Update docs: `docs/config-reference.md`, `README.md`

### Adding a Config Field

1. Add to model struct in `internal/model/config.go` with struct tag
2. Add default in `internal/config/defaults.go`
3. Add validation in `internal/validate/config.go` (if needed)
4. Add init prompt in `internal/cli/init.go` (if user-facing)
5. Update docs

### Running Tests

```bash
go test -race ./...              # Full suite with race detection (required)
go test -cover ./...             # With coverage
go test -v ./internal/config     # Specific package, verbose
go test -short ./integration     # Skip slow E2E tests
```

## Code Review Checklist

Before committing:

- ✅ Tests pass: `go test -race ./...`
- ✅ Coverage is 90%+ per package (use `go test -cover ./...`)
- ✅ No hardcoded paths; respect config (especially in generators)
- ✅ Errors include field context: `fmt.Errorf("field X: %w", err)`
- ✅ Commit message follows Conventional Commits format
- ✅ No breaking changes to frozen contracts
- ✅ Documentation updated (CLAUDE.md, docs/, README.md if user-facing)

## When to Suggest a New Session

Use the phrase: **"This is a complex change. I've created a plan at `.tmp/plans/<name>.md`. Let's start a fresh session to execute it, which will preserve context for implementation details."**

Or: **"I can help with this using a subagent. Let me break it down into a plan first."**

## Key Files to Always Check

- `CLAUDE.md` — This file; project conventions
- `ARCHITECTURE.md` — Design decisions, frozen contracts, diagrams
- `CONTRIBUTING.md` — PR checklist, code style
- `.tmp/plans/` — Existing implementation plans

## Red Flags

🚩 **If you see these, stop and ask:**
- Changes to `internal/model/` struct fields or tags
- Changes to `generator.Generator` interface signature
- Changes to `naming.ArchiveName()` behavior
- Hardcoded paths in generators (should use config/context)
- Error messages without field context
- Untested code

## Helpful Reminders

- All generators are in `internal/generator/<name>/`; the registry is in `internal/generator/generator.go`
- Use `ctx.ArchiveName(os, arch)` for all filename generation
- Use `ctx.Render(template, data)` for all template rendering
- Validation errors should include field path for user clarity
- CLI layer handles I/O; generators handle data transformation
