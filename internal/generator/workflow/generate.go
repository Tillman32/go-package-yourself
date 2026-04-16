// Package workflow implements the GitHub Actions workflow generator for go-package-yourself.
// It generates release.yml workflows that build and publish Go binaries to GitHub Releases.
package workflow

import (
	"fmt"
	"strings"

	"go-package-yourself/internal/generator"
	"go-package-yourself/internal/model"

	"gopkg.in/yaml.v3"
)

// WorkflowGenerator generates GitHub Actions workflows.
// Implements the generator.Generator interface.
type WorkflowGenerator struct{}

// Name returns the canonical name of this generator.
func (w *WorkflowGenerator) Name() string {
	return "workflow"
}

// Generate produces the GitHub Actions workflow file.
// Returns a single FileOutput containing the release.yml workflow.
func (w *WorkflowGenerator) Generate(ctx generator.Context) ([]generator.FileOutput, error) {
	if len(ctx.Config.Release.Platforms) == 0 {
		return nil, fmt.Errorf("no platforms configured for workflow generation")
	}

	// Build matrix entries for all platforms
	matrixEntries, err := buildMatrixEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("build matrix: %w", err)
	}

	// Render tag patterns from config
	tagPatterns := ctx.Config.GitHub.Workflows.TagPatterns
	if len(tagPatterns) == 0 {
		tagPatterns = []string{"v*"} // default
	}

	// Create the workflow structure
	workflow := newWorkflow(
		ctx.Config.Project.Name,
		ctx.Config.Project.Repo,
		ctx.Config.Go,
		ctx.Config.Release,
		ctx.Config.Packages.Docker,
		ctx.Config.Packages.NPM,
		ctx.Config.Packages.Homebrew,
		ctx.Config.Packages.Chocolatey,
		matrixEntries,
		tagPatterns,
	)

	// Marshal to YAML
	yamlContent, err := marshalWorkflow(workflow)
	if err != nil {
		return nil, fmt.Errorf("marshal workflow to YAML: %w", err)
	}

	// Use workflow file path from config
	workflowFile := ctx.Config.GitHub.Workflows.WorkflowFile
	if workflowFile == "" {
		return nil, fmt.Errorf("WorkflowFile not set in config (should have been set by ApplyDefaults)")
	}

	return []generator.FileOutput{
		{
			Path:    workflowFile,
			Content: yamlContent,
			Mode:    0o644,
		},
	}, nil
}

// MatrixEntry represents a single build variant in the workflow matrix.
type MatrixEntry struct {
	OS      string `yaml:"os"`
	Arch    string `yaml:"arch"`
	RunsOn  string `yaml:"runs-on"`
	Archive string `yaml:"archive"`
	BinPath string `yaml:"bin-path"`
	Ext     string `yaml:"ext"`
}

// buildMatrixEntries generates matrix entries for all configured platforms.
func buildMatrixEntries(ctx generator.Context) ([]MatrixEntry, error) {
	entries := make([]MatrixEntry, 0, len(ctx.Config.Release.Platforms))

	for _, platform := range ctx.Config.Release.Platforms {
		archFilename, binPath, err := ctx.ArchiveName(platform.OS, platform.Arch)
		if err != nil {
			return nil, fmt.Errorf("archive name for %s/%s: %w", platform.OS, platform.Arch, err)
		}

		runsOn := runnerFor(platform.OS)

		// Extract extension from archive filename
		ext := ""
		if strings.HasSuffix(archFilename, ".tar.gz") {
			ext = ".tar.gz"
		} else if strings.HasSuffix(archFilename, ".zip") {
			ext = ".zip"
		}

		entries = append(entries, MatrixEntry{
			OS:      platform.OS,
			Arch:    platform.Arch,
			RunsOn:  runsOn,
			Archive: archFilename,
			BinPath: binPath,
			Ext:     ext,
		})
	}

	return entries, nil
}

// runnerFor returns the GitHub Actions runner for a given OS.
func runnerFor(os string) string {
	switch os {
	case "darwin":
		return "macos-latest"
	case "windows":
		return "windows-latest"
	case "linux":
		return "ubuntu-latest"
	default:
		return "ubuntu-latest" // fallback
	}
}

// WorkflowDoc represents the complete workflow YAML structure.
type WorkflowDoc struct {
	Name string                 `yaml:"name"`
	On   WorkflowTrigger        `yaml:"on"`
	Jobs map[string]WorkflowJob `yaml:"jobs"`
}

// WorkflowTrigger defines what events trigger the workflow.
type WorkflowTrigger struct {
	Push         PushTrigger `yaml:"push"`
	WorkflowCall interface{} `yaml:"workflow_call,omitempty"` // Empty object enables reusable workflow
}

// PushTrigger defines push-based triggers.
type PushTrigger struct {
	Tags []string `yaml:"tags"`
}

// WorkflowJob represents a job in the workflow.
type WorkflowJob struct {
	Needs       []string          `yaml:"needs,omitempty"`
	Permissions map[string]string `yaml:"permissions,omitempty"`
	RunsOn      string            `yaml:"runs-on"`
	Strategy    *JobStrategy      `yaml:"strategy,omitempty"`
	Steps       []WorkflowStep    `yaml:"steps"`
}

// JobStrategy defines the matrix strategy.
type JobStrategy struct {
	Matrix JobMatrix `yaml:"matrix"`
}

// JobMatrix holds the matrix configuration.
type JobMatrix struct {
	Include []MatrixEntry `yaml:"include"`
}

// WorkflowStep represents a single step in a job.
type WorkflowStep struct {
	ID   string                 `yaml:"id,omitempty"`
	Name string                 `yaml:"name,omitempty"`
	Uses string                 `yaml:"uses,omitempty"`
	With map[string]interface{} `yaml:"with,omitempty"`
	Env  map[string]string      `yaml:"env,omitempty"`
	If   string                 `yaml:"if,omitempty"`
	Run  string                 `yaml:"run,omitempty"`
}

// newWorkflow constructs the complete workflow document.
func newWorkflow(projectName string, projectRepo string, goConfig model.Go, release model.Release, dockerCfg model.Docker, npmCfg model.NPM, homebrewCfg model.Homebrew, chocolateyCfg model.Chocolatey, matrixEntries []MatrixEntry, tagPatterns []string) WorkflowDoc {
	// Determine Go version to use (hard-coded to minimize complexity for now)
	goVersion := "1.22"

	steps := []WorkflowStep{
		{
			Name: "Checkout code",
			Uses: "actions/checkout@v4",
			With: map[string]interface{}{
				"fetch-depth": 0,
			},
		},
		{
			Name: "Setup Go",
			Uses: "actions/setup-go@v4",
			With: map[string]interface{}{
				"go-version": goVersion,
			},
		},
		{
			Name: "Build binary",
			Run:  buildStepRun(projectName, goConfig),
			Env: map[string]string{
				"GOOS":   "${{ matrix.os }}",
				"GOARCH": "${{ matrix.arch }}",
			},
		},
		{
			Name: "Create archive",
			Run:  archiveStepRun(),
		},
		{
			Name: "Generate checksums",
			Run:  checksumStepRun(release.Checksums.File),
		},
		{
			Name: "Upload to GitHub Release",
			Uses: "softprops/action-gh-release@v1",
			With: map[string]interface{}{
				"files": "release-files.txt",
			},
			Env: map[string]string{
				"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
			},
		},
	}

	jobs := map[string]WorkflowJob{
		"build": {
			RunsOn: "${{ matrix.runs-on }}",
			Strategy: &JobStrategy{
				Matrix: JobMatrix{
					Include: matrixEntries,
				},
			},
			Steps: steps,
		},
	}

	if dockerCfg.Enabled {
		jobs["publish-docker"] = publishDockerJob(projectRepo, dockerCfg.ImageName)
	}

	if npmCfg.Enabled {
		jobs["publish-npm"] = publishNpmJob(projectRepo, npmCfg)
	}

	if homebrewCfg.Enabled {
		jobs["publish-homebrew"] = publishHomebrewJob(projectRepo, homebrewCfg)
	}

	if chocolateyCfg.Enabled {
		jobs["publish-chocolatey"] = publishChocolateyJob(projectRepo, chocolateyCfg)
	}

	return WorkflowDoc{
		Name: "Release",
		On: WorkflowTrigger{
			Push: PushTrigger{
				Tags: tagPatterns,
			},
			WorkflowCall: struct{}{}, // Enable workflow_call for reusability
		},
		Jobs: jobs,
	}
}

// publishDockerJob creates the publish-docker job for the workflow.
func publishDockerJob(projectRepo, imageName string) WorkflowJob {
	return WorkflowJob{
		Needs:  []string{"build"},
		RunsOn: "ubuntu-latest",
		Permissions: map[string]string{
			"contents": "read",
			"packages": "write",
		},
		Steps: []WorkflowStep{
			{
				Name: "Checkout code",
				Uses: "actions/checkout@v4",
			},
			{
				Name: "Set up Docker Buildx",
				Uses: "docker/setup-buildx-action@v3",
			},
			{
				Name: "Log in to GitHub Container Registry",
				Uses: "docker/login-action@v3",
				With: map[string]interface{}{
					"registry": "ghcr.io",
					"username": "${{ github.actor }}",
					"password": "${{ secrets.GITHUB_TOKEN }}",
				},
			},
			{
				ID:   "meta",
				Name: "Extract Docker metadata",
				Uses: "docker/metadata-action@v5",
				With: map[string]interface{}{
					"images": "ghcr.io/${{ github.repository }}",
					"tags": "type=semver,pattern={{version}}\n" +
						"type=semver,pattern={{major}}.{{minor}}\n" +
						"type=semver,pattern={{major}}\n" +
						"type=raw,value=latest,enable=${{ github.ref == format('refs/heads/{0}', github.event.repository.default_branch) }}",
				},
			},
			{
				Name: "Build and push to GHCR",
				Uses: "docker/build-push-action@v5",
				With: map[string]interface{}{
					"context":    ".",
					"push":       true,
					"tags":       "${{ steps.meta.outputs.tags }}",
					"labels":     "${{ steps.meta.outputs.labels }}",
					"cache-from": "type=gha",
					"cache-to":   "type=gha,mode=max",
				},
			},
			{
				ID:   "dockerhub-token",
				Name: "Detect Docker Hub token",
				Run: "if [ -z \"$DOCKER_HUB_TOKEN\" ]; then\n" +
					"  echo \"available=false\" >> \"$GITHUB_OUTPUT\"\n" +
					"else\n" +
					"  echo \"available=true\" >> \"$GITHUB_OUTPUT\"\n" +
					"fi",
				Env: map[string]string{
					"DOCKER_HUB_TOKEN": "${{ secrets.DOCKER_HUB_TOKEN }}",
				},
			},
			{
				Name: "Log in to Docker Hub",
				If:   "vars.DOCKER_HUB_USERNAME != '' && steps.dockerhub-token.outputs.available == 'true'",
				Uses: "docker/login-action@v3",
				With: map[string]interface{}{
					"username": "${{ vars.DOCKER_HUB_USERNAME }}",
					"password": "${{ secrets.DOCKER_HUB_TOKEN }}",
				},
			},
			{
				Name: "Build and push to Docker Hub",
				If:   "vars.DOCKER_HUB_USERNAME != '' && steps.dockerhub-token.outputs.available == 'true'",
				Uses: "docker/build-push-action@v5",
				With: map[string]interface{}{
					"context": ".",
					"push":    true,
					"tags": fmt.Sprintf(
						"${{ vars.DOCKER_HUB_USERNAME }}/%s:${{ github.ref_name }}\n"+
							"${{ vars.DOCKER_HUB_USERNAME }}/%s:latest",
						imageName, imageName,
					),
					"cache-from": "type=gha",
					"cache-to":   "type=gha,mode=max",
				},
			},
		},
	}
}

// publishNpmJob creates the publish-npm job for the workflow.
func publishNpmJob(projectRepo string, npmCfg model.NPM) WorkflowJob {
	return WorkflowJob{
		Needs:  []string{"build"},
		RunsOn: "ubuntu-latest",
		Steps: []WorkflowStep{
			{
				Name: "Checkout code",
				Uses: "actions/checkout@v4",
			},
			{
				Name: "Setup Node.js",
				Uses: "actions/setup-node@v4",
				With: map[string]interface{}{
					"node-version": "18",
					"registry-url": "https://registry.npmjs.org/",
				},
			},
			{
				Name: "Setup Go",
				Uses: "actions/setup-go@v4",
				With: map[string]interface{}{
					"go-version": "1.22",
				},
			},
			{
				Name: "Generate and publish npm package",
				Env: map[string]string{
					"NODE_AUTH_TOKEN": "${{ secrets.NPM_TOKEN }}",
				},
				If: "env.NODE_AUTH_TOKEN != ''",
				Run: fmt.Sprintf(`#!/bin/bash
set -e

# Create temporary directory for package generation
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Generate npm package using Go
go run ./cmd/gpy --config gpy.yaml package --only npm --output "$TEMP_DIR/npm"

# Navigate to generated package and publish
cd "$TEMP_DIR/npm/%s"
npm publish`, npmCfg.PackageName),
			},
		},
	}
}

// publishHomebrewJob creates the publish-homebrew job for the workflow.
func publishHomebrewJob(projectRepo string, homebrewCfg model.Homebrew) WorkflowJob {
	return WorkflowJob{
		Needs:  []string{"build"},
		RunsOn: "ubuntu-latest",
		Steps: []WorkflowStep{
			{
				Name: "Checkout code",
				Uses: "actions/checkout@v4",
			},
			{
				Name: "Setup Go",
				Uses: "actions/setup-go@v4",
				With: map[string]interface{}{
					"go-version": "1.22",
				},
			},
			{
				Name: "Generate Homebrew formula and create PR",
				Env: map[string]string{
					"GH_TOKEN": "${{ secrets.HOMEBREW_CORE_PAT }}",
				},
				If: "env.GH_TOKEN != ''",
				Run: `#!/bin/bash
set -e

# Create temporary directory for formula generation
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Generate homebrew formula
go run ./cmd/gpy --config gpy.yaml package --only homebrew --output "$TEMP_DIR/homebrew"

# Configure git
git config --global user.name "gpy-bot"
git config --global user.email "gpy@github.local"

# Clone/fork homebrew-core (using GitHub CLI)
BRANCH_NAME="gpy-$(git rev-parse --short HEAD)"
gh repo fork homebrew/homebrew-core --clone --branch "$BRANCH_NAME"

# Copy generated formula to the fork
cd homebrew-core
FORMULA_FILE=$(find ../temp_dir/homebrew -name "*.rb" -type f | head -1)
FORMULA_NAME=$(basename "$FORMULA_FILE")
cp "$FORMULA_FILE" "Formula/$FORMULA_NAME"

# Commit and push
git add "Formula/$FORMULA_NAME"
git commit -m "feat: add gpy-generated formula for $(basename $FORMULA_FILE .rb)"
git push origin "$BRANCH_NAME"

# Create PR
gh pr create \
  --repo homebrew/homebrew-core \
  --title "feat: add formula for $(basename $FORMULA_FILE .rb)" \
  --body "Generated by [gpy](https://github.com/users/gpy-bot/repos)" \
  --head "$(git config --global user.name):$BRANCH_NAME"`,
			},
		},
	}
}

// publishChocolateyJob creates the publish-chocolatey job for the workflow.
func publishChocolateyJob(projectRepo string, chocolateyCfg model.Chocolatey) WorkflowJob {
	return WorkflowJob{
		Needs:  []string{"build"},
		RunsOn: "windows-latest",
		Steps: []WorkflowStep{
			{
				Name: "Checkout code",
				Uses: "actions/checkout@v4",
			},
			{
				Name: "Setup Go",
				Uses: "actions/setup-go@v4",
				With: map[string]interface{}{
					"go-version": "1.22",
				},
			},
			{
				Name: "Generate Chocolatey package",
				Run: `# Generate chocolatey package using gpy
go run ./cmd/gpy --config gpy.yaml package --only chocolatey --output "pkg"`,
			},
			{
				Name: "Pack Chocolatey package",
				Run: `# Navigate to generated package and pack it
cd "pkg\chocolatey\*"
choco pack`,
			},
			{
				Name: "Push to Chocolatey",
				Env: map[string]string{
					"ChocolateyApiKey": "${{ secrets.CHOCOLATEY_API_KEY }}",
				},
				If: "env.ChocolateyApiKey != ''",
				Run: `# Push the generated .nupkg to Chocolatey.org
cd "pkg\chocolatey\*"
$nupkg = Get-ChildItem -Filter "*.nupkg" | Select-Object -First 1
choco push "$($nupkg.FullName)" --source=https://push.chocolatey.org/`,
			},
		},
	}
}

// buildStepRun generates the build step shell script.
func buildStepRun(projectName string, goConfig model.Go) string {
	ldflags := ""
	if goConfig.LDFlags != "" {
		ldflags = fmt.Sprintf(" -ldflags \"%s -X main.Version=${{ github.ref_name }}\"", goConfig.LDFlags)
	} else {
		ldflags = " -ldflags \"-X main.Version=${{ github.ref_name }}\""
	}

	return fmt.Sprintf(
		"go build -o %s%s %s",
		projectName,
		ldflags,
		goConfig.Main,
	)
}

// archiveStepRun generates the archive creation step script.
func archiveStepRun() string {
	return `if [ "${{ runner.os }}" = "Windows" ]; then
  powershell -Command "Compress-Archive -Path '${{ matrix.bin-path }}' -DestinationPath '${{ matrix.archive }}' -Force"
else
  tar czf "${{ matrix.archive }}" "${{ matrix.bin-path }}"
fi`
}

// checksumStepRun generates the checksum generation step script.
func checksumStepRun(checksumFile string) string {
	if checksumFile == "" {
		checksumFile = "checksums.txt"
	}

	return fmt.Sprintf(`sha256sum "${{ matrix.archive }}" | sed 's/  .*\//  /' >> %s
echo "${{ matrix.archive }}" >> release-files.txt
echo %s >> release-files.txt`, checksumFile, checksumFile)
}

// marshalWorkflow converts the workflow document to YAML bytes.
func marshalWorkflow(workflow WorkflowDoc) ([]byte, error) {
	yamlData, err := yaml.Marshal(&workflow)
	if err != nil {
		return nil, fmt.Errorf("yaml.Marshal: %w", err)
	}

	// Add a header comment
	header := "# GitHub Actions workflow for releasing binaries\n" +
		"# Generated by gpy (go-package-yourself)\n" +
		"# DO NOT EDIT - regenerate with: gpy workflow --write\n\n"

	result := append([]byte(header), yamlData...)
	return result, nil
}

// New creates a new WorkflowGenerator instance.
func New() generator.Generator {
	return &WorkflowGenerator{}
}
