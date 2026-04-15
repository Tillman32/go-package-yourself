package docker

import (
	"strings"
	"testing"

	"go-package-yourself/internal/generator"
	"go-package-yourself/internal/model"
)

// makeCtx creates a minimal generator.Context for docker tests.
func makeCtx(cfg *model.Config) generator.Context {
	return generator.Context{
		Config: cfg,
	}
}

// baseConfig returns a valid config with docker enabled.
func baseConfig() *model.Config {
	return &model.Config{
		Project: model.Project{
			Name: "mytool",
			Repo: "owner/mytool",
		},
		Go: model.Go{
			Main: "./cmd/mytool",
		},
		Packages: model.Packages{
			Docker: model.Docker{
				Enabled:   true,
				ImageName: "mytool",
				Cmd:       "./{{name}}",
			},
		},
	}
}

// TestDockerGeneratorName verifies the generator name.
func TestDockerGeneratorName(t *testing.T) {
	gen := New()
	if gen.Name() != "docker" {
		t.Errorf("Name() = %q, want %q", gen.Name(), "docker")
	}
}

// TestDockerGenerator_Disabled verifies nil,nil is returned when disabled.
func TestDockerGenerator_Disabled(t *testing.T) {
	cfg := baseConfig()
	cfg.Packages.Docker.Enabled = false

	gen := New()
	outputs, err := gen.Generate(makeCtx(cfg))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outputs != nil {
		t.Errorf("expected nil outputs when disabled, got %d", len(outputs))
	}
}

// TestDockerGenerator_BasicOutput verifies Dockerfile and .dockerignore are produced.
func TestDockerGenerator_BasicOutput(t *testing.T) {
	gen := New()
	outputs, err := gen.Generate(makeCtx(baseConfig()))
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if len(outputs) != 2 {
		t.Fatalf("expected 2 outputs (Dockerfile + .dockerignore), got %d", len(outputs))
	}

	paths := map[string]bool{}
	for _, o := range outputs {
		paths[o.Path] = true
		if o.Mode != 0o644 {
			t.Errorf("%s: Mode = %o, want %o", o.Path, o.Mode, 0o644)
		}
		if len(o.Content) == 0 {
			t.Errorf("%s: Content is empty", o.Path)
		}
	}

	if !paths["Dockerfile"] {
		t.Errorf("Dockerfile not in outputs")
	}
	if !paths[".dockerignore"] {
		t.Errorf(".dockerignore not in outputs")
	}
}

// TestDockerGenerator_DockerfileContent verifies the Dockerfile structure.
func TestDockerGenerator_DockerfileContent(t *testing.T) {
	gen := New()
	outputs, err := gen.Generate(makeCtx(baseConfig()))
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var dockerfile string
	for _, o := range outputs {
		if o.Path == "Dockerfile" {
			dockerfile = string(o.Content)
		}
	}

	checks := []struct {
		desc string
		want string
	}{
		{"multi-stage builder", "FROM golang:1.23-alpine AS builder"},
		{"runtime base", "FROM alpine:latest"},
		{"ARG VERSION", "ARG VERSION=dev"},
		{"binary name in build", "-o mytool"},
		{"go.main path", "./cmd/mytool"},
		{"CGO disabled by default", "CGO_ENABLED=0"},
		{"version ldflags", "-X main.version=${VERSION}"},
		{"non-root user creation", "adduser"},
		{"binary copy", "COPY --from=builder /app/mytool ."},
		{"CMD default", `CMD ["./mytool"]`},
		{"do-not-edit header", "DO NOT EDIT"},
	}

	for _, c := range checks {
		if !strings.Contains(dockerfile, c.want) {
			t.Errorf("%s: Dockerfile missing %q", c.desc, c.want)
		}
	}
}

// TestDockerGenerator_WithPort verifies EXPOSE is emitted when port > 0.
func TestDockerGenerator_WithPort(t *testing.T) {
	cfg := baseConfig()
	cfg.Packages.Docker.Port = 8080

	gen := New()
	outputs, err := gen.Generate(makeCtx(cfg))
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var dockerfile string
	for _, o := range outputs {
		if o.Path == "Dockerfile" {
			dockerfile = string(o.Content)
		}
	}

	if !strings.Contains(dockerfile, "EXPOSE 8080") {
		t.Errorf("expected EXPOSE 8080 in Dockerfile:\n%s", dockerfile)
	}
}

// TestDockerGenerator_NoExpose verifies EXPOSE is absent when port is 0.
func TestDockerGenerator_NoExpose(t *testing.T) {
	cfg := baseConfig()
	cfg.Packages.Docker.Port = 0

	gen := New()
	outputs, err := gen.Generate(makeCtx(cfg))
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var dockerfile string
	for _, o := range outputs {
		if o.Path == "Dockerfile" {
			dockerfile = string(o.Content)
		}
	}

	if strings.Contains(dockerfile, "EXPOSE") {
		t.Errorf("expected no EXPOSE in Dockerfile when port=0")
	}
}

// TestDockerGenerator_WithEnv verifies ENV lines are emitted sorted.
func TestDockerGenerator_WithEnv(t *testing.T) {
	cfg := baseConfig()
	cfg.Packages.Docker.Env = map[string]string{
		"LOG_LEVEL": "info",
		"CACHE_DIR": "/data",
		"APP_MODE":  "production",
	}

	gen := New()
	outputs, err := gen.Generate(makeCtx(cfg))
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var dockerfile string
	for _, o := range outputs {
		if o.Path == "Dockerfile" {
			dockerfile = string(o.Content)
		}
	}

	// Verify all ENV lines present
	for _, want := range []string{"ENV APP_MODE=production", "ENV CACHE_DIR=/data", "ENV LOG_LEVEL=info"} {
		if !strings.Contains(dockerfile, want) {
			t.Errorf("expected %q in Dockerfile", want)
		}
	}

	// Verify sorted order: APP_MODE before CACHE_DIR before LOG_LEVEL
	posApp := strings.Index(dockerfile, "ENV APP_MODE")
	posCache := strings.Index(dockerfile, "ENV CACHE_DIR")
	posLog := strings.Index(dockerfile, "ENV LOG_LEVEL")
	if posApp >= posCache || posCache >= posLog {
		t.Errorf("ENV lines not sorted: APP_MODE=%d, CACHE_DIR=%d, LOG_LEVEL=%d", posApp, posCache, posLog)
	}
}

// TestDockerGenerator_WithCGO verifies CGO_ENABLED=1 when cgo is true.
func TestDockerGenerator_WithCGO(t *testing.T) {
	cfg := baseConfig()
	cfg.Go.CGO = true

	gen := New()
	outputs, err := gen.Generate(makeCtx(cfg))
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var dockerfile string
	for _, o := range outputs {
		if o.Path == "Dockerfile" {
			dockerfile = string(o.Content)
		}
	}

	if !strings.Contains(dockerfile, "CGO_ENABLED=1") {
		t.Errorf("expected CGO_ENABLED=1 when cgo=true")
	}
}

// TestDockerGenerator_WithLDFlags verifies user ldflags are prepended.
func TestDockerGenerator_WithLDFlags(t *testing.T) {
	cfg := baseConfig()
	cfg.Go.LDFlags = "-s -w"

	gen := New()
	outputs, err := gen.Generate(makeCtx(cfg))
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var dockerfile string
	for _, o := range outputs {
		if o.Path == "Dockerfile" {
			dockerfile = string(o.Content)
		}
	}

	if !strings.Contains(dockerfile, "-s -w -X main.version=${VERSION}") {
		t.Errorf("expected combined ldflags in Dockerfile:\n%s", dockerfile)
	}
}

// TestDockerGenerator_CustomCmd verifies CMD is rendered from config.
func TestDockerGenerator_CustomCmd(t *testing.T) {
	cfg := baseConfig()
	cfg.Packages.Docker.Cmd = "/usr/local/bin/{{name}}"

	gen := New()
	outputs, err := gen.Generate(makeCtx(cfg))
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var dockerfile string
	for _, o := range outputs {
		if o.Path == "Dockerfile" {
			dockerfile = string(o.Content)
		}
	}

	if !strings.Contains(dockerfile, `CMD ["/usr/local/bin/mytool"]`) {
		t.Errorf("expected custom CMD in Dockerfile:\n%s", dockerfile)
	}
}

// TestDockerGenerator_DockerignoreContent verifies key paths are excluded.
func TestDockerGenerator_DockerignoreContent(t *testing.T) {
	gen := New()
	outputs, err := gen.Generate(makeCtx(baseConfig()))
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	var dockerignore string
	for _, o := range outputs {
		if o.Path == ".dockerignore" {
			dockerignore = string(o.Content)
		}
	}

	for _, want := range []string{".git", "packaging", "*.exe", "*.test", "dist", "vendor"} {
		if !strings.Contains(dockerignore, want) {
			t.Errorf(".dockerignore missing %q", want)
		}
	}
}

// TestDockerGenerator_DeterministicOutput verifies same config produces identical output.
func TestDockerGenerator_DeterministicOutput(t *testing.T) {
	cfg := baseConfig()
	cfg.Packages.Docker.Env = map[string]string{
		"B": "2",
		"A": "1",
		"C": "3",
	}
	cfg.Packages.Docker.Port = 9090

	gen := New()
	ctx := makeCtx(cfg)

	out1, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("first Generate failed: %v", err)
	}
	out2, err := gen.Generate(ctx)
	if err != nil {
		t.Fatalf("second Generate failed: %v", err)
	}

	for i, o := range out1 {
		if string(o.Content) != string(out2[i].Content) {
			t.Errorf("%s: outputs differ between runs", o.Path)
		}
	}
}
