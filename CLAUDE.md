# gpy — Go Package Yourself

`gpy` is a Go CLI tool that generates multi-channel packaging artifacts — npm/npx, Homebrew, Chocolatey, and GitHub Actions workflows — from a single `gpy.yaml` config file. Go developers run it at their project root to produce release-ready packaging without writing boilerplate.

---

## Commands

### Build

```bash
go build -o gpy ./cmd/gpy
```

### Test

```bash
go test -race ./...              # Full suite with race detection (required to pass)
go test -cover ./...             # With coverage report
go test -v ./internal/config     # Specific package, verbose
go test -race ./integration -v   # Integration tests only
go test -short ./integration     # Skip slow E2E tests
```

### Run

```bash
./gpy init                       # Interactive wizard → creates gpy.yaml
./gpy package                    # Generate npm/homebrew/chocolatey artifacts
./gpy package --only npm         # Generate a subset
./gpy workflow                   # Print GitHub Actions workflow to stdout
./gpy workflow --write           # Write .github/workflows/gpy-release.yaml
```

---

## Project Structure

```text
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
```

---

## Architecture: Key Invariants

- **All generators use `internal/naming`** for archive filenames — zero per-generator naming logic.
- **All template rendering uses `internal/templatex`** — no custom string-substitution in generators.
- **Generators are pure**: no filesystem access during `Generate()`, only in the CLI layer.
- **Output is deterministic**: same config → identical byte output, always.
- **Single external dependency**: `gopkg.in/yaml.v3` only. Generated artifacts have zero deps.

### Generator Interface (frozen)

```go
type Generator interface {
    Name() string
    Generate(ctx Context) ([]FileOutput, error)
}
```

### Template Placeholders (frozen)

`{{name}}`, `{{version}}`, `{{os}}`, `{{arch}}`, `{{ext}}` — unknown placeholders are hard errors.

### Platform Matrix (frozen)

Valid OS: `darwin`, `linux`, `windows`. Valid Arch: `amd64`, `arm64`.

---

## Frozen v1 Contracts

These MUST NOT change without a v2 version bump:

- `internal/model/` struct tags and field names
- `generator.Generator` interface signature
- `naming.ArchiveName()` signature and behavior
- `templatex.Renderer` placeholder syntax and error format

See the [Frozen v1 Contracts](ARCHITECTURE.md#frozen-v1-contracts) section in ARCHITECTURE.md.

---

## Development Standards

- **TDD**: write tests first; coverage target is 90%+ per package
- **Always run with race detection**: `go test -race ./...` must pass before any commit
- **Errors must include field context**: `fmt.Errorf("field Release.Archive.NameTemplate: %w", err)`
- **No reflection** in hot paths (naming, template rendering)
- **No unnecessary allocations**: pre-allocate slices/maps when size is known
- **Commit format**: [Conventional Commits](https://www.conventionalcommits.org/) — `feat:`, `fix:`, `docs:`, `test:`, `refactor:`

---

## Config Discovery Order

1. `--config <path>` flag (explicit wins)
2. `gpy.yaml` / `gpy.yml`
3. `.gpy.yaml` / `.gpy.yml`
4. `go-package-yourself.yaml` / `go-package-yourself.yml`
5. `.go-package-yourself.yaml` / `.go-package-yourself.yml`

---

## Adding a New Generator

1. Create `internal/generator/<name>/generate.go` implementing `Generator` interface
2. Add tests in `generate_test.go`
3. Register in `internal/generator/generator.go`
4. Update `internal/model/config.go`, `internal/config/defaults.go`, `internal/validate/config.go`
5. Update `docs/config-reference.md` and `README.md`

---

## Planning & Execution Workflow

**All non-trivial changes must follow this two-session pattern to manage context efficiently:**

1. **Planning Session (THIS SESSION)**
   - Research the problem and gather context
   - Create a detailed plan file in `.tmp/plans/` (e.g., `.tmp/plans/feature-name.md`)
   - Document scope, tasks, testing strategy, and success criteria
   - Commit the plan mentally, but don't execute it in the same session
   - Reason: Plan files can be 2-5KB, implementation spans 5-10 code changes; combined context would exceed limits

2. **Execution Session (SEPARATE SESSION)**
   - Load the plan file from `.tmp/plans/`
   - Follow tasks step-by-step
   - Each implementation task is marked `in_progress` then `completed`
   - Run tests, commit changes, update docs
   - Reason: Fresh session preserves context budget for implementation details

**Example:**
```
Session 1: "I want to make workflows reusable"
  → Creates .tmp/plans/workflow-reusability.md (this pattern)
  
Session 2: "Execute: .tmp/plans/workflow-reusability.md"
  → Implements all tasks from the plan
  → Tests and commits
```

**For Small Changes** (single file, <50 lines):
- Implement immediately; no plan needed
- Example: typo fix, simple constant rename

**For Complex Changes** (>3 files, architectural decisions):
- Always create a plan file
- Keeps both sessions focused and efficient

---

## Reference Docs

| File | Purpose |
| --- | --- |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Design, frozen v1 contracts, diagrams, package usage examples |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Dev setup, PR checklist, code style |
| [docs/config-reference.md](docs/config-reference.md) | Complete YAML field reference |
| [docs/architecture.md](docs/architecture.md) | Internal architecture guide |
| [docs/troubleshooting.md](docs/troubleshooting.md) | Common errors and fixes |
| [.claude/decisions.md](.claude/decisions.md) | Key v1 design decisions and rationale |
| [.tmp/plans/](../.tmp/plans/) | Implementation plans (gitignored, local only) |
