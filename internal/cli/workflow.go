package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"go-package-yourself/internal/config"
	"go-package-yourself/internal/generator"
	"go-package-yourself/internal/generator/workflow"
	"go-package-yourself/internal/validate"
)

// Workflow implements the `gpy workflow` command.
// It generates a GitHub Actions release workflow file.
func Workflow(opts *GlobalOpts, args []string) error {
	// Parse command-specific flags
	fs := flag.NewFlagSet("workflow", flag.ContinueOnError)
	write := fs.Bool("write", false, "write workflow to the configured file path")

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			printWorkflowUsage()
			return nil
		}
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	// Change to project root
	if err := os.Chdir(opts.ProjectRoot); err != nil {
		return fmt.Errorf("failed to change to project root %q: %w", opts.ProjectRoot, err)
	}

	// Get absolute project root
	absProjectRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Load configuration
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w\n\nNext step: Run 'gpy init --yes' to create a starter config", err)
	}

	// Validate configuration
	if err := validate.Config(cfg); err != nil {
		return fmt.Errorf("invalid configuration: %w\n\nNext step: Edit gpy.yaml to fix the issues", err)
	}

	// Create generator context (no version for workflow, uses runtime values)
	ctx := createGeneratorContext(cfg, absProjectRoot, "")

	// Generate workflow (stub for now)
	outputs, err := generatorWorkflow(ctx)
	if err != nil {
		return fmt.Errorf("workflow generation failed: %w", err)
	}

	if len(outputs) == 0 {
		return fmt.Errorf("workflow generation produced no output")
	}

	workflowOutput := outputs[0]
	workflowYAML := string(workflowOutput.Content)

	if *write {
		// Use configured workflow file path (should be set by ApplyDefaults)
		workflowFilePath := cfg.GitHub.Workflows.WorkflowFile
		if workflowFilePath == "" {
			return fmt.Errorf("WorkflowFile not configured (should have been set by config defaults)")
		}
		workflowPath := filepath.Join(absProjectRoot, workflowFilePath)
		dir := filepath.Dir(workflowPath)

		// Create .github and .github/workflows directories if they don't exist
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create workflow directory %q: %w", dir, err)
		}

		// Check if workflow file already exists to prevent overwriting
		if _, err := os.Stat(workflowPath); err == nil {
			return fmt.Errorf("workflow file already exists at %q\n\nTo replace it, delete the existing file and run gpy workflow --write again", workflowFilePath)
		}

		// Write workflow file
		if err := os.WriteFile(workflowPath, []byte(workflowYAML), 0o644); err != nil {
			return fmt.Errorf("failed to write workflow file %q: %w", workflowPath, err)
		}

		fmt.Printf("✓ Workflow written to: %s\n", workflowFilePath)
		fmt.Printf("✓ Workflow file created successfully\n")
		fmt.Printf("Next step: Push to GitHub to enable automated releases\n")
	} else {
		// Print to stdout
		fmt.Println(workflowYAML)
	}

	return nil
}

// generatorWorkflow delegates to the workflow generator to create GitHub Actions workflows.
//
// The workflow generator implements the Generator interface and produces
// a single FileOutput containing the complete release.yml workflow.
func generatorWorkflow(ctx generator.Context) ([]generator.FileOutput, error) {
	gen := workflow.New()
	return gen.Generate(ctx)
}

func printWorkflowUsage() {
	fmt.Fprintf(os.Stderr, `gpy workflow - Generate GitHub Actions workflow

Usage:
  gpy workflow [flags]

Flags:
  --write                Write workflow to the configured file path
                        Without this flag, prints YAML to stdout
  -h, --help             Show this help message

Global flags:
  --config <path>        Path to config file (auto-detected if omitted)
  --project-root <path>  Project root directory (default: current working directory)

Examples:
   gpy workflow          # Print workflow YAML to stdout
   gpy workflow --write  # Write workflow to .github/workflows/gpy-release.yaml
   gpy --project-root /path/to/project workflow --write

Description:
  Generates a GitHub Actions workflow file for automating releases.
  The workflow file path is configured in gpy.yaml.

  Without --write, prints the generated YAML to stdout so you can review it.
  With --write, creates the workflow file in the configured location.

  Exit codes:
    0  Success
    1  Error (invalid config, file I/O error, workflows disabled, etc.)

`)
}
