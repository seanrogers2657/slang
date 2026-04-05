package slpackage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// PackageResolver translates import paths to filesystem directories
// within the packages/ directory under the project root.
type PackageResolver struct {
	RootDir     string // project root (entry file's directory)
	PackagesDir string // RootDir + "/packages"
}

// NewResolver creates a PackageResolver for the given project root.
func NewResolver(rootDir string) *PackageResolver {
	return &PackageResolver{
		RootDir:     rootDir,
		PackagesDir: filepath.Join(rootDir, "packages"),
	}
}

// Resolve validates an import path and returns the absolute directory path.
// Returns an error if the path is invalid or the package doesn't exist.
func (r *PackageResolver) Resolve(importPath string) (string, error) {
	// Reject relative/absolute path prefixes
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") || strings.HasPrefix(importPath, "/") {
		return "", fmt.Errorf("import path %q must not start with './', '../', or '/'; use a package-relative path like \"math\" or \"utils/helpers\"", importPath)
	}

	// Reject empty path
	if importPath == "" {
		return "", fmt.Errorf("import path must not be empty")
	}

	// Validate path segments
	segments := strings.Split(importPath, "/")
	for _, seg := range segments {
		if !isValidPathSegment(seg) {
			return "", fmt.Errorf("invalid import path segment %q: must match [a-z][a-z0-9_]*", seg)
		}
	}

	// Reject reserved paths
	if importPath == "main" {
		return "", fmt.Errorf("import path \"main\" is reserved for the root package")
	}
	if importPath == "std" || strings.HasPrefix(importPath, "std/") {
		return "", fmt.Errorf("import path %q is reserved for the future standard library", importPath)
	}

	// Check packages/ directory exists
	info, err := os.Stat(r.PackagesDir)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("no 'packages' directory found; create a 'packages/' directory in the project root to use imports")
	}
	if err != nil {
		return "", fmt.Errorf("error accessing packages directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("'packages' exists but is not a directory")
	}

	// Resolve to absolute path
	pkgDir := filepath.Join(r.PackagesDir, filepath.FromSlash(importPath))

	// Check package directory exists
	pkgInfo, err := os.Stat(pkgDir)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("package %q not found; expected a directory at %s", importPath, pkgDir)
	}
	if err != nil {
		return "", fmt.Errorf("error accessing package %q: %w", importPath, err)
	}
	if !pkgInfo.IsDir() {
		return "", fmt.Errorf("package path %q is not a directory", importPath)
	}

	// Check directory has .sl files
	files, err := DiscoverSlFiles(pkgDir)
	if err != nil {
		return "", fmt.Errorf("error reading package %q: %w", importPath, err)
	}
	if len(files) == 0 {
		return "", fmt.Errorf("package %q has no .sl files", importPath)
	}

	return pkgDir, nil
}

// DiscoverSlFiles finds all .sl files in a directory, sorted alphabetically.
// It does not recurse into subdirectories.
func DiscoverSlFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sl") {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}

// isValidPathSegment checks that a path segment matches [a-z][a-z0-9_]*.
func isValidPathSegment(seg string) bool {
	if len(seg) == 0 {
		return false
	}
	first := rune(seg[0])
	if !unicode.IsLower(first) || !unicode.IsLetter(first) {
		return false
	}
	for _, ch := range seg[1:] {
		if !unicode.IsLower(ch) && !unicode.IsDigit(ch) && ch != '_' {
			return false
		}
	}
	return true
}
