# Contributing to gpy

We welcome contributions! This guide explains how to set up your development environment, understand the project structure, and submit changes.

---

## Development Setup

### Prerequisites

- **Go 1.22+** — Download from [golang.org](https://golang.org/dl)
- **Git** — For cloning and version control
- **GitHub account** — To fork and submit PRs

### Clone the Repository

```bash
git clone https://github.com/Tillman32/go-package-yourself
cd go-package-yourself
```

### Install Dependencies

gpy uses `gopkg.in/yaml.v3` for YAML parsing. Dependencies are managed via `go.mod`.

```bash
go mod download
```

### Verify Setup

Build and test:

```bash
go build -o gpy ./cmd/gpy
./gpy --help

go test -race ./...  # Should pass all tests
```

---

## Project Structure

### Directory Layout

```
go-package-yourself/
├── cmd/gpy/
│   └── main.go                    # CLI entry point
├── internal/
│   ├── cli/                       # CLI commands (init, package, workflow)
│   ├── config/                    # Config loading + defaults
│   ├── generator/                 # Generator interface + implementations
│   ├── model/                     # Config schema (YAML structs)
│   ├── naming/                    # Archive naming + binary paths
│   ├── templatex/                 # Template rendering ({{placeholders}})
│   └── validate/                  # Config validation
├── docs/                          # User documentation
├── ARCHITECTURE.md                # Design documentation + frozen v1 contracts
├── README.md                      # User guide
└── go.mod / go.sum               # Dependency management
```

### Key Modules

- **`internal/model/`** — YAML schema (types for config)
- **`internal/config/`** — Config loading + defaults
- **`internal/validate/`** — Config validation
- **`internal/generator/`** — Generator interface + all implementations
- **`internal/naming/`** — Canonical archive naming
- **`internal/templatex/`** — Template rendering
- **`internal/cli/`** — CLI commands

---

## Workstreams Philosophy

gpy is organized into 9 workstreams (WS0-WS9), each with clear responsibilities:

- **WS0** — Architecture + Contracts (FROZEN ✓)
- **WS1** — Config Loading + Validation (COMPLETE ✓)
- **WS2** — Template + Naming Engine (COMPLETE ✓)
- **WS3** — CLI Commands + UX (COMPLETE ✓)
- **WS4** — npm/npx Generator (COMPLETE ✓)
- **WS5** — Homebrew Generator (COMPLETE ✓)
- **WS6** — Chocolatey Generator (COMPLETE ✓)
- **WS7** — GitHub Workflow Generator (COMPLETE ✓)
- **WS8** — Integration + E2E Validation (COMPLETE ✓)
- **WS9** — Docs + Examples (COMPLETE ✓)

**Principle:** Each workstream is independent and testable. All contracts are defined in WS0 and frozen for v1.

---

## Adding a Feature

### Step 1: Identify the Workstream

Where does your feature belong?

- **Config field?** → WS1 (config) or WS0 (contract)
- **Template placeholder?** → WS2 (templatex)
- **CLI command?** → WS3 (cli)
- **New package manager?** → Create new WS (WS4-7 pattern)
- **Bug fix?** → Relevant workstream

### Step 2: Check for Breaking Changes

**v1 Stability Requirement:**
- Config schema cannot change (see frozen v1 contracts in ARCHITECTURE.md)
- Generator interface cannot change
- Existing fields cannot be removed or renamed

For v2, breaking changes are allowed (with deprecation period).

### Step 3: Create an Issue or Discussion

Before implementing, discuss with maintainers:

```
Title: [RFC] Add feature X
- Problem: ...
- Proposed solution: ...
- Impact on contracts: ...
- Backward compatible: yes/no
```

### Step 4: Implement with Tests

Create tests FIRST (TDD):

```bash
# Write test in generate_test.go
# Run test (should fail)
go test -v ./internal/generator/npm

# Implement feature in generate.go
# Run test (should pass)
go test -v ./internal/generator/npm

# Run all tests with race detection
go test -race ./...
```

### Step 5: Ensure Tests Pass

Run the full test suite:

```bash
go test -race ./...
```

All tests must pass with `-race` flag enabled (no race conditions).

Coverage target: 90%+ per package.

### Step 6: Update Documentation

Update relevant docs:
- [README.md](README.md) — User-facing changes
- [docs/config-reference.md](docs/config-reference.md) — New config fields
- [docs/architecture.md](docs/architecture.md) — Internal changes
- [CHANGELOG.md](CHANGELOG.md) — Breaking changes or major features

### Step 7: Submit PR

```bash
git checkout -b feature/my-feature
git add -A
git commit -m "feat: add support for X"
git push origin feature/my-feature
```

Then open a PR on GitHub with:
- Descriptive title
- Summary of changes
- Link to issue (if applicable)
- Checklist completion

---

## Testing Guidelines

### Unit Tests

Test one function/method in isolation:

```go
func TestRenderPlaceholder(t *testing.T) {
    result, err := Render("{{name}}_{{version}}", map[string]string{
        "name":    "mycli",
        "version": "1.2.3",
    })
    if err != nil {
        t.Fatal(err)
    }
    expected := "mycli_1.2.3"
    if result != expected {
        t.Errorf("expected %q, got %q", expected, result)
    }
}
```

**Guidelines:**
- One assertion per test (usually)
- Test error cases too
- Use table-driven tests for multiple scenarios

### Table-Driven Tests

For testing multiple input/output combinations:

```go
func TestArchiveExtension(t *testing.T) {
    tests := []struct {
        name      string
        format    string
        os        string
        expected  string
        shouldErr bool
    }{
        {"tar.gz on linux", "tar.gz", "linux", ".tar.gz", false},
        {"zip on windows", "zip", "windows", ".zip", false},
        {"invalid format", "7z", "linux", "", true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := ExtensionFor(tt.format, tt.os)
            if (err != nil) != tt.shouldErr {
                t.Errorf("unexpected error: %v", err)
            }
            if result != tt.expected {
                t.Errorf("expected %q, got %q", tt.expected, result)
            }
        })
    }
}
```

### Integration Tests

Test full workflows (config loading → validation → generation):

```go
func TestFullPackageWorkflow(t *testing.T) {
    // Load config
    cfg, err := config.Load("testdata/valid.yaml")
    if err != nil {
        t.Fatal(err)
    }
    
    // Validate
    if err := validate.Config(cfg); err != nil {
        t.Fatal(err)
    }
    
    // Generate
    ctx := generator.Context{Config: cfg}
    gen := npm.NewGenerator()
    outputs, err := gen.Generate(ctx)
    if err != nil {
        t.Fatal(err)
    }
    
    if len(outputs) == 0 {
        t.Error("expected outputs, got none")
    }
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with race detection (all tests must pass)
go test -race ./...

# Run specific package
go test -v ./internal/config

# Run specific test
go test -v -run TestRenderPlaceholder ./internal/templatex

# Show coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test File Naming

- `*_test.go` — Unit tests (same package)
- `*_integration_test.go` — Integration tests (same package)
- `testdata/` — Test fixtures (golden files, example configs)

---

## Code Style

### Go Conventions

Follow [Effective Go](https://golang.org/doc/effective_go):
- Use `gofmt` for formatting (automatically via most editors)
- Exported functions: PascalCase
- Unexported functions: camelCase
- Use `var` for variable declarations
- Use `:=` for short assignments in functions

### Naming

- `Config` not `config`
- `Generate()` not `gen()` or `generate()`
- `ArchiveName()` not `GetArchiveName()` or `archive_name()`

### Error Handling

Always return errors with context:

```go
// Good
return fmt.Errorf("field Release.Archive.NameTemplate: %w", err)

// Bad
return err

// Bad
fmt.Println("error:", err)
```

### Comments

Document exported functions and types:

```go
// ArchiveName computes the archive filename and binary path inside the archive.
// It returns (archiveName, binPath, error).
func ArchiveName(params ArchiveNameParams) (string, string, error) {
    ...
}
```

### Packages

Keep packages small and focused:
- `internal/config/` — Config loading + defaults (2-3 files)
- `internal/generator/npm/` — npm generator (1-2 files)

Each package has responsibility; don't mix concerns.

---

## Commit Message Format

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <subject>

<body>

<footer>
```

**Types:**
- `feat` — New feature
- `fix` — Bug fix
- `docs` — Documentation update
- `test` — Test addition/update
- `refactor` — Code cleanup (no logic change)
- `perf` — Performance improvement
- `chore` — Build, dependency updates

**Examples:**

```
feat: add support for custom archive formats

- Add Format field to Archive config
- Update validation to accept "tar.gz" and "zip"
- Add tests for ExtensionFor() with new formats

Fixes #42
```

```
fix: handle missing config file gracefully

Previously, gpy would crash with unclear error message.
Now it searches all known filenames and provides helpful error.

Fixes #38
```

---

## PR Checklist

Before submitting a PR, verify:

- ✅ **Tests added/updated** — New logic has tests
- ✅ **Tests pass** — `go test -race ./...` succeeds
- ✅ **Docs updated** — Comments, README, config-reference.md
- ✅ **No breaking changes** — v1 contracts intact
- ✅ **Code formatted** — Run `gofmt -w .`
- ✅ **One commit per feature** — (or squash if needed)
- ✅ **PR title descriptive** — Follows Conventional Commits format

---

## Review Process

### What Reviewers Look For

1. **Correctness** — Does it work? Any edge cases missed?
2. **Tests** — Are tests comprehensive? Do they pass?
3. **Contracts** — Any v1 contract violations?
4. **Performance** — Any unnecessary allocations or syscalls?
5. **Docs** — Is user/contributor documentation updated?

### Addressing Feedback

- Respond to all comments
- Make changes in new commits (don't rewrite history)
- Re-request review when done
- Don't merge your own PR (wait for maintainer)

---

## Common Workflows

### Fix a Bug

```bash
# Create issue if not exists
# Create branch from main
git checkout -b fix/issue-number

# Make changes + tests
# Commit with Conventional Commits
git add -A
git commit -m "fix: resolve issue where config fails with empty project.name"

# Push and create PR
git push origin fix/issue-number
# Go to GitHub and open PR
```

### Add a Config Field

```bash
# 1. Update model: internal/model/config.go
# 2. Add to defaults: internal/config/defaults.go
# 3. Add validation: internal/validate/config.go
# 4. Add tests for all three
# 5. Update docs: docs/config-reference.md
# 6. Commit and PR
```

### Add a New Generator

```bash
# 1. Create internal/generator/newtool/
# 2. Implement Generator interface (generate.go)
# 3. Add tests (generate_test.go)
# 4. Register in internal/generator/generator.go
# 5. Update model, defaults, validation
# 6. Update README.md and docs
# 7. Commit and PR
```

---

## Key Requirements

### v1 Stability

- **No breaking changes** to existing config fields
- **No changes** to Generator interface
- **No changes** to public API signatures
- Document all changes in CHANGELOG.md

### Quality Standards

- **90%+ test coverage** per package
- **Deterministic output** (same input → same output always)
- **Race condition free** (pass `go test -race ./...`)
- **Zero external dependencies** in generated code (npm launchers, formulas, etc.)

### Performance

- **No unnecessary allocations** in hot paths
- **No reflection** in template rendering or naming
- **Pre-allocate** slices/maps when size is known

---

## Resources

- [ARCHITECTURE.md](ARCHITECTURE.md) — Internal design + frozen v1 contracts
- [README.md](README.md) — User guide
- [Effective Go](https://golang.org/doc/effective_go) — Go style guide
- [Conventional Commits](https://www.conventionalcommits.org/) — Commit format

---

## Getting Help

- **Issues** — Ask in GitHub Issues
- **Discussions** — Use GitHub Discussions for questions
- **Code Review** — Ask questions in PR comments
- **Documentation** — File issues for unclear docs

---

## License

By contributing, you agree to license your changes under the same license as the project (MIT).

---

Thank you for contributing! 🎉
