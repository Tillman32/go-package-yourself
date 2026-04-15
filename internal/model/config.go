// Package model defines the core domain types for gpy configuration and artifact generation.
// All types in this package are final contracts and must not be modified after freeze.
package model

// Config represents the top-level gpy configuration loaded from YAML.
//
// Schema version v1 only. All optional fields have sensible defaults applied during parsing.
//
// Contract: This type is frozen and must not be changed. All other workstreams depend on this structure.
type Config struct {
	SchemaVersion int      `yaml:"schemaVersion"`
	Project       Project  `yaml:"project"`
	Go            Go       `yaml:"go"`
	Release       Release  `yaml:"release"`
	Packages      Packages `yaml:"packages"`
	GitHub        GitHub   `yaml:"github"`
}

// Project contains project metadata.
//
// Required fields: Name, Repo
type Project struct {
	Name        string `yaml:"name"`        // required: binary name (e.g., "mytool")
	Repo        string `yaml:"repo"`        // required: GitHub owner/repo (e.g., "owner/mytool")
	Description string `yaml:"description"` // optional: short description
	Homepage    string `yaml:"homepage"`    // optional: project homepage URL
	License     string `yaml:"license"`     // optional: SPDX license identifier
}

// Go contains Go build configuration.
//
// Required fields: Main
type Go struct {
	Main    string `yaml:"main"`    // required: build target path (e.g., "./cmd/mytool")
	CGO     bool   `yaml:"cgo"`     // optional: enable CGO (default false)
	LDFlags string `yaml:"ldflags"` // optional: linker flags passed to -ldflags
}

// Release contains release and archive configuration.
type Release struct {
	TagTemplate string     `yaml:"tagTemplate"` // optional: template for release tags (default "v{{version}}")
	Platforms   []Platform `yaml:"platforms"`   // optional: target platforms (default: darwin/linux/windows with amd64/arm64)
	Archive     Archive    `yaml:"archive"`     // archive naming and formatting
	Checksums   Checksums  `yaml:"checksums"`   // checksum configuration
}

// Platform represents a build target OS/architecture combination.
//
// Supported OS values: darwin, linux, windows
// Supported Arch values: amd64, arm64
type Platform struct {
	OS   string `yaml:"os"`   // required: darwin, linux, or windows
	Arch string `yaml:"arch"` // required: amd64 or arm64
}

// Archive contains archive naming and format configuration.
type Archive struct {
	// NameTemplate is the template for archive filenames.
	// Supported placeholders: {{name}}, {{version}}, {{os}}, {{arch}}
	// Default: "{{name}}_{{version}}_{{os}}_{{arch}}"
	// Example output: "mytool_1.2.3_darwin_arm64.tar.gz"
	NameTemplate string `yaml:"nameTemplate"`

	// Format defines archive compression formats by OS.
	// Global default is "tar.gz". Windows default overrides to "zip".
	Format ArchiveFormat `yaml:"format"`

	// BinPathInArchive is the template for the binary path inside the archive.
	// Supported placeholders: {{name}}, {{os}}, {{arch}}
	// Default: "{{name}}"
	// Example: "bin/{{name}}" for "bin/mytool" inside the archive
	// For Windows, ".exe" is appended automatically if not present.
	BinPathInArchive string `yaml:"binPathInArchive"`
}

// ArchiveFormat defines compression format selection by OS.
type ArchiveFormat struct {
	Default string `yaml:"default"` // default format for all OS (default "tar.gz")
	Windows string `yaml:"windows"` // Windows-specific format override (default "zip")
}

// Checksums contains checksum generation configuration.
// In v1, algorithm and format are fixed.
type Checksums struct {
	File      string `yaml:"file"`      // output filename (default "checksums.txt")
	Algorithm string `yaml:"algorithm"` // fixed v1: "sha256" only
	Format    string `yaml:"format"`    // fixed v1: "goreleaser" format ("<sha>  <filename>")
}

// Packages contains packaging configuration for each distribution channel.
type Packages struct {
	NPM        NPM        `yaml:"npm"`
	Homebrew   Homebrew   `yaml:"homebrew"`
	Chocolatey Chocolatey `yaml:"chocolatey"`
	Docker     Docker     `yaml:"docker"`
}

// NPM contains npm/npx wrapper package configuration.
type NPM struct {
	Enabled     bool   `yaml:"enabled"`     // default false
	PackageName string `yaml:"packageName"` // optional: npm package name (default project.name)
	BinName     string `yaml:"binName"`     // optional: executable name (default project.name)
	NodeEngines string `yaml:"nodeEngines"` // optional: Node.js version requirement (default ">=18")
}

// Homebrew contains Homebrew formula configuration.
type Homebrew struct {
	Enabled     bool   `yaml:"enabled"`     // default false
	Tap         string `yaml:"tap"`         // optional: tap to publish to (e.g., "owner/homebrew-tap")
	FormulaName string `yaml:"formulaName"` // optional: formula name (default CamelCase(project.name))
}

// Chocolatey contains Chocolatey package configuration.
type Chocolatey struct {
	Enabled   bool   `yaml:"enabled"`   // default false
	PackageID string `yaml:"packageId"` // optional: package ID (default project.name)
	Authors   string `yaml:"authors"`   // optional: package authors
}

// Docker contains Docker image generation configuration.
type Docker struct {
	Enabled   bool              `yaml:"enabled"`   // default false
	ImageName string            `yaml:"imageName"` // optional: Docker image name (default: project.name)
	Port      int               `yaml:"port"`      // optional: EXPOSE port in the generated Dockerfile
	Env       map[string]string `yaml:"env"`       // optional: ENV directives in generated Dockerfile
	Cmd       string            `yaml:"cmd"`       // optional: CMD override (default: "./{{name}}")
}

// GitHub contains GitHub-specific configuration.
type GitHub struct {
	Workflows GitHubWorkflows `yaml:"workflows"`
}

// GitHubWorkflows contains GitHub Actions workflow generation configuration.
type GitHubWorkflows struct {
	Enabled      bool     `yaml:"enabled"`      // default false
	WorkflowFile string   `yaml:"workflowFile"` // path to workflow file (default ".github/workflows/release.yml")
	TagPatterns  []string `yaml:"tagPatterns"`  // patterns for workflow trigger (default ["v*"])
}
