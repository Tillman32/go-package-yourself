# Plan: Make GitHub Actions Workflow Reusable (gpy-release.yaml)

**Date Created**: 2026-04-15  
**Status**: Planning  
**Priority**: High

## Problem Statement

Currently, the generated GitHub Actions workflow outputs to `.github/workflows/release.yml` (configurable but defaults to this). If a user already has an existing `release.yml` workflow for other release tasks, they face a collision and can't use both workflows.

**Root Cause**: The workflow generator creates a standalone workflow that conflicts with existing release workflows. There's also an internal inconsistency where the generator hardcodes the filename instead of respecting the config.

## Solution Overview

Transform the generated workflow into a **reusable workflow** (via GitHub's `workflow_call` trigger) that can be:
1. **Used standalone** (triggered on push tags)
2. **Called from other workflows** (as a job dependency)

Users with existing `release.yml` workflows can now simply call the gpy workflow as an additional job:

```yaml
jobs:
  my-custom-publishing:
    runs-on: ubuntu-latest
    steps: [...]
  
  publish-via-gpy:
    uses: ./.github/workflows/gpy-release.yaml
    secrets: inherit
```

## Scope

### In Scope
- Rename default workflow output from `release.yml` → `gpy-release.yaml`
- Add `workflow_call` trigger to generated workflow
- Fix generator to respect config's `WorkflowFile` path (remove hardcoding)
- Update tests to reflect new behavior
- Update CLAUDE.md documentation with new workflow path
- Update CLI workflow command help text

### Out of Scope
- Custom inputs/outputs to the reusable workflow (phase 2)
- Nested workflow orchestration (phase 2)
- Conditional job logic based on package types (phase 2)

## Implementation Tasks

### 1. Fix Generator Hardcoding (`internal/generator/workflow/generate.go`)
- **Current**: Line 64 hardcodes `.github/workflows/package-release.yaml`
- **Fix**: Generator should NOT determine the path. The CLI layer should pass it via context.
- **Approach**: Add `WorkflowFile` field to `generator.Context` and use it in `Generate()`
- **Tests**: Update `generate_test.go` to verify the generator respects the context path

### 2. Update Config Defaults (`internal/config/defaults.go`)
- Change default `WorkflowFile` from `.github/workflows/release.yml` → `.github/workflows/gpy-release.yaml`
- Add comment explaining why the name includes `gpy-` prefix (avoid collision with user workflows)

### 3. Update Model Config (`internal/model/config.go:137`)
- Update documentation comment to reflect new default path
- Document that this path should avoid collisions with existing workflows

### 4. Add `workflow_call` Support to Generator (`internal/generator/workflow/generate.go`)
- Modify `WorkflowTrigger` struct to support both:
  ```go
  type WorkflowTrigger struct {
      Push         PushTrigger  `yaml:"push"`
      WorkflowCall interface{}  `yaml:"workflow_call,omitempty"` // Minimal support
  }
  ```
- Set `workflow_call: {}` in `newWorkflow()` to enable reusability
- Ensure YAML output includes both triggers

### 5. Update CLI Workflow Command (`internal/cli/workflow.go`)
- Verify it respects the config's `WorkflowFile` path
- Update help text to reference `gpy-release.yaml` as new default
- Update error message if file exists (mention the new filename)

### 6. Update Tests
- **`internal/generator/workflow/generate_test.go`**: Update golden files with `gpy-release.yaml` and `workflow_call` trigger
- **`internal/cli/workflow_test.go`** (if exists): Update to use new default path
- **`integration/e2e_test.go`**: Update workflow path expectations

### 7. Update Documentation
- **CLAUDE.md**: Update run examples to show new default workflow path
- **docs/config-reference.md**: Document `github.workflows.workflowFile` with new default
- **README.md**: Show how to call the workflow from another workflow (reusability example)
- **CONTRIBUTING.md**: Document workflow testing expectations

### 8. Update .gitignore
- (Already covered by `packaging/` entry)

## Testing Strategy

### Unit Tests
- Generator produces workflow with `workflow_call` trigger
- Generator respects `WorkflowFile` from context
- Default path resolves to `gpy-release.yaml`

### Integration Tests
- `gpy workflow --write` creates file at correct path
- File collision detection works with new name
- Generated workflow is valid YAML

### Manual Testing
- `gpy init --yes && gpy workflow --write` produces `gpy-release.yaml`
- Workflow file contains `on: push:` and `on: workflow_call:`
- User can call the workflow from another workflow without changes

## Rollout Plan

1. **Phase 1**: Implement and merge all changes (this plan)
2. **Phase 2** (future): Add structured `workflow_call` inputs/secrets for advanced users
3. **Phase 3** (future): Document nested workflow orchestration patterns

## Success Criteria

- ✅ Generated workflow outputs to `.github/workflows/gpy-release.yaml` by default
- ✅ Generated workflow includes `on: workflow_call:` trigger
- ✅ Existing workflows with `release.yml` can coexist with gpy workflow
- ✅ Users can call gpy workflow from their own workflows
- ✅ All tests pass with race detection
- ✅ Documentation updated with new default path and reusability example

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Breaking change for existing users | Medium | High | Document migration in CHANGELOG; suggest `--config` flag with custom path |
| YAML generation bugs with new trigger | Low | Medium | Comprehensive unit tests of trigger structure |
| Config path not respected by generator | Low | High | Add context field; test coverage |

## Notes

- The v1 config contract frozen in `internal/model/config.go` may need a v2 bump if we add structured `workflow_call` inputs later, but renaming the path and adding the trigger is additive and non-breaking.
- Users can override the path in their `gpy.yaml` if needed: `github: workflows: workflowFile: ".github/workflows/custom-release.yaml"`
