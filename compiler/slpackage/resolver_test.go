package slpackage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolverValidPaths(t *testing.T) {
	// Set up temp project with packages
	rootDir := t.TempDir()
	pkgsDir := filepath.Join(rootDir, "packages")
	os.MkdirAll(filepath.Join(pkgsDir, "math"), 0755)
	os.WriteFile(filepath.Join(pkgsDir, "math", "math.sl"), []byte("add = (a: s64, b: s64) -> s64 { a + b }"), 0644)
	os.MkdirAll(filepath.Join(pkgsDir, "utils", "helpers"), 0755)
	os.WriteFile(filepath.Join(pkgsDir, "utils", "helpers", "helpers.sl"), []byte("// helpers"), 0644)

	r := NewResolver(rootDir)

	tests := []struct {
		name       string
		importPath string
	}{
		{"simple path", "math"},
		{"nested path", "utils/helpers"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := r.Resolve(tt.importPath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dir == "" {
				t.Fatal("expected non-empty directory path")
			}
			// Verify the directory exists
			info, err := os.Stat(dir)
			if err != nil {
				t.Fatalf("resolved dir does not exist: %v", err)
			}
			if !info.IsDir() {
				t.Fatal("resolved path is not a directory")
			}
		})
	}
}

func TestResolverInvalidSegments(t *testing.T) {
	rootDir := t.TempDir()
	pkgsDir := filepath.Join(rootDir, "packages")
	os.MkdirAll(pkgsDir, 0755)

	r := NewResolver(rootDir)

	tests := []struct {
		name       string
		importPath string
		errContain string
	}{
		{"uppercase", "Math", "invalid import path segment"},
		{"hyphen", "my-pkg", "invalid import path segment"},
		{"starts with digit", "3d", "invalid import path segment"},
		{"dot in name", "my.pkg", "invalid import path segment"},
		{"empty segment", "utils//helpers", "invalid import path segment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.Resolve(tt.importPath)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errContain) {
				t.Errorf("expected error containing %q, got: %v", tt.errContain, err)
			}
		})
	}
}

func TestResolverRelativePaths(t *testing.T) {
	rootDir := t.TempDir()
	r := NewResolver(rootDir)

	tests := []struct {
		name       string
		importPath string
	}{
		{"dot-slash", "./math"},
		{"dot-dot-slash", "../math"},
		{"absolute", "/math"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.Resolve(tt.importPath)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "must not start with") {
				t.Errorf("expected path prefix error, got: %v", err)
			}
		})
	}
}

func TestResolverReservedPaths(t *testing.T) {
	rootDir := t.TempDir()
	pkgsDir := filepath.Join(rootDir, "packages")
	os.MkdirAll(pkgsDir, 0755)

	r := NewResolver(rootDir)

	tests := []struct {
		name       string
		importPath string
		errContain string
	}{
		{"main reserved", "main", "reserved for the root package"},
		{"std reserved", "std", "reserved for the future standard library"},
		{"std prefix reserved", "std/math", "reserved for the future standard library"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.Resolve(tt.importPath)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errContain) {
				t.Errorf("expected error containing %q, got: %v", tt.errContain, err)
			}
		})
	}
}

func TestResolverMissingPackagesDir(t *testing.T) {
	rootDir := t.TempDir()
	// No packages/ directory created
	r := NewResolver(rootDir)

	_, err := r.Resolve("math")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no 'packages' directory found") {
		t.Errorf("expected missing packages dir error, got: %v", err)
	}
}

func TestResolverMissingPackage(t *testing.T) {
	rootDir := t.TempDir()
	os.MkdirAll(filepath.Join(rootDir, "packages"), 0755)

	r := NewResolver(rootDir)

	_, err := r.Resolve("math")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found error, got: %v", err)
	}
}

func TestResolverEmptyPackage(t *testing.T) {
	rootDir := t.TempDir()
	os.MkdirAll(filepath.Join(rootDir, "packages", "math"), 0755)
	// No .sl files

	r := NewResolver(rootDir)

	_, err := r.Resolve("math")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "has no .sl files") {
		t.Errorf("expected empty package error, got: %v", err)
	}
}

func TestDiscoverSlFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "beta.sl"), []byte("//b"), 0644)
	os.WriteFile(filepath.Join(dir, "alpha.sl"), []byte("//a"), 0644)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# readme"), 0644)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "subdir", "nested.sl"), []byte("//n"), 0644)

	files, err := DiscoverSlFiles(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}

	// Should be sorted alphabetically
	if filepath.Base(files[0]) != "alpha.sl" {
		t.Errorf("expected first file to be alpha.sl, got %s", filepath.Base(files[0]))
	}
	if filepath.Base(files[1]) != "beta.sl" {
		t.Errorf("expected second file to be beta.sl, got %s", filepath.Base(files[1]))
	}
}

func TestIsValidPathSegment(t *testing.T) {
	tests := []struct {
		seg   string
		valid bool
	}{
		{"math", true},
		{"my_utils", true},
		{"a1", true},
		{"abc_123", true},
		{"Math", false},
		{"3d", false},
		{"my-pkg", false},
		{"", false},
		{"_private", false},
	}

	for _, tt := range tests {
		t.Run(tt.seg, func(t *testing.T) {
			got := isValidPathSegment(tt.seg)
			if got != tt.valid {
				t.Errorf("isValidPathSegment(%q) = %v, want %v", tt.seg, got, tt.valid)
			}
		})
	}
}
