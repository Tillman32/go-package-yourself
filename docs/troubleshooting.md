# Troubleshooting Guide

Common issues and solutions when using gpy.

---

## Configuration Issues

### "Config not found" Error

**Error Message:**
```
error: config file not found in search paths:
  (1) gpy.yaml
  (2) gpy.yml
  (3) .gpy.yaml
  (4) .gpy.yml
  (5) go-package-yourself.yaml
  (6) go-package-yourself.yml
  (7) .go-package-yourself.yaml
  (8) .go-package-yourself.yml

use --config flag to specify explicit path
```

**Solutions:**

1. **Verify config file exists in project root:**
   ```bash
   ls -la gpy.yaml
   ```
   If not found, create it:
   ```bash
   gpy init
   ```

2. **Use explicit path if config is elsewhere:**
   ```bash
   gpy package --config /path/to/gpy.yaml
   ```

3. **Check file name spelling:**
   - Valid names: `gpy.yaml`, `gpy.yml`, `go-package-yourself.yaml`, etc.
   - Invalid: `gpy.YAML`, `GPY.yaml`, `config.yaml`
   - gpy is case-sensitive on Linux/macOS

4. **If file is in current directory but error persists:**
   ```bash
   # Verify you're in the right directory
   pwd
   ls -la gpy.yaml
   
   # Try explicit path
   gpy package --config ./gpy.yaml
   ```

---

### "Invalid field" or "Unknown placeholder" Error

**Error Message:**
```
error: field Release.Archive.NameTemplate: unknown placeholder {{typo}}
(supported: {{name}}, {{version}}, {{os}}, {{arch}}, {{ext}})
```

**Solutions:**

1. **Check placeholder spelling:**
   - ✅ Valid: `{{name}}`, `{{version}}`, `{{os}}`, `{{arch}}`, `{{ext}}`
   - ❌ Invalid: `{{project}}`, `{{binary}}`, `{{platform}}`, `{{edition}}`

2. **Supported placeholders:**
   - `{{name}}` — Project name (from `project.name`)
   - `{{version}}` — Version/tag (from git tag in workflow)
   - `{{os}}` — Operating system (darwin, linux, windows)
   - `{{arch}}` — Architecture (amd64, arm64)
   - `{{ext}}` — File extension (.tar.gz or .zip)

3. **Check your config:**
   ```yaml
   release:
     archive:
       nameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}"  # ✅ Valid
       # nameTemplate: "{{name}}_{{version}}_{{edition}}"    # ❌ Invalid
   ```

4. **Field path tells you where the error is:**
   - `field Release.Archive.NameTemplate` → Look in `release.archive.nameTemplate`
   - `field Release.TagTemplate` → Look in `release.tagTemplate`
   - `field Go.LDFlags` → Look in `go.ldflags`

---

### "Invalid platform" Error

**Error Message:**
```
error: field Release.Platforms[0].OS: invalid value "macos" (must be "darwin", "linux", or "windows")
```

**Solutions:**

1. **Use correct OS names:**
   - ✅ `darwin` (not `macos`, `osx`, `mac`)
   - ✅ `linux` (not `ubuntu`, `debian`, `gnu`)
   - ✅ `windows` (not `win`, `win32`, `win64`)

2. **Use correct architecture names:**
   - ✅ `amd64` (not `x86_64`, `x64`, `64`)
   - ✅ `arm64` (not `aarch64`, `arm32`)

3. **Check your config:**
   ```yaml
   release:
     platforms:
       - os: darwin      # ✅ Correct
         arch: amd64
       # - os: macos     # ❌ Wrong (should be "darwin")
       # - os: linux
       #   arch: i386    # ❌ Wrong (should be "amd64" or "arm64")
   ```

4. **Error context tells you which platform:**
   - `Platforms[0]` → First platform in the list
   - `Platforms[1]` → Second platform in the list
   - Check your YAML carefully at that index

---

### "Invalid format" Error

**Error Message:**
```
error: Release.Archive.Format.Windows: Windows archives must be "zip", not "tar.gz"
```

**Solutions:**

1. **Windows must use ZIP format:**
   ```yaml
   release:
     archive:
       format:
         default: "tar.gz"
         windows: "zip"  # ✅ Always required for Windows
   ```

2. **Valid formats:**
   - ✅ `tar.gz` (Linux, macOS)
   - ✅ `zip` (Windows, or any OS)

3. **Why Windows needs ZIP:**
   - Some Windows tools don't support tar.gz
   - Chocolatey expects ZIP format
   - ZIP is the Windows standard

---

## Validation Issues

### "Missing required field" Error

**Error Message:**
```
error: field Project.Name: required field is missing
error: field Project.Repo: required field is missing
error: field Go.Main: required field is missing
```

**Solutions:**

1. **Ensure required fields are present:**
   ```yaml
   project:
     name: mycli        # ✅ Required
     repo: myorg/mycli  # ✅ Required
   
   go:
     main: ./cmd/mycli  # ✅ Required
   ```

2. **Valid field values:**
   - `project.name` — 1-64 alphanumeric + hyphens (no spaces)
   - `project.repo` — `owner/repo` format (exactly one `/`)
   - `go.main` — Path to main package (must exist)

3. **Common mistakes:**
   - ❌ `project.Name` (incorrect indentation/case)
   - ❌ `project:\n    name: mycli` (looks fine, check for typos)
   - ❌ Empty/blank values (looks present but is None)

---

## Package Generation Issues

### Generated Files Look Wrong

**Symptoms:**
- npm `package.json` has wrong binary name
- Homebrew formula has wrong paths
- Chocolatey spec checksum doesn't match

**Solutions:**

1. **Check your config is correct:**
   ```bash
   cat gpy.yaml | grep -A 10 "packages:"
   ```

2. **Regenerate in clean directory:**
   ```bash
   mkdir -p /tmp/test-gpy
   cp gpy.yaml /tmp/test-gpy/
   cd /tmp/test-gpy
   gpy package
   ls -la
   ```

3. **Verify version placeholder substitution:**
   If using `{{version}}` in ldflags or archive naming:
   ```yaml
   go:
     ldflags: "-X main.Version={{version}}"
   ```
   The version is substituted by GitHub Actions workflow (not `gpy package`).
   Locally generated files will show `{{version}}` literally (correct behavior).

4. **Check archive naming:**
   ```bash
   # Run and check generated checksums.txt
   gpy package
   cat checksums.txt
   # Filenames should match: {{name}}_{{version}}_{{os}}_{{arch}}
   ```

---

### npm `package.json` Issues

**Symptoms:**
- Binary not executable after install
- Wrong binary name
- Wrong Node.js version requirement

**Solutions:**

1. **Verify npm config:**
   ```yaml
   packages:
     npm:
       enabled: true
       packageName: "mycli"   # npm install -g mycli
       binName: "mycli"       # Run: mycli --help
        nodeEngines: ">=24"    # Node.js version requirement
   ```

2. **Test npm package locally:**
   ```bash
   # Generate packages
   gpy package
   
   # Install locally (link for testing)
   npm link
   
   # Test command works
   mycli --help
   
   # Unlink when done
   npm unlink
   ```

3. **For scoped packages:**
   ```yaml
   packages:
     npm:
       packageName: "@myorg/mycli"  # Publish as scoped
       binName: "mycli"             # Run as: mycli --help (not @myorg/mycli)
   ```

---

### Homebrew Formula Issues

**Symptoms:**
- Formula doesn't install
- Wrong architecture support
- Checksum verification fails

**Solutions:**

1. **Verify Homebrew config:**
   ```yaml
   packages:
     homebrew:
       enabled: true
       formulaName: "MyCli"    # brew install mycli
       tap: "myorg/homebrew-tools"  # Optional custom tap
   ```

2. **Test formula locally (if you have Homebrew):**
   ```bash
   # Generate
   gpy package
   
   # Install from local formula (requires Ruby environment)
   brew install ./homebrew.rb
   
   # Verify
   mycli --help
   ```

3. **Check SHA256 hash:**
   ```bash
   # Generated homebrew.rb should have correct SHA256
   grep sha256 homebrew.rb
   
   # Should match checksums.txt
   cat checksums.txt | grep darwin_amd64
   ```

---

### Chocolatey Package Issues

**Symptoms:**
- PowerShell hash mismatch
- 32-bit vs 64-bit detection fails
- XML spec is invalid

**Solutions:**

1. **Verify Chocolatey config:**
   ```yaml
   packages:
     chocolatey:
       enabled: true
       packageId: "mycli"
       authors: "Your Name"
   ```

2. **Check Windows platform configuration:**
   ```yaml
   release:
     platforms:
       - os: windows
         arch: amd64  # ✅ Only amd64 supported for Chocolatey (no arm64)
   ```

3. **Verify checksum format:**
   ```bash
   # Generated chocolatey.ps1 should have correct SHA256
   # that matches checksums.txt (Windows archive, .zip format)
   cat checksums.txt | grep windows_amd64
   ```

---

## GitHub Actions Workflow Issues

### Workflow Not Triggered

**Symptoms:**
- Pushed tag `v1.0.0` but workflow didn't run
- Workflow appears disabled

**Solutions:**

1. **Check GitHub Actions workflow file exists:**
   ```bash
   ls -la .github/workflows/release.yml
   ```

2. **Generate workflow if missing:**
   ```bash
   gpy workflow --write
   ```

3. **Verify tag pattern matches:**
   ```yaml
   github:
     workflows:
       tagPatterns: ["v*"]  # Matches: v1.0.0, v2.1.3, etc.
   ```

4. **Check your tag matches pattern:**
   ```bash
   # If tagPatterns is ["v*"], tags must start with 'v'
   git tag v1.0.0          # ✅ Will trigger
   # git tag 1.0.0          # ❌ Won't trigger (if pattern is "v*")
   
   # If tagPatterns is ["release-*"]
   git tag release-1.0.0   # ✅ Will trigger
   ```

5. **Check GitHub Actions is enabled:**
   - Go to your repo → Settings → Actions → General
   - Ensure "Actions permissions" is not disabled

6. **Check workflow syntax:**
   ```bash
   # View workflow file
   cat .github/workflows/release.yml
   
   # GitHub validates on push; check Actions tab for errors
   ```

---

### Build Fails in Workflow

**Symptoms:**
- Workflow runs but build step fails
- "go build" failed

**Solutions:**

1. **Verify Go build works locally:**
   ```bash
   go build -o gpy ./cmd/mycli
   ```

2. **Check build flags/ldflags:**
   ```yaml
   go:
     main: ./cmd/mycli
     ldflags: "-X main.Version={{version}}"
   ```

3. **Workflow uses git tag as version:**
   In the workflow, `{{version}}` is replaced with Git tag:
   ```bash
   go build -ldflags "-X main.Version=${{ github.ref_name }}"
   ```

4. **Check CGO setting:**
   If your code uses CGO:
   ```yaml
   go:
     cgo: true  # Enables CGO_ENABLED=1 in workflow
   ```

---

## Performance & Size

### Generated Files Are Very Large

**Symptoms:**
- npm `package.json` is large
- Homebrew formula has lots of duplication

**Cause:** Multi-platform support (multiple OS/arch combinations)

**This is expected.** Each platform needs:
- Its own archive download URL
- Its own SHA256 hash
- Platform-specific detection logic

---

### Build Takes Too Long

**Symptoms:**
- GitHub Actions workflow takes 10+ minutes

**Solutions:**

1. **Reduce platforms:**
   ```yaml
   release:
     platforms:
       - os: darwin
         arch: amd64
       - os: linux
         arch: amd64
       # Removed: arm64 variants, windows
   ```

2. **Use parallel matrix builds (already enabled):**
   Workflow matrix automatically builds platforms in parallel.

3. **Optimize Go build:**
   ```yaml
   go:
     ldflags: "-s -w"  # Strip debug symbols
   ```

---

## Common Workflows & Solutions

### I need to rebuild the npm package

```bash
# Regenerate all packages
gpy package

# Review generated package.json
cat package.json

# Publish to npm (if auth configured)
npm publish
```

### I need to update my config

```bash
# Edit existing config
nano gpy.yaml

# Or interactive wizard
gpy init

# Regenerate packages
gpy package
```

### I need to test locally before releasing

```bash
# Generate all packages
gpy package

# Test npm locally
npm link
mycli --help
npm unlink

# Commit and push
git add gpy.yaml package.json homebrew.rb chocolatey.ps1
git commit -m "chore: update gpy packages"
git push

# When ready, tag and push tag (triggers workflow)
git tag v1.0.0
git push origin v1.0.0
```

### I need to skip a package manager

```yaml
packages:
  npm:
    enabled: true
  homebrew:
    enabled: false    # Skip this
  chocolatey:
    enabled: true
```

Then regenerate:
```bash
gpy package
```

---

## Getting Help

If your issue isn't listed here:

1. **Check the config file matches examples:**
   - See [docs/config-reference.md](config-reference.md)
   - See [docs/examples/](examples/)

2. **Run with more verbose output:**
   - Most errors include field path context
   - Error message tells you exactly what's wrong

3. **Create an issue:**
   - Include your `gpy.yaml` (sanitize secrets)
   - Include error message and steps to reproduce
   - Include Go version: `go version`

---

## Tips & Tricks

### Use `--config` for testing

```bash
# Test with alternative config
gpy package --config /tmp/test.yaml
```

### Preview changes without writing

```bash
# Print to stdout (don't write files)
gpy workflow
cat << 'EOF'  # Copy output
EOF
```

### Validate config without generating

```bash
# Load and validate config
gpy package --yes --config gpy.yaml
```

---

**Still stuck?** Open a GitHub issue with:
- Error message (exact)
- Your `gpy.yaml` (sanitized)
- Steps to reproduce
- Go version (`go version`)
