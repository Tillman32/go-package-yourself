package naming

import (
	"strings"
	"testing"
)

func TestExtensionFor(t *testing.T) {
	tests := []struct {
		format      string
		os          string
		expected    string
		expectError bool
	}{
		{"tar.gz", "linux", ".tar.gz", false},
		{"targz", "linux", ".tar.gz", false},
		{"zip", "windows", ".zip", false},
		{"ZIP", "windows", ".zip", false},
		{"", "linux", ".tar.gz", false}, // Default
		{"invalid", "linux", "", true},
		{"tar", "linux", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.format+"_"+tt.os, func(t *testing.T) {
			result, err := ExtensionFor(tt.format, tt.os)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestArchiveName(t *testing.T) {
	tests := []struct {
		name            string
		params          ArchiveNameParams
		expectedArchive string
		expectedBinPath string
		expectError     bool
		errorMsg        string
	}{
		{
			name: "linux_amd64_tar_gz",
			params: ArchiveNameParams{
				Name:                "mytool",
				Version:             "1.2.3",
				OS:                  "linux",
				Arch:                "amd64",
				Format:              "tar.gz",
				ArchiveNameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathTemplate:     "{{name}}",
			},
			expectedArchive: "mytool_1.2.3_linux_amd64.tar.gz",
			expectedBinPath: "mytool",
			expectError:     false,
		},
		{
			name: "darwin_arm64_tar_gz",
			params: ArchiveNameParams{
				Name:                "mytool",
				Version:             "1.2.3",
				OS:                  "darwin",
				Arch:                "arm64",
				Format:              "tar.gz",
				ArchiveNameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathTemplate:     "bin/{{name}}",
			},
			expectedArchive: "mytool_1.2.3_darwin_arm64.tar.gz",
			expectedBinPath: "bin/mytool",
			expectError:     false,
		},
		{
			name: "windows_amd64_zip",
			params: ArchiveNameParams{
				Name:                "mytool",
				Version:             "1.2.3",
				OS:                  "windows",
				Arch:                "amd64",
				Format:              "zip",
				ArchiveNameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathTemplate:     "{{name}}",
			},
			expectedArchive: "mytool_1.2.3_windows_amd64.zip",
			expectedBinPath: "mytool.exe",
			expectError:     false,
		},
		{
			name: "windows_already_has_exe",
			params: ArchiveNameParams{
				Name:                "mytool",
				Version:             "1.2.3",
				OS:                  "windows",
				Arch:                "arm64",
				Format:              "zip",
				ArchiveNameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathTemplate:     "{{name}}.exe",
			},
			expectedArchive: "mytool_1.2.3_windows_arm64.zip",
			expectedBinPath: "mytool.exe",
			expectError:     false,
		},
		{
			name: "empty_name",
			params: ArchiveNameParams{
				Name:                "",
				Version:             "1.2.3",
				OS:                  "linux",
				Arch:                "amd64",
				Format:              "tar.gz",
				ArchiveNameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathTemplate:     "{{name}}",
			},
			expectError: true,
			errorMsg:    "name is required",
		},
		{
			name: "invalid_os",
			params: ArchiveNameParams{
				Name:                "mytool",
				Version:             "1.2.3",
				OS:                  "freebsd",
				Arch:                "amd64",
				Format:              "tar.gz",
				ArchiveNameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathTemplate:     "{{name}}",
			},
			expectError: true,
			errorMsg:    "invalid os",
		},
		{
			name: "invalid_arch",
			params: ArchiveNameParams{
				Name:                "mytool",
				Version:             "1.2.3",
				OS:                  "linux",
				Arch:                "386",
				Format:              "tar.gz",
				ArchiveNameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathTemplate:     "{{name}}",
			},
			expectError: true,
			errorMsg:    "invalid arch",
		},
		{
			name: "invalid_format",
			params: ArchiveNameParams{
				Name:                "mytool",
				Version:             "1.2.3",
				OS:                  "linux",
				Arch:                "amd64",
				Format:              "7z",
				ArchiveNameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathTemplate:     "{{name}}",
			},
			expectError: true,
			errorMsg:    "unsupported archive format",
		},
		{
			name: "empty_version_preserves_placeholder",
			params: ArchiveNameParams{
				Name:                "mytool",
				Version:             "",
				OS:                  "linux",
				Arch:                "amd64",
				Format:              "tar.gz",
				ArchiveNameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
				BinPathTemplate:     "{{name}}",
			},
			expectedArchive: "mytool__linux_amd64.tar.gz",
			expectedBinPath: "mytool",
			expectError:     false,
		},
		{
			name: "custom_bin_path_template",
			params: ArchiveNameParams{
				Name:                "cli",
				Version:             "2.0.0",
				OS:                  "darwin",
				Arch:                "arm64",
				Format:              "tar.gz",
				ArchiveNameTemplate: "{{name}}-{{version}}-{{os}}.tar.gz",
				BinPathTemplate:     "bin/{{name}}",
			},
			expectedArchive: "cli-2.0.0-darwin.tar.gz.tar.gz",
			expectedBinPath: "bin/cli",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive, binPath, err := ArchiveName(tt.params)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if archive != tt.expectedArchive {
					t.Errorf("archive: expected %q, got %q", tt.expectedArchive, archive)
				}
				if binPath != tt.expectedBinPath {
					t.Errorf("binPath: expected %q, got %q", tt.expectedBinPath, binPath)
				}
			}
		})
	}
}

func TestArchiveNameAllCombinations(t *testing.T) {
	// Test all valid OS/arch combinations
	oses := []string{"darwin", "linux", "windows"}
	arches := []string{"amd64", "arm64"}
	formats := []string{"tar.gz", "zip"}

	for _, os := range oses {
		for _, arch := range arches {
			for _, format := range formats {
				t.Run(os+"_"+arch+"_"+format, func(t *testing.T) {
					params := ArchiveNameParams{
						Name:                "tool",
						Version:             "1.0.0",
						OS:                  os,
						Arch:                arch,
						Format:              format,
						ArchiveNameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
						BinPathTemplate:     "{{name}}",
					}

					archive, binPath, err := ArchiveName(params)
					if err != nil {
						t.Errorf("unexpected error: %v", err)
						return
					}

					// Validate archive ends with correct extension
					expectedExt := ".tar.gz"
					if format == "zip" {
						expectedExt = ".zip"
					}
					if !strings.HasSuffix(archive, expectedExt) {
						t.Errorf("archive %q should end with %q", archive, expectedExt)
					}

					// For Windows, binPath should end with .exe
					if os == "windows" && !strings.HasSuffix(binPath, ".exe") {
						t.Errorf("Windows binPath %q should end with .exe", binPath)
					}
					// For non-Windows, binPath should NOT end with .exe
					if os != "windows" && strings.HasSuffix(binPath, ".exe") {
						t.Errorf("Non-Windows binPath %q should not end with .exe", binPath)
					}
				})
			}
		}
	}
}

func BenchmarkArchiveName(b *testing.B) {
	params := ArchiveNameParams{
		Name:                "mytool",
		Version:             "1.2.3",
		OS:                  "linux",
		Arch:                "amd64",
		Format:              "tar.gz",
		ArchiveNameTemplate: "{{name}}_{{version}}_{{os}}_{{arch}}",
		BinPathTemplate:     "bin/{{name}}",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = ArchiveName(params)
	}
}
