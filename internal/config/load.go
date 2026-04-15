// Package config provides configuration loading and discovery for gpy.
package config

import (
	"fmt"
	"os"

	"go-package-yourself/internal/model"

	"gopkg.in/yaml.v3"
)

// Load discovers and loads a gpy configuration file.
//
// If configPath is provided, it loads from that explicit path.
// Otherwise, it searches the current working directory for the first matching filename:
//   - gpy.yaml
//   - gpy.yml
//   - .gpy.yaml
//   - .gpy.yml
//   - go-package-yourself.yaml
//   - go-package-yourself.yml
//   - .go-package-yourself.yaml
//   - .go-package-yourself.yml
//
// The YAML is parsed into a Config struct, and sensible defaults are applied.
// Returns a helpful error message if no config is found after searching.
//
// Example:
//
//	cfg, err := Load("")  // auto-discover from current dir
//	if err != nil {
//		log.Fatalf("failed to load config: %v", err)
//	}
//
//	cfg, err := Load("./custom/gpy.yaml")  // explicit path
//	if err != nil {
//		log.Fatalf("failed to load config: %v", err)
//	}
func Load(configPath string) (*model.Config, error) {
	var path string
	var err error

	if configPath != "" {
		// Explicit path provided; use it directly
		path = configPath
	} else {
		// Auto-discover from current working directory
		path, err = discover()
		if err != nil {
			return nil, err
		}
	}

	// Read and parse YAML
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", path, err)
	}

	cfg := &model.Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w", path, err)
	}

	// Apply sensible defaults
	ApplyDefaults(cfg)

	return cfg, nil
}

// discover searches for a gpy config file in the current working directory.
// Returns the path to the first file found, or an error if no config file exists.
func discover() (string, error) {
	// Filenames to search for, in order of preference
	candidates := []string{
		"gpy.yaml",
		"gpy.yml",
		".gpy.yaml",
		".gpy.yml",
		"go-package-yourself.yaml",
		"go-package-yourself.yml",
		".go-package-yourself.yaml",
		".go-package-yourself.yml",
	}

	for _, name := range candidates {
		if _, err := os.Stat(name); err == nil {
			return name, nil
		}
	}

	return "", fmt.Errorf(
		"no gpy config file found in current directory\n"+
			"expected one of: %v\n"+
			"run 'gpy init' to create one",
		candidates,
	)
}
