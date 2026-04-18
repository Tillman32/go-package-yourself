package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitInteractiveWritesProjectMetadata(t *testing.T) {
	tmpdir := t.TempDir()

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Fatalf("failed to restore cwd: %v", err)
		}
	}()

	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	input := strings.Join([]string{
		"mytool",
		"owner/mytool",
		"A tool for packaging Go binaries",
		"https://example.com/mytool",
		"MIT",
		"",
		"n",
		"n",
		"n",
		"n",
		"n",
	}, "\n") + "\n"

	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	if _, err := stdinWriter.WriteString(input); err != nil {
		t.Fatalf("failed to write test input: %v", err)
	}
	if err := stdinWriter.Close(); err != nil {
		t.Fatalf("failed to close stdin writer: %v", err)
	}

	os.Stdin = stdinReader
	os.Stdout = stdoutWriter

	err = Init(&GlobalOpts{ProjectRoot: tmpdir}, nil)

	if err := stdoutWriter.Close(); err != nil {
		t.Fatalf("failed to close stdout writer: %v", err)
	}

	var stdout bytes.Buffer
	if _, readErr := stdout.ReadFrom(stdoutReader); readErr != nil {
		t.Fatalf("failed to read stdout: %v", readErr)
	}

	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	configPath := filepath.Join(tmpdir, "gpy.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read generated config: %v", err)
	}

	config := string(data)
	checks := []string{
		"name: mytool",
		"repo: owner/mytool",
		"description: \"A tool for packaging Go binaries\"",
		"homepage: \"https://example.com/mytool\"",
		"license: \"MIT\"",
		"main: ./cmd/mytool",
	}
	for _, check := range checks {
		if !strings.Contains(config, check) {
			t.Fatalf("generated config missing %q\nconfig:\n%s", check, config)
		}
	}

	if strings.Contains(config, "# description: Short description of your tool") {
		t.Fatalf("generated config kept description comment instead of writing value\nconfig:\n%s", config)
	}

	if !strings.Contains(stdout.String(), "Created config file") {
		t.Fatalf("stdout missing success message: %s", stdout.String())
	}
}

func TestApplyDefaultsSetsHomepageFromGitRemote(t *testing.T) {
	tmpdir := t.TempDir()

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Fatalf("failed to restore cwd: %v", err)
		}
	}()

	commands := [][]string{
		{"init"},
		{"remote", "add", "origin", "https://github.com/example/testtool.git"},
	}
	for _, args := range commands {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpdir
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
		}
	}

	if err := os.Chdir(tmpdir); err != nil {
		t.Fatalf("failed to chdir to temp repo: %v", err)
	}

	cfg := defaultConfig()
	if err := applyDefaults(cfg, tmpdir); err != nil {
		t.Fatalf("applyDefaults failed: %v", err)
	}

	if cfg.Project.Repo != "example/testtool" {
		t.Fatalf("Project.Repo = %q, want %q", cfg.Project.Repo, "example/testtool")
	}
	if cfg.Project.Homepage != "https://github.com/example/testtool" {
		t.Fatalf("Project.Homepage = %q, want %q", cfg.Project.Homepage, "https://github.com/example/testtool")
	}
	if cfg.Project.License != "" {
		t.Fatalf("Project.License = %q, want empty string", cfg.Project.License)
	}
	if cfg.Project.Description != "" {
		t.Fatalf("Project.Description = %q, want empty string", cfg.Project.Description)
	}
}

func TestDetectLicense(t *testing.T) {
	tmpdir := t.TempDir()

	licenseText := `MIT License

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software")`
	if err := os.WriteFile(filepath.Join(tmpdir, "LICENSE.md"), []byte(licenseText), 0o644); err != nil {
		t.Fatalf("failed to write license file: %v", err)
	}

	got, source := detectLicense(tmpdir)
	if got != "MIT" {
		t.Fatalf("detectLicense() = %q, want %q", got, "MIT")
	}
	if source != "LICENSE.md" {
		t.Fatalf("detectLicense() source = %q, want %q", source, "LICENSE.md")
	}
}

func TestInitInteractiveUsesDetectedLicenseDefault(t *testing.T) {
	tmpdir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpdir, "LICENSE"), []byte("MIT License\n\nPermission is hereby granted, free of charge"), 0o644); err != nil {
		t.Fatalf("failed to write LICENSE: %v", err)
	}

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldCwd); err != nil {
			t.Fatalf("failed to restore cwd: %v", err)
		}
	}()

	oldStdin := os.Stdin
	oldStdout := os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()

	input := strings.Join([]string{
		"mytool",
		"owner/mytool",
		"A tool for packaging Go binaries",
		"https://example.com/mytool",
		"",
		"",
		"n",
		"n",
		"n",
		"n",
		"n",
	}, "\n") + "\n"

	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	if _, err := stdinWriter.WriteString(input); err != nil {
		t.Fatalf("failed to write test input: %v", err)
	}
	if err := stdinWriter.Close(); err != nil {
		t.Fatalf("failed to close stdin writer: %v", err)
	}

	os.Stdin = stdinReader
	os.Stdout = stdoutWriter

	err = Init(&GlobalOpts{ProjectRoot: tmpdir}, nil)

	if err := stdoutWriter.Close(); err != nil {
		t.Fatalf("failed to close stdout writer: %v", err)
	}

	var stdout bytes.Buffer
	if _, readErr := stdout.ReadFrom(stdoutReader); readErr != nil {
		t.Fatalf("failed to read stdout: %v", readErr)
	}

	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpdir, "gpy.yaml"))
	if err != nil {
		t.Fatalf("failed to read generated config: %v", err)
	}

	config := string(data)
	if !strings.Contains(config, "license: \"MIT\"") {
		t.Fatalf("generated config missing detected license\nconfig:\n%s", config)
	}
	if !strings.Contains(stdout.String(), "Project license (SPDX identifier) [MIT from LICENSE]:") {
		t.Fatalf("stdout missing detected license prompt: %s", stdout.String())
	}
}
