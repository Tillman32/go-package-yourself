// Package cli implements the gpy command-line interface.
package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// GlobalOpts holds parsed global flags that apply to all commands.
type GlobalOpts struct {
	ConfigPath  string
	ProjectRoot string
	NoTUI       bool
	Yes         bool
}

// Execute parses command-line arguments and routes to the appropriate subcommand.
func Execute(args []string) error {
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	// Parse global flags that come before the command
	globalOpts, remaining, err := parseGlobalFlags(args)
	if err != nil {
		return err
	}

	// --yes implies --no-tui
	if globalOpts.Yes {
		globalOpts.NoTUI = true
	}

	if len(remaining) == 0 {
		printUsage()
		os.Exit(1)
	}

	command := remaining[0]
	commandArgs := remaining[1:]

	// Route to subcommand
	switch command {
	case "help", "-h", "--help":
		printUsage()
		return nil

	case "init":
		return Init(globalOpts, commandArgs)

	case "package":
		return Package(globalOpts, commandArgs)

	case "workflow":
		return Workflow(globalOpts, commandArgs)

	default:
		return fmt.Errorf("unknown command %q\n%s", command, suggestCommand(command))
	}
}

// parseGlobalFlags parses global flags that come before the subcommand.
// Returns the parsed options, remaining args (command + command args), and any error.
func parseGlobalFlags(args []string) (*GlobalOpts, []string, error) {
	opts := &GlobalOpts{
		ProjectRoot: ".",
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			// End of global flags; this is the command name
			return opts, args[i:], nil
		}

		if arg == "--help" || arg == "-h" {
			printUsage()
			os.Exit(0)
		}

		if arg == "--no-tui" {
			opts.NoTUI = true
			continue
		}

		if arg == "--yes" {
			opts.Yes = true
			continue
		}

		if arg == "--config" {
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag --config requires a value")
			}
			opts.ConfigPath = args[i+1]
			i++ // skip next arg
			continue
		}

		if arg == "--project-root" {
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag --project-root requires a value")
			}
			opts.ProjectRoot = args[i+1]
			i++ // skip next arg
			continue
		}

		// Handle --flag=value format for global flags
		if strings.HasPrefix(arg, "--config=") {
			opts.ConfigPath = strings.TrimPrefix(arg, "--config=")
			continue
		}

		if strings.HasPrefix(arg, "--project-root=") {
			opts.ProjectRoot = strings.TrimPrefix(arg, "--project-root=")
			continue
		}

		return nil, nil, fmt.Errorf("unknown global flag: %s", arg)
	}

	// All args were flags; no command found
	return opts, nil, nil
}

// suggestCommand suggests possible commands based on a misspelled input.
func suggestCommand(input string) string {
	commands := []string{"init", "package", "workflow"}
	for _, cmd := range commands {
		if levenshteinDistance(input, cmd) <= 1 {
			return fmt.Sprintf("Did you mean: gpy %s?", cmd)
		}
	}
	return "Available commands: init, package, workflow"
}

// levenshteinDistance computes edit distance between two strings.
func levenshteinDistance(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			curr[j] = min(
				curr[j-1]+1, // insert
				min(prev[j]+1, // delete
					prev[j-1]+cost), // substitute
			)
		}
		prev, curr = curr, prev
	}

	return prev[len(b)]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `gpy - Go Package Yourself: YAML-driven multi-channel packager

Usage:
  gpy [global flags] <command> [command flags]

Global flags:
  --config <path>        Path to config file (auto-detected if omitted)
  --project-root <path>  Project root directory (default: current working directory)
  --no-tui               Disable interactive prompts; fail if required fields are missing
  --yes                  Accept all defaults; implies --no-tui
  -h, --help            Show this help message

Commands:
  init                   Create a new go-package-yourself.yaml config
  package                Generate packaging artifacts (npm, homebrew, chocolatey)
  workflow               Generate GitHub Actions workflow file

Examples:
  gpy init
  gpy init --yes
  gpy package
  gpy package --only npm,homebrew
  gpy workflow --write

For more information on a command:
  gpy <command> -h

`)
}

// parseCommandFlags parses flags for a specific subcommand using the flag package.
// This returns a function that can be called to parse flag.NewFlagSet results.
func newFlagSet(commandName string) *flag.FlagSet {
	fs := flag.NewFlagSet(commandName, flag.ContinueOnError)
	return fs
}
