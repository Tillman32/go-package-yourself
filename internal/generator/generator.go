// Package generator defines the interface and types for all packaging artifact generators.
// All types and interfaces in this package are frozen contracts.
package generator

import (
	"os"

	"go-package-yourself/internal/model"
)

// Generator is the interface all artifact generators must implement.
//
// Contract: This interface is frozen. All generators (npm, homebrew, chocolatey, workflow)
// must implement these two methods. No breaking changes.
type Generator interface {
	// Name returns the canonical name of this generator (e.g., "npm", "homebrew").
	// Used for logging, debugging, and --only flag in CLI.
	Name() string

	// Generate produces the artifact files for this generator.
	// It receives a Context with all necessary configuration and helpers.
	// Returns a slice of FileOutput and any error.
	// Errors should be actionable and include field context where possible.
	Generate(ctx Context) ([]FileOutput, error)
}

// FileOutput represents a single file to be written by a generator.
//
// Path is relative to the project root (e.g., "packaging/npm/package.json").
// Content is the raw file bytes.
// Mode is the Unix file mode (e.g., 0o644 for regular files, 0o755 for executables).
//
// Contract: This struct is frozen. All generators produce FileOutput slices.
type FileOutput struct {
	// Path is the relative path from project root where this file should be written.
	// Example: "packaging/npm/mytool/package.json"
	Path string

	// Content is the raw file bytes to write.
	// For text files, this is the UTF-8 encoded content.
	// For binary files, this is the binary content.
	Content []byte

	// Mode is the Unix file permission mode to apply.
	// Use 0o644 for regular files, 0o755 for executable scripts.
	Mode os.FileMode
}

// Context provides generators with parsed configuration and helper functions.
//
// All helper funcs are pre-bound to the parsed config and project state,
// so generators don't need to re-implement common logic.
//
// Contract: This struct is frozen. Generators receive this; no external modifications.
type Context struct {
	// Config is the fully parsed and validated gpy configuration.
	Config *model.Config

	// ProjectRoot is the absolute path to the project root directory.
	// This is where the gpy config file was found or explicitly specified.
	ProjectRoot string

	// Version is the optional resolved version for this generation run.
	// If set by CLI flag, this is the concrete version string.
	// Otherwise (e.g., for "gpy package" without --version), this may be empty
	// and generators should preserve {{version}} placeholders instead of resolving.
	//
	// For "gpy workflow" generation, Version is often empty because the workflow
	// uses github.ref_name at runtime.
	Version string

	// ArchiveName computes the canonical archive filename and binary path for given parameters.
	// This function is deterministic and must be used by all generators.
	// See internal/naming for full specification.
	ArchiveName func(os, arch string) (archiveFilename, binPathInArchive string, err error)

	// RenderTemplate processes a template string with placeholders like {{name}}, {{version}}, {{os}}, {{arch}}, {{ext}}.
	// fieldPath is included in error messages for clarity (e.g., "field Release.Archive.NameTemplate").
	// See internal/templatex for full specification.
	RenderTemplate func(template, fieldPath string) (string, error)
}

// NewContext creates a Context for generator execution.
// This is called by the CLI before invoking generators.
//
// projectRoot: absolute path to project root
// config: parsed gpy configuration
// version: optional concrete version (may be empty for runtime resolution)
func NewContext(projectRoot string, config *model.Config, version string) *Context {
	return &Context{
		Config:      config,
		ProjectRoot: projectRoot,
		Version:     version,
		// ArchiveName and RenderTemplate will be bound during CLI setup
		// before passing Context to generators. See ARCHITECTURE.md for details.
	}
}

// Registry is a registry of available generators.
// Each generator is identified by name for --only and logging purposes.
type Registry map[string]Generator

// NewRegistry creates an empty generator registry.
func NewRegistry() Registry {
	return make(Registry)
}

// Register adds a generator to the registry by name.
// Panics if a generator with the same name already exists.
func (r Registry) Register(gen Generator) {
	name := gen.Name()
	if _, exists := r[name]; exists {
		panic("generator already registered: " + name)
	}
	r[name] = gen
}

// Get retrieves a generator by name, returning nil if not found.
func (r Registry) Get(name string) Generator {
	return r[name]
}

// GenerateAll runs all generators in the registry and collects their outputs.
// Returns a map of generator name -> []FileOutput.
// If any generator returns an error, collection stops and the error is returned.
//
// This is called by "gpy package" to invoke all selected generators.
func (r Registry) GenerateAll(genctx Context, names ...string) (map[string][]FileOutput, error) {
	if len(names) == 0 {
		// If no names specified, use all registered generators
		names = make([]string, 0, len(r))
		for name := range r {
			names = append(names, name)
		}
	}

	results := make(map[string][]FileOutput, len(names))
	for _, name := range names {
		gen := r[name]
		if gen == nil {
			continue // Skip unknown generators gracefully
		}

		outputs, err := gen.Generate(genctx)
		if err != nil {
			return nil, err
		}
		results[name] = outputs
	}
	return results, nil
}
